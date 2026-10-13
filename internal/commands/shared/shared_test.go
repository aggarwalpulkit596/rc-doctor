package shared

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/dashboard"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func TestParseAllowsFlagsAfterPositionals(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "")
	fix := fs.Bool("fix", false, "")
	pos, err := Parse(fs, []string{"./app", "--format", "json", "--fix"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pos, []string{"./app"}) || *format != "json" || !*fix {
		t.Errorf("pos = %q, format = %s, fix = %v", pos, *format, *fix)
	}
}

func TestParseRoot(t *testing.T) {
	newFS := func() *flag.FlagSet {
		fs := flag.NewFlagSet("check", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		return fs
	}
	if root, _, ok := ParseRoot(newFS(), nil, io.Discard); !ok || root != "." {
		t.Errorf("default root = %q, %v", root, ok)
	}
	if _, code, ok := ParseRoot(newFS(), []string{"a", "b"}, io.Discard); ok || code != cli.ExitError {
		t.Errorf("two roots: code = %d, ok = %v", code, ok)
	}
	if _, code, ok := ParseRoot(newFS(), []string{"-h"}, io.Discard); ok || code != cli.ExitOK {
		t.Errorf("-h: code = %d, ok = %v", code, ok)
	}
	if _, code, ok := ParseRoot(newFS(), []string{"--nope"}, io.Discard); ok || code != cli.ExitError {
		t.Errorf("unknown flag: code = %d, ok = %v", code, ok)
	}
}

func TestDashboardModes(t *testing.T) {
	// A fake rc on PATH stands in for the RevenueCat CLI.
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "rc"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", bin)
	auto := (&ScanFlags{Dashboard: "auto", ProjectID: "p1"}).Options(".")
	if rc, ok := auto.Dashboard.(dashboard.RCCLI); !ok || rc.ProjectID != "p1" || auto.StrictDashboard {
		t.Errorf("auto with rc installed = %+v", auto)
	}

	t.Setenv("PATH", t.TempDir())
	offline := (&ScanFlags{Dashboard: "auto"}).Options(".")
	if offline.Dashboard != nil || !strings.Contains(offline.OfflineNote, "isn't installed") {
		t.Errorf("auto without rc = %+v", offline)
	}

	if o := (&ScanFlags{Dashboard: "rc"}).Options("."); !o.StrictDashboard {
		t.Errorf("--dashboard rc must be strict: %+v", o)
	}
	if o := (&ScanFlags{Dashboard: "offline"}).Options("."); o.Dashboard != nil || o.OfflineNote != "" {
		t.Errorf("--dashboard offline = %+v", o)
	}
	snap := (&ScanFlags{Dashboard: "project.json", Platforms: "ios, android"}).Options(".")
	if s, ok := snap.Dashboard.(dashboard.Snapshot); !ok || s.Path != "project.json" || !snap.StrictDashboard {
		t.Errorf("snapshot = %+v", snap)
	}
	if !reflect.DeepEqual(snap.Platforms, []string{"ios", "android"}) {
		t.Errorf("platforms = %q", snap.Platforms)
	}
}

func TestThresholdAndExitCode(t *testing.T) {
	warning := []model.Finding{{Severity: model.Warning}}
	for _, tc := range []struct {
		failOn string
		want   int
	}{
		{"error", cli.ExitOK},
		{"warning", cli.ExitFindings},
		{"info", cli.ExitFindings},
		{"never", cli.ExitOK},
	} {
		th, err := Threshold(tc.failOn)
		if err != nil {
			t.Fatal(err)
		}
		if got := ExitCode(warning, th); got != tc.want {
			t.Errorf("--fail-on %s with a warning: exit %d, want %d", tc.failOn, got, tc.want)
		}
	}
	if _, err := Threshold("sometimes"); err == nil {
		t.Error("unknown --fail-on value should fail")
	}
}
