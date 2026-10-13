// Package migrate implements `rc-doctor migrate`: plan a move to RevenueCat
// from another SDK, and apply the automatable part with --apply.
//
// The command is source-agnostic. Each source (Superwall, Adapty, raw
// StoreKit 2, ...) is a migrations.Migrator plug-in; see internal/migrate.
package migrate

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/commands/shared"
	"github.com/aggarwalpulkit596/rc-doctor/internal/fix"
	migrations "github.com/aggarwalpulkit596/rc-doctor/internal/migrate"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/report"
)

func init() { cli.Register(Command{}) }

// Command is `rc-doctor migrate`.
type Command struct{}

func (Command) Name() string { return "migrate" }

func (Command) Summary() string {
	return "Plan a migration to RevenueCat from another SDK, and apply it with --apply"
}

func (c Command) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := shared.NewFlagSet(c, "[dir] ", stderr)
	from := fs.String("from", "", "source to migrate from (default: detect)")
	apply := fs.Bool("apply", false, "apply the plan's safe edits")
	suggested := fs.Bool("suggested", false, "with --apply, also apply suggested edits")
	root, code, ok := shared.ParseRoot(fs, args, stderr)
	if !ok {
		return code
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return shared.Fail(stderr, err)
	}

	available := migrations.Names()
	if len(available) == 0 {
		fmt.Fprintf(stdout, "No migration sources are available yet. Planned: %s.\n", strings.Join(migrations.Planned, ", "))
		fmt.Fprintln(stdout, "Each source plugs in as a package under internal/migrate (see the README).")
		if *from != "" {
			return cli.ExitError
		}
		return cli.ExitOK
	}

	m, code, ok := pick(abs, *from, available, stdout, stderr)
	if !ok {
		return code
	}
	plan, err := m.Plan(ctx, abs)
	if err != nil {
		return shared.Fail(stderr, fmt.Errorf("plan migration from %s: %w", m.Name(), err))
	}
	printPlan(stdout, abs, plan)

	level := model.Safe
	if *suggested {
		level = model.Suggested
	}
	edits := fix.Select(plan.Edits(), level)
	if !*apply {
		safe, sug := shared.CountSafety(fix.Select(plan.Edits(), model.Suggested))
		fmt.Fprintf(stdout, "\nNothing was changed. Run with --apply to apply %s (add --suggested for %s).\n",
			shared.Plural(safe, "safe edit", "safe edits"), shared.Plural(sug, "suggested edit", "suggested edits"))
		return cli.ExitOK
	}
	res, err := fix.Apply(edits)
	fmt.Fprintf(stdout, "\nApplied %s:\n", shared.Plural(len(res.Applied), "edit", "edits"))
	shared.PrintChanges(stdout, abs, res.Applied)
	if len(res.Skipped) > 0 {
		fmt.Fprintf(stdout, "\nSkipped %s:\n", shared.Plural(len(res.Skipped), "edit", "edits"))
		shared.PrintSkipped(stdout, abs, res.Skipped)
	}
	if err != nil {
		return shared.Fail(stderr, err)
	}
	fmt.Fprintln(stdout, "\nFinish the manual steps above, then run `rc-doctor check`.")
	return cli.ExitOK
}

// pick returns the source named by --from, or the single detected one.
func pick(root, from string, available []string, stdout, stderr io.Writer) (migrations.Migrator, int, bool) {
	if from != "" {
		m, ok := migrations.Get(from)
		if !ok {
			fmt.Fprintf(stderr, "rc-doctor migrate: unknown source %q (available: %s)\n", from, strings.Join(available, ", "))
			return nil, cli.ExitError, false
		}
		return m, 0, true
	}
	var found []migrations.Migrator
	for _, m := range migrations.All() {
		if m.Detect(root) {
			found = append(found, m)
		}
	}
	switch len(found) {
	case 0:
		fmt.Fprintf(stdout, "Nothing to migrate: no supported SDK found. Sources: %s.\n", strings.Join(available, ", "))
		return nil, cli.ExitOK, false
	case 1:
		return found[0], 0, true
	}
	var names []string
	for _, m := range found {
		names = append(names, m.Name())
	}
	fmt.Fprintf(stderr, "rc-doctor migrate: found several SDKs (%s); choose one with --from\n", strings.Join(names, ", "))
	return nil, cli.ExitError, false
}

func printPlan(w io.Writer, root string, p *migrations.Plan) {
	fmt.Fprintf(w, "Migration plan: %s -> RevenueCat (%s)\n", p.From, shared.Plural(len(p.Steps), "step", "steps"))
	for i, s := range p.Steps {
		fmt.Fprintf(w, "\n%d. %s\n", i+1, s.Title)
		for _, e := range s.Edits {
			fmt.Fprintf(w, "   [%s] %s:%d  %s\n", e.Safety, report.Path(root, e.File), e.Line, e.Description)
		}
		if s.Manual != "" {
			fmt.Fprintf(w, "   manual: %s\n", s.Manual)
		}
	}
}
