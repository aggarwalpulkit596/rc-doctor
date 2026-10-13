// Package shared holds the flags and helpers rc-doctor's commands have in
// common, so every command parses and reports the same way.
package shared

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/config"
	"github.com/aggarwalpulkit596/rc-doctor/internal/dashboard"
	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/fix"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/redact"
	"github.com/aggarwalpulkit596/rc-doctor/internal/report"
)

// NewFlagSet returns a flag set whose -h output names the command.
func NewFlagSet(c cli.Command, args string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: rc-doctor %s %s[flags]\n\n%s.\n\nFlags:\n", c.Name(), args, c.Summary())
		fs.PrintDefaults()
	}
	return fs
}

// Parse parses flags that may come before or after positional arguments, so
// "check ./app --format json" and "check --format json ./app" both work.
func Parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

// FlagExit is the exit code for a flag parsing error: 0 for -h, 2 otherwise.
// The flag package has already printed the message.
func FlagExit(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return cli.ExitOK
	}
	return cli.ExitError
}

// ParseRoot parses args and returns the directory to work on (default ".").
// When ok is false the command should return code.
func ParseRoot(fs *flag.FlagSet, args []string, stderr io.Writer) (root string, code int, ok bool) {
	pos, err := Parse(fs, args)
	if err != nil {
		return "", FlagExit(err), false
	}
	switch len(pos) {
	case 0:
		return ".", 0, true
	case 1:
		return pos[0], 0, true
	}
	fmt.Fprintf(stderr, "rc-doctor %s: expected at most one directory, got %q\n", fs.Name(), pos)
	return "", cli.ExitError, false
}

// ScanFlags select what to scan and which RevenueCat project to check against.
type ScanFlags struct {
	Platforms  string
	Dashboard  string
	ProjectID  string
	ConfigPath string
}

// BindScan registers --platform, --dashboard and --project-id.
func BindScan(fs *flag.FlagSet) *ScanFlags {
	f := &ScanFlags{}
	fs.StringVar(&f.Platforms, "platform", "", "only these platforms, comma-separated (default: auto-detect)")
	fs.StringVar(&f.Dashboard, "dashboard", "auto", "RevenueCat project source: auto, rc, offline, or a snapshot .json file")
	fs.StringVar(&f.ProjectID, "project-id", "", "RevenueCat project ID, passed to the rc CLI")
	fs.StringVar(&f.ConfigPath, "config", "", "config file (default: "+config.FileName+" in the checked directory, if present)")
	return f
}

// PlatformList returns the --platform values.
func (f *ScanFlags) PlatformList() []string {
	var out []string
	for _, p := range strings.Split(f.Platforms, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Options turns the flags into engine options for root.
//
//	auto     the rc CLI if it is installed, offline otherwise (the default)
//	rc       the rc CLI, and fail if it can't load the project
//	offline  no project: only the checks that work from code alone
//	<file>   a snapshot written by `rc-doctor snapshot`
func (f *ScanFlags) Options(root string) (engine.Options, error) {
	cfg, err := config.Load(root, f.ConfigPath)
	if err != nil {
		return engine.Options{}, err
	}
	opts := engine.Options{Root: root, Platforms: f.PlatformList(), Config: cfg}
	switch f.Dashboard {
	case "", "auto":
		if path, err := exec.LookPath("rc"); err == nil {
			opts.Dashboard = dashboard.RCCLI{Binary: path, ProjectID: f.ProjectID, Dir: root}
		} else {
			opts.OfflineNote = "The RevenueCat CLI (rc) isn't installed, so checks that need your project were skipped. Install it, or pass --dashboard with a snapshot file."
		}
	case "rc":
		opts.Dashboard = dashboard.RCCLI{ProjectID: f.ProjectID, Dir: root}
		opts.StrictDashboard = true
	case "offline", "none":
	default:
		opts.Dashboard = dashboard.Snapshot{Path: f.Dashboard}
		opts.StrictDashboard = true
	}
	return opts, nil
}

// Run runs the engine on root with these flags.
func (f *ScanFlags) Run(ctx context.Context, root string) (*engine.Result, error) {
	opts, err := f.Options(root)
	if err != nil {
		return nil, err
	}
	return engine.Run(ctx, opts)
}

// BindFormat registers --format with every registered reporter as a choice.
func BindFormat(fs *flag.FlagSet) *string {
	return fs.String("format", "text", "output format: "+strings.Join(report.Names(), ", "))
}

// Reporter returns the reporter for --format.
func Reporter(name string) (report.Reporter, error) {
	r, ok := report.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown format %q (available: %s)", name, strings.Join(report.Names(), ", "))
	}
	return r, nil
}

// BindFailOn registers --fail-on.
func BindFailOn(fs *flag.FlagSet) *string {
	return fs.String("fail-on", "error", "exit with code 1 when a finding is at least this severe: error, warning, info, never")
}

// Threshold parses --fail-on. "never" returns the empty severity.
func Threshold(s string) (model.Severity, error) {
	switch sev := model.Severity(s); sev {
	case model.Error, model.Warning, model.Info:
		return sev, nil
	case "never":
		return "", nil
	}
	return "", fmt.Errorf("unknown --fail-on value %q (use error, warning, info or never)", s)
}

// ExitCode returns ExitFindings when a finding reaches the threshold.
func ExitCode(findings []model.Finding, threshold model.Severity) int {
	if threshold == "" {
		return cli.ExitOK
	}
	for _, f := range findings {
		if f.Severity.Rank() >= threshold.Rank() {
			return cli.ExitFindings
		}
	}
	return cli.ExitOK
}

// Fail prints err and returns ExitError.
func Fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "rc-doctor: %v\n", err)
	return cli.ExitError
}

// PrintChanges shows each change as a small diff. Keys are masked.
func PrintChanges(w io.Writer, root string, changes []fix.Change) {
	for _, c := range changes {
		fmt.Fprintf(w, "  [%s] %s:%d\n", c.Edit.Safety, report.Path(root, c.Edit.File), c.Edit.Line)
		if c.Edit.Description != "" {
			fmt.Fprintf(w, "      %s\n", c.Edit.Description)
		}
		fmt.Fprintf(w, "      - %s\n", redact.Keys(strings.TrimSpace(c.Before)))
		fmt.Fprintf(w, "      + %s\n", redact.Keys(strings.TrimSpace(c.After)))
	}
}

// PrintSkipped lists edits that could not be applied.
func PrintSkipped(w io.Writer, root string, skipped []fix.Skipped) {
	for _, s := range skipped {
		fmt.Fprintf(w, "  %s:%d: %s\n", report.Path(root, s.Edit.File), s.Edit.Line, s.Reason)
	}
}

// CountSafety counts edits by safety level.
func CountSafety(edits []model.Edit) (safe, suggested int) {
	for _, e := range edits {
		if e.Safety == model.Safe {
			safe++
		} else {
			suggested++
		}
	}
	return safe, suggested
}

// Plural formats a count with a noun: "1 fix", "2 fixes".
func Plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
