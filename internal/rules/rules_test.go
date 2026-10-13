package rules

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

const (
	iosKey     = "appl_IOSKEY1234567890abcd"
	androidKey = "goog_ANDROIDKEY1234567890"
	amazonKey  = "amzn_AMAZONKEY12345678901"
	secretKey  = "sk_SECRETKEY1234567890abcd"
)

var prefixes = map[string]string{"appl_": "ios", "goog_": "android", "amzn_": "android"}

func loc(file string, line int) model.Location { return model.Location{File: file, Line: line} }

func androidApp(keys ...string) model.Facts {
	f := model.Facts{
		Platform:    "android",
		Root:        "/repo",
		AppID:       "com.example.app",
		AppIDAt:     loc("app/build.gradle.kts", 9),
		KeyPrefixes: []string{"goog_", "amzn_"},
	}
	for i, k := range keys {
		f.APIKeys = append(f.APIKeys, model.APIKey{Value: k, Location: loc("app/App.kt", 10+i)})
	}
	return f
}

func iosApp(keys ...string) model.Facts {
	f := model.Facts{
		Platform:    "ios",
		Root:        "/repo",
		AppID:       "com.example.app",
		AppIDAt:     loc("App.xcodeproj/project.pbxproj", 40),
		KeyPrefixes: []string{"appl_"},
	}
	for i, k := range keys {
		f.APIKeys = append(f.APIKeys, model.APIKey{Value: k, Location: loc("App/App.swift", 5+i)})
	}
	return f
}

func project() *model.Dashboard {
	return &model.Dashboard{
		Entitlements: []string{"premium", "pro"},
		Offerings:    []string{"default", "annual_promo"},
		Apps: []model.App{
			{ID: "app1", Platform: "android", AppID: "com.example.app", PublicKeys: []string{androidKey}},
			{ID: "app2", Platform: "ios", AppID: "com.example.app", PublicKeys: []string{iosKey}},
		},
	}
}

func run(t *testing.T, id string, d *model.Dashboard, facts ...model.Facts) []model.Finding {
	t.Helper()
	r, ok := Get(id)
	if !ok {
		t.Fatalf("rule %q is not registered", id)
	}
	return r.Check(&Context{Facts: facts, Dashboard: d, KeyPrefixes: prefixes})
}

func TestRegistry(t *testing.T) {
	var ids []string
	for _, r := range All() {
		ids = append(ids, r.ID())
		if r.Description() == "" {
			t.Errorf("%s has no description", r.ID())
		}
	}
	if got, want := strings.Join(ids, ","), "api-key,app-id,unknown-entitlement,unknown-offering"; got != want {
		t.Errorf("rules = %s, want %s", got, want)
	}
	if r, _ := Get("api-key"); r.NeedsDashboard() {
		t.Error("api-key must work offline")
	}
}

func TestUnknownEntitlementSuggestsTypoFix(t *testing.T) {
	f := androidApp()
	f.Entitlements = []model.Ref{{ID: "premum", Location: loc("app/Paywall.kt", 3)}, {ID: "pro", Location: loc("app/Paywall.kt", 4)}}

	got := run(t, "unknown-entitlement", project(), f)
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(got), got)
	}
	fd := got[0]
	if fd.Severity != model.Error || fd.Location != loc("app/Paywall.kt", 3) || !strings.Contains(fd.Message, `"premum"`) {
		t.Errorf("finding = %+v", fd)
	}
	if fd.Fix != `Did you mean "premium"?` {
		t.Errorf("Fix = %q", fd.Fix)
	}
	want := model.Edit{
		Path: filepath.Join("/repo", "app", "Paywall.kt"), File: "app/Paywall.kt", Line: 3,
		Old: `"premum"`, New: `"premium"`, Safety: model.Suggested,
		Description: `Change entitlement "premum" to "premium".`,
	}
	if len(fd.Edits) != 1 || fd.Edits[0] != want {
		t.Errorf("Edits = %+v\nwant %+v", fd.Edits, want)
	}
}

func TestUnknownOfferingWithoutCloseMatchListsExisting(t *testing.T) {
	f := iosApp()
	f.Offerings = []model.Ref{{ID: "summer_sale", Location: loc("App/Paywall.swift", 7)}}
	got := run(t, "unknown-offering", project(), f)
	if len(got) != 1 || len(got[0].Edits) != 0 {
		t.Fatalf("want one finding without edits, got %+v", got)
	}
	if want := `Create "summer_sale" in the RevenueCat dashboard, or use one that exists: "default", "annual_promo".`; got[0].Fix != want {
		t.Errorf("Fix = %q\nwant %q", got[0].Fix, want)
	}
}

func TestUnknownIDsEdgeCases(t *testing.T) {
	f := androidApp()
	f.Entitlements = []model.Ref{{ID: "prx", Location: loc("a.kt", 1)}}

	tie := &model.Dashboard{Entitlements: []string{"pre", "pro"}}
	if got := run(t, "unknown-entitlement", tie, f); len(got) != 1 || len(got[0].Edits) != 0 {
		t.Errorf("a tie between two close IDs must not suggest either: %+v", got)
	}
	empty := &model.Dashboard{}
	if got := run(t, "unknown-entitlement", empty, f); len(got) != 1 || !strings.Contains(got[0].Fix, "has no entitlements yet") {
		t.Errorf("empty project: %+v", got)
	}
	exact := &model.Dashboard{Entitlements: []string{"prx"}}
	if got := run(t, "unknown-entitlement", exact, f); len(got) != 0 {
		t.Errorf("known IDs must pass: %+v", got)
	}
}

func TestLevenshtein(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"premum", "premium", 1},
		{"Premium", "premium", 1},
		{"pro", "pro", 0},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
	} {
		if got := levenshtein(tc.a, tc.b); got != tc.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestAPIKey(t *testing.T) {
	strangerKey := "goog_SOMEONEELSE123456789"
	twoAndroidApps := project()
	twoAndroidApps.Apps = append(twoAndroidApps.Apps, model.App{ID: "app3", Platform: "android", AppID: "com.example.amazon", PublicKeys: []string{amazonKey}})
	noKeys := &model.Dashboard{Apps: []model.App{{ID: "app1", Platform: "android", AppID: "com.example.app"}}}

	for _, tc := range []struct {
		name     string
		facts    model.Facts
		dash     *model.Dashboard
		severity model.Severity // "" means no finding
		safety   model.Safety   // "" means no edit
		newKey   string
	}{
		{"secret key offline", iosApp(secretKey), nil, model.Error, "", ""},
		// The replacement can't be proven to be the same project, so it is only suggested.
		{"secret key with project", iosApp(secretKey), project(), model.Error, model.Suggested, iosKey},
		{"wrong platform offline", androidApp(iosKey), nil, model.Error, "", ""},
		// The iOS key is this project's, so the project's Android key is certainly right.
		{"wrong platform, same project", androidApp(iosKey), project(), model.Error, model.Safe, androidKey},
		{"wrong platform, other project", androidApp("appl_OTHERPROJECT123456789"), project(), model.Error, model.Suggested, androidKey},
		{"right key", androidApp(androidKey), project(), "", "", ""},
		{"right platform, other project", androidApp(strangerKey), project(), model.Warning, model.Suggested, androidKey},
		{"right platform, keys not listed", androidApp(strangerKey), noKeys, "", "", ""},
		{"prefix no platform claims", androidApp("rcb_WEBBILLINGKEY123456789"), project(), "", "", ""},
		// The app ID picks the Play Store app out of two Android apps.
		{"app id picks the app", androidApp(iosKey), twoAndroidApps, model.Error, model.Safe, androidKey},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, "api-key", tc.dash, tc.facts)
			if tc.severity == "" {
				if len(got) != 0 {
					t.Fatalf("want no finding, got %+v", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("got %d findings, want 1: %+v", len(got), got)
			}
			fd := got[0]
			if fd.Severity != tc.severity {
				t.Errorf("severity = %s, want %s", fd.Severity, tc.severity)
			}
			if fd.Fixable() != tc.safety {
				t.Errorf("fixable = %q, want %q", fd.Fixable(), tc.safety)
			}
			if tc.safety != "" && (fd.Edits[0].New != tc.newKey || fd.Edits[0].Old != tc.facts.APIKeys[0].Value) {
				t.Errorf("edit = %+v, want %s -> %s", fd.Edits[0], tc.facts.APIKeys[0].Value, tc.newKey)
			}
			for _, k := range []string{iosKey, androidKey, secretKey, strangerKey} {
				if strings.Contains(fd.Message+fd.Fix, k) {
					t.Errorf("full key %s leaked into %q / %q", k, fd.Message, fd.Fix)
				}
			}
		})
	}
}

func TestAPIKeyAmbiguousReplacement(t *testing.T) {
	d := project()
	// Two Android apps and the code's app ID matches neither: no single right key.
	d.Apps = append(d.Apps, model.App{ID: "app3", Platform: "android", AppID: "com.example.amazon", PublicKeys: []string{amazonKey}})
	f := androidApp(iosKey)
	f.AppID = "com.example.unknown"
	got := run(t, "api-key", d, f)
	if len(got) != 1 || len(got[0].Edits) != 0 {
		t.Fatalf("want one finding without an edit, got %+v", got)
	}
	if !strings.Contains(got[0].Fix, "rc apps keys") {
		t.Errorf("Fix should explain where to find the key: %q", got[0].Fix)
	}
}

func TestSecretKeyFixMentionsRotation(t *testing.T) {
	got := run(t, "api-key", project(), iosApp(secretKey))
	if len(got) != 1 || !strings.Contains(got[0].Fix, "rotate") || !strings.Contains(got[0].Edits[0].Description, "rotate") {
		t.Fatalf("the fix and the edit must both say to rotate the secret key: %+v", got)
	}
}

func TestAppID(t *testing.T) {
	other := androidApp()
	other.AppID = "com.example.other"
	noAppID := androidApp()
	noAppID.AppID = ""
	iosOnly := &model.Dashboard{Apps: []model.App{{ID: "a", Platform: "ios", AppID: "com.example.app"}}}

	for _, tc := range []struct {
		name  string
		facts model.Facts
		dash  *model.Dashboard
		want  string // substring of the message, "" for no finding
	}{
		{"matches", androidApp(), project(), ""},
		{"mismatch", other, project(), `"com.example.other" doesn't match any android app`},
		{"platform missing from project", androidApp(), iosOnly, "has no android app"},
		{"apps not listed", androidApp(), &model.Dashboard{}, ""},
		{"app id unknown", noAppID, project(), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, "app-id", tc.dash, tc.facts)
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("want no finding, got %+v", got)
				}
				return
			}
			if len(got) != 1 || got[0].Severity != model.Warning || !strings.Contains(got[0].Message, tc.want) {
				t.Fatalf("got %+v, want one warning containing %q", got, tc.want)
			}
		})
	}
}

func TestAPIKeyPlaceholder(t *testing.T) {
	for _, placeholder := range []string{"REVENUECAT_API_KEY", "googl_api_key", "appl_short"} {
		got := run(t, "api-key", nil, iosApp(placeholder))
		if len(got) != 1 || got[0].Severity != model.Error || !strings.Contains(got[0].Message, "placeholder") {
			t.Errorf("%s: %+v", placeholder, got)
		}
	}
}
