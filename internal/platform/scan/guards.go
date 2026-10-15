package scan

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// lookback bounds how far KeyGuards walks up from a key looking for the
// blocks that enclose it. Guards are close to the keys they protect.
const lookback = 80

var (
	// debugSourceSet matches Android source sets that only debug builds
	// compile: src/debug/ and a flavor's src/<flavor>Debug/.
	debugSourceSet = regexp.MustCompile(`(^|/)src/(?:debug|[a-z]\w*Debug)/`)

	// debugConfigName matches per-configuration settings files such as
	// Debug.xcconfig, Config-Debug.xcconfig or secrets.debug.properties.
	debugConfigName = regexp.MustCompile(`(?i)(^|[._-])debug([._-]|$)`)

	// debugGuard matches a condition or block that only debug builds enter:
	// Swift's #if DEBUG, Android's BuildConfig.DEBUG (not negated), and a
	// Gradle debug build type.
	debugGuard = regexp.MustCompile(`#if\s+(?:[^!\n]*\s)?DEBUG\b|(?:^|[^!\w])(?:[\w.]*\.)?BuildConfig\.DEBUG\b|\bdebug\s*$|\b(?:getByName|named|create|maybeCreate)\(\s*"debug"\s*\)\s*$`)

	elseWord = regexp.MustCompile(`\belse\b`)
)

// KeyGuards reads the code around the key at k.Location and reports whether
// the key is debug-only, and which platform the condition around it names.
//
// A key is debug-only in a debug source set (src/debug/, src/freeDebug/) or
// a debug settings file (Debug.xcconfig, secrets-debug.properties), or when
// it sits behind a debug guard: before the else of a guard on its own line
// (if (BuildConfig.DEBUG) "test_..." else "goog_..."), or inside an #if DEBUG
// block, an if (BuildConfig.DEBUG) { ... } block or a Gradle debug { ... }
// build type.
//
// The platform is the one named by the innermost enclosing block or #if that
// names one: "android" for a key inside if (platform is Platform.Android) { },
// "ios" inside #if os(iOS). A key in the else branch of a condition that names
// one platform belongs to the other. The key's own line is left to the caller.
//
// KeyGuards reads files without parsing them, so it is a heuristic: a guard
// it doesn't recognize leaves the key reachable from every build.
func KeyGuards(root string, k model.APIKey) (debugOnly bool, platform string) {
	rel := k.Location.File
	if debugSourceSet.MatchString(rel) {
		debugOnly = true
	}
	switch path.Ext(rel) {
	case ".xcconfig", ".plist", ".properties":
		if debugConfigName.MatchString(path.Base(rel)) {
			debugOnly = true
		}
	}
	lines := readLines(filepath.Join(root, filepath.FromSlash(rel)), k.Location.Line)
	if k.Location.Line < 1 || len(lines) < k.Location.Line {
		return debugOnly, ""
	}
	own, heads, excluded := enclosing(lines, k.Location.Line-1, k.Value)
	debugOnly = debugOnly || guardBefore(own)
	for _, h := range heads {
		debugOnly = debugOnly || guardBefore(h)
		if platform == "" {
			platform = PlatformHint(h)
		}
	}
	if platform == "" {
		platform = otherPlatform[PlatformHint(excluded)]
	}
	return debugOnly, platform
}

// otherPlatform names the platform an else branch is left with.
var otherPlatform = map[string]string{"android": "ios", "ios": "android"}

// enclosing walks up from the key on line i (0-based). It returns the key's
// own line up to the key, and the heads of the blocks around the key,
// innermost first: the text before each { that opens one, and each #if the
// key sits in. The walk ends after a head with an else in it, or at an #else:
// the key is in a branch that excludes a condition, which is returned too.
func enclosing(lines []string, i int, key string) (own string, heads []string, excluded string) {
	line := lines[i]
	col := strings.Index(line, key)
	if col < 0 {
		col = len(line)
	}
	own = line[:col]
	need := 0   // closing braces seen, whose openers are further up
	endifs := 0 // #endif lines seen, whose #if is further up
	for j := i; j >= 0 && j >= i-lookback; j-- {
		text := lines[j]
		if j == i {
			text = own
		}
		t := strings.TrimSpace(text)
		switch {
		case strings.HasPrefix(t, "#endif"):
			endifs++
			continue
		case strings.HasPrefix(t, "#else"), strings.HasPrefix(t, "#elseif"), strings.HasPrefix(t, "#elif"):
			if endifs == 0 {
				return own, heads, ifDirective(lines, j) // the key is in the other branch of an #if
			}
			continue
		case strings.HasPrefix(t, "#if"):
			if endifs > 0 {
				endifs--
				continue
			}
			heads = append(heads, t)
			continue
		case strings.HasPrefix(t, "//"), strings.HasPrefix(t, "*"), strings.HasPrefix(t, "/*"):
			continue
		}
		for c := len(text) - 1; c >= 0; c-- {
			switch text[c] {
			case '}':
				need++
			case '{':
				if need > 0 {
					need--
					continue
				}
				head := text[:c] // text[c] opens a block around the key
				heads = append(heads, head)
				if elseWord.MatchString(head) {
					return own, heads, ifHead(lines, j, c, i-lookback)
				}
			}
		}
	}
	return own, heads, ""
}

// ifHead finds the condition of the block an else on line j, before column
// c, follows: the head of the block the first } to its left closes.
func ifHead(lines []string, j, c, stop int) string {
	need, seen := 0, false
	for ; j >= 0 && j >= stop; j-- {
		text := lines[j][:c]
		for k := len(text) - 1; k >= 0; k-- {
			switch text[k] {
			case '}':
				need++
				seen = true
			case '{':
				if need--; seen && need == 0 {
					return text[:k]
				}
			}
		}
		if j > 0 {
			c = len(lines[j-1])
		}
	}
	return ""
}

// ifDirective finds the #if that the #else on line j belongs to.
func ifDirective(lines []string, j int) string {
	depth := 0
	for j--; j >= 0; j-- {
		t := strings.TrimSpace(lines[j])
		switch {
		case strings.HasPrefix(t, "#endif"):
			depth++
		case strings.HasPrefix(t, "#if"):
			if depth == 0 {
				return t
			}
			depth--
		}
	}
	return ""
}

// guardBefore reports whether head ends in a debug guard: the last guard in
// it is not followed by an else.
func guardBefore(head string) bool {
	locs := debugGuard.FindAllStringIndex(head, -1)
	if len(locs) == 0 {
		return false
	}
	return !elseWord.MatchString(head[locs[len(locs)-1][1]:])
}

// readLines returns the first n lines of the file at p.
func readLines(p string, n int) []string {
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }() // read only
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var out []string
	for len(out) < n && sc.Scan() {
		out = append(out, sc.Text())
	}
	return out
}
