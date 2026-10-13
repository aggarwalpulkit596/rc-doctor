// Package dashboard loads the configured state of a RevenueCat project.
//
// Providers are swappable: the rc CLI for real projects, a JSON snapshot for
// tests, CI without credentials, and reproducible bug reports.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// Provider returns the dashboard state, or nil when running offline.
type Provider interface {
	Name() string
	Load(ctx context.Context) (*model.Dashboard, error)
}

// Offline is used when no dashboard source is configured. Rules that need the
// dashboard skip themselves and say so.
type Offline struct{}

func (Offline) Name() string                                   { return "offline" }
func (Offline) Load(context.Context) (*model.Dashboard, error) { return nil, nil }

// Static serves a dashboard that is already in memory: the state a previous run
// loaded (so `review --fix` re-checks against exactly the same project state),
// or fixtures in tests.
type Static struct {
	Label string
	Data  *model.Dashboard
}

func (s Static) Name() string {
	if s.Label == "" {
		return "static"
	}
	return s.Label
}

func (s Static) Load(context.Context) (*model.Dashboard, error) { return s.Data, nil }

// Snapshot reads a model.Dashboard from a JSON file, such as one written by
// `rc-doctor snapshot`. Useful in CI without rc credentials and in bug reports.
type Snapshot struct{ Path string }

func (s Snapshot) Name() string { return "snapshot:" + s.Path }

func (s Snapshot) Load(context.Context) (*model.Dashboard, error) {
	b, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, fmt.Errorf("read dashboard snapshot: %w", err)
	}
	var d model.Dashboard
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, fmt.Errorf("parse dashboard snapshot %s: %w", s.Path, err)
	}
	return &d, nil
}
