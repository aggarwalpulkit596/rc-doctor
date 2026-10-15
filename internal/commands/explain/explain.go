// Package explain implements `rc-doctor explain <rule>`: print a rule's
// documentation, the same page that reports link to.
package explain

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	rcdoctor "github.com/aggarwalpulkit596/rc-doctor"
	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/commands/shared"
	"github.com/aggarwalpulkit596/rc-doctor/internal/rules"
)

func init() { cli.Register(Command{}) }

// Command is `rc-doctor explain`.
type Command struct{}

func (Command) Name() string { return "explain" }
func (Command) Summary() string {
	return "Explain a rule: what it checks, why it matters and how to fix it"
}

func (c Command) Run(_ context.Context, args []string, stdout, stderr io.Writer) int {
	fs := shared.NewFlagSet(c, "[rule] ", stderr)
	pos, err := shared.Parse(fs, args)
	if err != nil {
		return shared.FlagExit(err)
	}
	switch len(pos) {
	case 0:
		tw := tabwriter.NewWriter(stdout, 0, 0, 3, ' ', 0)
		for _, r := range rules.All() {
			fmt.Fprintf(tw, "%s\t%s\n", r.ID(), r.Description())
		}
		tw.Flush()
		fmt.Fprintln(stdout, "\nRun `rc-doctor explain <rule>` for details.")
		return cli.ExitOK
	case 1:
	default:
		return shared.Fail(stderr, fmt.Errorf("explain takes one rule, got %q", pos))
	}
	id := pos[0]
	if _, ok := rules.Get(id); !ok {
		var ids []string
		for _, r := range rules.All() {
			ids = append(ids, r.ID())
		}
		return shared.Fail(stderr, fmt.Errorf("unknown rule %q (rules: %s)", id, strings.Join(ids, ", ")))
	}
	page, err := rcdoctor.RuleDocs.ReadFile("docs/rules/" + id + ".md")
	if err != nil {
		return shared.Fail(stderr, fmt.Errorf("no documentation for %s: %w", id, err))
	}
	if _, err := stdout.Write(page); err != nil {
		return shared.Fail(stderr, err)
	}
	fmt.Fprintf(stdout, "\nOnline: %s\n", rules.DocsURL(id))
	return cli.ExitOK
}
