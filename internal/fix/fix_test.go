package fix

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func edit(path string, line int, old, with string, s model.Safety) model.Edit {
	return model.Edit{Path: path, File: filepath.Base(path), Line: line, Old: old, New: with, Safety: s}
}

func TestApplyReplacesFirstOccurrenceAndKeepsMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "App.kt")
	if err := os.WriteFile(path, []byte("one\nconfigure(\"appl_x\") // appl_x\nthree\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Apply([]model.Edit{edit(path, 2, "appl_x", "goog_y", model.Safe)})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := testutil.ReadFile(t, path), "one\nconfigure(\"goog_y\") // appl_x\nthree\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if len(res.Applied) != 1 || res.Applied[0].Before != `configure("appl_x") // appl_x` || res.Applied[0].After != `configure("goog_y") // appl_x` {
		t.Errorf("Applied = %+v", res.Applied)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
	if matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".*rc-doctor*")); len(matches) != 0 {
		t.Errorf("temporary files left behind: %v", matches)
	}
}

func TestPreviewDoesNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.swift")
	const content = "let key = \"sk_x\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Preview([]model.Edit{edit(path, 1, "sk_x", "appl_y", model.Suggested)})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Applied) != 1 || res.Applied[0].After != `let key = "appl_y"` {
		t.Errorf("Preview = %+v", res)
	}
	if got := testutil.ReadFile(t, path); got != content {
		t.Errorf("Preview wrote the file: %q", got)
	}
}

func TestSkipsStaleOrInvalidEdits(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.kt")
	if err := os.WriteFile(path, []byte("val a = \"premium\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Apply([]model.Edit{
		edit(path, 1, `"premum"`, `"premium"`, model.Suggested), // already changed by hand
		edit(path, 7, "x", "y", model.Safe),                     // line out of range
		edit(path, 1, "", "y", model.Safe),                      // nothing to match
		edit(filepath.Join(dir, "missing.kt"), 1, "x", "y", model.Safe),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Applied) != 0 || len(res.Skipped) != 4 {
		t.Fatalf("want 0 applied and 4 skipped, got %+v", res)
	}
	if got := testutil.ReadFile(t, path); got != "val a = \"premium\"\n" {
		t.Errorf("file changed: %q", got)
	}
}

func TestSeveralEditsOnOneLineAndCRLF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.kt")
	if err := os.WriteFile(path, []byte("first\r\nent[\"premum\"] || off[\"defualt\"]\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Apply([]model.Edit{
		edit(path, 2, `"defualt"`, `"default"`, model.Suggested),
		edit(path, 2, `"premum"`, `"premium"`, model.Suggested),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Applied) != 2 {
		t.Fatalf("Applied = %+v", res)
	}
	if got, want := testutil.ReadFile(t, path), "first\r\nent[\"premium\"] || off[\"default\"]\r\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestSelect(t *testing.T) {
	safe := edit("/a", 1, "x", "y", model.Safe)
	suggested := edit("/a", 2, "p", "q", model.Suggested)
	in := []model.Edit{safe, suggested, safe}

	if got := Select(in, model.Safe); !reflect.DeepEqual(got, []model.Edit{safe}) {
		t.Errorf("Select(safe) = %+v", got)
	}
	if got := Select(in, model.Suggested); !reflect.DeepEqual(got, []model.Edit{safe, suggested}) {
		t.Errorf("Select(suggested) = %+v", got)
	}
}
