package rcdoctor_test

import (
	"io/fs"
	"strings"
	"testing"

	rcdoctor "github.com/aggarwalpulkit596/rc-doctor"
	"github.com/aggarwalpulkit596/rc-doctor/internal/rules"
)

// Every rule has a documentation page, and every page belongs to a rule, so
// the docs that reports link to can't drift from the rules that exist.
func TestEveryRuleIsDocumented(t *testing.T) {
	pages := map[string]bool{}
	entries, err := fs.ReadDir(rcdoctor.RuleDocs, "docs/rules")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		pages[strings.TrimSuffix(e.Name(), ".md")] = true
	}
	for _, r := range rules.All() {
		if !pages[r.ID()] {
			t.Errorf("rule %s has no docs/rules/%s.md", r.ID(), r.ID())
			continue
		}
		page, _ := rcdoctor.RuleDocs.ReadFile("docs/rules/" + r.ID() + ".md")
		if !strings.HasPrefix(string(page), "# "+r.ID()+"\n") {
			t.Errorf("docs/rules/%s.md must start with its rule ID as the title", r.ID())
		}
		delete(pages, r.ID())
	}
	for page := range pages {
		t.Errorf("docs/rules/%s.md documents a rule that doesn't exist", page)
	}
}
