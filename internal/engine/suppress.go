package engine

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/config"
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// Directive is the comment that silences findings on its own line or the line
// below it. It works in any comment syntax:
//
//	val key = "goog_..." // rc-doctor:ignore api-key -- staging key, checked by hand
//	# rc-doctor:ignore
//	<!-- rc-doctor:ignore android-application-class -->
//
// With rule IDs, only those rules are silenced; without, every rule is.
// Anything after "--" is a free-text reason.
const Directive = "rc-doctor:ignore"

var ruleToken = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// suppress applies severity overrides and splits findings into those to
// report and those silenced by config ignore patterns or ignore comments.
func suppress(root string, findings []model.Finding, cfg *config.Config) (kept []model.Finding, silenced []Suppressed) {
	files := map[string][]string{}
	lines := func(rel string) []string {
		if l, ok := files[rel]; ok {
			return l
		}
		var l []string
		if f, err := os.Open(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
			for sc.Scan() {
				l = append(l, sc.Text())
			}
			_ = f.Close() // read only
		}
		files[rel] = l
		return l
	}
	for _, f := range findings {
		if sev, ok := cfg.Severity(f.Rule); ok {
			f.Severity = sev
		}
		switch {
		case f.Location.File != "" && cfg.Ignored(f.Location.File):
			silenced = append(silenced, Suppressed{Finding: f, By: SuppressedByConfig})
		case f.Location.File != "" && ignoredByComment(lines(f.Location.File), f.Location.Line, f.Rule):
			silenced = append(silenced, Suppressed{Finding: f, By: SuppressedByComment})
		default:
			kept = append(kept, f)
		}
	}
	return kept, silenced
}

// ignoredByComment reports whether line (1-based) or the line above it holds
// a directive that covers rule.
func ignoredByComment(lines []string, line int, rule string) bool {
	for _, n := range []int{line, line - 1} {
		if n < 1 || n > len(lines) {
			continue
		}
		if covers(lines[n-1], rule) {
			return true
		}
	}
	return false
}

func covers(text, rule string) bool {
	i := strings.Index(text, Directive)
	if i < 0 {
		return false
	}
	rest := text[i+len(Directive):]
	for _, end := range []string{"--", "*/", "-->"} {
		if j := strings.Index(rest, end); j >= 0 {
			rest = rest[:j]
		}
	}
	var named []string
	for _, tok := range strings.FieldsFunc(rest, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' }) {
		if ruleToken.MatchString(tok) {
			named = append(named, tok)
		}
	}
	if len(named) == 0 {
		return true
	}
	for _, n := range named {
		if n == rule {
			return true
		}
	}
	return false
}
