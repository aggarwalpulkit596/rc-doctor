package main

// End-to-end tests: the real binary's commands, plug-ins and fixtures, run
// through cli.Main exactly as a user would.

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/cli"
	"github.com/aggarwalpulkit596/rc-doctor/internal/migrate"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

// Full keys from the fixtures. No output may ever contain them.
var fixtureKeys = []string{"appl_AbCdEfGhIjKlMnOpQrSt", "sk_AbCdEfGhIjKlMnOpQrSt", "goog_ZyXwVuTsRqPoNmLkJiHg", "appl_KmpIosKey123456789012", "goog_KmpAndroidKey12345678"}

func run(t *testing.T, args ...string) (string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := cli.Main(context.Background(), args, &out, &errOut)
	all := out.String() + errOut.String()
	for _, k := range fixtureKeys {
		if strings.Contains(all, k) {
			t.Fatalf("rc-doctor %s printed the full key %s:\n%s", strings.Join(args, " "), k, all)
		}
	}
	return all, code
}

func wantAll(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("output is missing %q:\n%s", w, out)
		}
	}
}

func sample(t *testing.T) (dir, dash string) {
	return testutil.Testdata(t, "sample"), testutil.Testdata(t, "dashboard.json")
}

func TestCheckAgainstSnapshot(t *testing.T) {
	dir, dash := sample(t)
	out, code := run(t, "check", dir, "--dashboard", dash)
	if code != cli.ExitFindings {
		t.Fatalf("exit %d, want %d", code, cli.ExitFindings)
	}
	wantAll(t, out,
		"rc-doctor checked android (com.example.paywall), ios (com.example.paywall)",
		"App.kt:11", `Entitlement "premum"`, `Did you mean "premium"?`,
		`Offering "summer_sale"`, "Secret API key sk_AbCd…", "4 errors, 0 warnings.",
		"Automatic fixes available: 1 safe, 2 suggested.",
	)
}

func TestCheckIsTheDefaultAndOfflineWorks(t *testing.T) {
	dir, _ := sample(t)
	out, code := run(t, dir, "--dashboard", "offline")
	if code != cli.ExitFindings {
		t.Fatalf("exit %d", code)
	}
	wantAll(t, out, "offline: no RevenueCat project loaded", "2 errors", "skipped app-id, unknown-entitlement, unknown-offering")
}

func TestFailOn(t *testing.T) {
	dir, dash := sample(t)
	if _, code := run(t, "check", dir, "--dashboard", dash, "--fail-on", "never"); code != cli.ExitOK {
		t.Errorf("--fail-on never: exit %d", code)
	}
}

func TestCheckJSONAndSARIF(t *testing.T) {
	dir, dash := sample(t)

	out, _ := run(t, "check", dir, "--dashboard", dash, "--format", "json")
	var report struct {
		Summary  map[string]int `json:"summary"`
		Findings []struct {
			Rule    string `json:"rule"`
			Fixable string `json:"fixable"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if report.Summary["error"] != 4 || len(report.Findings) != 4 || report.Findings[0].Fixable != "safe" {
		t.Errorf("JSON report = %+v", report)
	}

	out, _ = run(t, "check", dir, "--dashboard", dash, "--format", "sarif")
	var sarif struct {
		Version string `json:"version"`
		Runs    []struct {
			Results []json.RawMessage `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(out), &sarif); err != nil || sarif.Version != "2.1.0" || len(sarif.Runs[0].Results) != 4 {
		t.Errorf("SARIF = %+v, err = %v", sarif, err)
	}
}

func TestReviewPreviewChangesNothing(t *testing.T) {
	dir := testutil.CopyTree(t, testutil.Testdata(t, "sample"))
	_, dash := sample(t)
	app := filepath.Join(dir, "android/app/src/main/java/com/example/paywall/App.kt")
	before := testutil.ReadFile(t, app)

	out, code := run(t, "review", dir, "--dashboard", dash)
	if code != cli.ExitFindings {
		t.Fatalf("exit %d", code)
	}
	wantAll(t, out, "Proposed fixes (1 safe, 2 suggested):",
		`- Purchases.configure(PurchasesConfiguration.Builder(this, "appl_AbCd…").build())`,
		`+ Purchases.configure(PurchasesConfiguration.Builder(this, "goog_ZyXw…").build())`,
		"Nothing was changed.")
	if testutil.ReadFile(t, app) != before {
		t.Error("review without --fix changed a file")
	}
}

func TestReviewFixAppliesSafeFixesOnly(t *testing.T) {
	dir := testutil.CopyTree(t, testutil.Testdata(t, "sample"))
	_, dash := sample(t)

	out, code := run(t, "review", dir, "--dashboard", dash, "--fix")
	if code != cli.ExitFindings {
		t.Fatalf("exit %d", code)
	}
	wantAll(t, out, "Applied 1 fix:", "After fixes:", "3 errors, 0 warnings.", "2 suggested fixes left")

	app := testutil.ReadFile(t, filepath.Join(dir, "android/app/src/main/java/com/example/paywall/App.kt"))
	if !strings.Contains(app, `"goog_ZyXwVuTsRqPoNmLkJiHg"`) {
		t.Errorf("the Android key was not replaced:\n%s", app)
	}
	swift := testutil.ReadFile(t, filepath.Join(dir, "ios/Paywall/PaywallApp.swift"))
	if !strings.Contains(swift, `"sk_AbCdEfGhIjKlMnOpQrSt"`) {
		t.Error("a suggested fix was applied without --suggested")
	}
}

func TestReviewFixSuggested(t *testing.T) {
	dir := testutil.CopyTree(t, testutil.Testdata(t, "sample"))
	_, dash := sample(t)

	out, code := run(t, "review", dir, "--dashboard", dash, "--fix", "--suggested")
	if code != cli.ExitFindings {
		t.Fatalf("exit %d", code)
	}
	wantAll(t, out, "Applied 3 fixes:", "1 error, 0 warnings.", `Offering "summer_sale"`)

	paywall := testutil.ReadFile(t, filepath.Join(dir, "android/app/src/main/java/com/example/paywall/Paywall.kt"))
	if !strings.Contains(paywall, `entitlements["premium"]`) || !strings.Contains(paywall, `entitlements["pro_legacy"]`) {
		t.Errorf("want the typo fixed and the commented-out code untouched:\n%s", paywall)
	}

	// Running again finds nothing left to fix automatically.
	out, _ = run(t, "review", dir, "--dashboard", dash, "--fix", "--suggested")
	wantAll(t, out, "Nothing to fix automatically.")
}

func TestFacts(t *testing.T) {
	dir, _ := sample(t)
	out, code := run(t, "facts", dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, out)
	}
	var got struct {
		Platforms []string      `json:"platforms"`
		Facts     []model.Facts `json:"facts"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.Platforms, ",") != "android,ios" || got.Facts[0].APIKeys[0].Value != "appl_AbCd…" {
		t.Errorf("facts = %+v", got)
	}
	if strings.Contains(out, "fake_for_test") {
		t.Error("test sources leaked into facts")
	}
}

func TestListAndVersion(t *testing.T) {
	out, code := run(t, "list")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	wantAll(t, out, "platforms:", "android", "ios", "rules:", "api-key", "migrators:", "(none yet)", "formats:", "sarif", "commands:", "review")

	out, _ = run(t, "list", "rules")
	if strings.Contains(out, "platforms:") || !strings.Contains(out, "unknown-offering") {
		t.Errorf("list rules:\n%s", out)
	}
	if _, code := run(t, "list", "nope"); code != cli.ExitError {
		t.Errorf("unknown section: exit %d", code)
	}
	if out, _ := run(t, "version"); !strings.HasPrefix(out, "rc-doctor ") {
		t.Errorf("version: %q", out)
	}
}

func TestErrors(t *testing.T) {
	dir, _ := sample(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"check", t.TempDir(), "--dashboard", "offline"}, "no app using RevenueCat found"},
		{[]string{"check", dir, "--format", "xml"}, `unknown format "xml"`},
		{[]string{"check", dir, "--platform", "web"}, `unknown platform "web"`},
		{[]string{"check", dir, "--dashboard", filepath.Join(t.TempDir(), "missing.json")}, "read dashboard snapshot"},
		{[]string{"bogus-command"}, `unknown command or directory "bogus-command"`},
	} {
		out, code := run(t, tc.args...)
		if code != cli.ExitError || !strings.Contains(out, tc.want) {
			t.Errorf("rc-doctor %s: exit %d, output:\n%s\nwant exit 2 and %q", strings.Join(tc.args, " "), code, out, tc.want)
		}
	}
}

func TestMigrateWithoutSources(t *testing.T) {
	dir, _ := sample(t)
	out, code := run(t, "migrate", dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	wantAll(t, out, "No migration sources are available yet. Planned: superwall")
	if _, code := run(t, "migrate", dir, "--from", "superwall"); code != cli.ExitError {
		t.Errorf("--from an unavailable source: exit %d", code)
	}
}

// fakeSDK is a stand-in migration source: it proves the migrate pipeline end
// to end (detect, plan, preview, apply) before any real source exists.
type fakeSDK struct{}

func (fakeSDK) Name() string        { return "fakesdk" }
func (fakeSDK) Description() string { return "test source" }
func (fakeSDK) Detect(root string) bool {
	_, err := os.Stat(filepath.Join(root, "Billing.kt"))
	return err == nil
}

func (fakeSDK) Plan(_ context.Context, root string) (*migrate.Plan, error) {
	return &migrate.Plan{From: "fakesdk", Steps: []migrate.Step{
		{
			Title: "Replace SDK setup",
			Edits: []model.Edit{{
				Path: filepath.Join(root, "Billing.kt"), File: "Billing.kt", Line: 1,
				Old: "FakeSdk.init()", New: "Purchases.configure(config)",
				Safety: model.Safe, Description: "Configure RevenueCat instead",
			}},
		},
		{Title: "Create entitlements", Manual: "Create the premium entitlement in the RevenueCat dashboard."},
	}}, nil
}

func TestReviewWithNothingToFix(t *testing.T) {
	dir, _ := sample(t)
	// Offline, the key findings can't know the right replacement key.
	out, _ := run(t, "review", dir, "--dashboard", "offline")
	wantAll(t, out, "2 errors", "No automatic fixes available.")
}

// fakeRCOnPath puts a shell script named rc on PATH that answers like the
// RevenueCat CLI, so the real exec path is tested without network or login.
func fakeRCOnPath(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	script := `#!/bin/sh
case "$1 $2" in
  "entitlements list") echo '[{"id":"entl1","lookup_key":"premium"}]' ;;
  "offerings list") echo '{"items":[{"id":"o1","lookup_key":"default"},{"id":"o2","lookup_key":"annual_promo"}]}' ;;
  "apps list") echo '{"items":[{"id":"app1","type":"play_store","play_store":{"package_name":"com.example.paywall"}},{"id":"app2","type":"app_store","app_store":{"bundle_id":"com.example.paywall"}}]}' ;;
  "apps keys")
    if [ "$3" = "app1" ]; then echo '{"items":[{"key":"goog_ZyXwVuTsRqPoNmLkJiHg"}]}'; else echo '{"items":[{"key":"appl_AbCdEfGhIjKlMnOpQrSt"}]}'; fi ;;
  *) echo "unexpected: $*" >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "rc"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestRCCLIEndToEnd(t *testing.T) {
	fakeRCOnPath(t)
	dir, _ := sample(t)

	// auto finds the rc on PATH; the result matches the snapshot fixture.
	out, code := run(t, "check", dir)
	if code != cli.ExitFindings {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	wantAll(t, out, "against RevenueCat project: rc-cli", "4 errors", "Automatic fixes available: 1 safe, 2 suggested.")

	snap := filepath.Join(t.TempDir(), "project.json")
	if out, code := run(t, "snapshot", "-o", snap); code != cli.ExitOK {
		t.Fatalf("snapshot: exit %d:\n%s", code, out)
	}
	out, _ = run(t, "check", dir, "--dashboard", snap)
	wantAll(t, out, "4 errors", "Automatic fixes available: 1 safe, 2 suggested.")
}

func TestMigrateSourceSelection(t *testing.T) {
	migrate.Register(fakeSDK{})
	migrate.Register(otherSDK{})
	t.Cleanup(func() {
		migrate.Unregister("fakesdk")
		migrate.Unregister("othersdk")
	})
	dir := testutil.WriteFiles(t, map[string]string{"Billing.kt": "FakeSdk.init()\n"})

	if out, code := run(t, "migrate", dir); code != cli.ExitError || !strings.Contains(out, "found several SDKs (fakesdk, othersdk)") {
		t.Errorf("two detected sources: exit %d\n%s", code, out)
	}
	if out, code := run(t, "migrate", dir, "--from", "nope"); code != cli.ExitError || !strings.Contains(out, `unknown source "nope"`) {
		t.Errorf("unknown source: exit %d\n%s", code, out)
	}
	if out, code := run(t, "migrate", t.TempDir()); code != cli.ExitOK || !strings.Contains(out, "Nothing to migrate") {
		t.Errorf("nothing detected: exit %d\n%s", code, out)
	}
	if out, code := run(t, "migrate", dir, "--from", "othersdk"); code != cli.ExitOK || !strings.Contains(out, "othersdk -> RevenueCat") {
		t.Errorf("--from othersdk: exit %d\n%s", code, out)
	}
}

type otherSDK struct{ fakeSDK }

func (otherSDK) Name() string { return "othersdk" }
func (otherSDK) Plan(context.Context, string) (*migrate.Plan, error) {
	return &migrate.Plan{From: "othersdk", Steps: []migrate.Step{{Title: "Read the docs", Manual: "Nothing to automate yet."}}}, nil
}

func TestMigrateWithASource(t *testing.T) {
	migrate.Register(fakeSDK{})
	t.Cleanup(func() { migrate.Unregister("fakesdk") })
	dir := testutil.WriteFiles(t, map[string]string{"Billing.kt": "FakeSdk.init()\n"})

	out, code := run(t, "migrate", dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	wantAll(t, out, "Migration plan: fakesdk -> RevenueCat (2 steps)", "1. Replace SDK setup", "[safe] ", "manual: Create the premium entitlement", "Nothing was changed.")
	if got := testutil.ReadFile(t, filepath.Join(dir, "Billing.kt")); got != "FakeSdk.init()\n" {
		t.Fatalf("plan without --apply changed the file: %q", got)
	}

	out, code = run(t, "migrate", dir, "--apply")
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	wantAll(t, out, "Applied 1 edit:")
	if got := testutil.ReadFile(t, filepath.Join(dir, "Billing.kt")); got != "Purchases.configure(config)\n" {
		t.Errorf("after --apply: %q", got)
	}

	if out, _ := run(t, "list", "migrators"); !strings.Contains(out, "fakesdk") {
		t.Errorf("list doesn't show the registered source:\n%s", out)
	}
}

func TestKMPFixesAConstantWhereItIsDefined(t *testing.T) {
	dir := testutil.CopyTree(t, testutil.Testdata(t, "kmp"))
	dash := testutil.Testdata(t, "dashboard-kmp.json")
	out, _ := run(t, "check", dir, "--dashboard", dash)
	wantAll(t, out, "android (com.example.kmp), ios (com.example.kmp.ios)",
		`Entitlement "premum" (RC_ENTITLEMENT, set at composeApp/build.gradle.kts:17)`, "1 error, 0 warnings.")

	out, code := run(t, "review", dir, "--dashboard", dash, "--fix", "--suggested")
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	gradle := testutil.ReadFile(t, filepath.Join(dir, "composeApp/build.gradle.kts"))
	if !strings.Contains(gradle, `getProperty("RC_ENTITLEMENT", "premium")`) {
		t.Errorf("the default was not fixed:\n%s", gradle)
	}
}

func TestKMPKeyInTheWrongSlot(t *testing.T) {
	dir := testutil.CopyTree(t, testutil.Testdata(t, "kmp"))
	secrets := filepath.Join(dir, "secrets.properties")
	if err := os.WriteFile(secrets, []byte("RC_ANDROID_KEY=appl_KmpIosKey123456789012\nRC_IOS_KEY=appl_KmpIosKey123456789012\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _ := run(t, "check", dir, "--dashboard", testutil.Testdata(t, "dashboard-kmp.json"))
	wantAll(t, out, "secrets.properties:1", "is a public key for ios apps, but this is the android app", "[safe fix]")
	if strings.Contains(out, "secrets.properties:2") {
		t.Errorf("the iOS slot holds the right key and must not be flagged:\n%s", out)
	}
}
