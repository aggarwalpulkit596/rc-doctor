package rules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func init() { Register(playBillingDesugaring{}) }

// playBillingDesugaring warns about Android apps that can crash on old
// devices. RevenueCat's Android SDK 8.10.8 moved to Play Billing Library
// 7.1.1 (see its CHANGELOG), which uses Java 8 types some devices below
// Android 7.0 (API 24) lack. RevenueCat's troubleshooting guide: enable core
// library desugaring, or raise minSdk to 24.
type playBillingDesugaring struct{}

// desugaringSince is the first Android SDK release on Play Billing Library 7.1.
var desugaringSince = []int{8, 10, 8}

func (playBillingDesugaring) ID() string { return "play-billing-desugaring" }

func (playBillingDesugaring) Description() string {
	return "Android apps below minSdk 24 enable core library desugaring for Play Billing Library 7.1+"
}

func (playBillingDesugaring) NeedsDashboard() bool { return false }

func (r playBillingDesugaring) Check(ctx *Context) []model.Finding {
	var out []model.Finding
	for _, f := range ctx.Facts {
		minSDK, ok := f.Build[model.BuildMinSDK]
		if f.Platform != "android" || !ok {
			continue
		}
		n, err := strconv.Atoi(minSDK.Value)
		if err != nil || n >= 24 {
			continue
		}
		if _, enabled := f.Build[model.BuildDesugaring]; enabled {
			continue
		}
		// Only the native SDK's version maps directly to a Billing Library
		// version. An older native SDK is fine; anything else is assumed current.
		if f.Build[model.BuildSDKArtifact].Value == "purchases" {
			if v, ok := parseVersion(f.Build[model.BuildSDKVersion].Value); ok && versionLess(v, desugaringSince) {
				continue
			}
		}
		out = append(out, model.Finding{
			Rule:     r.ID(),
			Severity: model.Warning,
			Platform: f.Platform,
			Location: minSDK.Location,
			Message:  fmt.Sprintf("minSdk %d without core library desugaring. The Play Billing Library in RevenueCat's Android SDK 8.10.8 and later uses Java 8 types that some devices below Android 7.0 (API 24) lack, and the app can crash there with NoClassDefFoundError.", n),
			Fix:      "Enable core library desugaring (isCoreLibraryDesugaringEnabled = true in compileOptions, plus the com.android.tools:desugar_jdk_libs dependency), or raise minSdk to 24.",
		})
	}
	return out
}

// parseVersion reads the leading numeric parts of a version such as "8.10.8"
// or "9.1.0-beta". It fails on anything else, such as a Gradle variable.
func parseVersion(s string) ([]int, bool) {
	if s == "" {
		return nil, false
	}
	s = strings.SplitN(s, "-", 2)[0]
	var out []int
	for _, part := range strings.Split(s, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

func versionLess(a, b []int) bool {
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x < y
		}
	}
	return false
}
