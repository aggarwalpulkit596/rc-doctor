package engine_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/config"
	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func runWith(t *testing.T, root string, cfg *config.Config) *engine.Result {
	t.Helper()
	res, err := engine.Run(context.Background(), engine.Options{Root: root, Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func edit(t *testing.T, path, old, new string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), old) {
		t.Fatalf("%s doesn't contain %q", path, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(b), old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInlineIgnoreComments(t *testing.T) {
	dir := testutil.CopyTree(t, testutil.Testdata(t, "sample"))
	// Same line, naming the rule, with a reason.
	edit(t, filepath.Join(dir, "android/app/src/main/java/com/example/paywall/App.kt"),
		`.build())`, `.build()) // rc-doctor:ignore api-key -- testing the wrong key on purpose`)
	// Line above, every rule.
	edit(t, filepath.Join(dir, "ios/Paywall/PaywallApp.swift"),
		`        let apiKey = "sk_`, "        // rc-doctor:ignore\n        let apiKey = \"sk_")

	res := runWith(t, dir, nil)
	if len(res.Findings) != 0 || len(res.Suppressed) != 2 {
		t.Fatalf("findings = %+v\nsuppressed = %+v", res.Findings, res.Suppressed)
	}
	for _, s := range res.Suppressed {
		if s.By != engine.SuppressedByComment {
			t.Errorf("suppressed by %q", s.By)
		}
	}
}

func TestIgnoreCommentForAnotherRuleDoesNothing(t *testing.T) {
	dir := testutil.CopyTree(t, testutil.Testdata(t, "sample"))
	edit(t, filepath.Join(dir, "android/app/src/main/java/com/example/paywall/App.kt"),
		`.build())`, `.build()) // rc-doctor:ignore app-id, restore-purchases`)
	if res := runWith(t, dir, nil); len(res.Findings) != 2 || len(res.Suppressed) != 0 {
		t.Fatalf("findings = %+v, suppressed = %+v", res.Findings, res.Suppressed)
	}
}

func TestConfigRulesAndIgnore(t *testing.T) {
	dir := testutil.CopyTree(t, testutil.Testdata(t, "sample"))
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(`{
	  "rules": {"api-key": "warning", "restore-purchases": "off"},
	  "ignore": ["ios/"]
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	res := runWith(t, dir, cfg)
	if len(res.Findings) != 1 || res.Findings[0].Platform != "android" || res.Findings[0].Severity != model.Warning {
		t.Fatalf("findings = %+v", res.Findings)
	}
	if len(res.Suppressed) != 1 || res.Suppressed[0].By != engine.SuppressedByConfig {
		t.Errorf("suppressed = %+v", res.Suppressed)
	}
	off := false
	for _, s := range res.Skipped {
		off = off || (s.Rule == "restore-purchases" && strings.Contains(s.Reason, config.FileName))
	}
	if !off {
		t.Errorf("a rule turned off must be listed as skipped: %+v", res.Skipped)
	}
}

func TestConfigWithUnknownRuleFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(`{"rules": {"api-keys": "off"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = engine.Run(context.Background(), engine.Options{Root: testutil.Testdata(t, "sample"), Config: cfg})
	if err == nil || !strings.Contains(err.Error(), `unknown rule "api-keys"`) {
		t.Errorf("err = %v", err)
	}
}
