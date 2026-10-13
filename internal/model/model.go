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

	// Notes are messages from the provider about a partial load (for example,
	// apps couldn't be listed). They are shown in reports, never saved.
	Notes []string `json:"-"`
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
