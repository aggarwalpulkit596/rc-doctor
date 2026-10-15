package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// RCCLI loads project state by running RevenueCat's own `rc` CLI. Reusing rc
// means rc-doctor never handles credentials: login, profiles, $RC_API_KEY and
// project selection behave exactly as they do for rc itself.
//
// It relies on these rc behaviors, checked against rc's source
// (github.com/RevenueCat/cli):
//   - `--json` prints {"data": <result>, "schema_version": 1}
//     (internal/output/output.go); a list result is a REST API v2 page,
//     {"items": [...], "next_page": <url or null>} (internal/api/client.go);
//   - `rc api GET <path>` prints the API's response as is, which is how
//     rc-doctor follows next_page: the list commands read one page only;
//   - `--no-input` fails instead of prompting, and with --json errors are
//     {"error": {"message", "hint", ...}} on stderr (internal/cli/run.go);
//   - exit codes: 1 error, 2 bad usage, 4 auth, 5 not found, 6 rate limited
//     (ExitCodeFor in internal/cli/runtime.go);
//   - the project comes from --project-id, then $RC_PROJECT_ID, then a
//     .revenuecat.json in the working directory's tree, then the profile.
//
// Object fields read: lookup_key and id (entitlements, offerings, packages),
// is_current (offerings), type plus app_store.bundle_id, mac_app_store.bundle_id,
// play_store.package_name and amazon.package_name (apps), key (public API
// keys), store_identifier and type (products; a package's product list nests
// each product under "product").
type RCCLI struct {
	Binary    string // defaults to "rc"
	ProjectID string // optional; passed through as --project-id
	Dir       string // where rc runs, so a .revenuecat.json in the checked repo picks the project

	// run replaces exec in tests.
	run func(ctx context.Context, args ...string) ([]byte, error)
}

// Name identifies the provider in output.
func (r RCCLI) Name() string { return "rc-cli" }

// maxPages stops a pagination loop that never ends.
const maxPages = 200

// Load fetches entitlements and offerings (required), then apps with their
// public keys and the catalog that links offerings, packages, products and
// entitlements (both best effort: a failure there adds a note, and the rules
// that need the missing part stay silent).
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

	catalog, err := r.catalog(ctx, ents, offs)
	if err != nil {
		d.Notes = append(d.Notes, fmt.Sprintf("Couldn't load offerings, packages and products with rc (%v); catalog checks were skipped.", err))
	}
	if catalog != nil {
		products, err := r.list(ctx, "products", "list")
		if err != nil {
			d.Notes = append(d.Notes, fmt.Sprintf("Couldn't list products with rc (%v); checks that depend on product types were limited.", err))
		} else {
			catalog.ProductTypes = map[string]string{}
			for _, p := range products {
				if id := str(p, "store_identifier"); id != "" {
					catalog.ProductTypes[id] = str(p, "type")
				}
			}
		}
	}
	d.Catalog = catalog
	return d, nil
}

// storePlatforms maps RevenueCat app types to rc-doctor platforms. The other
// app types (rc_billing, stripe, paddle, roku, test_store) have no native SDK
// for rc-doctor to check.
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
		typ := str(it, "type")
		platform, ok := storePlatforms[typ]
		if !ok {
			continue
		}
		a := model.App{ID: str(it, "id"), Platform: platform}
		if store, ok := it[typ].(map[string]any); ok {
			a.AppID = str(store, "bundle_id")
			if a.AppID == "" {
				a.AppID = str(store, "package_name")
			}
		}
		if a.ID != "" {
			keys, err := r.list(ctx, "apps", "keys", a.ID)
			if err != nil {
				notes = append(notes, fmt.Sprintf("Couldn't list public keys for app %s (%v).", a.ID, err))
			}
			for _, k := range keys {
				if v := str(k, "key"); v != "" {
					a.PublicKeys = append(a.PublicKeys, v)
				}
			}
		}
		apps = append(apps, a)
	}
	return apps, notes, nil
}

// catalog follows each entitlement to its products, and each offering to its
// packages and their products. Every rc command takes RevenueCat's internal
// object id, while code uses lookup keys, so both are kept.
func (r RCCLI) catalog(ctx context.Context, ents, offs []map[string]any) (*model.Catalog, error) {
	c := &model.Catalog{
		Packages:            map[string][]model.Package{},
		EntitlementProducts: map[string][]string{},
	}
	for _, e := range ents {
		key, id := str(e, "lookup_key"), str(e, "id")
		if key == "" || id == "" {
			continue
		}
		products, err := r.list(ctx, "entitlements", "products", id)
		if err != nil {
			return nil, err
		}
		c.EntitlementProducts[key] = storeIdentifiers(products)
	}
	for _, o := range offs {
		key, id := str(o, "lookup_key"), str(o, "id")
		if key == "" || id == "" {
			continue
		}
		if current, _ := o["is_current"].(bool); current {
			c.CurrentOffering = key
		}
		pkgs, err := r.list(ctx, "offerings", "packages", id)
		if err != nil {
			return nil, err
		}
		c.Packages[key] = []model.Package{}
		for _, p := range pkgs {
			pkg := model.Package{ID: str(p, "lookup_key"), Products: []string{}}
			if pid := str(p, "id"); pid != "" {
				products, err := r.list(ctx, "packages", "products", pid)
				if err != nil {
					return nil, err
				}
				pkg.Products = storeIdentifiers(products)
			}
			c.Packages[key] = append(c.Packages[key], pkg)
		}
	}
	return c, nil
}

// storeIdentifiers reads product identifiers from a product listing. A
// package's product list wraps each product ({"product": {...}}).
func storeIdentifiers(items []map[string]any) []string {
	out := []string{}
	for _, it := range items {
		if inner, ok := it["product"].(map[string]any); ok {
			it = inner
		}
		if v := str(it, "store_identifier"); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// list runs an rc list subcommand with --json and returns every listed
// object, following next_page through `rc api GET`.
func (r RCCLI) list(ctx context.Context, args ...string) ([]map[string]any, error) {
	command := strings.Join(args, " ")
	out, err := r.exec(ctx, args...)
	if err != nil {
		return nil, err
	}
	p, err := parsePage(out)
	if err != nil {
		// Other tools are called rc too (the npm "rc" config loader, the Plan 9
		// shell), so say which binary answered.
		return nil, fmt.Errorf("`%s %s --json` returned %v; is %s the RevenueCat CLI?", r.bin(), command, err, r.bin())
	}
	all := p.items
	for pages := 1; p.next != ""; pages++ {
		if pages >= maxPages {
			return nil, fmt.Errorf("`rc %s` still had more pages after %d", command, maxPages)
		}
		path, err := apiPath(p.next)
		if err != nil {
			return nil, fmt.Errorf("`rc %s` returned a next page rc-doctor can't follow (%q)", command, p.next)
		}
		out, err := r.exec(ctx, "api", "GET", path)
		if err != nil {
			return nil, err
		}
		next, err := parsePage(out)
		if err != nil {
			return nil, fmt.Errorf("`rc api GET %s` returned %v", path, err)
		}
		if next.next == p.next {
			return nil, fmt.Errorf("`rc api GET %s` returned the same next page again", path)
		}
		all = append(all, next.items...)
		p = next
	}
	return all, nil
}

// page is one page of a REST API v2 list.
type page struct {
	items []map[string]any
	next  string
}

// parsePage reads a list page from rc's --json envelope ({"data": page}) or
// from `rc api`, which prints the page itself.
func parsePage(b []byte) (page, error) {
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		return page{}, errors.New("output that isn't a JSON object")
	}
	if data, ok := v["data"].(map[string]any); ok {
		v = data
	}
	raw, ok := v["items"].([]any)
	if !ok {
		return page{}, errors.New("JSON without an items list")
	}
	p := page{items: make([]map[string]any, 0, len(raw))}
	for _, e := range raw {
		m, ok := e.(map[string]any)
		if !ok {
			return page{}, errors.New("an items list that holds something other than objects")
		}
		p.items = append(p.items, m)
	}
	p.next, _ = v["next_page"].(string)
	return p, nil
}

// apiPath turns a next_page value (a full URL or a /v2/... path) into the
// path `rc api` expects, which is relative to the v2 base URL.
func apiPath(next string) (string, error) {
	u, err := url.Parse(next)
	if err != nil {
		return "", err
	}
	p, ok := strings.CutPrefix(u.Path, "/v2/")
	if !ok {
		return "", errors.New("not a v2 API path")
	}
	path := "/" + p
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	return path, nil
}

func (r RCCLI) bin() string {
	if r.Binary == "" {
		return "rc"
	}
	return r.Binary
}

// exitMeanings explains rc's exit codes when rc itself gives no hint.
var exitMeanings = map[int]string{
	2: "bad usage (is this rc version too old for rc-doctor?)",
	4: "not authenticated: run `rc auth login`, or set RC_API_KEY",
	5: "not found: check --project-id",
	6: "rate limited by the RevenueCat API: try again shortly",
}

func (r RCCLI) exec(ctx context.Context, args ...string) ([]byte, error) {
	command := strings.Join(args, " ")
	args = append(args, "--json", "--no-input")
	if r.ProjectID != "" {
		args = append(args, "--project-id", r.ProjectID)
	}
	if r.run != nil {
		return r.run(ctx, args...)
	}
	bin := r.bin()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = r.Dir
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("`%s %s` failed (exit %d): %s", bin, command, ee.ExitCode(), explainExit(ee.ExitCode(), ee.Stderr))
		}
		return nil, fmt.Errorf("could not run %s (install the RevenueCat CLI with `npm install -g @revenuecat/cli` or `brew install RevenueCat/tap/rc`): %w", bin, err)
	}
	return out, nil
}

// explainExit describes a failed rc run from its exit code and stderr. With
// --json, rc writes {"error": {"message": ..., "hint": ...}} there; its own
// hint wins over rc-doctor's generic one.
func explainExit(code int, stderr []byte) string {
	msg, hint := rcError(stderr)
	if hint != "" {
		return strings.TrimSuffix(msg, ".") + ". " + hint
	}
	meaning, known := exitMeanings[code]
	switch {
	case known && msg != "":
		return meaning + " (" + msg + ")"
	case known:
		return meaning
	case msg != "":
		return msg
	}
	return "no error output"
}

// rcError reads rc's JSON error envelope, falling back to the raw text.
func rcError(stderr []byte) (msg, hint string) {
	var env struct {
		Error struct {
			Message string `json:"message"`
			Hint    string `json:"hint"`
		} `json:"error"`
	}
	text := strings.TrimSpace(string(stderr))
	if json.Unmarshal(stderr, &env) == nil && env.Error.Message != "" {
		return strings.TrimSpace(env.Error.Message), strings.TrimSpace(env.Error.Hint)
	}
	return text, ""
}

// lookupKeys extracts the identifiers apps use in code. RevenueCat objects
// carry an internal id ("entl...") and the lookup_key the SDK sees.
func lookupKeys(items []map[string]any) []string {
	var out []string
	for _, it := range items {
		if v := str(it, "lookup_key"); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// str returns m[key] when it is a non-empty string.
func str(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}
