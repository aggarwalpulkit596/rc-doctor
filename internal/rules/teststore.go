package rules

import (
	"fmt"
	"slices"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/redact"
)

func init() { Register(testStoreKey{}) }

// testStorePrefix marks Test Store API keys. Every SDK accepts them, so no
// platform claims the prefix.
const testStorePrefix = "test_"

// testStoreKey reports Test Store API keys that release builds can reach.
//
// Both native SDKs stop a release build that is configured with a Test Store
// key: it crashes on launch (after showing an error, on Android) unless the
// app sets forceAllowTestStoreInReleaseBuilds. App Review rejects apps that
// use one either way, and Test Store purchases earn no revenue.
//
// Keys the platform module found behind a debug-only guard are fine. A key
// that reaches Purchases.configure is a warning (the rule can't see build
// configurations, so it doesn't err); a key that is only defined, in a
// constant or a settings file rc-doctor can't follow to the call, is info.
type testStoreKey struct{}

func (testStoreKey) ID() string { return "test-store-key" }

func (testStoreKey) Description() string {
	return "Test Store API keys are only used by debug builds"
}

func (testStoreKey) NeedsDashboard() bool { return false }

func (r testStoreKey) Check(ctx *Context) []model.Finding {
	var out []model.Finding
	for _, f := range ctx.Facts {
		storeKey := false
		for _, k := range f.APIKeys {
			storeKey = storeKey || slices.Contains(f.KeyPrefixes, keyPrefix(k.Value))
		}
		optedOut := f.HasSignal(model.SignalTestStoreRelease)
		for _, k := range f.APIKeys {
			if keyPrefix(k.Value) != testStorePrefix || k.DebugOnly {
				continue
			}
			masked := redact.Keys(k.Value)
			fd := model.Finding{
				Rule:     r.ID(),
				Severity: model.Warning,
				Platform: f.Platform,
				Location: k.Location,
				Fix: "Use the Test Store key in debug builds only (#if DEBUG, BuildConfig.DEBUG, a debug build type or a debug source set) " +
					"and the store's public SDK key in release builds (`rc apps keys <app-id>`). " +
					"If only debug builds reach this line, add `rc-doctor:ignore test-store-key -- debug only` to it.",
			}
			switch {
			case !k.Configured:
				// Defined here, used somewhere rc-doctor can't follow.
				fd.Severity = model.Info
				fd.Message = fmt.Sprintf("Test Store API key %s is defined here, and rc-doctor can't tell which builds use it. A release build configured with a Test Store key crashes on launch, and App Review rejects it.", masked)
			case optedOut:
				fd.Message = fmt.Sprintf("Test Store API key %s isn't behind a debug-only check that rc-doctor recognizes, and forceAllowTestStoreInReleaseBuilds lets release builds run with it. App Review rejects apps that use a Test Store key, and Test Store purchases earn no revenue.", masked)
			case !storeKey:
				fd.Message = fmt.Sprintf("Test Store API key %s is the only RevenueCat key rc-doctor found for this app. A release build configured with a Test Store key crashes on launch, and App Review rejects it.", masked)
			default:
				fd.Message = fmt.Sprintf("Test Store API key %s isn't behind a debug-only check that rc-doctor recognizes. A release build configured with a Test Store key crashes on launch, and App Review rejects it.", masked)
			}
			out = append(out, fd)
		}
	}
	return out
}
