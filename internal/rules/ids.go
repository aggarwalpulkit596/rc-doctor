package rules

import (
	"fmt"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func init() {
	Register(unknownIDs{
		id:    "unknown-entitlement",
		noun:  "entitlement",
		refs:  func(f model.Facts) []model.Ref { return f.Entitlements },
		known: func(d *model.Dashboard) []string { return d.Entitlements },
	})
	Register(unknownIDs{
		id:    "unknown-offering",
		noun:  "offering",
		refs:  func(f model.Facts) []model.Ref { return f.Offerings },
		known: func(d *model.Dashboard) []string { return d.Offerings },
	})
}

// unknownIDs flags identifiers used in code that the RevenueCat project
// doesn't define. A mistyped entitlement ID compiles, ships, and then never
// unlocks anything, even after a successful purchase.
type unknownIDs struct {
	id, noun string
	refs     func(model.Facts) []model.Ref
	known    func(*model.Dashboard) []string
}

func (r unknownIDs) ID() string { return r.id }

func (r unknownIDs) Description() string {
	return fmt.Sprintf("%s IDs used in code exist in the RevenueCat project", capitalize(r.noun))
}

func (r unknownIDs) NeedsDashboard() bool { return true }

func (r unknownIDs) Check(ctx *Context) []model.Finding {
	known := unique(r.known(ctx.Dashboard))
	exists := make(map[string]bool, len(known))
	for _, k := range known {
		exists[k] = true
	}
	var out []model.Finding
	for _, f := range ctx.Facts {
		for _, ref := range r.refs(f) {
			if exists[ref.ID] {
				continue
			}
			fd := model.Finding{
				Rule:     r.id,
				Severity: model.Error,
				Platform: f.Platform,
				Location: ref.Location,
				Message:  fmt.Sprintf("%s %q is used in code but doesn't exist in your RevenueCat project.", capitalize(r.noun), ref.ID),
			}
			if ref.Def != nil {
				fd.Message = fmt.Sprintf("%s %q (%s, set at %s) is used in code but doesn't exist in your RevenueCat project.",
					capitalize(r.noun), ref.ID, ref.Def.Name, ref.Def.Location)
			}
			switch match, ok := closest(ref.ID, known); {
			case ok:
				fd.Fix = fmt.Sprintf("Did you mean %q?", match)
				// A literal is fixed where it's used; a constant where it's defined.
				at, old := ref.Location, `"`+ref.ID+`"`
				if ref.Def != nil {
					at, old = ref.Def.Location, ref.Def.Text
				}
				fd.Edits = []model.Edit{{
					Path:        pathOf(f, at),
					File:        at.File,
					Line:        at.Line,
					Old:         old,
					New:         strings.Replace(old, ref.ID, match, 1),
					Safety:      model.Suggested,
					Description: fmt.Sprintf("Change %s %q to %q.", r.noun, ref.ID, match),
				}}
			case len(known) == 0:
				fd.Fix = fmt.Sprintf("Create %q in the RevenueCat dashboard. The project has no %ss yet.", ref.ID, r.noun)
			default:
				fd.Fix = fmt.Sprintf("Create %q in the RevenueCat dashboard, or use one that exists: %s.", ref.ID, quoteList(known, 5))
			}
			out = append(out, fd)
		}
	}
	return out
}

// closest returns the single known value within edit distance 2 of s. A tie
// means there is no clear intended value, so nothing is suggested.
func closest(s string, known []string) (string, bool) {
	best, bestDist, tie := "", 3, false
	for _, k := range known {
		switch d := levenshtein(s, k); {
		case d < bestDist:
			best, bestDist, tie = k, d, false
		case d == bestDist:
			tie = true
		}
	}
	return best, best != "" && !tie
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

func unique(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// quoteList renders up to limit values as "a", "b", and N more.
func quoteList(vals []string, limit int) string {
	var q []string
	for i, v := range vals {
		if i == limit {
			q = append(q, fmt.Sprintf("and %d more", len(vals)-limit))
			break
		}
		q = append(q, fmt.Sprintf("%q", v))
	}
	return strings.Join(q, ", ")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
