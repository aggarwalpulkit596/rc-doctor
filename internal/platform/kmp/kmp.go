// Package kmp collects RevenueCat facts from Kotlin Multiplatform apps that use
// the RevenueCat KMP SDK (purchases-kmp).
//
// One KMP repo ships two apps, so Collect returns two Facts, "android" and
// "ios", and every rule checks each one as it would a native app:
//   - androidMain belongs to Android; iosMain, appleMain and friends, and the
//     Xcode wrapper's Swift code, belong to iOS;
//   - commonMain and config files are shared. Their entitlement and offering
//     IDs go to both apps (the engine reports a shared line once), and each
//     key goes to the app its own name or prefix points at, so
//     REVENUECAT_IOS_KEY=appl_... in secrets.properties is the iOS app's key
//     even though the Android build reads that file.
package kmp

import (
	"regexp"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform/android"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform/ios"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform/scan"
)

func init() { platform.Register(KMP{}) }

// KMP implements platform.Platform for Kotlin Multiplatform apps.
type KMP struct{}

var (
	buildFiles = []string{"build.gradle", "build.gradle.kts", "libs.versions.toml"}
	sdkModule  = "purchases-kmp"

	// Test source sets and test targets, on both sides.
	exclude = regexp.MustCompile(`(^|/)((?:test|androidTest)(?:[A-Z]\w*)?|[a-z]\w*Test|\w*Tests)(/|$)`)

	androidSets = regexp.MustCompile(`(^|/)android\w*Main/`)
	appleSets   = regexp.MustCompile(`(^|/)(?:ios|apple|macos|tvos|watchos)\w*Main/`)
	sharedSets  = regexp.MustCompile(`(^|/)common\w*Main/`)

	// The RevenueCat KMP SDK spells offering lookup like the native SDKs.
	kotlinSet = scan.Patterns{
		Extensions:        []string{".kt"},
		APIKey:            []*regexp.Regexp{scan.KeyQuoted},
		Configure:         []*regexp.Regexp{scan.ConfigureCall},
		Entitlements:      []*regexp.Regexp{scan.EntitlementSubscript, scan.EntitlementCall, scan.PaywallEntitlement},
		Offerings:         []*regexp.Regexp{scan.OfferingSubscript, scan.OfferingCall},
		EntitlementIdents: scan.EntitlementIdents,
		OfferingIdents:    scan.OfferingIdents,
		KeyIdents:         scan.KeyIdents,
	}
	swiftSet = scan.Patterns{
		Extensions:        []string{".swift", ".m"},
		APIKey:            []*regexp.Regexp{scan.KeyQuoted},
		Configure:         []*regexp.Regexp{scan.ConfigureCall},
		Entitlements:      []*regexp.Regexp{scan.EntitlementSubscript, scan.EntitlementCall, scan.PaywallEntitlement},
		Offerings:         []*regexp.Regexp{scan.OfferingSubscript},
		EntitlementIdents: scan.EntitlementIdents,
		OfferingIdents:    scan.OfferingIdents,
		KeyIdents:         scan.KeyIdents,
	}
	configSet = scan.Patterns{
		Extensions: []string{".gradle", ".gradle.kts", ".properties", ".xcconfig", ".plist", ".xml"},
		APIKey:     []*regexp.Regexp{scan.KeyQuoted, scan.KeyAssigned, scan.KeyXMLText},
	}
)

func (KMP) Name() string { return "kmp" }

// KeyPrefixes: a KMP repo ships both apps. The engine checks each app's Facts
// against its own platform's prefixes.
func (KMP) KeyPrefixes() []string {
	return append(android.Android{}.KeyPrefixes(), ios.IOS{}.KeyPrefixes()...)
}

// Supersedes: a KMP app also looks like an Android app (Gradle declares a
// com.revenuecat.purchases artifact) and like an iOS app, but read as either
// alone, half its code would be checked against the wrong platform.
func (KMP) Supersedes() []string { return []string{"android", "ios"} }

func (KMP) Detect(root string) bool { return scan.Contains(root, buildFiles, sdkModule) }

func (KMP) Collect(root string) ([]model.Facts, error) {
	consts := scan.Constants(root, exclude)
	scanWith := func(sets ...scan.Patterns) (scan.Result, error) {
		return scan.Source(root, scan.Spec{Sets: sets, Exclude: exclude, Constants: consts})
	}
	only := func(p scan.Patterns, include *regexp.Regexp) scan.Patterns {
		p.Include = include
		return p
	}

	droid, err := scanWith(only(kotlinSet, androidSets))
	if err != nil {
		return nil, err
	}
	apple, err := scanWith(only(kotlinSet, appleSets), swiftSet)
	if err != nil {
		return nil, err
	}
	shared, err := scanWith(only(kotlinSet, sharedSets), configSet)
	if err != nil {
		return nil, err
	}

	a := model.Facts{Platform: "android", Root: root}
	i := model.Facts{Platform: "ios", Root: root}
	for _, part := range []struct {
		f   *model.Facts
		src scan.Result
	}{{&a, droid}, {&i, apple}, {&a, shared}, {&i, shared}} {
		part.f.ConfigureAt = append(part.f.ConfigureAt, part.src.ConfigureAt...)
		part.f.Entitlements = append(part.f.Entitlements, part.src.Entitlements...)
		part.f.Offerings = append(part.f.Offerings, part.src.Offerings...)
	}
	a.APIKeys = droid.APIKeys
	i.APIKeys = apple.APIKeys
	for n, k := range shared.APIKeys {
		switch owner(shared.KeyHints[n], k.Value) {
		case "android":
			a.APIKeys = append(a.APIKeys, k)
		case "ios":
			i.APIKeys = append(i.APIKeys, k)
		default: // a secret key, or a key nothing ties to one app: check it as both
			a.APIKeys = append(a.APIKeys, k)
			i.APIKeys = append(i.APIKeys, k)
		}
	}

	if m, ok := android.ApplicationID(root); ok {
		a.AppID, a.AppIDAt = m.Value, m.Location
	}
	if m, ok := ios.BundleID(root); ok {
		i.AppID, i.AppIDAt = m.Value, m.Location
	}
	return []model.Facts{a, i}, nil
}

// owner decides which app a shared key belongs to: the platform its line names
// (REVENUECAT_IOS_KEY), else the platform its prefix belongs to.
func owner(hint, key string) string {
	if hint != "" {
		return hint
	}
	for _, p := range []platform.Platform{android.Android{}, ios.IOS{}} {
		for _, prefix := range p.KeyPrefixes() {
			if len(key) > len(prefix) && key[:len(prefix)] == prefix {
				return p.Name()
			}
		}
	}
	return ""
}
