package engine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/dashboard"
	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/platform/android"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/platform/ios"
	_ "github.com/aggarwalpulkit596/rc-doctor/internal/platform/kmp"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

type failingProvider struct{}

func (failingProvider) Name() string { return "broken" }
func (failingProvider) Load(context.Context) (*model.Dashboard, error) {
	return nil, errors.New("not logged in")
}

func rulesOf(r *engine.Result) string {
	var ids []string
	for _, f := range r.Findings {
		ids = append(ids, f.Platform+":"+f.Rule)
	}
	return strings.Join(ids, " ")
}

func TestRunAgainstSnapshot(t *testing.T) {
	res, err := engine.Run(context.Background(), engine.Options{
		Root:      testutil.Testdata(t, "sample"),
		Dashboard: dashboard.Snapshot{Path: testutil.Testdata(t, "dashboard.json")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(res.Platforms, ","), "android,ios"; got != want {
		t.Errorf("platforms = %s, want %s", got, want)
	}
	// Sorted by platform, then file, then line.
	if got, want := rulesOf(res), "android:api-key android:unknown-entitlement ios:unknown-offering ios:api-key"; got != want {
		t.Errorf("findings = %s\nwant %s", got, want)
	}
	if res.Count(model.Error) != 4 || len(res.Skipped) != 0 || res.DashboardData == nil {
		t.Errorf("errors = %d, skipped = %v, dashboard = %v", res.Count(model.Error), res.Skipped, res.DashboardData)
	}
	if len(res.Edits()) != 3 {
		t.Errorf("edits = %d, want 3", len(res.Edits()))
	}
	for _, f := range res.Facts {
		if len(f.KeyPrefixes) == 0 {
			t.Errorf("%s facts have no key prefixes", f.Platform)
		}
	}
}

func TestRunOffline(t *testing.T) {
	res, err := engine.Run(context.Background(), engine.Options{
		Root:        testutil.Testdata(t, "sample"),
		OfflineNote: "no rc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := rulesOf(res), "android:api-key ios:api-key"; got != want {
		t.Errorf("findings = %s, want %s", got, want)
	}
	if len(res.Skipped) != 3 || res.Dashboard != "offline" || len(res.Notes) != 1 {
		t.Errorf("skipped = %v, dashboard = %s, notes = %v", res.Skipped, res.Dashboard, res.Notes)
	}
}

func TestDashboardFailure(t *testing.T) {
	opts := engine.Options{Root: testutil.Testdata(t, "sample"), Dashboard: failingProvider{}}

	res, err := engine.Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("a non-strict failure should fall back to offline: %v", err)
	}
	if res.DashboardData != nil || len(res.Notes) != 1 || !strings.Contains(res.Notes[0], "not logged in") {
		t.Errorf("notes = %v", res.Notes)
	}

	opts.StrictDashboard = true
	if _, err := engine.Run(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "not logged in") {
		t.Errorf("strict failure: err = %v", err)
	}
}

func TestCollectErrors(t *testing.T) {
	if _, err := engine.Collect(t.TempDir(), nil); !errors.Is(err, engine.ErrNoApp) {
		t.Errorf("empty dir: err = %v, want ErrNoApp", err)
	}
	if _, err := engine.Collect(t.TempDir()+"/missing", nil); err == nil || !strings.Contains(err.Error(), "doesn't exist") {
		t.Errorf("missing dir: err = %v", err)
	}
	if _, err := engine.Collect(testutil.Testdata(t, "dashboard.json"), nil); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("file: err = %v", err)
	}
	if _, err := engine.Collect(testutil.Testdata(t, "sample"), []string{"web"}); err == nil || !strings.Contains(err.Error(), `unknown platform "web"`) {
		t.Errorf("unknown platform: err = %v", err)
	}
}

func TestCollectNamedPlatform(t *testing.T) {
	col, err := engine.Collect(testutil.Testdata(t, "sample"), []string{"ios"})
	if err != nil {
		t.Fatal(err)
	}
	if len(col.Facts) != 1 || col.Facts[0].Platform != "ios" {
		t.Errorf("facts = %+v", col.Facts)
	}
}

func TestKeyPrefixes(t *testing.T) {
	got := engine.KeyPrefixes()
	if got["appl_"] != "ios" || got["goog_"] != "android" || got["amzn_"] != "android" {
		t.Errorf("KeyPrefixes = %v", got)
	}
}

func TestKMPSupersedesNativeModulesAndReportsSharedCodeOnce(t *testing.T) {
	res, err := engine.Run(context.Background(), engine.Options{
		Root:      testutil.Testdata(t, "kmp"),
		Dashboard: dashboard.Snapshot{Path: testutil.Testdata(t, "dashboard-kmp.json")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(res.Platforms, ","); got != "kmp" {
		t.Errorf("platforms = %s, want kmp alone", got)
	}
	// One commonMain typo, checked for both apps, reported once.
	if got := rulesOf(res); got != "android:unknown-entitlement" {
		t.Errorf("findings = %s", got)
	}
	for _, f := range res.Facts {
		if f.Platform == "ios" && strings.Join(f.KeyPrefixes, ",") != "appl_" {
			t.Errorf("the iOS app must be checked with iOS prefixes, got %v", f.KeyPrefixes)
		}
	}
	if engine.KeyPrefixes()["appl_"] != "ios" {
		t.Error("appl_ belongs to ios, not to the kmp umbrella")
	}
}
