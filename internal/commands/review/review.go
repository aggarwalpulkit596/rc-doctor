// Package review implements `rc-doctor review`: the check report plus the
// fixes rc-doctor can make, previewed as diffs. With --fix it applies the safe
// ones (and, with --suggested, the best guesses too), then checks again.
package review

import (
	"context"
	"fmt"
	"io"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/commands/shared"
	"github.com/aggarwalpulkit596/rc-doctor/internal/dashboard"
	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/fix"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/report"
)

func init() { cli.Register(Command{}) }

// Command is `rc-doctor review`.
type Command struct{}

func (Command) Name() string { return "review" }

func (Command) Summary() string {
	return "Preview the fixes rc-doctor can make, and apply them with --fix"
}

// text is the report review prints: it shows fixes itself, so no pointer to review.
var text = report.Text{NoFixHint: true}

func (c Command) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := shared.NewFlagSet(c, "[dir] ", stderr)
	scan := shared.BindScan(fs)
	failOn := shared.BindFailOn(fs)
	apply := fs.Bool("fix", false, "apply safe fixes, then check again")
	suggested := fs.Bool("suggested", false, "with --fix, also apply suggested fixes: best guesses such as typo corrections")
	root, code, ok := shared.ParseRoot(fs, args, stderr)
	if !ok {
		return code
	}
	threshold, err := shared.Threshold(*failOn)
	if err != nil {
		return shared.Fail(stderr, err)
	}
	res, err := scan.Run(ctx, root)
	if err != nil {
		return shared.Fail(stderr, err)
	}
	if !*apply {
		if err := text.Write(stdout, res); err != nil {
			return shared.Fail(stderr, err)
		}
		return preview(stdout, res, threshold)
	}

	level := model.Safe
	if *suggested {
		level = model.Suggested
	}
	edits := fix.Select(res.Edits(), level)
	if len(edits) == 0 {
		if err := text.Write(stdout, res); err != nil {
			return shared.Fail(stderr, err)
		}
		fmt.Fprintln(stdout, "\nNothing to fix automatically.")
		hintSuggested(stdout, res, *suggested)
		return shared.ExitCode(res.Findings, threshold)
	}

	applied, err := fix.Apply(edits)
	fmt.Fprintf(stdout, "Applied %s:\n", shared.Plural(len(applied.Applied), "fix", "fixes"))
	shared.PrintChanges(stdout, res.Root, applied.Applied)
	if len(applied.Skipped) > 0 {
		fmt.Fprintf(stdout, "\nSkipped %s:\n", shared.Plural(len(applied.Skipped), "fix", "fixes"))
		shared.PrintSkipped(stdout, res.Root, applied.Skipped)
	}
	if err != nil {
		return shared.Fail(stderr, err)
	}

	// Check again against the same project state, so the result reflects the
	// edits and nothing else.
	opts := scan.Options(root)
	opts.Dashboard, opts.StrictDashboard, opts.OfflineNote = nil, false, ""
	if res.DashboardData != nil {
		opts.Dashboard = dashboard.Static{Label: res.Dashboard, Data: res.DashboardData}
	}
	after, err := engine.Run(ctx, opts)
	if err != nil {
		return shared.Fail(stderr, err)
	}
	fmt.Fprintln(stdout, "\nAfter fixes:")
	if err := text.Write(stdout, after); err != nil {
		return shared.Fail(stderr, err)
	}
	hintSuggested(stdout, after, *suggested)
	return shared.ExitCode(after.Findings, threshold)
}

func preview(w io.Writer, res *engine.Result, threshold model.Severity) int {
	edits := fix.Select(res.Edits(), model.Suggested)
	if len(edits) == 0 {
		fmt.Fprintln(w, "\nNo automatic fixes available.")
		return shared.ExitCode(res.Findings, threshold)
	}
	changes, err := fix.Preview(edits)
	if err != nil {
		fmt.Fprintf(w, "\nCouldn't preview fixes: %v\n", err)
		return cli.ExitError
	}
	safe, sug := shared.CountSafety(edits)
	fmt.Fprintf(w, "\nProposed fixes (%d safe, %d suggested):\n", safe, sug)
	shared.PrintChanges(w, res.Root, changes.Applied)
	if len(changes.Skipped) > 0 {
		fmt.Fprintln(w, "\nCan't apply:")
		shared.PrintSkipped(w, res.Root, changes.Skipped)
	}
	fmt.Fprintln(w, "\nNothing was changed. Apply the safe fixes with --fix; add --suggested to apply the suggested ones too.")
	return shared.ExitCode(res.Findings, threshold)
}

// hintSuggested points at --suggested when only suggested fixes are left.
func hintSuggested(w io.Writer, res *engine.Result, already bool) {
	if already {
		return
	}
	if _, sug := shared.CountSafety(fix.Select(res.Edits(), model.Suggested)); sug > 0 {
		fmt.Fprintf(w, "%s left: run `rc-doctor review` to preview, then --fix --suggested to apply.\n",
			shared.Plural(sug, "suggested fix", "suggested fixes"))
	}
}
