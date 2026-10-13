// Package rules turns Facts and the Dashboard into Findings.
//
// Rules are plug-ins. To add one:
//  1. Write a type implementing Rule, in this package or a new one.
//  2. Register it from init(). A rule in a new package is linked in with a
//     blank import in cmd/rc-doctor/plugins.go.
//  3. Test it with a hand-built Context. Rules never touch the file system,
//     never call the network and never import a platform package, so a rule
//     works for every platform, current and future.
//
// A rule that knows the exact fix attaches a model.Edit marked Safe, which
// `review --fix` applies. A best guess, such as a likely typo, is marked
// Suggested and only applies with `review --fix --suggested`.
package rules

import (
	"path/filepath"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/registry"
)

// Context is everything a rule may look at.
type Context struct {
	Facts     []model.Facts
	Dashboard *model.Dashboard // nil when running offline

	// KeyPrefixes maps each registered platform's public key prefixes to that
	// platform ("appl_" -> "ios"), so a rule can tell which platform a key is for.
	KeyPrefixes map[string]string
}

// Rule checks one thing.
type Rule interface {
	ID() string
	Description() string
	// NeedsDashboard rules are skipped, and reported as skipped, when offline.
	NeedsDashboard() bool
	Check(ctx *Context) []model.Finding
}

var reg = registry.New[Rule]("rule", Rule.ID)

// Register adds a rule. It panics on duplicate IDs.
func Register(r Rule) { reg.Register(r) }

// All returns registered rules sorted by ID.
func All() []Rule { return reg.All() }

// Get returns a rule by ID.
func Get(id string) (Rule, bool) { return reg.Get(id) }

// pathOf returns the absolute path of a location reported for f.
func pathOf(f model.Facts, loc model.Location) string {
	return filepath.Join(f.Root, filepath.FromSlash(loc.File))
}
