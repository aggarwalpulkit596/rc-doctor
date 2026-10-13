// Package list implements `rc-doctor list`: show what this binary was built
// with. Every section is read from a plug-in registry, so new plug-ins show up
// here without touching this file.
package list

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/commands/shared"
	"github.com/aggarwalpulkit596/rc-doctor/internal/migrate"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform"
	"github.com/aggarwalpulkit596/rc-doctor/internal/report"
	"github.com/aggarwalpulkit596/rc-doctor/internal/rules"
)

func init() { cli.Register(Command{}) }

// Command is `rc-doctor list`.
type Command struct{}

func (Command) Name() string { return "list" }

func (Command) Summary() string {
	return "List the platforms, rules, migration sources, formats and commands built in"
}

type section struct {
	name string
	rows func() [][2]string
}

var sections = []section{
	{"platforms", func() (rows [][2]string) {
		for _, p := range platform.All() {
			rows = append(rows, [2]string{p.Name(), "public key prefixes: " + strings.Join(p.KeyPrefixes(), ", ")})
		}
		return rows
	}},
	{"rules", func() (rows [][2]string) {
		for _, r := range rules.All() {
			d := r.Description()
			if r.NeedsDashboard() {
				d += " (needs your RevenueCat project)"
			}
			rows = append(rows, [2]string{r.ID(), d})
		}
		return rows
	}},
	{"migrators", func() (rows [][2]string) {
		for _, m := range migrate.All() {
			rows = append(rows, [2]string{m.Name(), m.Description()})
		}
		if len(rows) == 0 {
			rows = append(rows, [2]string{"(none yet)", "planned: " + strings.Join(migrate.Planned, ", ")})
		}
		return rows
	}},
	{"formats", func() (rows [][2]string) {
		for _, r := range report.All() {
			rows = append(rows, [2]string{r.Name(), r.Description()})
		}
		return rows
	}},
	{"commands", func() (rows [][2]string) {
		for _, c := range cli.All() {
			rows = append(rows, [2]string{c.Name(), c.Summary()})
		}
		return rows
	}},
}

func (c Command) Run(_ context.Context, args []string, stdout, stderr io.Writer) int {
	fs := shared.NewFlagSet(c, "[platforms|rules|migrators|formats|commands] ", stderr)
	pos, err := shared.Parse(fs, args)
	if err != nil {
		return shared.FlagExit(err)
	}
	var names []string
	for _, s := range sections {
		names = append(names, s.name)
	}
	want := map[string]bool{}
	for _, p := range pos {
		found := false
		for _, n := range names {
			found = found || n == p
		}
		if !found {
			fmt.Fprintf(stderr, "rc-doctor list: unknown section %q (choose from %s)\n", p, strings.Join(names, ", "))
			return cli.ExitError
		}
		want[p] = true
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 3, ' ', 0)
	first := true
	for _, s := range sections {
		if len(want) > 0 && !want[s.name] {
			continue
		}
		if !first {
			fmt.Fprintln(tw)
		}
		first = false
		fmt.Fprintf(tw, "%s:\n", s.name)
		for _, row := range s.rows() {
			fmt.Fprintf(tw, "  %s\t%s\n", row[0], row[1])
		}
	}
	if err := tw.Flush(); err != nil {
		return shared.Fail(stderr, err)
	}
	return cli.ExitOK
}
