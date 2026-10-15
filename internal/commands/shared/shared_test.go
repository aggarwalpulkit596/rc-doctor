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
	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/fix"
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
	auto := mustOptions(t, &ScanFlags{Dashboard: "auto", ProjectID: "p1"})
	if rc, ok := auto.Dashboard.(dashboard.RCCLI); !ok || rc.ProjectID != "p1" || rc.Dir == "" || auto.StrictDashboard {
		t.Errorf("auto with rc installed = %+v", auto)
	}

	t.Setenv("PATH", t.TempDir())
	offline := mustOptions(t, &ScanFlags{Dashboard: "auto"})
	if offline.Dashboard != nil || !strings.Contains(offline.OfflineNote, "isn't installed") {
		t.Errorf("auto without rc = %+v", offline)
	}

	if o := mustOptions(t, &ScanFlags{Dashboard: "rc"}); !o.StrictDashboard {
		t.Errorf("--dashboard rc must be strict: %+v", o)
	}
	if o := mustOptions(t, &ScanFlags{Dashboard: "offline"}); o.Dashboard != nil || o.OfflineNote != "" {
		t.Errorf("--dashboard offline = %+v", o)
	}
	snap := mustOptions(t, &ScanFlags{Dashboard: "project.json", Platforms: "ios, android"})
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

func mustOptions(t *testing.T, f *ScanFlags) engine.Options {
	t.Helper()
	opts, err := f.Options(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return opts
}

func TestConfigFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.json")
	if err := os.WriteFile(path, []byte(`{"rules": {"api-key": "warning"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	opts, err := (&ScanFlags{ConfigPath: path}).Options(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if sev, ok := opts.Config.Severity("api-key"); !ok || sev != model.Warning {
		t.Errorf("config not loaded: %+v", opts.Config)
	}
	if _, err := (&ScanFlags{ConfigPath: filepath.Join(dir, "missing.json")}).Options(dir); err == nil {
		t.Error("a missing --config file must fail")
	}
}

func TestPrintSkipped(t *testing.T) {
	var b strings.Builder
	PrintSkipped(&b, "/repo", []fix.Skipped{{
		Edit:   model.Edit{File: "app/App.kt", Line: 11},
		Reason: "the line changed since the scan",
	}})
	if got := b.String(); !strings.Contains(got, "app/App.kt:11: the line changed since the scan") {
		t.Errorf("PrintSkipped = %q", got)
	}
}
