package scan

import (
	"regexp"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func TestPlatformHint(t *testing.T) {
	for _, tc := range []struct{ line, want string }{
		{"REVENUECAT_IOS_KEY=", "ios"},
		{"REVENUECAT_ANDROID_KEY=", "android"},
		{`const val revenueCatIosKey = ""`, "ios"},
		{`val playStoreKey = ""`, "android"},
		{"<key>RCAppleKey</key>", "ios"},
		{"RC_KEY=", ""},
		{"IOS_AND_ANDROID_KEY=", ""}, // names both: no hint
		{"RATIOS_BIOS=", ""},         // not words
		{"let iOSKey =", "ios"},      // Apple's spelling survives the camelCase split
		{"#if os(macOS)", "ios"},
		{"if (platform is Platform.Android) {", "android"},
	} {
		if got := PlatformHint(tc.line); got != tc.want {
			t.Errorf("PlatformHint(%q) = %q, want %q", tc.line, got, tc.want)
		}
	}
}

func TestConstants(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"app/build.gradle.kts": `val e = props.getProperty("ENTITLEMENT", "from_gradle")`,
		"secrets.properties":   "ENTITLEMENT=from_properties\nEMPTY=\n",
		"app/src/Config.kt":    "object Config {\n    const val OFFERING = \"main\"\n    const val TWICE = \"a\"\n}\n",
		"app/src/Other.kt":     `const val TWICE = "b"`,
		"ios/Config.swift":     `static let swiftOffering = "ios_main"`,
		"app/src/test/Fake.kt": `const val OFFERING = "test_only"`,
	})
	consts := Constants(root, regexp.MustCompile(`(^|/)test(/|$)`))

	for _, tc := range []struct{ ident, want, file string }{
		{"Config.ENTITLEMENT", "from_properties", "secrets.properties"}, // properties beat the Gradle default
		{"OFFERING", "main", "app/src/Config.kt"},                       // the test source set is excluded
		{"swiftOffering", "ios_main", "ios/Config.swift"},
	} {
		c, ok := consts.Lookup(tc.ident)
		if !ok || c.Value != tc.want || c.Location.File != tc.file {
			t.Errorf("Lookup(%s) = %+v, %v; want %s from %s", tc.ident, c, ok, tc.want, tc.file)
		}
	}
	for _, ident := range []string{"TWICE", "EMPTY", "MISSING"} {
		if c, ok := consts.Lookup(ident); ok {
			t.Errorf("Lookup(%s) = %+v; want unresolved (ambiguous, empty or unknown)", ident, c)
		}
	}
}

func TestKeyIdentsResolvePlaceholdersAtTheirDefinition(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"app/Constants.kt":    "object Constants {\n    const val GOOGLE_API_KEY = \"googl_api_key\"\n}\n",
		"app/App.kt":          "Purchases.configure(PurchasesConfiguration.Builder(this, Constants.GOOGLE_API_KEY).build())\n",
		"ios/Constants.swift": "static let apiKey = \"REVENUECAT_API_KEY\"\n",
		"ios/App.swift":       "Purchases.configure(with: Configuration.Builder(withAPIKey: Constants.apiKey).build())\n",
		// The KMP SDK's builder and configure DSL take the key alone.
		"shared/Keys.kt":    "const val RC_KEY = \"rc_placeholder\"\nconst val RC_DSL_KEY = \"rc_dsl_placeholder\"\n",
		"shared/Billing.kt": "Purchases.configure(PurchasesConfiguration.Builder(RC_KEY).build())\nPurchases.configure(apiKey = RC_DSL_KEY) { }\n",
	})
	res, err := Source(root, Spec{Sets: []Patterns{{Extensions: []string{".kt", ".swift"}, KeyIdents: KeyIdents}}})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for i, k := range res.APIKeys {
		got = append(got, k.Value+"@"+k.Location.String()+"/"+res.KeyHints[i])
	}
	want := []string{"googl_api_key@app/Constants.kt:2/android", "REVENUECAT_API_KEY@ios/Constants.swift:1/",
		"rc_placeholder@shared/Keys.kt:1/", "rc_dsl_placeholder@shared/Keys.kt:2/"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("keys = %q, want %q", got, want)
	}
}
