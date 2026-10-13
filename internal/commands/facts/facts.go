// Package facts implements `rc-doctor facts`: print, as JSON, what the
// platform modules found in the code. It is the debugging view behind every
// rule, and a stable input for scripts and AI agents. Keys are masked.
package facts

import (
	"context"
	"encoding/json"
	"io"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/commands/shared"
	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/redact"
)

func init() { cli.Register(Command{}) }

// Command is `rc-doctor facts`.
type Command struct{}

func (Command) Name() string    { return "facts" }
func (Command) Summary() string { return "Print what rc-doctor found in your code, as JSON" }

func (c Command) Run(_ context.Context, args []string, stdout, stderr io.Writer) int {
	fs := shared.NewFlagSet(c, "[dir] ", stderr)
	platforms := fs.String("platform", "", "only these platforms, comma-separated (default: auto-detect)")
	root, code, ok := shared.ParseRoot(fs, args, stderr)
	if !ok {
		return code
	}
	sf := shared.ScanFlags{Platforms: *platforms}
	col, err := engine.Collect(root, sf.PlatformList())
	if err != nil {
		return shared.Fail(stderr, err)
	}
	facts := make([]model.Facts, len(col.Facts))
	for i, f := range col.Facts {
		f.APIKeys = append([]model.APIKey(nil), f.APIKeys...)
		for j := range f.APIKeys {
			f.APIKeys[j].Value = redact.Keys(f.APIKeys[j].Value)
		}
		facts[i] = f
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(struct {
		Root      string        `json:"root"`
		Platforms []string      `json:"platforms"`
		Facts     []model.Facts `json:"facts"`
	}{col.Root, col.Platforms, facts}); err != nil {
		return shared.Fail(stderr, err)
	}
	return cli.ExitOK
}
