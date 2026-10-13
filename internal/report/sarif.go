package report

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/buildinfo"
	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/rules"
)

func init() { Register(SARIF{}) }

// SARIF writes SARIF 2.1.0, the format GitHub code scanning and most CI
// systems use to annotate pull requests. File URIs are relative to the working
// directory, so run rc-doctor from the repository root in CI.
type SARIF struct{}

func (SARIF) Name() string        { return "sarif" }
func (SARIF) Description() string { return "SARIF 2.1.0, for GitHub code scanning and CI annotations" }

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	ShortDescription sarifMessage `json:"shortDescription"`
	HelpURI          string       `json:"helpUri"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID       string             `json:"ruleId"`
	Level        string             `json:"level"`
	Message      sarifMessage       `json:"message"`
	Locations    []sarifLocation    `json:"locations,omitempty"`
	Suppressions []sarifSuppression `json:"suppressions,omitempty"`
}

// sarifSuppression marks a result someone silenced on purpose: "inSource"
// for an rc-doctor:ignore comment, "external" for .rc-doctor.json.
type sarifSuppression struct {
	Kind string `json:"kind"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           *sarifRegion  `json:"region,omitempty"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func (SARIF) Write(w io.Writer, r *engine.Result) error {
	driver := sarifDriver{
		Name:           "rc-doctor",
		Version:        buildinfo.String(),
		InformationURI: "https://github.com/aggarwalpulkit596/rc-doctor",
	}
	for _, rule := range rules.All() {
		driver.Rules = append(driver.Rules, sarifRule{
			ID:               rule.ID(),
			ShortDescription: sarifMessage{Text: rule.Description()},
			HelpURI:          rules.DocsURL(rule.ID()),
		})
	}
	run := sarifRun{Tool: sarifTool{Driver: driver}, Results: []sarifResult{}}
	for _, f := range r.Findings {
		run.Results = append(run.Results, sarifResultFor(r.Root, f))
	}
	for _, s := range r.Suppressed {
		res := sarifResultFor(r.Root, s.Finding)
		kind := "external"
		if s.By == engine.SuppressedByComment {
			kind = "inSource"
		}
		res.Suppressions = []sarifSuppression{{Kind: kind}}
		run.Results = append(run.Results, res)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs:    []sarifRun{run},
	})
}

func sarifResultFor(root string, f model.Finding) sarifResult {
	text := f.Message
	if f.Fix != "" {
		text += " Fix: " + f.Fix
	}
	res := sarifResult{RuleID: f.Rule, Level: sarifLevel(f.Severity), Message: sarifMessage{Text: text}}
	if f.Location.File != "" {
		loc := sarifPhysical{ArtifactLocation: sarifArtifact{URI: uri(root, f.Location.File)}}
		if f.Location.Line > 0 {
			loc.Region = &sarifRegion{StartLine: f.Location.Line}
		}
		res.Locations = []sarifLocation{{PhysicalLocation: loc}}
	}
	return res
}

func sarifLevel(s model.Severity) string {
	switch s {
	case model.Error:
		return "error"
	case model.Warning:
		return "warning"
	}
	return "note"
}

// uri returns a relative URI when possible. SARIF consumers resolve relative
// URIs against the repository root, which is where CI runs rc-doctor.
func uri(root, rel string) string {
	p := Path(root, rel)
	if !filepath.IsAbs(p) {
		return p
	}
	s := filepath.ToSlash(p)
	if !strings.HasPrefix(s, "/") {
		s = "/" + s // Windows: C:/x becomes file:///C:/x
	}
	return "file://" + s
}
