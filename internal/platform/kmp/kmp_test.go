package kmp

import (
	"reflect"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func TestDetect(t *testing.T) {
	if !(KMP{}).Detect(testutil.Testdata(t, "kmp")) {
		t.Error("the KMP fixture declares purchases-kmp")
	}
	if (KMP{}).Detect(testutil.Testdata(t, "sample")) {
		t.Error("native apps are not KMP apps")
	}
}

func TestCollectSplitsAppsAndRoutesKeys(t *testing.T) {
	facts, err := KMP{}.Collect(testutil.Testdata(t, "kmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 2 || facts[0].Platform != "android" || facts[1].Platform != "ios" {
		t.Fatalf("facts = %+v", facts)
	}
	a, i := facts[0], facts[1]
	if a.AppID != "com.example.kmp" || i.AppID != "com.example.kmp.ios" {
		t.Errorf("app ids = %q, %q", a.AppID, i.AppID)
	}

	// Both keys sit in one properties file; each goes to the app its name points at.
	keys := func(f model.Facts) (out []string) {
		for _, k := range f.APIKeys {
			out = append(out, k.Value)
		}
		return out
	}
	if got := keys(a); !reflect.DeepEqual(got, []string{"goog_KmpAndroidKey12345678"}) {
		t.Errorf("android keys = %q", got)
	}
	if got := keys(i); !reflect.DeepEqual(got, []string{"appl_KmpIosKey123456789012"}) {
		t.Errorf("ios keys = %q", got)
	}

	// commonMain uses a constant that the Gradle script defaults to "premum";
	// both apps see it, and the test source set is ignored.
	want := []model.Ref{{
		ID:       "premum",
		Location: model.Location{File: "composeApp/src/commonMain/kotlin/com/example/kmp/Billing.kt", Line: 14},
		Def: &model.Definition{
			Name:     "RC_ENTITLEMENT",
			Location: model.Location{File: "composeApp/build.gradle.kts", Line: 17},
			Text:     `"premum"`,
		},
	}}
	if !reflect.DeepEqual(a.Entitlements, want) || !reflect.DeepEqual(i.Entitlements, want) {
		t.Errorf("entitlements = %+v / %+v", a.Entitlements, i.Entitlements)
	}
	if len(a.ConfigureAt) != 1 {
		t.Errorf("ConfigureAt = %+v", a.ConfigureAt)
	}
}

func TestSharedKeyOwner(t *testing.T) {
	for _, tc := range []struct{ hint, key, want string }{
		{"ios", "goog_x123", "ios"}, // the name wins: a wrong key in the iOS slot stays the iOS app's
		{"", "appl_x123", "ios"},
		{"", "amzn_x123", "android"},
		{"", "sk_x123", ""},
	} {
		if got := owner(tc.hint, tc.key); got != tc.want {
			t.Errorf("owner(%q, %q) = %q, want %q", tc.hint, tc.key, got, tc.want)
		}
	}
}

func TestCollectAndroidModulesAndDeadBranches(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"shared/build.gradle.kts": `implementation("com.revenuecat.purchases:purchases-kmp-core:2.1.0+16.2.0")`,
		// The Android app module is a plain Android module.
		"androidApp/src/main/kotlin/App.kt": "Purchases.configure(PurchasesConfiguration.Builder(\"goog_AndroidModuleKey12345\").build())\n",
		// A flavor's source set.
		"shared/src/androidFull/kotlin/Billing.kt": "val entitlement = customerInfo.entitlements[\"pro\"]\n",
		// Copied from common code: in iosMain the Android branch never runs.
		"shared/src/iosMain/kotlin/Billing.ios.kt": "fun key() =\n  if (platform is Platform.Android) {\n    \"goog_DeadBranchKey1234567\"\n  } else {\n    \"appl_IosBranchKey12345678\"\n  }\n",
		// Not Android: a JVM test set.
		"shared/src/androidUnitTest/kotlin/BillingTest.kt": "val entitlement = customerInfo.entitlements[\"test_only\"]\n",
	})
	facts, err := KMP{}.Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	a, i := facts[0], facts[1]
	if len(a.ConfigureAt) != 1 || a.ConfigureAt[0].File != "androidApp/src/main/kotlin/App.kt" {
		t.Errorf("android configure = %+v", a.ConfigureAt)
	}
	if len(a.Entitlements) != 1 || a.Entitlements[0].ID != "pro" {
		t.Errorf("android entitlements = %+v", a.Entitlements)
	}
	var iosKeys []string
	for _, k := range i.APIKeys {
		iosKeys = append(iosKeys, k.Value)
	}
	if !reflect.DeepEqual(iosKeys, []string{"appl_IosBranchKey12345678"}) {
		t.Errorf("ios keys = %v, want the iOS branch only", iosKeys)
	}
}

func TestSourceSetSides(t *testing.T) {
	for set, want := range map[string]side{
		"main": androidSide, "androidMain": androidSide, "androidFull": androidSide, "androidDebug": androidSide,
		"iosMain": appleSide, "iosArm64Main": appleSide, "appleMain": appleSide, "nativeMain": appleSide, "macosMain": appleSide,
		"commonMain": sharedSide, "mobileMain": sharedSide, "nonWebMain": sharedSide,
		"jvmMain": "", "desktopMain": "", "wasmJsMain": "", "jsMain": "", "commonTest": "", "billing": "",
	} {
		if got := sideOf(set); got != want {
			t.Errorf("sideOf(%q) = %q, want %q", set, got, want)
		}
	}
}

func TestCollectSkipsJVMModulesAndReadsIntermediateSets(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"shared/build.gradle.kts": "plugins { kotlin(\"multiplatform\") }\nimplementation(\"com.revenuecat.purchases:purchases-kmp-core:2.1.0+16.2.0\")\n",
		// mobileMain is shared by the Android and iOS targets.
		"shared/src/mobileMain/kotlin/Billing.kt": "Purchases.configure(apiKey = \"appl_MobileMainKey123456789\") { }\n",
		// A Ktor server: its secret key is where a secret key belongs.
		"server/build.gradle.kts":              "plugins {\n    kotlin(\"jvm\")\n    application\n}\n",
		"server/src/main/kotlin/RevenueCat.kt": "val secret = \"sk_ServerSideSecretKey12345\"\n",
	})
	facts, err := KMP{}.Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	a, i := facts[0], facts[1]
	if len(a.ConfigureAt) != 1 || len(i.ConfigureAt) != 1 {
		t.Errorf("mobileMain configures both apps: %v, %v", a.ConfigureAt, i.ConfigureAt)
	}
	for _, f := range facts {
		for _, k := range f.APIKeys {
			if strings.HasPrefix(k.Value, "sk_") {
				t.Errorf("%s: the server's key was read as app code: %+v", f.Platform, k)
			}
		}
	}
}
