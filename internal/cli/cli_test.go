package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

// recorder is a fake command that remembers how it was called.
type recorder struct {
	name string
	got  *[]string
}

func (r recorder) Name() string    { return r.name }
func (r recorder) Summary() string { return "fake " + r.name }
func (r recorder) Run(_ context.Context, args []string, stdout, _ io.Writer) int {
	*r.got = append([]string{r.name}, args...)
	io.WriteString(stdout, "ran "+r.name)
	return 7
}

func TestMainDispatch(t *testing.T) {
	var got []string
	for _, name := range []string{"check", "version", "fake"} {
		Register(recorder{name, &got})
		t.Cleanup(func() { Unregister(name) })
	}
	dir := t.TempDir()

	for _, tc := range []struct {
		args []string
		want string // the recorded call, "" when no command should run
		code int
	}{
		{nil, "check", 7},
		{[]string{"--format", "json"}, "check --format json", 7},
		{[]string{dir}, "check " + dir, 7},
		{[]string{"fake", "a", "-b"}, "fake a -b", 7},
		{[]string{"--version"}, "version", 7},
		{[]string{"help", "fake"}, "fake -h", 7},
		{[]string{"help"}, "", ExitOK},
		{[]string{"nope"}, "", ExitError},
	} {
		got = nil
		var out, errOut bytes.Buffer
		code := Main(context.Background(), tc.args, &out, &errOut)
		if code != tc.code || strings.Join(got, " ") != tc.want {
			t.Errorf("Main(%q) ran %q with code %d, want %q with %d", tc.args, got, code, tc.want, tc.code)
		}
	}
}

func TestUsageListsCommands(t *testing.T) {
	var got []string
	Register(recorder{"zz-cmd", &got})
	t.Cleanup(func() { Unregister("zz-cmd") })
	var b bytes.Buffer
	Usage(&b)
	if !strings.Contains(b.String(), "zz-cmd") || !strings.Contains(b.String(), "fake zz-cmd") {
		t.Errorf("usage doesn't list registered commands:\n%s", b.String())
	}
}
