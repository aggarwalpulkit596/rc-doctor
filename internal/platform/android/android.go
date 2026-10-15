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

	// FinishesPurchase is the app acknowledging or consuming a Play purchase
	// itself, which PurchasesAreCompletedBy.MY_APP makes its job.
	FinishesPurchase = regexp.MustCompile(`\b(?:acknowledgePurchase|consumeAsync|consumePurchase)\s*\(`)

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
				AppUserIDs:        scan.AppUserIDLiterals,
				AppUserIDIdents:   scan.AppUserIDIdents,
				Signals:           scan.WithSignals(model.SignalFinishesPurchase, FinishesPurchase),
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

// KeyPrefixes: the Android SDK runs Google Play, Amazon Appstore and Galaxy
// Store apps. The prefixes are the ones the SDK itself checks
// (APIKeyValidator.kt in RevenueCat/purchases-android). Test Store keys
// (test_) work on every platform, so no platform claims them.
func (Android) KeyPrefixes() []string { return []string{"goog_", "amzn_", "galx_"} }

func (Android) Detect(root string) bool {
	return scan.Contains(root, buildFiles, sdkGroup)
}

func (Android) Collect(root string) ([]model.Facts, error) {
	spec := spec
	// A JVM server or tool next to the app isn't part of it.
	spec.Exclude = scan.ExcludeDirs(spec.Exclude, scan.JVMModules(root, spec.Exclude))
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
		AppUserIDs:   src.AppUserIDs,
		Signals:      src.Signals,
		Build:        BuildSettings(root, spec.Exclude, src.ConfigureAt),
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
