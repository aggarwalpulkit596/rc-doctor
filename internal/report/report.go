// Package report renders an engine.Result.
//
// Reporters are plug-ins: implement Reporter, register it from init(), and it
// becomes available as --format <name>. Built in: text (people), json (scripts
// and AI agents) and sarif (GitHub code scanning and other CI annotations).
package report

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/registry"
)

// Reporter writes a Result in one format.
type Reporter interface {
	Name() string
	Description() string
	Write(w io.Writer, r *engine.Result) error
}

var reg = registry.New[Reporter]("reporter", Reporter.Name)

// Register adds a reporter. It panics on duplicate names.
func Register(r Reporter) { reg.Register(r) }

// Get returns a reporter by name.
func Get(name string) (Reporter, bool) { return reg.Get(name) }

// All returns registered reporters sorted by name.
func All() []Reporter { return reg.All() }

// Names returns the registered reporter names.
func Names() []string { return reg.Names() }

// Path turns a root-relative location into a path for display: relative to the
// working directory when the file is under it (so terminals and editors can
// open it), absolute otherwise.
func Path(root, rel string) string {
	if rel == "" {
		return ""
	}
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if wd, err := os.Getwd(); err == nil {
		if r, err := filepath.Rel(wd, abs); err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(r)
		}
	}
	return abs
}
