// Package migrate is the plug-in point for moving an app to RevenueCat from
// another SDK (Superwall, Adapty, Qonversion) or from raw StoreKit 2 / Play
// Billing code.
//
// A migrator detects its source and returns a Plan: ordered steps, each with
// Edits the fix engine can apply and/or Manual instructions for what can't be
// automated (creating products and entitlements, mapping paywalls). Plans
// never write anything themselves: `rc-doctor migrate --apply` applies them
// through the same engine, and the same safety levels, as `review --fix`.
//
// To add a source:
//  1. Create internal/migrate/<source> with a type implementing Migrator.
//  2. Call migrate.Register from its init().
//  3. Blank-import it in cmd/rc-doctor/plugins.go.
//  4. Add a fixture app that uses the source SDK and a Plan test.
package migrate

import (
	"context"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/registry"
)

// Step is one stage of a migration.
type Step struct {
	Title  string       `json:"title"`
	Edits  []model.Edit `json:"edits,omitempty"`
	Manual string       `json:"manual,omitempty"` // what the developer must do by hand
}

// Plan is a migrator's proposal. Nothing in it has been applied.
type Plan struct {
	From  string `json:"from"`
	Steps []Step `json:"steps"`
}

// Edits returns every edit in the plan, in step order.
func (p *Plan) Edits() []model.Edit {
	var out []model.Edit
	for _, s := range p.Steps {
		out = append(out, s.Edits...)
	}
	return out
}

// Migrator plans a migration from one source.
type Migrator interface {
	// Name is the value of --from ("superwall").
	Name() string
	Description() string
	// Detect reports whether root uses this source.
	Detect(root string) bool
	// Plan inspects root (an absolute path) and proposes the steps.
	Plan(ctx context.Context, root string) (*Plan, error)
}

// Planned lists the sources on the roadmap, shown while none is registered.
var Planned = []string{"superwall", "adapty", "qonversion", "storekit2", "play-billing"}

var reg = registry.New[Migrator]("migrator", Migrator.Name)

// Register adds a migration source. It panics on duplicate names.
func Register(m Migrator) { reg.Register(m) }

// Unregister removes a source. It exists for tests.
func Unregister(name string) { reg.Unregister(name) }

// All returns registered sources sorted by name.
func All() []Migrator { return reg.All() }

// Get returns a source by name.
func Get(name string) (Migrator, bool) { return reg.Get(name) }

// Names returns the registered source names.
func Names() []string { return reg.Names() }
