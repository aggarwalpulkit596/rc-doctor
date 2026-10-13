// Package model holds the platform-neutral types every module shares.
//
// Platforms produce Facts, the dashboard produces a Dashboard, rules turn both
// into Findings, and Findings (or migrations) can carry Edits that the fix
// engine applies. Nothing in here knows about Android, iOS or the rc CLI.
package model

import "fmt"

// Location points at the place in the user's code a fact came from.
// File is relative to the scanned root.
type Location struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

func (l Location) String() string {
	if l.File == "" {
		return "(unknown)"
	}
	return fmt.Sprintf("%s:%d", l.File, l.Line)
}

// Ref is one identifier referenced in code, such as the entitlement "pro".
type Ref struct {
	ID       string   `json:"id"`
	Location Location `json:"location"` // where the code uses it

	// Def is set when the code uses a constant (ENTITLEMENT_ID) rather than a
	// literal: it says where the value is defined, which is where a fix goes.
	Def *Definition `json:"defined_at,omitempty"`
}

// Definition is where a constant's value is written.
type Definition struct {
	Name     string   `json:"name"`     // the constant or property name
	Location Location `json:"location"` // the line holding the value
	Text     string   `json:"text"`     // the value exactly as written there: "plus" or plus
}

// APIKey is a RevenueCat key literal found in code.
type APIKey struct {
	Value    string   `json:"value"`
	Location Location `json:"location"`

	// Configured is set when the key reaches Purchases.configure: it is
	// written in the configure call, or in the constant the call passes.
	Configured bool `json:"configured,omitempty"`

	// DebugOnly is set when the key, or the configure call that passes it,
	// sits behind a debug-only guard the platform module recognizes: an
	// #if DEBUG block, a BuildConfig.DEBUG check, a Gradle debug build type,
	// or a debug source set or config file.
	DebugOnly bool `json:"debug_only,omitempty"`
}

// Facts is everything one platform module learned about one app target.
// A repo can produce several Facts (for example an Android app and an iOS app).
type Facts struct {
	Platform     string     `json:"platform"` // "android", "ios", ...
	Root         string     `json:"root"`     // absolute directory the facts were collected from
	AppID        string     `json:"app_id"`   // applicationId or bundle identifier
	AppIDAt      Location   `json:"app_id_at"`
	APIKeys      []APIKey   `json:"api_keys"`
	ConfigureAt  []Location `json:"configure_at"` // Purchases.configure call sites
	Entitlements []Ref      `json:"entitlements"` // entitlement IDs referenced in code
	Offerings    []Ref      `json:"offerings"`    // offering IDs referenced in code

	// KeyPrefixes are the public API key prefixes valid for this platform.
	// The engine fills this from the platform module so rules stay platform-neutral.
	KeyPrefixes []string `json:"key_prefixes"`

	// AppUserIDs are fixed app user IDs passed to the SDK (configure's
	// appUserID, logIn): literals, or constants resolved to one.
	AppUserIDs []Ref `json:"app_user_ids,omitempty"`

	// Signals record whether the app does something anywhere in its code, and
	// where: restoring purchases, showing a RevenueCat paywall, and so on.
	// Keys are the Signal* constants.
	Signals map[string][]Location `json:"signals,omitempty"`

	// Build holds build settings the platform read, keyed by the Build*
	// constants. An absent key means the setting wasn't found, not "false".
	Build map[string]Setting `json:"build,omitempty"`
}

// Signal names. A platform module records a signal for every place its code
// does the thing; rules ask whether a signal is present at all.
const (
	SignalRestore          = "restore"             // restorePurchases and equivalents
	SignalPaywallUI        = "paywall-ui"          // a RevenueCatUI paywall, which includes a restore button
	SignalCustomerCenter   = "customer-center"     // RevenueCatUI Customer Center, which includes restore
	SignalCurrentOffering  = "current-offering"    // reads offerings.current
	SignalCompletedByMyApp = "completed-by-my-app" // purchases are completed (acknowledged or finished) by the app
	SignalFinishesPurchase = "finishes-purchase"   // the app acknowledges, consumes or finishes a store transaction
	SignalTestStoreRelease = "test-store-release"  // the app lets release builds use a Test Store key (forceAllowTestStoreInReleaseBuilds)
)

// Build setting keys.
const (
	BuildSDKArtifact      = "revenuecat.sdk_artifact"           // e.g. "purchases", "purchases-kmp-core"
	BuildSDKVersion       = "revenuecat.sdk_version"            // as declared, e.g. "8.10.8"
	BuildMinSDK           = "android.min_sdk"                   // e.g. "21"
	BuildDesugaring       = "android.core_library_desugaring"   // "true" when enabled
	BuildApplicationClass = "android.application_class"         // fully qualified Application subclass that configures the SDK
	BuildManifest         = "android.manifest"                  // a main AndroidManifest.xml was found (value: its path)
	BuildManifestAppName  = "android.manifest_application_name" // fully qualified android:name of <application>, if set
)

// Setting is one build setting and where it was read.
type Setting struct {
	Value    string   `json:"value"`
	Location Location `json:"location"`
}

// HasSignal reports whether any of the named signals was found.
func (f Facts) HasSignal(names ...string) bool {
	for _, n := range names {
		if len(f.Signals[n]) > 0 {
			return true
		}
	}
	return false
}

// App is one app configured in the RevenueCat project.
type App struct {
	ID         string   `json:"id"`
	Platform   string   `json:"platform"`              // "android", "ios", ...
	AppID      string   `json:"app_id"`                // package name or bundle identifier
	PublicKeys []string `json:"public_keys,omitempty"` // public SDK keys for this app
}

// Dashboard is the configured state of a RevenueCat project.
type Dashboard struct {
	Entitlements []string `json:"entitlements"` // entitlement lookup keys
	Offerings    []string `json:"offerings"`    // offering lookup keys
	Apps         []App    `json:"apps"`

	// Catalog holds how offerings, packages, products and entitlements connect.
	// It is nil when the provider couldn't load it; rules that need it then
	// stay silent rather than guess.
	Catalog *Catalog `json:"catalog,omitempty"`

	// Notes are messages from the provider about a partial load (for example,
	// apps couldn't be listed). They are shown in reports, never saved.
	Notes []string `json:"-"`
}

// Catalog is the part of a project that decides what a purchase unlocks and
// what a paywall can sell.
type Catalog struct {
	// CurrentOffering is the lookup key of the offering marked current, or ""
	// when none is.
	CurrentOffering string `json:"current_offering"`
	// Packages maps an offering's lookup key to its packages.
	Packages map[string][]Package `json:"packages"`
	// EntitlementProducts maps an entitlement's lookup key to the store
	// identifiers of the products that grant it.
	EntitlementProducts map[string][]string `json:"entitlement_products"`
	// ProductTypes maps every product's store identifier to its RevenueCat
	// type: subscription, consumable, non_consumable, non_renewing_subscription
	// or one_time. Nil when the products couldn't be listed.
	ProductTypes map[string]string `json:"product_types,omitempty"`
}

// OnlyConsumables reports whether the project lists products and every one of
// them is a consumable, which nothing can restore.
func (c *Catalog) OnlyConsumables() bool {
	if c == nil || len(c.ProductTypes) == 0 {
		return false
	}
	for _, t := range c.ProductTypes {
		if t != "consumable" {
			return false
		}
	}
	return true
}

// Package is one package in an offering.
type Package struct {
	ID       string   `json:"id"`       // lookup key, such as $rc_monthly
	Products []string `json:"products"` // store identifiers of the products attached
}

// AppsFor returns the configured apps for a platform.
func (d *Dashboard) AppsFor(platform string) []App {
	var out []App
	for _, a := range d.Apps {
		if a.Platform == platform {
			out = append(out, a)
		}
	}
	return out
}

// Severity of a finding.
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
	Info    Severity = "info"
)

// Rank orders severities so callers can compare them (higher is worse).
func (s Severity) Rank() int {
	switch s {
	case Error:
		return 3
	case Warning:
		return 2
	case Info:
		return 1
	}
	return 0
}

// Safety says how confident we are that an Edit is correct.
type Safety string

const (
	// Safe edits are correct by construction (for example, replacing a key with
	// the one the dashboard says belongs to this app). `review --fix` applies them.
	Safe Safety = "safe"
	// Suggested edits are a best guess (for example, a likely typo). They are
	// applied only when the user opts in with `review --fix --suggested`.
	Suggested Safety = "suggested"
)

// AllowedAt reports whether an edit with safety s may be applied when the user
// allowed edits up to max.
func (s Safety) AllowedAt(max Safety) bool {
	return s == Safe || max == Suggested
}

// Edit is a single-line text replacement the fix engine can apply.
// Edits are deliberately simple so every module (rules, migrations, future
// generators) can produce them and one engine applies them all.
type Edit struct {
	Path        string `json:"path"` // absolute path of the file to change
	File        string `json:"file"` // path relative to the scanned root, for display
	Line        int    `json:"line"` // 1-based
	Old         string `json:"old"`  // exact text expected on that line
	New         string `json:"new"`  // replacement text
	Safety      Safety `json:"safety"`
	Description string `json:"description"`
}

// Finding is one problem a rule found.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Platform string   `json:"platform"`
	Message  string   `json:"message"`
	Location Location `json:"location"`
	Fix      string   `json:"fix,omitempty"`   // human-readable suggestion
	Edits    []Edit   `json:"edits,omitempty"` // machine-applicable fixes, if any
}

// Fixable returns the safety of the most confident edit attached to the
// finding, or "" when it has none.
func (f Finding) Fixable() Safety {
	var best Safety
	for _, e := range f.Edits {
		if e.Safety == Safe {
			return Safe
		}
		best = e.Safety
	}
	return best
}
