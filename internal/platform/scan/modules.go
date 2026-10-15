package scan

import (
	"os"
	"path"
	"regexp"
	"strings"
)

var (
	// jvmModule marks a Gradle module built for the JVM: a server, a
	// command-line tool, a backend for the app.
	jvmModule = regexp.MustCompile(`kotlin\(\s*"jvm"\s*\)|org\.jetbrains\.kotlin\.jvm|plugins\.kotlin\.jvm|\bio\.ktor\.plugin\b|plugins\.ktor\b|^\s*application\s*$|id\(?\s*["']application["']`)

	// clientModule marks a module that builds an app or a library for one:
	// the Android Gradle Plugin, or Kotlin Multiplatform.
	clientModule = regexp.MustCompile(`com\.android\.|plugins\.android|androidApplication|androidLibrary|kotlin\(\s*"multiplatform"\s*\)|kotlin\.multiplatform|plugins\.kotlin\.multiplatform|kotlinMultiplatform|\bandroid\s*\{`)
)

// JVMModules returns the root-relative directories of Gradle modules that
// build for the JVM only, such as a Ktor server next to the app. Their code
// never ships in the app, so a secret key there is where it belongs. A module
// counts only when its build file says so and mentions neither the Android
// Gradle Plugin nor Kotlin Multiplatform.
func JVMModules(root string, exclude *regexp.Regexp) []string {
	var out []string
	_ = Walk(root, exclude, func(p, rel string) bool {
		base := path.Base(rel)
		if base != "build.gradle" && base != "build.gradle.kts" {
			return true
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return true
		}
		text := string(b)
		if dir := path.Dir(rel); dir != "." && multiline(jvmModule, text) && !clientModule.MatchString(text) {
			out = append(out, dir)
		}
		return true
	})
	return out
}

// multiline matches re against each line of text, so ^ and $ anchor to lines.
func multiline(re *regexp.Regexp, text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// ExcludeDirs returns a pattern that matches what base matches, plus
// everything under dirs (root-relative, slash-separated).
func ExcludeDirs(base *regexp.Regexp, dirs []string) *regexp.Regexp {
	if len(dirs) == 0 {
		return base
	}
	quoted := make([]string, len(dirs))
	for i, d := range dirs {
		quoted[i] = regexp.QuoteMeta(d)
	}
	under := `^(?:` + strings.Join(quoted, "|") + `)/`
	if base == nil {
		return regexp.MustCompile(under)
	}
	return regexp.MustCompile(`(?:` + base.String() + `)|` + under)
}
