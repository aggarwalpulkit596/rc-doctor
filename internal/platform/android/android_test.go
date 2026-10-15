package android

import (
	"reflect"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func TestDetect(t *testing.T) {
	if !(Android{}).Detect(testutil.Testdata(t, "sample")) {
		t.Error("sample has an Android app using RevenueCat")
	}
	if (Android{}).Detect(testutil.Testdata(t, "sample", "ios")) {
		t.Error("the iOS app alone has no Gradle build")
	}
	catalog := testutil.WriteFiles(t, map[string]string{
		"gradle/libs.versions.toml": "[libraries]\nrevenuecat = { module = \"com.revenuecat.purchases:purchases\", version = \"8.10.0\" }\n",
	})
	if !(Android{}).Detect(catalog) {
		t.Error("a version catalog declaring the SDK should be detected")
	}
}

func TestCollectSample(t *testing.T) {
	facts, err := Android{}.Collect(testutil.Testdata(t, "sample"))
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 {
		t.Fatalf("got %d facts, want 1", len(facts))
	}
	f := facts[0]
	const dir = "android/app/src/main/java/com/example/paywall/"
	at := func(file string, line int) model.Location { return model.Location{File: file, Line: line} }

	if f.Platform != "android" || f.AppID != "com.example.paywall" || f.AppIDAt != at("android/app/build.gradle.kts", 11) {
		t.Errorf("app = %s %q at %v", f.Platform, f.AppID, f.AppIDAt)
	}
	if want := []model.APIKey{{Value: "appl_AbCdEfGhIjKlMnOpQrSt", Location: at(dir+"App.kt", 11), Configured: true}}; !reflect.DeepEqual(f.APIKeys, want) {
		t.Errorf("APIKeys = %+v, want %+v", f.APIKeys, want)
	}
	// The test source set and the commented-out block must not show up.
	if want := []model.Ref{{ID: "premum", Location: at(dir+"Paywall.kt", 12)}}; !reflect.DeepEqual(f.Entitlements, want) {
		t.Errorf("Entitlements = %+v, want %+v", f.Entitlements, want)
	}
	if want := []model.Ref{{ID: "default", Location: at(dir+"Paywall.kt", 15)}}; !reflect.DeepEqual(f.Offerings, want) {
		t.Errorf("Offerings = %+v, want %+v", f.Offerings, want)
	}
	if want := []model.Location{at(dir+"App.kt", 11)}; !reflect.DeepEqual(f.ConfigureAt, want) {
		t.Errorf("ConfigureAt = %+v, want %+v", f.ConfigureAt, want)
	}
}

func TestCollectBuildTimeKeysAndMultiplatformSourceSets(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"app/build.gradle": "android { defaultConfig { applicationId 'com.example.groovy' } }\n" +
			"dependencies { implementation 'com.revenuecat.purchases:purchases:8.10.0' }\n" +
			"android { defaultConfig { buildConfigField \"String\", \"RC_KEY\", \"\\\"goog_1234567890abcdefghij\\\"\" } }\n",
		"app/src/main/res/values/keys.xml":          `<resources><string name="rc_key">amzn_1234567890abcdefghij</string></resources>`,
		"shared/src/iosMain/kotlin/Keys.kt":         `val key = "appl_1234567890abcdefghij"`,
		"shared/src/iosArm64Main/kotlin/Keys.kt":    `val key = "appl_ABCDEFGHIJ1234567890"`,
		"shared/src/commonMain/kotlin/Paywall.kt":   `val pro = customerInfo.entitlements.active.containsKey("pro")`,
		"shared/src/commonTest/kotlin/PaywallT.kt":  `val t = customerInfo.entitlements["test_only"]`,
		"app/src/androidTestDebug/java/UiTest.java": `boolean t = info.getEntitlements().get("ui_test");`,
	})
	if !(Android{}).Detect(root) {
		t.Fatal("Groovy build file declares the SDK")
	}
	facts, err := Android{}.Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	f := facts[0]
	if f.AppID != "com.example.groovy" {
		t.Errorf("AppID = %q, want com.example.groovy", f.AppID)
	}
	var keys []string
	for _, k := range f.APIKeys {
		keys = append(keys, k.Value+"@"+k.Location.String())
	}
	want := []string{
		"goog_1234567890abcdefghij@app/build.gradle:3",
		"amzn_1234567890abcdefghij@app/src/main/res/values/keys.xml:1",
	}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("keys = %q, want %q (Apple-only source sets must be skipped)", keys, want)
	}
	if len(f.Entitlements) != 1 || f.Entitlements[0].ID != "pro" {
		t.Errorf("Entitlements = %+v, want only \"pro\" (test source sets must be skipped)", f.Entitlements)
	}
}
