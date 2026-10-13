// Package version implements `rc-doctor version`.
package version

import (
	"context"
	"fmt"
	"io"

	"github.com/aggarwalpulkit596/rc-doctor/internal/buildinfo"
	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
)

func init() { cli.Register(Command{}) }

// Command is `rc-doctor version`.
type Command struct{}

func (Command) Name() string    { return "version" }
func (Command) Summary() string { return "Print the rc-doctor version" }

func (Command) Run(_ context.Context, _ []string, stdout, _ io.Writer) int {
	fmt.Fprintf(stdout, "rc-doctor %s\n", buildinfo.String())
	return cli.ExitOK
}
