// Package android collects RevenueCat facts from Gradle-based Android apps.
package android

import (
	"regexp"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform/scan"
)

func init() { platform.Register(Android{}) }

// Android implements platform.Platform for Gradle-based Android apps.
type Android struct{}

var (
	// buildFiles can declare the SDK, directly or through a version catalog.
	buildFiles  = []string{"build.gradle", "build.gradle.kts", "libs.versions.toml"}
	gradleFiles = []string{"build.gradle", "build.gradle.kts"}
	sdkGroup    = "com.revenuecat.purchases"

	applicationID = regexp.MustCompile(`\bapplicationId\s*=?\s*["']([^"']+)["']`)

	spec = scan.Spec{
		// Test source sets (test, testDebug, androidTest, commonTest, jvmTest, ...)
		// and Kotlin Multiplatform source sets that only compile for Apple targets.
		Exclude: regexp.MustCompile(`(^|/)((?:test|androidTest)(?:[A-Z]\w*)?|[a-z]\w*Test|(?:ios|apple|macos|tvos|watchos)\w*Main)(/|$)`),
		Sets: []scan.Patterns{
			{
				Extensions:   []string{".kt", ".java"},
				APIKey:       []*regexp.Regexp{scan.KeyQuoted},
				Configure:    []*regexp.Regexp{scan.ConfigureCall},
				Entitlements: []*regexp.Regexp{scan.EntitlementSubscript, scan.EntitlementCall, scan.PaywallEntitlement},
				Offerings:    []*regexp.Regexp{scan.OfferingSubscript, scan.OfferingCall},

				EntitlementIdents: scan.EntitlementIdents,
				OfferingIdents:    scan.OfferingIdents,
				KeyIdents:         scan.KeyIdents,
			},
			{
				// Keys kept out of code: buildConfigField and resValue in Gradle
				// files, properties files, string resources and the manifest.
				Extensions: []string{".gradle", ".gradle.kts", ".properties", ".xml"},
				APIKey:     []*regexp.Regexp{scan.KeyQuoted, scan.KeyAssigned, scan.KeyXMLText},
			},
		},
	}
)

func (Android) Name() string { return "android" }

// KeyPrefixes: Google Play and Amazon Appstore apps both run the Android SDK.
// TODO(verify): confirm the prefixes against RevenueCat's API keys documentation.
func (Android) KeyPrefixes() []string { return []string{"goog_", "amzn_"} }

func (Android) Detect(root string) bool {
	return scan.Contains(root, buildFiles, sdkGroup)
}

func (Android) Collect(root string) ([]model.Facts, error) {
	src, err := scan.Source(root, spec)
	if err != nil {
		return nil, err
	}
	f := model.Facts{
		Platform:     "android",
		Root:         root,
		APIKeys:      src.KeysNotFor("ios"),
		ConfigureAt:  src.ConfigureAt,
		Entitlements: src.Entitlements,
		Offerings:    src.Offerings,
	}
	if m, ok := ApplicationID(root); ok {
		f.AppID, f.AppIDAt = m.Value, m.Location
	}
	return []model.Facts{f}, nil
}

// ApplicationID returns the first applicationId declared in a Gradle build file.
func ApplicationID(root string) (scan.Match, bool) {
	return scan.FirstMatch(root, gradleFiles, applicationID, nil)
}
