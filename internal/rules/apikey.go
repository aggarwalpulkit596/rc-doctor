package rules

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/redact"
)

func init() { Register(apiKey{}) }

// keyShaped is the shape of a real RevenueCat key. Anything else that reaches
// Purchases.configure is a placeholder ("REVENUECAT_API_KEY") or a typo.
var keyShaped = regexp.MustCompile(`^[a-z]{2,4}_[A-Za-z0-9]{16,}$`)

// secretPrefix marks RevenueCat secret API keys, which belong on servers only.
const secretPrefix = "sk_"

// apiKey checks every key found in code and config files:
//   - a placeholder passed to Purchases.configure is an error: nothing works;
//   - a secret key in an app is an error: it must never ship;
//   - a public key for another platform is an error: each app has its own key;
//   - with the dashboard loaded, a key that isn't from the project being
//     checked is a warning.
//
// It runs offline. The dashboard only adds the exact replacement key.
type apiKey struct{}

func (apiKey) ID() string { return "api-key" }

func (apiKey) Description() string {
	return "No secret keys in app code, and each app uses its own platform's public SDK key"
}

func (apiKey) NeedsDashboard() bool { return false }

func (r apiKey) Check(ctx *Context) []model.Finding {
	var out []model.Finding
	for _, f := range ctx.Facts {
		for _, k := range f.APIKeys {
			prefix := keyPrefix(k.Value)
			masked := redact.Keys(k.Value)
			fd := model.Finding{Rule: r.ID(), Platform: f.Platform, Location: k.Location}
			switch owner := ctx.KeyPrefixes[prefix]; {
			case !keyShaped.MatchString(k.Value):
				fd.Severity = model.Error
				fd.Message = fmt.Sprintf("Purchases is configured with %q, which isn't a RevenueCat API key. It looks like a placeholder that was never replaced.", k.Value)
				fd.Fix, fd.Edits = propose(ctx, f, k, "")
			case prefix == secretPrefix:
				fd.Severity = model.Error
				fd.Message = fmt.Sprintf("Secret API key %s is in app code. Secret keys are for servers only and must never ship in an app.", masked)
				fd.Fix, fd.Edits = propose(ctx, f, k, " Then rotate the secret key in the RevenueCat dashboard: anyone with your code or app binary can read it.")
			case slices.Contains(f.KeyPrefixes, prefix):
				if !foreignKey(ctx.Dashboard, k.Value) {
					continue
				}
				fd.Severity = model.Warning
				fd.Message = fmt.Sprintf("%s isn't a public key of any app in the RevenueCat project being checked. If it belongs to another project, purchases are recorded there.", masked)
				fd.Fix, fd.Edits = propose(ctx, f, k, "")
			case owner != "":
				fd.Severity = model.Error
				fd.Message = fmt.Sprintf("%s is a public key for %s apps, but this is the %s app. Each RevenueCat app has its own public SDK key.", masked, owner, f.Platform)
				fd.Fix, fd.Edits = propose(ctx, f, k, "")
			default:
				continue // a prefix no platform claims: web billing, test store, or not a key at all
			}
			out = append(out, fd)
		}
	}
	return out
}

// propose suggests the public key this app should use.
//
// The edit is Safe only when the dashboard proves the code belongs to the
// project being checked, which is when the key being replaced is itself one of
// the project's keys (an iOS key from this project pasted into the Android
// app). Otherwise it is Suggested: teams often keep staging and production in
// separate projects with the same bundle ID, and silently swapping in the
// wrong project's key would be worse than the original bug.
func propose(ctx *Context, f model.Facts, k model.APIKey, note string) (string, []model.Edit) {
	key, ok := publicKeyFor(ctx.Dashboard, f)
	if !ok {
		return fmt.Sprintf("Replace it with the %s app's public SDK key from the API keys page of the RevenueCat dashboard (or `rc apps keys <app-id>`).%s", f.Platform, note), nil
	}
	safety := model.Suggested
	if projectHasKey(ctx.Dashboard, k.Value) {
		safety = model.Safe
	}
	text := fmt.Sprintf("Use the %s app's public SDK key (%s).%s", f.Platform, redact.Keys(key), note)
	return text, []model.Edit{{
		Path:        pathOf(f, k.Location),
		File:        k.Location.File,
		Line:        k.Location.Line,
		Old:         k.Value,
		New:         key,
		Safety:      safety,
		Description: text,
	}}
}

// publicKeyFor returns the one public key the dashboard lists for f's
// platform, preferring apps whose identifier matches f.AppID. No key, or more
// than one candidate, means there is no single right answer to propose.
func publicKeyFor(d *model.Dashboard, f model.Facts) (string, bool) {
	if d == nil {
		return "", false
	}
	apps := d.AppsFor(f.Platform)
	if f.AppID != "" {
		var same []model.App
		for _, a := range apps {
			if a.AppID == f.AppID {
				same = append(same, a)
			}
		}
		if len(same) > 0 {
			apps = same
		}
	}
	var keys []string
	for _, a := range apps {
		for _, key := range a.PublicKeys {
			if slices.Contains(f.KeyPrefixes, keyPrefix(key)) && !slices.Contains(keys, key) {
				keys = append(keys, key)
			}
		}
	}
	if len(keys) != 1 {
		return "", false
	}
	return keys[0], true
}

func projectHasKey(d *model.Dashboard, key string) bool {
	if d == nil {
		return false
	}
	for _, a := range d.Apps {
		if slices.Contains(a.PublicKeys, key) {
			return true
		}
	}
	return false
}

// foreignKey reports whether the dashboard lists public keys and key isn't
// one of them. Without listed keys there is nothing to compare against.
func foreignKey(d *model.Dashboard, key string) bool {
	if d == nil {
		return false
	}
	listed := false
	for _, a := range d.Apps {
		listed = listed || len(a.PublicKeys) > 0
	}
	return listed && !projectHasKey(d, key)
}

// keyPrefix returns the key's prefix including the underscore ("appl_").
func keyPrefix(key string) string {
	if i := strings.IndexByte(key, '_'); i > 0 {
		return key[:i+1]
	}
	return ""
}
