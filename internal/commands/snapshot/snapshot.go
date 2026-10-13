// Package snapshot implements `rc-doctor snapshot`: save the RevenueCat
// project's state (entitlements, offerings, apps and their public keys) as
// JSON through the rc CLI.
//
// A snapshot lets CI run every check without rc credentials
// (`rc-doctor check --dashboard rc-project.json`), and makes a bug report
// reproducible. Public SDK keys are meant to ship inside apps, so a snapshot
// is safe to commit; it never contains secret keys.
package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/commands/shared"
	"github.com/aggarwalpulkit596/rc-doctor/internal/dashboard"
)

func init() { cli.Register(Command{}) }

// Command is `rc-doctor snapshot`.
type Command struct{}

func (Command) Name() string { return "snapshot" }

func (Command) Summary() string {
	return "Save your RevenueCat project's state as JSON, for CI and bug reports (uses the rc CLI)"
}

func (c Command) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := shared.NewFlagSet(c, "", stderr)
	projectID := fs.String("project-id", "", "RevenueCat project ID, passed to the rc CLI")
	out := fs.String("o", "", "write to this file instead of stdout")
	pos, err := shared.Parse(fs, args)
	if err != nil {
		return shared.FlagExit(err)
	}
	if len(pos) > 0 {
		return shared.Fail(stderr, fmt.Errorf("snapshot takes no arguments, got %q", pos))
	}
	d, err := dashboard.RCCLI{ProjectID: *projectID}.Load(ctx)
	if err != nil {
		return shared.Fail(stderr, err)
	}
	for _, n := range d.Notes {
		fmt.Fprintf(stderr, "note: %s\n", n)
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return shared.Fail(stderr, err)
	}
	b = append(b, '\n')
	if *out == "" {
		_, err = stdout.Write(b)
	} else {
		err = os.WriteFile(*out, b, 0o644)
	}
	if err != nil {
		return shared.Fail(stderr, err)
	}
	return cli.ExitOK
}
