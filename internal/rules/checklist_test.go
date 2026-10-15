package rules

import (
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func withSignals(f model.Facts, names ...string) model.Facts {
	f.Signals = map[string][]model.Location{}
	for i, n := range names {
		f.Signals[n] = []model.Location{loc("app/Billing.kt", 20+i)}
	}
	return f
}

func withBuild(f model.Facts, kv ...string) model.Facts {
	f.Build = map[string]model.Setting{}
	for i := 0; i+1 < len(kv); i += 2 {
		f.Build[kv[i]] = model.Setting{Value: kv[i+1], Location: loc("app/build.gradle", 10+i)}
	}
	return f
}

func configured(f model.Facts) model.Facts {
	f.ConfigureAt = []model.Location{loc("app/App.kt", 12)}
	return f
}

func TestHardcodedAppUserID(t *testing.T) {
	f := androidApp()
	f.AppUserIDs = []model.Ref{
		{ID: "demo_user", Location: loc("app/App.kt", 14)},
		{ID: "", Location: loc("app/App.kt", 15)},
		{ID: "qa_account", Location: loc("app/Login.kt", 3), Def: &model.Definition{Name: "QA_USER", Location: loc("app/Consts.kt", 2), Text: `"qa_account"`}},
	}
	got := run(t, "hardcoded-app-user-id", nil, f)
	if len(got) != 3 {
		t.Fatalf("got %+v", got)
	}
	for i, want := range []string{
		`The fixed app user ID "demo_user" is passed to the SDK.`,
		"An empty app user ID is passed to the SDK, which rejects it with INVALID_APP_USER_ID.",
		`"qa_account" (QA_USER, set at app/Consts.kt:2)`,
	} {
		if got[i].Severity != model.Error || !strings.Contains(got[i].Message, want) {
			t.Errorf("finding %d = %q, want it to contain %q", i, got[i].Message, want)
		}
	}
}

func TestRestorePurchases(t *testing.T) {
	for _, tc := range []struct {
		name  string
		facts model.Facts
		want  string // "" for no finding
	}{
		{"android without restore", configured(androidApp()), "Users who reinstall"},
		{"ios without restore", configured(iosApp()), "guideline 3.1.1"},
		{"restore call", withSignals(configured(iosApp()), model.SignalRestore), ""},
		{"RevenueCat paywall", withSignals(configured(androidApp()), model.SignalPaywallUI), ""},
		{"Customer Center", withSignals(configured(iosApp()), model.SignalCustomerCenter), ""},
		{"no configure call (a library, not the app)", androidApp(), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, "restore-purchases", nil, tc.facts)
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("want no finding, got %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Severity != model.Warning || !strings.Contains(got[0].Message, tc.want) || got[0].Location != loc("app/App.kt", 12) {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestPlayBillingDesugaring(t *testing.T) {
	for _, tc := range []struct {
		name  string
		facts model.Facts
		fires bool
	}{
		{"minSdk 21, current SDK", withBuild(androidApp(), model.BuildMinSDK, "21", model.BuildSDKArtifact, "purchases", model.BuildSDKVersion, "9.2.0"), true},
		{"minSdk 23, SDK version unknown", withBuild(androidApp(), model.BuildMinSDK, "23"), true},
		{"first SDK on Billing 7.1", withBuild(androidApp(), model.BuildMinSDK, "21", model.BuildSDKArtifact, "purchases", model.BuildSDKVersion, "8.10.8"), true},
		{"SDK before Billing 7.1", withBuild(androidApp(), model.BuildMinSDK, "21", model.BuildSDKArtifact, "purchases", model.BuildSDKVersion, "8.10.7"), false},
		{"KMP SDK versions don't map: assume current", withBuild(androidApp(), model.BuildMinSDK, "21", model.BuildSDKArtifact, "purchases-kmp-core", model.BuildSDKVersion, "2.1.0+16.2.0"), true},
		{"desugaring enabled", withBuild(androidApp(), model.BuildMinSDK, "21", model.BuildDesugaring, "true"), false},
		{"minSdk 24", withBuild(androidApp(), model.BuildMinSDK, "24"), false},
		{"minSdk unknown", androidApp(), false},
		{"ios", withBuild(iosApp(), model.BuildMinSDK, "21"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, "play-billing-desugaring", nil, tc.facts)
			if fired := len(got) == 1; fired != tc.fires {
				t.Fatalf("fired = %v, want %v: %+v", fired, tc.fires, got)
			}
			if tc.fires && (got[0].Severity != model.Warning || !strings.Contains(got[0].Fix, "raise minSdk to 24")) {
				t.Errorf("finding = %+v", got[0])
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	for _, tc := range []struct {
		v    string
		less bool
		ok   bool
	}{
		{"8.10.7", true, true},
		{"8.10.8", false, true},
		{"8.11", false, true},
		{"8.9.99", true, true},
		{"9.0.0-beta.1", false, true},
		{"$revenuecatVersion", false, false},
	} {
		v, ok := parseVersion(tc.v)
		if ok != tc.ok || (ok && versionLess(v, desugaringSince) != tc.less) {
			t.Errorf("%s: parsed %v (%v), less = %v", tc.v, v, ok, ok && versionLess(v, desugaringSince))
		}
	}
}

func TestAndroidApplicationClass(t *testing.T) {
	app := func(kv ...string) model.Facts {
		return withBuild(androidApp(), append([]string{
			model.BuildApplicationClass, "com.example.app.App",
			model.BuildManifest, "app/src/main/AndroidManifest.xml",
		}, kv...)...)
	}
	for _, tc := range []struct {
		name     string
		facts    model.Facts
		severity model.Severity // "" for no finding
	}{
		{"registered by full name", app(model.BuildManifestAppName, "com.example.app.App"), ""},
		{"registered relatively", app(model.BuildManifestAppName, ".App"), ""},
		{"not registered", app(), model.Error},
		{"another class registered", app(model.BuildManifestAppName, "com.example.app.DebugApp"), model.Warning},
		{"no manifest in the repo", withBuild(androidApp(), model.BuildApplicationClass, "com.example.app.App"), ""},
		{"configure isn't in an Application subclass", withBuild(androidApp(), model.BuildManifest, "app/src/main/AndroidManifest.xml"), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, "android-application-class", nil, tc.facts)
			if tc.severity == "" {
				if len(got) != 0 {
					t.Fatalf("want no finding, got %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Severity != tc.severity {
				t.Fatalf("got %+v", got)
			}
		})
	}

	// The manifest setting points at the <application line, where the edit goes.
	unregistered := app()
	unregistered.Build[model.BuildManifest] = model.Setting{Value: "app/src/main/AndroidManifest.xml", Location: loc("app/src/main/AndroidManifest.xml", 5)}
	got := run(t, "android-application-class", nil, unregistered)
	want := model.Edit{
		Path: "/repo/app/src/main/AndroidManifest.xml", File: "app/src/main/AndroidManifest.xml", Line: 5,
		Old: "<application", New: `<application android:name="com.example.app.App"`,
		Safety: model.Suggested, Description: "Register App in app/src/main/AndroidManifest.xml.",
	}
	if len(got[0].Edits) != 1 || got[0].Edits[0] != want {
		t.Errorf("edit = %+v\nwant %+v", got[0].Edits, want)
	}
}

func TestPurchasesCompletedByApp(t *testing.T) {
	for _, tc := range []struct {
		name     string
		facts    model.Facts
		severity model.Severity
	}{
		{"android without acknowledge", withSignals(androidApp(), model.SignalCompletedByMyApp), model.Error},
		{"ios without finish", withSignals(iosApp(), model.SignalCompletedByMyApp), model.Warning},
		{"android acknowledges", withSignals(androidApp(), model.SignalCompletedByMyApp, model.SignalFinishesPurchase), ""},
		{"RevenueCat completes purchases", androidApp(), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, "purchases-completed-by-app", nil, tc.facts)
			if tc.severity == "" {
				if len(got) != 0 {
					t.Fatalf("want no finding, got %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Severity != tc.severity {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func catalogProject(c *model.Catalog) *model.Dashboard {
	d := project()
	d.Offerings = []string{"default", "seasonal"}
	d.Catalog = c
	return d
}

func TestCatalogRules(t *testing.T) {
	healthy := &model.Catalog{
		CurrentOffering:     "default",
		Packages:            map[string][]model.Package{"default": {{ID: "$rc_monthly", Products: []string{"pro_monthly"}}}, "seasonal": {{ID: "$rc_annual", Products: []string{"pro_annual"}}}},
		EntitlementProducts: map[string][]string{"premium": {"pro_monthly"}, "pro": {"pro_monthly"}},
	}
	broken := &model.Catalog{
		CurrentOffering:     "",
		Packages:            map[string][]model.Package{"default": {{ID: "$rc_monthly", Products: []string{}}}, "seasonal": {}},
		EntitlementProducts: map[string][]string{"premium": {}},
	}
	code := withSignals(androidApp(), model.SignalCurrentOffering)
	code.Entitlements = []model.Ref{{ID: "premium", Location: loc("app/Gate.kt", 4)}}
	code.Offerings = []model.Ref{{ID: "seasonal", Location: loc("app/Paywall.kt", 8)}, {ID: "default", Location: loc("app/Paywall.kt", 9)}}

	for _, id := range []string{"current-offering", "empty-offering", "entitlement-products"} {
		if got := run(t, id, catalogProject(healthy), code); len(got) != 0 {
			t.Errorf("%s on a healthy catalog: %+v", id, got)
		}
		if got := run(t, id, catalogProject(nil), code); len(got) != 0 {
			t.Errorf("%s without a catalog must stay silent: %+v", id, got)
		}
	}

	if got := run(t, "current-offering", catalogProject(broken), code); len(got) != 1 || !strings.Contains(got[0].Message, "no offering in your RevenueCat project is marked current") {
		t.Errorf("current-offering: %+v", got)
	}
	got := run(t, "empty-offering", catalogProject(broken), code)
	if len(got) != 2 || !strings.Contains(got[0].Message, `Offering "seasonal" has no packages`) || !strings.Contains(got[1].Message, `Package "$rc_monthly" in offering "default" has no product`) {
		t.Errorf("empty-offering: %+v", got)
	}
	if got := run(t, "entitlement-products", catalogProject(broken), code); len(got) != 1 || got[0].Location != loc("app/Gate.kt", 4) {
		t.Errorf("entitlement-products: %+v", got)
	}

	// The current offering counts as used when the code reads it.
	currentEmpty := &model.Catalog{CurrentOffering: "seasonal", Packages: map[string][]model.Package{"seasonal": {}}}
	reader := withSignals(androidApp(), model.SignalCurrentOffering)
	if got := run(t, "empty-offering", catalogProject(currentEmpty), reader); len(got) != 1 || got[0].Location != loc("app/Billing.kt", 20) {
		t.Errorf("empty current offering: %+v", got)
	}
	// An ID the project doesn't have is unknown-offering's job, not this rule's.
	ghost := androidApp()
	ghost.Offerings = []model.Ref{{ID: "ghost", Location: loc("a.kt", 1)}}
	if got := run(t, "empty-offering", catalogProject(&model.Catalog{Packages: map[string][]model.Package{"ghost": {}}}), ghost); len(got) != 0 {
		t.Errorf("unknown offering reported twice: %+v", got)
	}
}

func TestTestStoreKey(t *testing.T) {
	mark := func(f model.Facts, debugOnly, configured bool) model.Facts {
		for i := range f.APIKeys {
			if strings.HasPrefix(f.APIKeys[i].Value, "test_") {
				f.APIKeys[i].DebugOnly, f.APIKeys[i].Configured = debugOnly, configured
			}
		}
		return f
	}
	debugOnly := func(f model.Facts) model.Facts { return mark(f, true, true) }
	used := func(f model.Facts) model.Facts { return mark(f, false, true) }
	for _, tc := range []struct {
		name  string
		facts model.Facts
		want  string // "" for no finding
	}{
		{"only a Test Store key", used(androidApp(testKey)), "is the only RevenueCat key rc-doctor found for this app"},
		{"next to a store key, no recognized guard", used(iosApp(testKey, iosKey)), "isn't behind a debug-only check"},
		{"behind a debug-only guard", debugOnly(iosApp(testKey, iosKey)), ""},
		{"debug-only and the only key", debugOnly(androidApp(testKey)), ""},
		{"release builds allowed to run with it", used(withSignals(androidApp(testKey, androidKey), model.SignalTestStoreRelease)), "forceAllowTestStoreInReleaseBuilds lets release builds run with it"},
		{"defined, not followed to configure", androidApp(testKey), "is defined here, and rc-doctor can't tell which builds use it"},
		{"no Test Store key", androidApp(androidKey), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, "test-store-key", nil, tc.facts)
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("want no finding, got %+v", got)
				}
				return
			}
			severity := model.Warning
			if strings.Contains(tc.want, "is defined here") {
				severity = model.Info
			}
			if len(got) != 1 || got[0].Severity != severity || !strings.Contains(got[0].Message, tc.want) {
				t.Fatalf("got %+v", got)
			}
			if strings.Contains(got[0].Message+got[0].Fix, testKey) {
				t.Errorf("the key must be masked: %q", got[0].Message)
			}
		})
	}
}

// A Test Store key belongs to no platform, so api-key leaves it alone.
func TestAPIKeyIgnoresTestStoreKeys(t *testing.T) {
	if got := run(t, "api-key", project(), androidApp(testKey), iosApp(testKey)); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestRestorePurchasesWithProductTypes(t *testing.T) {
	withTypes := func(types map[string]string) *model.Dashboard {
		return catalogProject(&model.Catalog{ProductTypes: types})
	}
	app := configured(androidApp())
	if got := run(t, "restore-purchases", withTypes(map[string]string{"tip_small": "consumable", "tip_large": "consumable"}), app); len(got) != 0 {
		t.Errorf("a project that only sells consumables has nothing to restore: %+v", got)
	}
	got := run(t, "restore-purchases", withTypes(map[string]string{"tip_small": "consumable", "pro_monthly": "subscription"}), app)
	if len(got) != 1 || strings.Contains(got[0].Message, "consumables") {
		t.Errorf("with a subscription the warning stands, without the consumables caveat: %+v", got)
	}
	if got := run(t, "restore-purchases", nil, app); len(got) != 1 || !strings.Contains(got[0].Message, "Apps that only sell consumables, such as tips, don't need one.") {
		t.Errorf("offline, the message says when the warning doesn't apply: %+v", got)
	}
}
