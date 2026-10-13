// Package check implements `rc-doctor check`, the default command: run every
// rule and report the findings. It never changes files, which makes it the
// command for CI.
package check

import (
	"context"
	"io"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/commands/shared"
)

func init() { cli.Register(Command{}) }

// Command is `rc-doctor check`.
type Command struct{}

func (Command) Name() string { return "check" }

func (Command) Summary() string {
	return "Check the RevenueCat integration and report problems (the default command)"
}

func (c Command) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := shared.NewFlagSet(c, "[dir] ", stderr)
	scan := shared.BindScan(fs)
	format := shared.BindFormat(fs)
	failOn := shared.BindFailOn(fs)
	root, code, ok := shared.ParseRoot(fs, args, stderr)
	if !ok {
		return code
	}
	rep, err := shared.Reporter(*format)
	if err != nil {
		return shared.Fail(stderr, err)
	}
	threshold, err := shared.Threshold(*failOn)
	if err != nil {
		return shared.Fail(stderr, err)
	}
	res, err := scan.Run(ctx, root)
	if err != nil {
		return shared.Fail(stderr, err)
	}
	if err := rep.Write(stdout, res); err != nil {
		return shared.Fail(stderr, err)
	}
	return shared.ExitCode(res.Findings, threshold)
}
