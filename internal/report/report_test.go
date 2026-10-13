package report

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

const fullKey = "appl_AbCdEfGhIjKlMnOpQrSt"

func sampleResult(root string) *engine.Result {
	return &engine.Result{
		Collection: engine.Collection{
			Root:      root,
			Platforms: []string{"android", "ios"},
			Facts: []model.Facts{
				{Platform: "android", AppID: "com.example.app"},
				{Platform: "ios", AppID: "com.example.app"},
			},
		},
		Dashboard:     "snapshot:dashboard.json",
		DashboardData: &model.Dashboard{},
		Findings: []model.Finding{
			{
				Rule: "api-key", Severity: model.Error, Platform: "android",
				Location: model.Location{File: "app/App.kt", Line: 11},
				Message:  "appl_AbCd… is a public key for ios apps, but this is the android app.",
				Fix:      "Use the android app's public SDK key (goog_ZyXw…).",
				// Edits carry full keys; reporters must never print them.
				Edits: []model.Edit{{Old: fullKey, New: "goog_ZyXwVuTsRqPoNmLkJiHg", Safety: model.Safe}},
			},
			{
				Rule: "app-id", Severity: model.Warning, Platform: "ios",
				Location: model.Location{File: "App.xcodeproj/project.pbxproj", Line: 40},
				Message:  `"com.example.app" doesn't match any ios app in the RevenueCat project.`,
			},
		},
		Notes:   []string{"a note"},
		Skipped: []engine.SkippedRule{{Rule: "x", Reason: "why"}, {Rule: "y", Reason: "why"}},
	}
}

func render(t *testing.T, r Reporter, res *engine.Result) string {
	t.Helper()
	var b bytes.Buffer
	if err := r.Write(&b, res); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if strings.Contains(out, fullKey) {
		t.Fatalf("%s output leaked a full key:\n%s", r.Name(), out)
	}
	return out
}

func TestRegistered(t *testing.T) {
	if got := strings.Join(Names(), ","); got != "json,sarif,text" {
		t.Errorf("reporters = %s", got)
	}
}

func TestText(t *testing.T) {
	root := t.TempDir()
	out := render(t, Text{}, sampleResult(root))
	for _, want := range []string{
		"rc-doctor checked android (com.example.app), ios (com.example.app)",
		"against RevenueCat project: snapshot:dashboard.json",
		"\nandroid\n",
		"error   api-key  " + filepath.Join(root, "app", "App.kt") + ":11",
		"fix: Use the android app's public SDK key (goog_ZyXw…).  [safe fix]",
		"\nios\n",
		"warning app-id",
		"note: a note",
		"skipped x, y (why)",
		"1 error, 1 warning.",
		"Automatic fixes available: 1 safe, 0 suggested.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("text output is missing %q:\n%s", want, out)
		}
	}
	if quiet := render(t, Text{NoFixHint: true}, sampleResult(root)); strings.Contains(quiet, "Automatic fixes") {
		t.Errorf("NoFixHint output still points at review:\n%s", quiet)
	}

	clean := sampleResult(root)
	clean.Findings = nil
	if out := render(t, Text{}, clean); !strings.Contains(out, "No problems found.") {
		t.Errorf("clean output:\n%s", out)
	}
}

func TestJSON(t *testing.T) {
	out := render(t, JSON{}, sampleResult("/repo"))
	var got struct {
		Tool     string         `json:"tool"`
		Root     string         `json:"root"`
		Summary  map[string]int `json:"summary"`
		Findings []map[string]any
		Skipped  []engine.SkippedRule
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if got.Tool != "rc-doctor" || got.Root != "/repo" || got.Summary["error"] != 1 || got.Summary["warning"] != 1 || len(got.Skipped) != 2 {
		t.Errorf("report = %+v", got)
	}
	first := got.Findings[0]
	if first["file"] != "app/App.kt" || first["line"] != float64(11) || first["fixable"] != "safe" {
		t.Errorf("finding = %v", first)
	}
	if _, ok := first["edits"]; ok {
		t.Error("raw edits must not be in the JSON report")
	}
	if _, ok := got.Findings[1]["fixable"]; ok {
		t.Error("a finding without edits must not be fixable")
	}

	empty := sampleResult("/repo")
	empty.Findings = nil
	if out := render(t, JSON{}, empty); !strings.Contains(out, `"findings": []`) {
		t.Errorf("no findings must render as an empty list:\n%s", out)
	}
}

func TestSARIF(t *testing.T) {
	root := t.TempDir()
	t.Chdir(filepath.Dir(root))
	out := render(t, SARIF{}, sampleResult(root))

	var got struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name  string `json:"name"`
					Rules []struct {
						ID string `json:"id"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID    string `json:"ruleId"`
				Level     string `json:"level"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid SARIF: %v\n%s", err, out)
	}
	if got.Version != "2.1.0" || len(got.Runs) != 1 || got.Runs[0].Tool.Driver.Name != "rc-doctor" {
		t.Fatalf("SARIF header = %+v", got)
	}
	res := got.Runs[0].Results
	if len(res) != 2 || res[0].Level != "error" || res[1].Level != "warning" {
		t.Fatalf("results = %+v", res)
	}
	loc := res[0].Locations[0].PhysicalLocation
	if want := filepath.Base(root) + "/app/App.kt"; loc.ArtifactLocation.URI != want || loc.Region.StartLine != 11 {
		t.Errorf("location = %+v, want %s:11 relative to the working directory", loc, want)
	}
}
