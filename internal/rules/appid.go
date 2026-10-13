package rules

import (
	"fmt"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func init() { Register(appID{}) }

// appID checks that the app's package name or bundle ID matches an app in the
// RevenueCat project. A mismatch usually means the code is being checked
// against the wrong project, or the store app was never added to it.
type appID struct{}

func (appID) ID() string { return "app-id" }

func (appID) Description() string {
	return "The app's package name or bundle ID matches an app in the RevenueCat project"
}

func (appID) NeedsDashboard() bool { return true }

func (r appID) Check(ctx *Context) []model.Finding {
	d := ctx.Dashboard
	if len(d.Apps) == 0 {
		return nil // the provider couldn't list apps, so there is nothing to compare
	}
	var out []model.Finding
	for _, f := range ctx.Facts {
		apps := d.AppsFor(f.Platform)
		if len(apps) == 0 {
			out = append(out, model.Finding{
				Rule:     r.ID(),
				Severity: model.Warning,
				Platform: f.Platform,
				Location: f.AppIDAt,
				Message:  fmt.Sprintf("This repo's %s app uses RevenueCat, but the RevenueCat project has no %s app.", f.Platform, f.Platform),
				Fix:      fmt.Sprintf("Add the %s app to the project in the RevenueCat dashboard, or check you're pointing at the right project.", f.Platform),
			})
			continue
		}
		if f.AppID == "" {
			continue
		}
		var configured []string
		match := false
		for _, a := range apps {
			match = match || a.AppID == f.AppID
			if a.AppID != "" {
				configured = append(configured, a.AppID)
			}
		}
		if match || len(configured) == 0 {
			continue
		}
		out = append(out, model.Finding{
			Rule:     r.ID(),
			Severity: model.Warning,
			Platform: f.Platform,
			Location: f.AppIDAt,
			Message:  fmt.Sprintf("%q doesn't match any %s app in the RevenueCat project (configured: %s).", f.AppID, f.Platform, quoteList(configured, 3)),
			Fix:      "Check you're pointing at the right RevenueCat project, or correct the bundle ID / package name in the app's settings in the dashboard.",
		})
	}
	return out
}
