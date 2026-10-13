// Command rc-doctor checks a RevenueCat integration against the app's code and
// its RevenueCat project, and fixes what it safely can.
//
//	rc-doctor                    check the current directory
//	rc-doctor review             preview fixes as diffs
//	rc-doctor review --fix       apply the safe ones
//	rc-doctor migrate            plan a move from another SDK
//
// See README.md for the full guide.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.Main(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
