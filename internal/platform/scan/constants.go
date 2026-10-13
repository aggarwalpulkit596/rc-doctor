package scan

import (
	"regexp"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// Const is one resolved constant: its value and where that value is written.
type Const struct {
	model.Definition
	Value string
}

// ConstTable maps constant names to their definitions. Build it with Constants.
type ConstTable map[string][]constEntry

type constEntry struct {
	Const
	tier int // lower wins: see the tier constants
}

// Where a value comes from decides which definition wins when a name is
// defined more than once: a properties file is what the build actually reads,
// so it beats a constant in code, which beats a Gradle getProperty fallback.
const (
	tierProperties = iota
	tierCode
	tierGradleDefault
)

var (
	// const val ID = "pro", val ID: String = "pro", static let id = "pro",
	// let id = "pro", static final String ID = "pro".
	codeConst = regexp.MustCompile(`(?:\bconst\s+val|\bval|\bvar|\blet|\bString)\s+([A-Za-z_]\w*)\s*(?::\s*String\s*)?=\s*("[^"\\]*")`)
	// NAME=value in a properties file.
	propertyLine = regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+)\s*[=:]\s*(\S(?:.*\S)?)\s*$`)
	// getProperty("NAME", "default") in a Gradle script.
	gradleDefault = regexp.MustCompile(`getProperty\(\s*"([A-Za-z0-9_.-]+)"\s*,\s*("[^"\\]*")\s*\)`)
)

// Constants collects string constants from source, properties and Gradle
// files under root, skipping anything exclude matches. It is deliberately
// simple: it resolves names, not scopes, and a name defined with different
// values at the same precedence is treated as unknown.
func Constants(root string, exclude *regexp.Regexp) ConstTable {
	t := ConstTable{}
	_ = Walk(root, exclude, func(path, rel string) bool {
		var re *regexp.Regexp
		tier := tierCode
		switch {
		case hasExt(path, []string{".properties"}):
			re, tier = propertyLine, tierProperties
		case hasExt(path, []string{".gradle", ".gradle.kts"}):
			re, tier = gradleDefault, tierGradleDefault
		case hasExt(path, []string{".kt", ".java", ".swift"}):
			re = codeConst
		default:
			return true
		}
		eachLine(path, func(n int, text string) {
			for _, m := range re.FindAllStringSubmatch(text, -1) {
				value := strings.Trim(m[2], `"`)
				t[m[1]] = append(t[m[1]], constEntry{tier: tier, Const: Const{
					Value: value,
					Definition: model.Definition{
						Name:     m[1],
						Location: model.Location{File: rel, Line: n},
						Text:     m[2],
					},
				}})
			}
		})
		return true
	})
	return t
}

// Lookup resolves an identifier such as Config.ENTITLEMENT_ID by its last
// segment. It fails when the name is unknown or ambiguous.
func (t ConstTable) Lookup(ident string) (Const, bool) {
	if i := strings.LastIndexByte(ident, '.'); i >= 0 {
		ident = ident[i+1:]
	}
	entries := t[ident]
	best := -1
	for _, e := range entries {
		if best == -1 || e.tier < best {
			best = e.tier
		}
	}
	var found *Const
	for i, e := range entries {
		if e.tier != best {
			continue
		}
		if found != nil && found.Value != e.Value {
			return Const{}, false // same precedence, different values: don't guess
		}
		if found == nil {
			found = &entries[i].Const
		}
	}
	if found == nil || found.Value == "" {
		return Const{}, false
	}
	return *found, true
}
