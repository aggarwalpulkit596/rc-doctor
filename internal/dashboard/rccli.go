package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// RCCLI loads project state by running RevenueCat's own `rc` CLI with --json.
// Reusing rc means rc-doctor never handles credentials: login, profiles and
// project selection behave exactly as they do for rc itself.
//
// TODO(verify): the JSON field names below are tolerant guesses based on
// RevenueCat's REST API v2 objects. Confirm them against `rc schema` and
// tighten the parsing.
type RCCLI struct {
	Binary    string // defaults to "rc"
	ProjectID string // optional; passed through as --project-id

	// run replaces exec in tests.
	run func(ctx context.Context, args ...string) ([]byte, error)
}

func (r RCCLI) Name() string { return "rc-cli" }

// Load fetches entitlements and offerings (required) and apps with their
// public keys (best effort: when that fails the result carries a note and the
// checks that need apps do less).
func (r RCCLI) Load(ctx context.Context) (*model.Dashboard, error) {
	ents, err := r.list(ctx, "entitlements", "list")
	if err != nil {
		return nil, err
	}
	offs, err := r.list(ctx, "offerings", "list")
	if err != nil {
		return nil, err
	}
	d := &model.Dashboard{Entitlements: lookupKeys(ents), Offerings: lookupKeys(offs)}
	apps, notes, err := r.apps(ctx)
	if err != nil {
		d.Notes = append(d.Notes, fmt.Sprintf("Couldn't list apps with rc (%v); app and key checks were limited.", err))
	}
	d.Apps = apps
	d.Notes = append(d.Notes, notes...)
	return d, nil
}

// storePlatforms maps RevenueCat app types to rc-doctor platforms. Other app
// types (web billing, Stripe, the test store) have no native SDK to check.
var storePlatforms = map[string]string{
	"app_store":     "ios",
	"mac_app_store": "ios",
	"play_store":    "android",
	"amazon":        "android",
}

func (r RCCLI) apps(ctx context.Context) ([]model.App, []string, error) {
	items, err := r.list(ctx, "apps", "list")
	if err != nil {
		return nil, nil, err
	}
	var apps []model.App
	var notes []string
	for _, it := range items {
		typ := firstString(it, "type")
		platform, ok := storePlatforms[typ]
		if !ok {
			continue
		}
		a := model.App{ID: firstString(it, "id"), Platform: platform}
		if store, ok := it[typ].(map[string]any); ok {
			a.AppID = firstString(store, "bundle_id", "package_name")
		}
		if a.ID != "" {
			keys, err := r.list(ctx, "apps", "keys", a.ID)
			if err != nil {
				notes = append(notes, fmt.Sprintf("Couldn't list public keys for app %s (%v).", a.ID, err))
			}
			for _, k := range keys {
				if v := firstString(k, "key"); v != "" {
					a.PublicKeys = append(a.PublicKeys, v)
				}
			}
		}
		apps = append(apps, a)
	}
	return apps, notes, nil
}

// list runs an rc subcommand with --json and returns the listed objects.
func (r RCCLI) list(ctx context.Context, args ...string) ([]map[string]any, error) {
	out, err := r.exec(ctx, args...)
	if err != nil {
		return nil, err
	}
	its, err := items(out)
	if err != nil {
		// Other tools are called rc too (the npm "rc" config loader, the Plan 9
		// shell), so say which binary answered.
		return nil, fmt.Errorf("`%s %s --json` returned %v; is %s the RevenueCat CLI?", r.bin(), strings.Join(args, " "), err, r.bin())
	}
	return its, nil
}

func (r RCCLI) bin() string {
	if r.Binary == "" {
		return "rc"
	}
	return r.Binary
}

func (r RCCLI) exec(ctx context.Context, args ...string) ([]byte, error) {
	args = append(args, "--json")
	if r.ProjectID != "" {
		args = append(args, "--project-id", r.ProjectID)
	}
	if r.run != nil {
		return r.run(ctx, args...)
	}
	bin := r.bin()
	out, err := exec.CommandContext(ctx, bin, args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("%s %s failed: %s", bin, strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("could not run %s (is the RevenueCat CLI installed and logged in?): %w", bin, err)
	}
	return out, nil
}

// lookupKeys extracts the identifiers apps use in code. RevenueCat objects
// carry an internal id ("entl...") and the lookup_key the SDK sees.
func lookupKeys(items []map[string]any) []string {
	var out []string
	for _, it := range items {
		if v := firstString(it, "lookup_key", "identifier", "id"); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// items accepts a bare JSON array or an object wrapping one
// ({"items": [...]}, {"data": [...]}, {"results": [...]}).
func items(b []byte) ([]map[string]any, error) {
	var arr []map[string]any
	if err := json.Unmarshal(b, &arr); err == nil {
		return arr, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, fmt.Errorf("output that isn't JSON")
	}
	for _, k := range []string{"items", "data", "results"} {
		if raw, ok := obj[k]; ok {
			if err := json.Unmarshal(raw, &arr); err == nil {
				return arr, nil
			}
		}
	}
	return nil, fmt.Errorf("JSON without a list in it")
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}
