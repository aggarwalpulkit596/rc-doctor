// Package fix applies model.Edits to files.
//
// Every module that changes user code (rule fixes today; migrations and
// generators later) emits Edits and goes through this one engine, so the
// safety checks live in one place:
//   - an edit applies only if its expected text is still on its line, so a
//     file changed since the scan is skipped rather than corrupted;
//   - files are replaced atomically and keep their permissions;
//   - Preview shows the exact change without writing anything.
package fix

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// Change is one edit applied (or previewed), with the line before and after.
type Change struct {
	Edit          model.Edit
	Before, After string
}

// Skipped is an edit that could not be applied, and why.
type Skipped struct {
	Edit   model.Edit
	Reason string
}

// Result of applying or previewing edits.
type Result struct {
	Applied []Change
	Skipped []Skipped
}

// Select keeps the edits allowed at level and drops exact duplicates.
func Select(edits []model.Edit, level model.Safety) []model.Edit {
	type key struct {
		path     string
		line     int
		old, new string
	}
	seen := map[key]bool{}
	var out []model.Edit
	for _, e := range edits {
		k := key{e.Path, e.Line, e.Old, e.New}
		if !e.Safety.AllowedAt(level) || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, e)
	}
	return out
}

// Preview computes what Apply would change, without writing.
func Preview(edits []model.Edit) (Result, error) { return run(edits, false) }

// Apply writes the edits to disk.
func Apply(edits []model.Edit) (Result, error) { return run(edits, true) }

func run(edits []model.Edit, write bool) (Result, error) {
	var res Result
	byPath := map[string][]model.Edit{}
	var paths []string
	for _, e := range edits {
		if _, ok := byPath[e.Path]; !ok {
			paths = append(paths, e.Path)
		}
		byPath[e.Path] = append(byPath[e.Path], e)
	}
	sort.Strings(paths)

	for _, path := range paths {
		es := byPath[path]
		sort.SliceStable(es, func(i, j int) bool { return es[i].Line < es[j].Line })
		info, err := os.Stat(path)
		if err != nil {
			for _, e := range es {
				res.Skipped = append(res.Skipped, Skipped{e, err.Error()})
			}
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			for _, e := range es {
				res.Skipped = append(res.Skipped, Skipped{e, err.Error()})
			}
			continue
		}
		lines := strings.SplitAfter(string(content), "\n")
		changed := false
		for _, e := range es {
			switch {
			case e.Old == "":
				res.Skipped = append(res.Skipped, Skipped{e, "edit has no expected text"})
				continue
			case e.Line < 1 || e.Line > len(lines):
				res.Skipped = append(res.Skipped, Skipped{e, fmt.Sprintf("line %d is out of range", e.Line)})
				continue
			}
			before := lines[e.Line-1]
			if !strings.Contains(before, e.Old) {
				res.Skipped = append(res.Skipped, Skipped{e, "expected text not found (the file changed since the scan)"})
				continue
			}
			after := strings.Replace(before, e.Old, e.New, 1)
			lines[e.Line-1] = after
			changed = true
			res.Applied = append(res.Applied, Change{Edit: e, Before: trimEOL(before), After: trimEOL(after)})
		}
		if changed && write {
			if err := writeAtomic(path, strings.Join(lines, ""), info.Mode().Perm()); err != nil {
				return res, fmt.Errorf("write %s: %w", path, err)
			}
		}
	}
	return res, nil
}

// writeAtomic replaces path via a temporary file in the same directory, so an
// interrupted run never leaves a half-written source file.
func writeAtomic(path, content string, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".rc-doctor-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func trimEOL(s string) string { return strings.TrimRight(s, "\r\n") }
