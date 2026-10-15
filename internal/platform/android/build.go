package android

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/platform/scan"
)

var (
	catalogFiles = []string{"libs.versions.toml"}

	// The SDK artifacts whose version decides the Play Billing Library in use.
	sdkCoordinate = regexp.MustCompile(`com\.revenuecat\.purchases:(purchases|purchases-kmp-core|purchases-kmp):([0-9][0-9A-Za-z.+-]*)`)
	catalogModule = regexp.MustCompile(`"com\.revenuecat\.purchases:(purchases|purchases-kmp-core|purchases-kmp)"\s*,\s*version(?:\.ref)?\s*=\s*"([^"]+)"`)
	catalogEntry  = regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+)\s*=\s*"([^"]+)"`)

	minSDKLiteral = regexp.MustCompile(`\bminSdk(?:Version)?\s*(?:=\s*|\s+)(\d+)\b`)
	minSDKCatalog = regexp.MustCompile(`\bminSdk(?:Version)?\s*=?\s*libs\.versions\.([A-Za-z0-9_.]+)\.get\(\)`)
	desugaring    = regexp.MustCompile(`\b(?:isCoreLibraryDesugaringEnabled|coreLibraryDesugaringEnabled)\s*(?:=\s*|\s+)true\b`)
	namespace     = regexp.MustCompile(`\bnamespace\s*=?\s*["']([\w.]+)["']`)

	packageDecl      = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)`)
	applicationClass = regexp.MustCompile(`(?m)^[^/\n]*\bclass\s+(\w+)(?:\s*\([^)]*\))?\s*(?::\s*(?:[\w.]+\.)?(?:MultiDex)?Application\s*\(|\s+extends\s+(?:[\w.]+\.)?(?:MultiDex)?Application\b)`)
	applicationTag   = regexp.MustCompile(`<application\b[^>]*>`)
	androidName      = regexp.MustCompile(`android:name\s*=\s*"([^"]+)"`)
	manifestPackage  = regexp.MustCompile(`<manifest\b[^>]*\bpackage\s*=\s*"([\w.]+)"`)
)

// BuildSettings reads the Gradle and manifest settings the rules need: the
// RevenueCat SDK artifact and version, minSdk, core library desugaring, the
// Application subclass that configures the SDK, and the manifest's
// <application android:name>. configureAt are the SDK's configure call sites.
// Settings that can't be read are left out.
func BuildSettings(root string, exclude *regexp.Regexp, configureAt []model.Location) map[string]model.Setting {
	out := map[string]model.Setting{}
	put := func(key, value string, at model.Location) {
		if _, done := out[key]; !done && value != "" {
			out[key] = model.Setting{Value: value, Location: at}
		}
	}

	appBuild := ""
	if m, ok := ApplicationID(root); ok {
		appBuild = m.Location.File
	}
	catalog := catalogVersions(root)

	if artifact, version, at, ok := sdkVersion(root, catalog); ok {
		put(model.BuildSDKArtifact, artifact, at)
		put(model.BuildSDKVersion, version, at)
	}
	if v, at, ok := minSDK(root, appBuild, catalog); ok {
		put(model.BuildMinSDK, v, at)
	}
	if m, ok := scan.FirstMatch(root, gradleFiles, regexp.MustCompile(`(`+desugaring.String()+`)`), nil); ok {
		put(model.BuildDesugaring, "true", m.Location)
	}

	ns := ""
	if appBuild != "" {
		for _, m := range scan.Matches(root, gradleFiles, namespace) {
			if m.Location.File == appBuild {
				ns = m.Value
				break
			}
		}
	}
	if cls, at, ok := applicationClassFor(root, configureAt); ok {
		put(model.BuildApplicationClass, cls, at)
	}
	if file, tag, name, at, ok := manifestApplication(root, exclude, appBuild); ok {
		put(model.BuildManifest, file, tag)
		if name != "" {
			put(model.BuildManifestAppName, qualify(name, ns), at)
		}
	}
	return out
}

// catalogVersions reads every name = "value" line of the version catalogs.
func catalogVersions(root string) map[string]scan.Match {
	out := map[string]scan.Match{}
	for _, m := range scan.Matches(root, catalogFiles, regexp.MustCompile(`^\s*([A-Za-z0-9_.-]+\s*=\s*"[^"]+")`)) {
		parts := catalogEntry.FindStringSubmatch(m.Value)
		if parts == nil {
			continue
		}
		if _, dup := out[parts[1]]; !dup {
			out[parts[1]] = scan.Match{Value: parts[2], Location: m.Location}
		}
	}
	return out
}

// sdkVersion finds the RevenueCat artifact and version, declared directly
// (com.revenuecat.purchases:purchases:8.10.8) or through a version catalog.
func sdkVersion(root string, catalog map[string]scan.Match) (artifact, version string, at model.Location, ok bool) {
	both := regexp.MustCompile(`(` + sdkCoordinate.String() + `)`)
	if m, found := scan.FirstMatch(root, append(append([]string{}, gradleFiles...), catalogFiles...), both, nil); found {
		parts := sdkCoordinate.FindStringSubmatch(m.Value)
		return parts[1], parts[2], m.Location, true
	}
	if m, found := scan.FirstMatch(root, catalogFiles, regexp.MustCompile(`(`+catalogModule.String()+`)`), nil); found {
		parts := catalogModule.FindStringSubmatch(m.Value)
		artifact, version, at = parts[1], parts[2], m.Location
		// version.ref = "x" names a [versions] entry; version = "1.2.3" is the version.
		if strings.Contains(m.Value, "version.ref") {
			ref, known := catalog[version]
			if !known {
				return "", "", model.Location{}, false
			}
			version, at = ref.Value, ref.Location
		}
		return artifact, version, at, true
	}
	return "", "", model.Location{}, false
}

// minSDK prefers the app module's build file, then any build file. It follows
// libs.versions.x.get() references into the version catalog.
func minSDK(root, appBuild string, catalog map[string]scan.Match) (string, model.Location, bool) {
	literals := scan.Matches(root, gradleFiles, minSDKLiteral)
	refs := scan.Matches(root, gradleFiles, minSDKCatalog)
	fromCatalog := func(ref string) (scan.Match, bool) {
		for _, key := range []string{ref, strings.ReplaceAll(ref, ".", "-"), strings.ReplaceAll(ref, ".", "_")} {
			if m, ok := catalog[key]; ok {
				return m, true
			}
		}
		return scan.Match{}, false
	}
	for _, preferApp := range []bool{true, false} {
		for _, m := range literals {
			if !preferApp || m.Location.File == appBuild {
				return m.Value, m.Location, true
			}
		}
		for _, r := range refs {
			if !preferApp || r.Location.File == appBuild {
				if m, ok := fromCatalog(r.Value); ok {
					return m.Value, m.Location, true
				}
			}
		}
		if appBuild == "" {
			break
		}
	}
	return "", model.Location{}, false
}

// applicationClassFor returns the Application subclass declared in a file
// that configures the SDK, fully qualified.
func applicationClassFor(root string, configureAt []model.Location) (string, model.Location, bool) {
	seen := map[string]bool{}
	for _, loc := range configureAt {
		if seen[loc.File] || !(strings.HasSuffix(loc.File, ".kt") || strings.HasSuffix(loc.File, ".java")) {
			continue
		}
		seen[loc.File] = true
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(loc.File)))
		if err != nil {
			continue
		}
		src := string(b)
		m := applicationClass.FindStringSubmatchIndex(src)
		if m == nil {
			continue
		}
		name := src[m[2]:m[3]]
		if p := packageDecl.FindStringSubmatch(src); p != nil {
			name = p[1] + "." + name
		}
		return name, model.Location{File: loc.File, Line: lineAt(src, m[0])}, true
	}
	return "", model.Location{}, false
}

// manifestApplication reads <application android:name> from the main manifest
// of the app module, or from the first main manifest found. tag is where the
// <application element starts.
func manifestApplication(root string, exclude *regexp.Regexp, appBuild string) (file string, tag model.Location, name string, at model.Location, ok bool) {
	var manifests []string
	_ = scan.Walk(root, exclude, func(_, rel string) bool {
		// src/main for Android modules, src/androidMain for Kotlin Multiplatform.
		if path.Base(rel) == "AndroidManifest.xml" && (strings.Contains(rel, "src/main/") || strings.Contains(rel, "src/androidMain/")) {
			manifests = append(manifests, rel)
		}
		return true
	})
	if len(manifests) == 0 {
		return "", model.Location{}, "", model.Location{}, false
	}
	file = manifests[0]
	if appBuild != "" {
		dir := path.Dir(appBuild)
		for _, m := range manifests {
			if m == path.Join(dir, "src/main/AndroidManifest.xml") || m == path.Join(dir, "src/androidMain/AndroidManifest.xml") {
				file = m
			}
		}
	}
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return "", model.Location{}, "", model.Location{}, false
	}
	src := string(b)
	span := applicationTag.FindStringIndex(src)
	if span == nil {
		return "", model.Location{}, "", model.Location{}, false // a library manifest
	}
	tag = model.Location{File: file, Line: lineAt(src, span[0])}
	attr := androidName.FindStringSubmatchIndex(src[span[0]:span[1]])
	if attr == nil {
		return file, tag, "", model.Location{}, true
	}
	name = src[span[0]+attr[2] : span[0]+attr[3]]
	if p := manifestPackage.FindStringSubmatch(src); p != nil && !strings.Contains(strings.TrimPrefix(name, "."), ".") {
		name = qualifyWith(name, p[1])
	}
	return file, tag, name, model.Location{File: file, Line: lineAt(src, span[0]+attr[0])}, true
}

// qualify resolves a manifest class name (".App", "App", "com.x.App")
// against the module namespace.
func qualify(name, ns string) string {
	if ns == "" || (!strings.HasPrefix(name, ".") && strings.Contains(name, ".")) {
		return name
	}
	return qualifyWith(name, ns)
}

func qualifyWith(name, pkg string) string {
	if strings.HasPrefix(name, ".") {
		return pkg + name
	}
	if !strings.Contains(name, ".") {
		return pkg + "." + name
	}
	return name
}

func lineAt(src string, offset int) int {
	return strings.Count(src[:offset], "\n") + 1
}
