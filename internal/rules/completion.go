package rules

import (
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func init() { Register(purchasesCompletedByApp{}) }

// purchasesCompletedByApp checks apps that complete purchases themselves.
// With PurchasesAreCompletedBy.MY_APP the SDK no longer acknowledges or
// finishes transactions, so the app must. RevenueCat's Android troubleshooting
// guide: an unacknowledged Play purchase is refunded after 3 days. On iOS,
// StoreKit keeps delivering unfinished transactions.
type purchasesCompletedByApp struct{}

func (purchasesCompletedByApp) ID() string { return "purchases-completed-by-app" }

func (purchasesCompletedByApp) Description() string {
	return "Apps that complete purchases themselves acknowledge (Android) or finish (iOS) them"
}

func (purchasesCompletedByApp) NeedsDashboard() bool { return false }

func (r purchasesCompletedByApp) Check(ctx *Context) []model.Finding {
	var out []model.Finding
	for _, f := range ctx.Facts {
		at := f.Signals[model.SignalCompletedByMyApp]
		if len(at) == 0 || f.HasSignal(model.SignalFinishesPurchase) {
			continue
		}
		fd := model.Finding{Rule: r.ID(), Platform: f.Platform, Location: at[0]}
		switch f.Platform {
		case "android":
			fd.Severity = model.Error
			fd.Message = "Purchases are completed by the app (PurchasesAreCompletedBy.MY_APP), but no acknowledgePurchase or consumeAsync call was found. Google Play refunds purchases that aren't acknowledged within 3 days."
			fd.Fix = "Acknowledge each purchase with BillingClient.acknowledgePurchase (or consume consumables) once access is granted, or let RevenueCat complete purchases, the default."
		case "ios":
			fd.Severity = model.Warning
			fd.Message = "Purchases are completed by the app (purchasesAreCompletedBy(.myApp)), but no transaction finish() call was found. StoreKit keeps unfinished transactions and delivers them again."
			fd.Fix = "Call finish() on each transaction once the purchase is delivered, or let RevenueCat complete purchases, the default."
		default:
			continue
		}
		out = append(out, fd)
	}
	return out
}
