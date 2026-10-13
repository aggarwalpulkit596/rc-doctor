// Package testutil has helpers shared by rc-doctor's tests.
package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Testdata returns the absolute path of elem under the repository's testdata directory.
func Testdata(t testing.TB, elem ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("testutil: runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "testdata")
	return filepath.Join(append([]string{root}, elem...)...)
}

// CopyTree copies the directory src into a new temporary directory and returns
// its path. Tests that apply fixes work on a copy, never on testdata itself.
func CopyTree(t testing.TB, src string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), filepath.Base(src))
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatalf("testutil: copy %s: %v", src, err)
	}
	return dst
}

// WriteFiles creates a temporary directory holding files, a map from
// slash-separated relative paths to contents, and returns its path.
func WriteFiles(t testing.TB, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// ReadFile returns the contents of path, failing the test on error.
func ReadFile(t testing.TB, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
