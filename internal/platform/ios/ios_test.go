package ios

import (
	"reflect"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func TestDetect(t *testing.T) {
	if !(IOS{}).Detect(testutil.Testdata(t, "sample")) {
		t.Error("sample has an iOS app using RevenueCat")
	}
	if (IOS{}).Detect(testutil.Testdata(t, "sample", "android")) {
		t.Error("the Android app alone has no Apple project")
	}
}

func TestCollectSample(t *testing.T) {
	facts, err := IOS{}.Collect(testutil.Testdata(t, "sample"))
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 1 {
		t.Fatalf("got %d facts, want 1", len(facts))
	}
	f := facts[0]
	at := func(file string, line int) model.Location { return model.Location{File: file, Line: line} }

	// The test target (com.example.paywallTests) and the widget extension
	// (com.example.paywall.widget) must lose to the app itself.
	if f.AppID != "com.example.paywall" || f.AppIDAt != at("ios/Paywall.xcodeproj/project.pbxproj", 27) {
		t.Errorf("AppID = %q at %v", f.AppID, f.AppIDAt)
	}
	if want := []model.APIKey{{Value: "sk_AbCdEfGhIjKlMnOpQrSt", Location: at("ios/Paywall/PaywallApp.swift", 8)}}; !reflect.DeepEqual(f.APIKeys, want) {
		t.Errorf("APIKeys = %+v, want %+v", f.APIKeys, want)
	}
	if want := []model.Ref{{ID: "premium", Location: at("ios/Paywall/ContentView.swift", 11)}}; !reflect.DeepEqual(f.Entitlements, want) {
		t.Errorf("Entitlements = %+v, want %+v (PaywallTests must be skipped)", f.Entitlements, want)
	}
	if want := []model.Ref{{ID: "summer_sale", Location: at("ios/Paywall/ContentView.swift", 15)}}; !reflect.DeepEqual(f.Offerings, want) {
		t.Errorf("Offerings = %+v, want %+v", f.Offerings, want)
	}
	if want := []model.Location{at("ios/Paywall/PaywallApp.swift", 8)}; !reflect.DeepEqual(f.ConfigureAt, want) {
		t.Errorf("ConfigureAt = %+v, want %+v", f.ConfigureAt, want)
	}
}

func TestCollectObjectiveCAndConfigFiles(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"Podfile": "pod 'RevenueCat'\n",
		"App/AppDelegate.m": "[RCPurchases configureWithAPIKey:@\"appl_1234567890abcdefghij\"];\n" +
			"BOOL pro = customerInfo.entitlements[@\"pro\"].isActive;\n",
		"App/Info.plist":          "<key>RCKey</key>\n<string>goog_1234567890abcdefghij</string>\n",
		"App/Paywall.swift":       `.presentPaywallIfNeeded(requiredEntitlementIdentifier: "pro_plus")`,
		"Config/Release.xcconfig": "REVENUECAT_API_KEY = appl_ABCDEFGHIJ1234567890\n",
		"App.xcodeproj/project.pbxproj": "PRODUCT_BUNDLE_IDENTIFIER = \"$(BUNDLE_PREFIX).app\";\n" +
			"PRODUCT_BUNDLE_IDENTIFIER = com.example.objc;\n",
	})
	if !(IOS{}).Detect(root) {
		t.Fatal("a Podfile with RevenueCat should be detected")
	}
	facts, err := IOS{}.Collect(root)
	if err != nil {
		t.Fatal(err)
	}
	f := facts[0]
	if f.AppID != "com.example.objc" {
		t.Errorf("AppID = %q, want com.example.objc (unresolved build settings are skipped)", f.AppID)
	}
	var keys []string
	for _, k := range f.APIKeys {
		keys = append(keys, k.Value+"@"+k.Location.String())
	}
	want := []string{
		"appl_1234567890abcdefghij@App/AppDelegate.m:1",
		"goog_1234567890abcdefghij@App/Info.plist:2",
		"appl_ABCDEFGHIJ1234567890@Config/Release.xcconfig:1",
	}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("keys = %q, want %q", keys, want)
	}
	var ents []string
	for _, e := range f.Entitlements {
		ents = append(ents, e.ID)
	}
	if !reflect.DeepEqual(ents, []string{"pro", "pro_plus"}) {
		t.Errorf("entitlements = %q, want [pro pro_plus]", ents)
	}
	if len(f.ConfigureAt) != 1 || f.ConfigureAt[0].File != "App/AppDelegate.m" {
		t.Errorf("ConfigureAt = %+v, want the Objective-C call", f.ConfigureAt)
	}
}
