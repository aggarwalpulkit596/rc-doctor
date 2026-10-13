package report

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/engine"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func init() { Register(Text{}) }

// Text is the human-readable report.
type Text struct {
	// NoFixHint drops the "preview them with rc-doctor review" pointer, for
	// commands that already show fixes themselves.
	NoFixHint bool
}

func (Text) Name() string        { return "text" }
func (Text) Description() string { return "for people (the default)" }

func (t Text) Write(w io.Writer, r *engine.Result) error {
	var b bytes.Buffer

	fmt.Fprintf(&b, "rc-doctor checked %s\n", describeApps(r))
	if r.DashboardData != nil {
		fmt.Fprintf(&b, "against RevenueCat project: %s\n", r.Dashboard)
	} else {
		fmt.Fprintln(&b, "offline: no RevenueCat project loaded")
	}

	platform := ""
	for _, f := range r.Findings {
		if f.Platform != platform {
			platform = f.Platform
			fmt.Fprintf(&b, "\n%s\n", platform)
		}
		where := Path(r.Root, f.Location.File)
		if where == "" {
			where = "(no file)"
		} else if f.Location.Line > 0 {
			where = fmt.Sprintf("%s:%d", where, f.Location.Line)
		}
		fmt.Fprintf(&b, "  %-7s %s  %s\n", f.Severity, f.Rule, where)
		fmt.Fprintf(&b, "          %s\n", f.Message)
		if f.Fix != "" {
			fmt.Fprintf(&b, "          fix: %s%s\n", f.Fix, fixTag(f))
		}
	}

	if len(r.Notes) > 0 || len(r.Skipped) > 0 {
		fmt.Fprintln(&b)
	}
	for _, n := range r.Notes {
		fmt.Fprintf(&b, "note: %s\n", n)
	}
	for _, g := range groupSkipped(r.Skipped) {
		fmt.Fprintf(&b, "skipped %s (%s)\n", strings.Join(g.rules, ", "), g.reason)
	}

	fmt.Fprintf(&b, "\n%s\n", summary(r, !t.NoFixHint))
	_, err := w.Write(b.Bytes())
	return err
}

func describeApps(r *engine.Result) string {
	var parts []string
	for _, f := range r.Facts {
		if f.AppID != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", f.Platform, f.AppID))
		} else {
			parts = append(parts, f.Platform)
		}
	}
	return strings.Join(parts, ", ")
}

func fixTag(f model.Finding) string {
	switch f.Fixable() {
	case model.Safe:
		return "  [safe fix]"
	case model.Suggested:
		return "  [suggested fix]"
	}
	return ""
}

type skippedGroup struct {
	reason string
	rules  []string
}

func groupSkipped(skipped []engine.SkippedRule) []skippedGroup {
	var out []skippedGroup
	index := map[string]int{}
	for _, s := range skipped {
		i, ok := index[s.Reason]
		if !ok {
			i = len(out)
			index[s.Reason] = i
			out = append(out, skippedGroup{reason: s.Reason})
		}
		out[i].rules = append(out[i].rules, s.Rule)
	}
	return out
}

func summary(r *engine.Result, fixHint bool) string {
	if len(r.Findings) == 0 {
		return "No problems found."
	}
	s := fmt.Sprintf("%s, %s", plural(r.Count(model.Error), "error"), plural(r.Count(model.Warning), "warning"))
	if n := r.Count(model.Info); n > 0 {
		s += ", " + plural(n, "info message")
	}
	s += "."
	safe, suggested := 0, 0
	for _, f := range r.Findings {
		switch f.Fixable() {
		case model.Safe:
			safe++
		case model.Suggested:
			suggested++
		}
	}
	if fixHint && safe+suggested > 0 {
		s += fmt.Sprintf("\nAutomatic fixes available: %d safe, %d suggested. Preview them by running `rc-doctor review` with the same arguments.", safe, suggested)
	}
	return s
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
