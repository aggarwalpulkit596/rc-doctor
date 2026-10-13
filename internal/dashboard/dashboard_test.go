package dashboard

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

// fakeRC answers rc subcommands from a table keyed by the joined arguments.
func fakeRC(t *testing.T, answers map[string]string, calls *[]string) func(context.Context, ...string) ([]byte, error) {
	return func(_ context.Context, args ...string) ([]byte, error) {
		key := strings.Join(args, " ")
		*calls = append(*calls, key)
		out, ok := answers[key]
		if !ok {
			return nil, errors.New("unexpected call: rc " + key)
		}
		return []byte(out), nil
	}
}

func TestRCCLILoadsProject(t *testing.T) {
	var calls []string
	rc := RCCLI{ProjectID: "proj1", run: fakeRC(t, map[string]string{
		"entitlements list --json --project-id proj1": `[{"id":"entl1","lookup_key":"premium"}]`,
		"offerings list --json --project-id proj1":    `{"items":[{"id":"ofrng1","lookup_key":"default"},{"id":"ofrng2","lookup_key":"annual"}]}`,
		"apps list --json --project-id proj1": `{"items":[
			{"id":"app1","type":"app_store","app_store":{"bundle_id":"com.example.app"}},
			{"id":"app2","type":"play_store","play_store":{"package_name":"com.example.app"}},
			{"id":"app3","type":"stripe"}
		]}`,
		"apps keys app1 --json --project-id proj1": `{"items":[{"key":"appl_KEY1234567890abcdef"}]}`,
		"apps keys app2 --json --project-id proj1": `[{"key":"goog_KEY1234567890abcdef"}]`,
	}, &calls)}

	d, err := rc.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := &model.Dashboard{
		Entitlements: []string{"premium"},
		Offerings:    []string{"default", "annual"},
		Apps: []model.App{
			{ID: "app1", Platform: "ios", AppID: "com.example.app", PublicKeys: []string{"appl_KEY1234567890abcdef"}},
			{ID: "app2", Platform: "android", AppID: "com.example.app", PublicKeys: []string{"goog_KEY1234567890abcdef"}},
		},
	}
	if !reflect.DeepEqual(d, want) {
		t.Errorf("Load = %+v\nwant %+v", d, want)
	}
	if len(calls) != 5 {
		t.Errorf("calls = %q", calls)
	}
}

func TestRCCLIAppsAreBestEffort(t *testing.T) {
	var calls []string
	rc := RCCLI{run: fakeRC(t, map[string]string{
		"entitlements list --json": `[]`,
		"offerings list --json":    `[]`,
	}, &calls)}
	d, err := rc.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Apps) != 0 || len(d.Notes) != 1 || !strings.Contains(d.Notes[0], "Couldn't list apps") {
		t.Errorf("Load = %+v", d)
	}
}

func TestRCCLIErrors(t *testing.T) {
	var calls []string
	failing := RCCLI{run: fakeRC(t, map[string]string{}, &calls)}
	if _, err := failing.Load(context.Background()); err == nil {
		t.Error("entitlements are required")
	}

	// The npm "rc" package echoes its arguments as JSON. That must read as
	// "not the RevenueCat CLI", not as an empty project.
	impostor := RCCLI{Binary: "/usr/bin/rc", run: fakeRC(t, map[string]string{
		"entitlements list --json": `{"_":["entitlements","list"],"json":true}`,
	}, &calls)}
	_, err := impostor.Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "is /usr/bin/rc the RevenueCat CLI?") {
		t.Errorf("err = %v", err)
	}
}
