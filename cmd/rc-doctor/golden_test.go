package main

import (
	"bytes"
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/buildinfo"
	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/golden")

// TestGolden pins rc-doctor's output for each fixture and format. The JSON
// and SARIF shapes are a contract with scripts, agents and CI, so a change
// must show up as a diff in review. Run `go test ./cmd/rc-doctor -update` to
// accept a deliberate change.
func TestGolden(t *testing.T) {
	repo := filepath.Clean(testutil.Testdata(t, ".."))
	t.Chdir(repo) // paths in the output are relative to the working directory
	defer func(v string) { buildinfo.Version = v }(buildinfo.Version)
	buildinfo.Version = "v0.0.0-golden"

	sample := []string{"testdata/sample", "--dashboard", "testdata/dashboard.json"}
	legacy := []string{"testdata/android-legacy", "--dashboard", "testdata/dashboard-legacy.json"}
	kmp := []string{"testdata/kmp", "--dashboard", "testdata/dashboard-kmp.json"}
	for _, tc := range []struct {
		golden string
		args   []string
	}{
		{"check-sample.txt", append([]string{"check"}, sample...)},
		{"check-sample.json", append([]string{"check", "--format", "json"}, sample...)},
		{"check-sample.sarif", append([]string{"check", "--format", "sarif"}, sample...)},
		{"check-sample-offline.txt", []string{"check", "testdata/sample", "--dashboard", "offline"}},
		{"review-sample.txt", append([]string{"review"}, sample...)},
		{"check-android-legacy.txt", append([]string{"check"}, legacy...)},
		{"check-android-legacy.json", append([]string{"check", "--format", "json"}, legacy...)},
		{"check-kmp.txt", append([]string{"check"}, kmp...)},
	} {
		t.Run(tc.golden, func(t *testing.T) {
			var out, errOut bytes.Buffer
			cli.Main(context.Background(), tc.args, &out, &errOut)
			if errOut.Len() > 0 {
				t.Fatalf("unexpected stderr: %s", errOut.String())
			}
			got := strings.ReplaceAll(out.String(), repo, "$REPO")
			path := filepath.Join("testdata", "golden", tc.golden)
			if *update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run `go test ./cmd/rc-doctor -update` to create it)", err)
			}
			if got != string(want) {
				t.Errorf("output changed for rc-doctor %s.\nRun `go test ./cmd/rc-doctor -update` if the change is intended.\n--- got\n%s", strings.Join(tc.args, " "), got)
			}
		})
	}
}
