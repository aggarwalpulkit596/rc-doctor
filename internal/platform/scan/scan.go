// Package scan holds the file-walking and matching helpers shared by platform
// modules.
//
// Platforms mostly differ in which files they read and which patterns they look
// for, so a new platform is usually a Spec plus an app-identifier extractor,
// not new scanning code.
package scan

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// skipDirs are never scanned: VCS and IDE state, build output and dependency checkouts.
var skipDirs = map[string]bool{
	".git": true, ".idea": true, ".gradle": true, ".kotlin": true, ".cxx": true, "build": true,
	".build": true, ".swiftpm": true, "DerivedData": true, "Pods": true, "Carthage": true,
	"node_modules": true, ".dart_tool": true, "vendor": true,
}

// Patterns says what to look for in files with the given extensions.
// Every regexp except Configure must have one capture group: the value to extract.
type Patterns struct {
	Extensions []string // matched as suffixes: ".kt", ".gradle.kts"
	// Include, when set, limits the set to files whose root-relative path
	// matches, such as one Kotlin Multiplatform source set.
	Include *regexp.Regexp

	APIKey       []*regexp.Regexp // captures a key
	Configure    []*regexp.Regexp // a match marks an SDK configure call site
	Entitlements []*regexp.Regexp // captures an entitlement ID literal
	Offerings    []*regexp.Regexp // captures an offering ID literal

	// EntitlementIdents and OfferingIdents capture an identifier used in place
	// of a literal (entitlements[ENTITLEMENT_ID]). It is resolved through the
	// Constants table; unresolved identifiers are dropped.
	EntitlementIdents []*regexp.Regexp
	OfferingIdents    []*regexp.Regexp
	// KeyIdents capture an identifier passed as the API key. The resolved
	// value is reported at its definition, where a fix belongs.
	KeyIdents []*regexp.Regexp
}

// Spec is everything a platform scans for in one walk of the tree.
// A file matching two sets is scanned by both.
type Spec struct {
	Sets []Patterns
	// Exclude skips files and directories whose root-relative, slash-separated
	// path it matches (directories are matched with a trailing "/"). Platforms
	// use it for test sources and for code that belongs to another platform.
	Exclude *regexp.Regexp
	// Constants resolves identifiers. When nil and a set has ident patterns,
	// Source builds it with Constants(root, Exclude).
	Constants ConstTable
}

// Result is what Source found.
type Result struct {
	APIKeys []model.APIKey
	// KeyHints is parallel to APIKeys: the platform the key's line names
	// ("ios" for REVENUECAT_IOS_KEY=...), or "" when it names none.
	KeyHints     []string
	ConfigureAt  []model.Location
	Entitlements []model.Ref
	Offerings    []model.Ref
}

// Source walks root once and applies each pattern set to the files it covers.
func Source(root string, spec Spec) (Result, error) {
	var res Result
	consts := spec.Constants
	if consts == nil && needsConstants(spec) {
		consts = Constants(root, spec.Exclude)
	}
	err := Walk(root, spec.Exclude, func(path, rel string) bool {
		for _, p := range spec.Sets {
			if hasExt(path, p.Extensions) && (p.Include == nil || p.Include.MatchString(rel)) {
				scanFile(path, rel, p, consts, &res)
			}
		}
		return true
	})
	return res, err
}

func needsConstants(spec Spec) bool {
	for _, p := range spec.Sets {
		if len(p.EntitlementIdents)+len(p.OfferingIdents)+len(p.KeyIdents) > 0 {
			return true
		}
	}
	return false
}

// Walk calls fn for every regular file under root in lexical order, skipping
// skipDirs and anything exclude matches. rel is slash-separated and relative
// to root. fn returns false to stop the walk early.
func Walk(root string, exclude *regexp.Regexp, fn func(path, rel string) bool) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			return nil // unreadable entries are skipped, not fatal
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if skipDirs[d.Name()] || (exclude != nil && exclude.MatchString(rel+"/")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || (exclude != nil && exclude.MatchString(rel)) {
			return nil
		}
		if !fn(path, rel) {
			return filepath.SkipAll
		}
		return nil
	})
}

// Match is one capture found by Matches.
type Match struct {
	Value    string
	Location model.Location
}

// Matches returns every first-group capture of re in the files under root
// whose base name is in names, in walk order. Comments are ignored.
func Matches(root string, names []string, re *regexp.Regexp) []Match {
	var out []Match
	_ = Walk(root, nil, func(path, rel string) bool {
		if !hasName(path, names) {
			return true
		}
		eachLine(path, func(n int, text string) {
			for _, m := range re.FindAllStringSubmatch(text, -1) {
				if len(m) > 1 && m[1] != "" {
					out = append(out, Match{Value: m[1], Location: model.Location{File: rel, Line: n}})
				}
			}
		})
		return true
	})
	return out
}

// FirstMatch returns the first match that keep accepts (nil keeps every match).
func FirstMatch(root string, names []string, re *regexp.Regexp, keep func(string) bool) (Match, bool) {
	for _, m := range Matches(root, names, re) {
		if keep == nil || keep(m.Value) {
			return m, true
		}
	}
	return Match{}, false
}

// Contains reports whether any file under root whose base name is in names
// contains one of substrs.
func Contains(root string, names []string, substrs ...string) bool {
	found := false
	_ = Walk(root, nil, func(path, _ string) bool {
		if !hasName(path, names) {
			return true
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return true
		}
		for _, s := range substrs {
			if strings.Contains(string(b), s) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

func scanFile(path, rel string, p Patterns, consts ConstTable, res *Result) {
	eachLine(path, func(n int, text string) {
		loc := model.Location{File: rel, Line: n}
		for _, v := range captures(p.APIKey, text) {
			res.addKey(model.APIKey{Value: v, Location: loc}, PlatformHint(strings.Replace(text, v, "", 1)))
		}
		for _, ident := range captures(p.KeyIdents, text) {
			if def, ok := consts.Lookup(ident); ok {
				res.addKey(model.APIKey{Value: def.Value, Location: def.Location}, PlatformHint(def.Name))
			}
		}
		for _, ident := range captures(p.EntitlementIdents, text) {
			if def, ok := consts.Lookup(ident); ok {
				res.Entitlements = append(res.Entitlements, model.Ref{ID: def.Value, Location: loc, Def: &def.Definition})
			}
		}
		for _, ident := range captures(p.OfferingIdents, text) {
			if def, ok := consts.Lookup(ident); ok {
				res.Offerings = append(res.Offerings, model.Ref{ID: def.Value, Location: loc, Def: &def.Definition})
			}
		}
		for _, re := range p.Configure {
			if re.MatchString(text) {
				res.ConfigureAt = append(res.ConfigureAt, loc)
				break
			}
		}
		for _, v := range captures(p.Entitlements, text) {
			res.Entitlements = append(res.Entitlements, model.Ref{ID: v, Location: loc})
		}
		for _, v := range captures(p.Offerings, text) {
			res.Offerings = append(res.Offerings, model.Ref{ID: v, Location: loc})
		}
	})
}

// eachLine calls fn for every line of the file that isn't a comment. It knows
// line comments (//, "# ") and /* */ blocks that start a line, which covers
// commented-out code in every language rc-doctor reads without parsing them.
func eachLine(path string, fn func(n int, text string)) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	inBlock := false
	for n := 1; sc.Scan(); n++ {
		text := sc.Text()
		t := strings.TrimSpace(text)
		switch {
		case inBlock:
			inBlock = !strings.Contains(t, "*/")
			continue
		case strings.HasPrefix(t, "/*"):
			inBlock = !strings.Contains(t[2:], "*/")
			continue
		case isLineComment(t):
			continue
		}
		fn(n, text)
	}
}

// isLineComment recognises // and KDoc/Javadoc continuation lines, plus "# "
// comments in properties, TOML and Ruby files. A bare "#" followed by a word
// is code in Swift (#if) and Objective-C (#define), so it is not a comment.
func isLineComment(t string) bool {
	return strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*") ||
		t == "#" || strings.HasPrefix(t, "# ") || strings.HasPrefix(t, "#\t") || strings.HasPrefix(t, "##")
}

// captures returns the distinct first-group captures of res in text.
func captures(res []*regexp.Regexp, text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, re := range res {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			if len(m) > 1 && m[1] != "" && !seen[m[1]] {
				seen[m[1]] = true
				out = append(out, m[1])
			}
		}
	}
	return out
}

func hasExt(path string, exts []string) bool {
	for _, e := range exts {
		if strings.HasSuffix(path, e) {
			return true
		}
	}
	return false
}

func hasName(path string, names []string) bool {
	base := filepath.Base(path)
	for _, n := range names {
		if base == n {
			return true
		}
	}
	return false
}

// KeysNotFor returns the keys whose line doesn't name another platform. A
// properties file read by the Android build can also hold REVENUECAT_IOS_KEY;
// that key belongs to the iOS app and must not be checked as Android's.
func (r Result) KeysNotFor(other string) []model.APIKey {
	var out []model.APIKey
	for i, k := range r.APIKeys {
		if r.KeyHints[i] != other {
			out = append(out, k)
		}
	}
	return out
}

// addKey records a key once per value and location: a constant's literal is
// found both by the key patterns and through the configure call that uses it.
func (r *Result) addKey(k model.APIKey, hint string) {
	for _, seen := range r.APIKeys {
		if seen == k {
			return
		}
	}
	r.APIKeys = append(r.APIKeys, k)
	r.KeyHints = append(r.KeyHints, hint)
}
