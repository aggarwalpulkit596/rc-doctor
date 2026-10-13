// Package engine runs the pipeline every command shares:
//
//	detect platforms -> collect facts -> load the RevenueCat project -> run rules
//
// Commands decide what to do with the Result: print it (check), apply its
// edits (review --fix), or dump the facts (facts).
package engine

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/config"
	"github.com/aggarwalpulkit596/rc-doctor/internal/dashboard"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform"
	"github.com/aggarwalpulkit596/rc-doctor/internal/rules"
)

// ErrNoApp means no registered platform found an app that uses RevenueCat.
var ErrNoApp = errors.New("no app using RevenueCat found")

// Options configures a run.
type Options struct {
	Root      string
	Platforms []string           // empty: auto-detect
	Dashboard dashboard.Provider // nil: offline

	// StrictDashboard makes a dashboard failure fatal instead of falling back
	// to offline checks. Commands set it when the user asked for a source.
	StrictDashboard bool
	// OfflineNote explains why Dashboard is nil, for the report.
	OfflineNote string

	// Config is the project's .rc-doctor.json; nil means the defaults.
	Config *config.Config
}

// SkippedRule is a rule that didn't run, and why.
type SkippedRule struct {
	Rule   string `json:"rule"`
	Reason string `json:"reason"`
}

// Collection is what the platform modules found in the code.
type Collection struct {
	Root      string // absolute
	Platforms []string
	Facts     []model.Facts
}

// Result of a full run.
type Result struct {
	Collection
	Dashboard     string           // provider name, "offline" when none was loaded
	DashboardData *model.Dashboard // nil when offline
	Notes         []string
	Findings      []model.Finding
	Skipped       []SkippedRule
	// Suppressed are findings silenced by an rc-doctor:ignore comment or an
	// ignore pattern in .rc-doctor.json. Reports count them; SARIF keeps them,
	// marked as suppressed, so code scanning knows they were reviewed.
	Suppressed []Suppressed
}

// Suppressed is a finding that was silenced, and how.
type Suppressed struct {
	Finding model.Finding
	By      string // SuppressedByComment or SuppressedByConfig
}

// How a finding was suppressed.
const (
	SuppressedByComment = "comment"
	SuppressedByConfig  = "config"
)

// Collect resolves root, picks platforms (the named ones, or every platform
// whose Detect matches) and collects their facts.
func Collect(root string, names []string) (*Collection, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	switch fi, err := os.Stat(abs); {
	case errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("%s doesn't exist", root)
	case err != nil:
		return nil, err
	case !fi.IsDir():
		return nil, fmt.Errorf("%s is not a directory", root)
	}
	ps, err := choose(abs, names)
	if err != nil {
		return nil, err
	}
	c := &Collection{Root: abs}
	for _, p := range ps {
		facts, err := p.Collect(abs)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.Name(), err)
		}
		for i := range facts {
			// A module can return Facts for other platforms (kmp returns
			// "android" and "ios"); each app is checked with its own prefixes.
			facts[i].KeyPrefixes = p.KeyPrefixes()
			if own, ok := platform.Get(facts[i].Platform); ok {
				facts[i].KeyPrefixes = own.KeyPrefixes()
			}
		}
		c.Platforms = append(c.Platforms, p.Name())
		c.Facts = append(c.Facts, facts...)
	}
	if len(c.Facts) == 0 {
		return nil, fmt.Errorf("%w under %s. rc-doctor looks for the RevenueCat SDK in build files (supported platforms: %s); use --platform to force one",
			ErrNoApp, root, strings.Join(platform.Names(), ", "))
	}
	return c, nil
}

func choose(root string, names []string) ([]platform.Platform, error) {
	var ps []platform.Platform
	if len(names) > 0 {
		for _, n := range names {
			p, ok := platform.Get(n)
			if !ok {
				return nil, fmt.Errorf("unknown platform %q (available: %s)", n, strings.Join(platform.Names(), ", "))
			}
			ps = append(ps, p)
		}
		return ps, nil
	}
	superseded := map[string]bool{}
	for _, p := range platform.All() {
		if p.Detect(root) {
			ps = append(ps, p)
			if s, ok := p.(platform.Superseder); ok {
				for _, name := range s.Supersedes() {
					superseded[name] = true
				}
			}
		}
	}
	var out []platform.Platform
	for _, p := range ps {
		if !superseded[p.Name()] {
			out = append(out, p)
		}
	}
	return out, nil
}

// Run executes the whole pipeline.
func Run(ctx context.Context, opts Options) (*Result, error) {
	col, err := Collect(opts.Root, opts.Platforms)
	if err != nil {
		return nil, err
	}
	res := &Result{Collection: *col, Dashboard: "offline"}
	if opts.Dashboard == nil {
		if opts.OfflineNote != "" {
			res.Notes = append(res.Notes, opts.OfflineNote)
		}
	} else {
		d, err := opts.Dashboard.Load(ctx)
		switch {
		case err != nil && opts.StrictDashboard:
			return nil, fmt.Errorf("load RevenueCat project from %s: %w", opts.Dashboard.Name(), err)
		case err != nil:
			res.Notes = append(res.Notes, fmt.Sprintf("Couldn't load your RevenueCat project (%v), so only offline checks ran.", err))
		case d != nil:
			res.Dashboard, res.DashboardData = opts.Dashboard.Name(), d
			res.Notes = append(res.Notes, d.Notes...)
		}
	}

	cfg := opts.Config
	if cfg == nil {
		cfg = &config.Config{}
	}
	if err := cfg.Validate(RuleIDs()); err != nil {
		return nil, err
	}

	rctx := &rules.Context{Facts: res.Facts, Dashboard: res.DashboardData, KeyPrefixes: KeyPrefixes()}
	for _, r := range rules.All() {
		if !cfg.Enabled(r.ID()) {
			res.Skipped = append(res.Skipped, SkippedRule{Rule: r.ID(), Reason: "turned off in " + config.FileName})
			continue
		}
		if r.NeedsDashboard() && rctx.Dashboard == nil {
			res.Skipped = append(res.Skipped, SkippedRule{
				Rule:   r.ID(),
				Reason: "needs your RevenueCat project: use --dashboard rc, or a snapshot file",
			})
			continue
		}
		res.Findings = append(res.Findings, r.Check(rctx)...)
	}
	res.Findings = dedupe(res.Findings)
	res.Findings, res.Suppressed = suppress(res.Root, res.Findings, cfg)
	sortFindings(res.Findings)
	return res, nil
}

// RuleIDs returns the IDs of every registered rule.
func RuleIDs() []string {
	var ids []string
	for _, r := range rules.All() {
		ids = append(ids, r.ID())
	}
	return ids
}

// dedupe drops findings that repeat one already reported for the same line.
// Code shared between apps (KMP commonMain) is checked once per app, but a
// typo on a shared line is one problem.
func dedupe(fs []model.Finding) []model.Finding {
	type key struct {
		rule, file, msg string
		line            int
	}
	seen := map[key]bool{}
	var out []model.Finding
	for _, f := range fs {
		k := key{f.Rule, f.Location.File, f.Message, f.Location.Line}
		if !seen[k] {
			seen[k] = true
			out = append(out, f)
		}
	}
	return out
}

// KeyPrefixes maps every registered platform's public key prefixes to it.
func KeyPrefixes() map[string]string {
	m := map[string]string{}
	for _, p := range platform.All() {
		if _, umbrella := p.(platform.Superseder); umbrella {
			continue // kmp's prefixes are android's and ios's; name the real owner
		}
		for _, prefix := range p.KeyPrefixes() {
			m[prefix] = p.Name()
		}
	}
	return m
}

// Count returns how many findings have severity s.
func (r *Result) Count(s model.Severity) int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == s {
			n++
		}
	}
	return n
}

// Edits returns every edit attached to a finding, in finding order.
func (r *Result) Edits() []model.Edit {
	var out []model.Edit
	for _, f := range r.Findings {
		out = append(out, f.Edits...)
	}
	return out
}

func sortFindings(fs []model.Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.Platform != b.Platform {
			return a.Platform < b.Platform
		}
		if a.Location.File != b.Location.File {
			return a.Location.File < b.Location.File
		}
		if a.Location.Line != b.Location.Line {
			return a.Location.Line < b.Location.Line
		}
		return a.Rule < b.Rule
	})
}
