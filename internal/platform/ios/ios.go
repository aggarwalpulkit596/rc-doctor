// Package ios collects RevenueCat facts from Apple-platform apps built with
// Xcode, Swift Package Manager or CocoaPods.
package ios

import (
	"regexp"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform/scan"
)

func init() { platform.Register(IOS{}) }

// IOS implements platform.Platform for iOS, macOS and other Apple targets.
type IOS struct{}

var (
	manifestFiles = []string{"Package.swift", "Package.resolved", "Podfile", "Podfile.lock", "project.pbxproj"}
	sdkMarkers    = []string{"purchases-ios", "RevenueCat"}

	bundleID = regexp.MustCompile(`PRODUCT_BUNDLE_IDENTIFIER\s*=\s*"?([^";]+?)"?\s*;`)

	// Swift spells offering lookup as offerings.offering(identifier: "main").
	offeringIdentifier      = regexp.MustCompile(`offering\(\s*identifier:\s*"([^"]+)"`)
	offeringIdentifierIdent = regexp.MustCompile(`offering\(\s*identifier:\s*([A-Za-z_][A-Za-z0-9_.]*)\s*\)`)
	// Objective-C configures with [RCPurchases configureWithAPIKey:@"..."].
	configureObjC = regexp.MustCompile(`\bRCPurchases\s+configureWith`)

	spec = scan.Spec{
		// Test targets: FooTests, FooUITests, and SwiftPM's Tests directory.
		Exclude: regexp.MustCompile(`(^|/)\w*Tests(/|$)`),
		Sets: []scan.Patterns{
			{
				Extensions:   []string{".swift", ".m", ".mm"},
				APIKey:       []*regexp.Regexp{scan.KeyQuoted},
				Configure:    []*regexp.Regexp{scan.ConfigureCall, configureObjC},
				Entitlements: []*regexp.Regexp{scan.EntitlementSubscript, scan.EntitlementCall, scan.PaywallEntitlement},
				Offerings:    []*regexp.Regexp{scan.OfferingSubscript, offeringIdentifier},

				EntitlementIdents: scan.EntitlementIdents,
				OfferingIdents:    append([]*regexp.Regexp{offeringIdentifierIdent}, scan.OfferingIdents...),
				KeyIdents:         scan.KeyIdents,
			},
			{
				// Keys kept out of code: build settings and Info.plist values.
				Extensions: []string{".xcconfig", ".plist"},
				APIKey:     []*regexp.Regexp{scan.KeyAssigned, scan.KeyXMLText},
			},
		},
	}
)

func (IOS) Name() string { return "ios" }

// KeyPrefixes: App Store and Mac App Store apps.
// TODO(verify): confirm the prefix against RevenueCat's API keys documentation.
func (IOS) KeyPrefixes() []string { return []string{"appl_"} }

func (IOS) Detect(root string) bool {
	return scan.Contains(root, manifestFiles, sdkMarkers...)
}

func (IOS) Collect(root string) ([]model.Facts, error) {
	src, err := scan.Source(root, spec)
	if err != nil {
		return nil, err
	}
	f := model.Facts{
		Platform:     "ios",
		Root:         root,
		APIKeys:      src.KeysNotFor("android"),
		ConfigureAt:  src.ConfigureAt,
		Entitlements: src.Entitlements,
		Offerings:    src.Offerings,
	}
	if m, ok := BundleID(root); ok {
		f.AppID, f.AppIDAt = m.Value, m.Location
	}
	return []model.Facts{f}, nil
}

// BundleID picks the app's bundle identifier from project.pbxproj files.
// Test targets and unresolved build settings are ignored. App extensions must
// use identifiers that start with their containing app's, so the shortest
// remaining candidate is the app itself.
func BundleID(root string) (scan.Match, bool) {
	var best scan.Match
	found := false
	for _, m := range scan.Matches(root, []string{"project.pbxproj"}, bundleID) {
		v := m.Value
		if strings.HasSuffix(v, "Tests") || strings.Contains(v, "$(") || strings.Contains(v, "${") {
			continue
		}
		if !found || len(v) < len(best.Value) {
			best, found = m, true
		}
	}
	return best, found
}
