package rules

import (
	"fmt"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func init() { Register(restorePurchases{}) }

// restorePurchases warns when an app that configures the SDK never offers to
// restore purchases. Users who reinstall or change devices need it to get
// their entitlements back, and App Review rejects apps with restorable
// purchases and no restore option (guideline 3.1.1). RevenueCat paywalls and
// the Customer Center include a restore button, so either counts. When the
// project shows that every product is a consumable, there is nothing to
// restore and the rule stays silent.
type restorePurchases struct{}

func (restorePurchases) ID() string { return "restore-purchases" }

func (restorePurchases) Description() string {
	return "The app offers a way to restore purchases (restorePurchases, a RevenueCat paywall, or the Customer Center)"
}

func (restorePurchases) NeedsDashboard() bool { return false }

func (r restorePurchases) Check(ctx *Context) []model.Finding {
	var out []model.Finding
	for _, f := range ctx.Facts {
		if len(f.ConfigureAt) == 0 || f.HasSignal(model.SignalRestore, model.SignalPaywallUI, model.SignalCustomerCenter) {
			continue
		}
		if ctx.Dashboard != nil && ctx.Dashboard.Catalog.OnlyConsumables() {
			continue // consumables (tips, coins) can't be restored
		}
		why := "Users who reinstall or switch devices can't get their purchases back without it."
		if f.Platform == "ios" {
			why = "App Review rejects apps with restorable purchases and no restore option (guideline 3.1.1), and users who reinstall can't get their purchases back."
		}
		if ctx.Dashboard == nil || ctx.Dashboard.Catalog == nil || ctx.Dashboard.Catalog.ProductTypes == nil {
			why += " Apps that only sell consumables, such as tips, don't need one."
		}
		out = append(out, model.Finding{
			Rule:     r.ID(),
			Severity: model.Warning,
			Platform: f.Platform,
			Location: f.ConfigureAt[0],
			Message:  fmt.Sprintf("The %s app never offers to restore purchases: no restorePurchases call, RevenueCat paywall or Customer Center was found. %s", f.Platform, why),
			Fix:      "Add a \"Restore purchases\" button that calls restorePurchases, on the paywall or in settings, or show a RevenueCat paywall or the Customer Center, which include one.",
		})
	}
	return out
}
