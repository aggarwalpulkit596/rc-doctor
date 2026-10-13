// Package cli dispatches rc-doctor's subcommands.
//
// Commands are plug-ins: implement Command, register it from init(), and
// blank-import its package in cmd/rc-doctor/plugins.go.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/aggarwalpulkit596/rc-doctor/internal/registry"
)

// Exit codes shared by every command.
const (
	ExitOK       = 0 // nothing at or above the --fail-on threshold
	ExitFindings = 1 // findings at or above the threshold
	ExitError    = 2 // bad usage, or rc-doctor itself failed
)

// DefaultCommand runs when no command is named: `rc-doctor`, `rc-doctor ./app`.
const DefaultCommand = "check"

// Command is one subcommand.
type Command interface {
	Name() string
	Summary() string
	// Run parses its own flags and returns an exit code.
	Run(ctx context.Context, args []string, stdout, stderr io.Writer) int
}

var reg = registry.New[Command]("command", Command.Name)

// Register adds a command. It panics on duplicate names.
func Register(c Command) { reg.Register(c) }

// Unregister removes a command. It exists for tests.
func Unregister(name string) { reg.Unregister(name) }

// Lookup returns a command by name.
func Lookup(name string) (Command, bool) { return reg.Get(name) }

// All returns registered commands sorted by name.
func All() []Command { return reg.All() }

// Main runs rc-doctor with args (without the program name).
func Main(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return run(ctx, DefaultCommand, nil, stdout, stderr)
	}
	switch first := args[0]; {
	case first == "help" || first == "-h" || first == "-help" || first == "--help":
		if first == "help" && len(args) > 1 {
			if c, ok := Lookup(args[1]); ok {
				return c.Run(ctx, []string{"-h"}, stdout, stderr)
			}
		}
		Usage(stdout)
		return ExitOK
	case first == "-v" || first == "-version" || first == "--version":
		return run(ctx, "version", nil, stdout, stderr)
	case strings.HasPrefix(first, "-"):
		return run(ctx, DefaultCommand, args, stdout, stderr)
	}
	if c, ok := Lookup(args[0]); ok {
		return c.Run(ctx, args[1:], stdout, stderr)
	}
	if fi, err := os.Stat(args[0]); err == nil && fi.IsDir() {
		return run(ctx, DefaultCommand, args, stdout, stderr)
	}
	fmt.Fprintf(stderr, "rc-doctor: unknown command or directory %q\n\n", args[0])
	Usage(stderr)
	return ExitError
}

func run(ctx context.Context, name string, args []string, stdout, stderr io.Writer) int {
	c, ok := Lookup(name)
	if !ok {
		fmt.Fprintf(stderr, "rc-doctor: command %q is not built into this binary\n", name)
		return ExitError
	}
	return c.Run(ctx, args, stdout, stderr)
}

// Usage prints the top-level help.
func Usage(w io.Writer) {
	fmt.Fprint(w, `rc-doctor checks a RevenueCat integration against your code and your
RevenueCat project, and fixes what it safely can.

Usage:
  rc-doctor [command] [dir] [flags]

Commands:
`)
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	for _, c := range All() {
		fmt.Fprintf(tw, "  %s\t%s\n", c.Name(), c.Summary())
	}
	tw.Flush()
	fmt.Fprintf(w, `
With no command, rc-doctor runs %q on the current directory.
Run "rc-doctor help <command>" for a command's flags.
`, DefaultCommand)
}
