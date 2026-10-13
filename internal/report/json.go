package report

import (
	"encoding/json"
	"io"

	"github.com/aggarwalpulkit596/rc-doctor/internal/buildinfo"
	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/rules"
)

func init() { Register(JSON{}) }

// JSON is the machine-readable report, for scripts and AI agents.
// Raw edits are left out because they contain full keys; "fixable" says
// whether `rc-doctor review --fix` can resolve a finding.
type JSON struct{}

func (JSON) Name() string        { return "json" }
func (JSON) Description() string { return "for scripts and AI agents" }

type jsonFinding struct {
	Rule     string         `json:"rule"`
	Severity model.Severity `json:"severity"`
	Platform string         `json:"platform"`
	File     string         `json:"file,omitempty"` // relative to root
	Line     int            `json:"line,omitempty"`
	Message  string         `json:"message"`
	Fix      string         `json:"fix,omitempty"`
	Fixable  model.Safety   `json:"fixable,omitempty"` // "safe" or "suggested"
	Docs     string         `json:"docs"`
}

type jsonApp struct {
	Platform string `json:"platform"`
	AppID    string `json:"app_id,omitempty"`
}

// SchemaVersion is the version of the JSON report's shape. It changes only
// when a field is removed or changes meaning; new fields don't bump it.
// docs/output.md documents the shape.
const SchemaVersion = 1

type jsonReport struct {
	SchemaVersion int                  `json:"schema_version"`
	Tool          string               `json:"tool"`
	Version       string               `json:"version"`
	Root          string               `json:"root"`
	Apps          []jsonApp            `json:"apps"`
	Dashboard     string               `json:"dashboard"`
	Summary       map[string]int       `json:"summary"`
	Findings      []jsonFinding        `json:"findings"`
	Suppressed    int                  `json:"suppressed"`
	Skipped       []engine.SkippedRule `json:"skipped,omitempty"`
	Notes         []string             `json:"notes,omitempty"`
}

func (JSON) Write(w io.Writer, r *engine.Result) error {
	out := jsonReport{
		SchemaVersion: SchemaVersion,
		Suppressed:    len(r.Suppressed),
		Tool:          "rc-doctor",
		Version:       buildinfo.String(),
		Root:          r.Root,
		Apps:          []jsonApp{},
		Dashboard:     r.Dashboard,
		Summary: map[string]int{
			"error":   r.Count(model.Error),
			"warning": r.Count(model.Warning),
			"info":    r.Count(model.Info),
		},
		Findings: []jsonFinding{},
		Skipped:  r.Skipped,
		Notes:    r.Notes,
	}
	for _, f := range r.Facts {
		out.Apps = append(out.Apps, jsonApp{Platform: f.Platform, AppID: f.AppID})
	}
	for _, f := range r.Findings {
		out.Findings = append(out.Findings, jsonFinding{
			Rule:     f.Rule,
			Severity: f.Severity,
			Platform: f.Platform,
			File:     f.Location.File,
			Line:     f.Location.Line,
			Message:  f.Message,
			Fix:      f.Fix,
			Fixable:  f.Fixable(),
			Docs:     rules.DocsURL(f.Rule),
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(out)
}
