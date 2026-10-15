package dashboard

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func TestSnapshot(t *testing.T) {
	d, err := Snapshot{Path: testutil.Testdata(t, "dashboard.json")}.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.Entitlements, []string{"premium"}) || len(d.Apps) != 2 || d.AppsFor("ios")[0].PublicKeys[0] != "appl_AbCdEfGhIjKlMnOpQrSt" {
		t.Errorf("snapshot = %+v", d)
	}

	legacy, err := Snapshot{Path: testutil.Testdata(t, "dashboard-legacy.json")}.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if c := legacy.Catalog; c == nil || c.CurrentOffering != "" || len(c.Packages["seasonal"]) != 0 || len(c.Packages["default"]) != 1 {
		t.Errorf("catalog = %+v", legacy.Catalog)
	}

	if _, err := (Snapshot{Path: filepath.Join(t.TempDir(), "missing.json")}).Load(context.Background()); err == nil {
		t.Error("missing snapshot should fail")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (Snapshot{Path: bad}).Load(context.Background()); err == nil {
		t.Error("invalid snapshot should fail")
	}
}

func TestOfflineAndStatic(t *testing.T) {
	if d, err := (Offline{}).Load(context.Background()); d != nil || err != nil {
		t.Errorf("Offline = %v, %v", d, err)
	}
	data := &model.Dashboard{Entitlements: []string{"pro"}}
	s := Static{Label: "snapshot:x.json", Data: data}
	if d, _ := s.Load(context.Background()); d != data || s.Name() != "snapshot:x.json" {
		t.Errorf("Static = %v (%s)", d, s.Name())
	}
}

// fakeRC answers rc subcommands from a table keyed by the arguments before
// the flags rc-doctor always adds.
func fakeRC(answers map[string]string, calls *[]string) func(context.Context, ...string) ([]byte, error) {
	return func(_ context.Context, args ...string) ([]byte, error) {
		key := strings.Join(args, " ")
		*calls = append(*calls, key)
		for _, suffix := range []string{" --json --no-input --project-id proj1", " --json --no-input"} {
			if cmd, ok := strings.CutSuffix(key, suffix); ok {
				if out, ok := answers[cmd]; ok {
					return []byte(out), nil
				}
				break
			}
		}
		return nil, errors.New("unexpected call: rc " + key)
	}
}

// The shapes below follow rc's --json envelope, {"data": <page>}, with REST API
// v2 objects (see the RCCLI doc comment for where each is defined).
var project = map[string]string{
	"entitlements list":           `{"data":{"items":[{"id":"entl1","lookup_key":"premium"},{"id":"entl2","lookup_key":"legacy"}]}}`,
	"offerings list":              `{"data":{"items":[{"id":"ofrng1","lookup_key":"default","is_current":true},{"id":"ofrng2","lookup_key":"annual","is_current":false}]}}`,
	"apps list":                   `{"data":{"items":[{"id":"app1","type":"app_store","app_store":{"bundle_id":"com.example.app"}},{"id":"app2","type":"play_store","play_store":{"package_name":"com.example.app"}},{"id":"app3","type":"stripe"}]}}`,
	"apps keys app1":              `{"data":{"items":[{"key":"appl_KEY1234567890abcdef"}]}}`,
	"apps keys app2":              `{"data":{"items":[{"key":"goog_KEY1234567890abcdef"}]}}`,
	"entitlements products entl1": `{"data":{"items":[{"id":"prod1","store_identifier":"premium_monthly"}]}}`,
	"entitlements products entl2": `{"data":{"items":[]}}`,
	"offerings packages ofrng1":   `{"data":{"items":[{"id":"pkge1","lookup_key":"$rc_monthly"}]}}`,
	"offerings packages ofrng2":   `{"data":{"items":[]}}`,
	"packages products pkge1":     `{"data":{"items":[{"product":{"id":"prod1","store_identifier":"premium_monthly"},"eligibility_criteria":"all"}]}}`,
	"products list":               `{"data":{"items":[{"id":"prod1","store_identifier":"premium_monthly","type":"subscription"},{"id":"prod2","store_identifier":"tip_small","type":"consumable"}]}}`,
}

func TestRCCLILoadsProject(t *testing.T) {
	var calls []string
	rc := RCCLI{ProjectID: "proj1", run: fakeRC(project, &calls)}

	d, err := rc.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := &model.Dashboard{
		Entitlements: []string{"premium", "legacy"},
		Offerings:    []string{"default", "annual"},
		Apps: []model.App{
			{ID: "app1", Platform: "ios", AppID: "com.example.app", PublicKeys: []string{"appl_KEY1234567890abcdef"}},
			{ID: "app2", Platform: "android", AppID: "com.example.app", PublicKeys: []string{"goog_KEY1234567890abcdef"}},
		},
		Catalog: &model.Catalog{
			CurrentOffering: "default",
			Packages: map[string][]model.Package{
				"default": {{ID: "$rc_monthly", Products: []string{"premium_monthly"}}},
				"annual":  {},
			},
			EntitlementProducts: map[string][]string{"premium": {"premium_monthly"}, "legacy": {}},
			ProductTypes:        map[string]string{"premium_monthly": "subscription", "tip_small": "consumable"},
		},
	}
	if !reflect.DeepEqual(d, want) {
		t.Errorf("Load = %+v\nwant %+v", d, want)
	}
	for _, c := range calls {
		if !strings.HasSuffix(c, "--json --no-input --project-id proj1") {
			t.Errorf("rc must always get --json --no-input and the project: %q", c)
		}
	}
}

func TestRCCLIPartialLoads(t *testing.T) {
	var calls []string
	empty := map[string]string{
		"entitlements list": `{"data":{"object":"list","items":[],"next_page":null},"schema_version":1}`,
		"offerings list":    `{"data":{"items":[]}}`,
	}
	d, err := RCCLI{run: fakeRC(empty, &calls)}.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	notes := strings.Join(d.Notes, "\n")
	for _, want := range []string{"Couldn't list apps", "Couldn't list products"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes are missing %q: %q", want, notes)
		}
	}
	if len(d.Apps) != 0 {
		t.Errorf("apps = %+v", d.Apps)
	}
	if d.Catalog == nil {
		t.Error("an empty project still has a (empty) catalog")
	}

	broken := withAnswers(map[string]string{"offerings packages ofrng2": ""})
	d, err = RCCLI{ProjectID: "proj1", run: fakeRC(broken, &calls)}.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.Catalog != nil || !strings.Contains(strings.Join(d.Notes, " "), "catalog checks were skipped") {
		t.Errorf("a failed catalog call must drop the whole catalog, not leave half of it: %+v %q", d.Catalog, d.Notes)
	}
}

// withAnswers copies project, replacing answers; an empty answer removes one.
func withAnswers(changes map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range project {
		out[k] = v
	}
	for k, v := range changes {
		if v == "" {
			delete(out, k)
			continue
		}
		out[k] = v
	}
	return out
}

func TestRCCLIFollowsPages(t *testing.T) {
	var calls []string
	paged := withAnswers(map[string]string{
		// rc's list commands print one page; next_page is a full URL or a /v2 path.
		"offerings list": `{"data":{"items":[{"id":"ofrng1","lookup_key":"default","is_current":true}],"next_page":"https://api.revenuecat.com/v2/projects/proj1/offerings?starting_after=ofrng1&limit=1"}}`,
		"api GET /projects/proj1/offerings?starting_after=ofrng1&limit=1": `{"object":"list","items":[{"id":"ofrng2","lookup_key":"annual","is_current":false}],"next_page":"/v2/projects/proj1/offerings?starting_after=ofrng2&limit=1"}`,
		"api GET /projects/proj1/offerings?starting_after=ofrng2&limit=1": `{"object":"list","items":[],"next_page":null}`,
	})
	d, err := RCCLI{ProjectID: "proj1", run: fakeRC(paged, &calls)}.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.Offerings, []string{"default", "annual"}) || d.Catalog == nil || len(d.Catalog.Packages) != 2 {
		t.Errorf("offerings = %v, catalog = %+v", d.Offerings, d.Catalog)
	}

	for name, tc := range map[string]struct {
		answers map[string]string
		want    string
	}{
		"a next page that never changes": {withAnswers(map[string]string{
			"entitlements list": `{"data":{"items":[],"next_page":"/v2/projects/proj1/entitlements?starting_after=x"}}`,
			"api GET /projects/proj1/entitlements?starting_after=x": `{"items":[],"next_page":"/v2/projects/proj1/entitlements?starting_after=x"}`,
		}), "returned the same next page again"},
		"a next page that isn't an API path": {withAnswers(map[string]string{
			"entitlements list": `{"data":{"items":[],"next_page":"cursor123"}}`,
		}), "can't follow"},
		"a later page that fails": {withAnswers(map[string]string{
			"entitlements list": `{"data":{"items":[],"next_page":"/v2/projects/proj1/entitlements?starting_after=x"}}`,
		}), "unexpected call: rc api GET /projects/proj1/entitlements?starting_after=x"},
	} {
		// Entitlements are required: a list rc-doctor can't read to the end
		// fails the load instead of reporting IDs on later pages as missing.
		if _, err := (RCCLI{ProjectID: "proj1", run: fakeRC(tc.answers, &calls)}).Load(context.Background()); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tc.want)
		}
	}
}

func TestRCCLIErrors(t *testing.T) {
	var calls []string
	failing := RCCLI{run: fakeRC(map[string]string{}, &calls)}
	if _, err := failing.Load(context.Background()); err == nil {
		t.Error("entitlements are required")
	}

	// The npm "rc" package echoes its arguments as JSON. That must read as
	// "not the RevenueCat CLI", not as an empty project.
	impostor := RCCLI{Binary: "/usr/bin/rc", run: fakeRC(map[string]string{
		"entitlements list": `{"_":["entitlements","list"],"json":true}`,
	}, &calls)}
	_, err := impostor.Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "is /usr/bin/rc the RevenueCat CLI?") {
		t.Errorf("err = %v", err)
	}
}

func TestRCCLIExitCodes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script as a stand-in rc")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	bin := filepath.Join(t.TempDir(), "rc")
	script := "#!/bin/sh\necho 'Not logged in.' >&2\nexit 4\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := RCCLI{Binary: bin}.Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "not authenticated: run `rc auth login`, or set RC_API_KEY (Not logged in.)") {
		t.Errorf("err = %v", err)
	}

	// rc runs in the checked repository, where its .revenuecat.json picks the
	// project. This stand-in only answers from files in its working directory.
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "list.json"), []byte(`{"data":{"items":[]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	reader := filepath.Join(t.TempDir(), "rc")
	if err := os.WriteFile(reader, []byte("#!/bin/sh\ncase \"$1 $2\" in\n\"entitlements list\"|\"offerings list\") cat list.json ;;\n*) exit 1 ;;\nesac\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := (RCCLI{Binary: reader, Dir: repo}).Load(context.Background()); err != nil {
		t.Errorf("rc must run in Dir: %v", err)
	}

	_, err = RCCLI{Binary: filepath.Join(t.TempDir(), "missing-rc")}.Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "npm install -g @revenuecat/cli") {
		t.Errorf("a missing binary should say how to install rc: %v", err)
	}
}

func TestExplainExit(t *testing.T) {
	for _, tc := range []struct {
		code   int
		stderr string
		want   string
	}{
		{6, "", "rate limited by the RevenueCat API: try again shortly"},
		{1, "boom", "boom"},
		{1, "", "no error output"},
		// With --json, rc reports errors as JSON on stderr; its own hint wins.
		{4, `{"error":{"type":"unauthorized","message":"not authenticated","exit_code":4,"hint":"Run ` + "`rc login`" + ` or set RC_API_KEY."},"schema_version":1}`, "not authenticated. Run `rc login` or set RC_API_KEY."},
		{5, `{"error":{"type":"resource_missing","message":"Project not found","exit_code":5},"schema_version":1}`, "not found: check --project-id (Project not found)"},
	} {
		if got := explainExit(tc.code, []byte(tc.stderr)); got != tc.want {
			t.Errorf("explainExit(%d, %q) = %q, want %q", tc.code, tc.stderr, got, tc.want)
		}
	}
}
