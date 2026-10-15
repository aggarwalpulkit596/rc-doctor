package rules

import (
	"fmt"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func init() { Register(androidApplicationClass{}) }

// androidApplicationClass catches an Application subclass that configures the
// SDK but isn't registered in AndroidManifest.xml. Android then never creates
// it, so Purchases.configure never runs. RevenueCat's Android troubleshooting
// guide names this as the cause when no Purchases log output appears at all.
type androidApplicationClass struct{}

func (androidApplicationClass) ID() string { return "android-application-class" }

func (androidApplicationClass) Description() string {
	return "The Application subclass that configures the SDK is registered in AndroidManifest.xml"
}

func (androidApplicationClass) NeedsDashboard() bool { return false }

func (r androidApplicationClass) Check(ctx *Context) []model.Finding {
	var out []model.Finding
	for _, f := range ctx.Facts {
		cls, hasClass := f.Build[model.BuildApplicationClass]
		manifest, hasManifest := f.Build[model.BuildManifest]
		if f.Platform != "android" || !hasClass || !hasManifest {
			continue
		}
		registered, hasName := f.Build[model.BuildManifestAppName]
		if hasName && sameClass(registered.Value, cls.Value) {
			continue
		}
		fd := model.Finding{
			Rule:     r.ID(),
			Platform: f.Platform,
			Location: cls.Location,
		}
		if hasName {
			// Possibly a subclass registered on purpose, so only a warning.
			fd.Severity = model.Warning
			fd.Message = fmt.Sprintf("%s configures RevenueCat, but %s registers %s as the Application class, so %s may never be created.",
				simpleName(cls.Value), manifest.Value, registered.Value, simpleName(cls.Value))
			fd.Fix = fmt.Sprintf("Register %s as android:name on <application>, or configure RevenueCat in %s.", cls.Value, simpleName(registered.Value))
		} else {
			fd.Severity = model.Error
			fd.Message = fmt.Sprintf("%s configures RevenueCat, but %s doesn't register it as android:name on <application>, so Android never creates it and Purchases.configure never runs.",
				simpleName(cls.Value), manifest.Value)
			fd.Fix = fmt.Sprintf("Add android:name=%q to the <application> element in %s.", cls.Value, manifest.Value)
			fd.Edits = []model.Edit{{
				Path:        pathOf(f, manifest.Location),
				File:        manifest.Location.File,
				Line:        manifest.Location.Line,
				Old:         "<application",
				New:         fmt.Sprintf("<application android:name=%q", cls.Value),
				Safety:      model.Suggested,
				Description: fmt.Sprintf("Register %s in %s.", simpleName(cls.Value), manifest.Value),
			}}
		}
		out = append(out, fd)
	}
	return out
}

// sameClass compares a manifest name with a class. Relative manifest names
// (".App") are compared by simple name.
func sameClass(manifestName, class string) bool {
	if manifestName == class {
		return true
	}
	if strings.HasPrefix(manifestName, ".") || !strings.Contains(manifestName, ".") || !strings.Contains(class, ".") {
		return simpleName(manifestName) == simpleName(class)
	}
	return false
}

func simpleName(class string) string {
	return class[strings.LastIndexByte(class, '.')+1:]
}
