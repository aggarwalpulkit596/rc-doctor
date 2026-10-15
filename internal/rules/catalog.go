package rules

import (
	"fmt"
	"slices"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func init() {
	Register(currentOffering{})
	Register(emptyOffering{})
	Register(entitlementProducts{})
}

// These rules follow the dashboard checks in RevenueCat's own troubleshooting
// playbook (the revenuecat-troubleshoot skill), applied only to what the code
// actually uses. They need the catalog, which the rc CLI provider and snapshots
// supply; without it they stay silent.

// currentOffering: the code reads offerings.current, but none is current.
type currentOffering struct{}

func (currentOffering) ID() string { return "current-offering" }

func (currentOffering) Description() string {
	return "Code that reads the current offering has one marked current in the project"
}

func (currentOffering) NeedsDashboard() bool { return true }

func (r currentOffering) Check(ctx *Context) []model.Finding {
	c := ctx.Dashboard.Catalog
	if c == nil || c.CurrentOffering != "" {
		return nil
	}
	var out []model.Finding
	for _, f := range ctx.Facts {
		at := f.Signals[model.SignalCurrentOffering]
		if len(at) == 0 {
			continue
		}
		out = append(out, model.Finding{
			Rule:     r.ID(),
			Severity: model.Error,
			Platform: f.Platform,
			Location: at[0],
			Message:  "The code reads the current offering, but no offering in your RevenueCat project is marked current, so it is null and the paywall has nothing to show.",
			Fix:      "Mark an offering as current in the dashboard, or run `rc offerings set-current <offering-id>`.",
		})
	}
	return out
}

// emptyOffering: an offering the code uses has no packages, or a package in
// it has no product.
type emptyOffering struct{}

func (emptyOffering) ID() string { return "empty-offering" }

func (emptyOffering) Description() string {
	return "Offerings the code uses have packages, and every package has a product"
}

func (emptyOffering) NeedsDashboard() bool { return true }

func (r emptyOffering) Check(ctx *Context) []model.Finding {
	d := ctx.Dashboard
	if d.Catalog == nil {
		return nil
	}
	var out []model.Finding
	for _, f := range ctx.Facts {
		used := append([]model.Ref{}, f.Offerings...)
		if at := f.Signals[model.SignalCurrentOffering]; len(at) > 0 && d.Catalog.CurrentOffering != "" {
			used = append(used, model.Ref{ID: d.Catalog.CurrentOffering, Location: at[0]})
		}
		for _, ref := range used {
			pkgs, known := d.Catalog.Packages[ref.ID]
			if !known || !slices.Contains(d.Offerings, ref.ID) {
				continue // unknown-offering reports IDs that don't exist
			}
			fd := model.Finding{Rule: r.ID(), Severity: model.Error, Platform: f.Platform, Location: ref.Location}
			if len(pkgs) == 0 {
				fd.Message = fmt.Sprintf("Offering %q has no packages, so its availablePackages list is empty and there is nothing to buy.", ref.ID)
				fd.Fix = "Add packages to the offering and attach a product to each, in the dashboard or with `rc packages create` and `rc packages attach`."
				out = append(out, fd)
				continue
			}
			for _, p := range pkgs {
				if len(p.Products) > 0 {
					continue
				}
				fd.Message = fmt.Sprintf("Package %q in offering %q has no product attached, so there is nothing to buy in it.", p.ID, ref.ID)
				fd.Fix = "Attach a product to the package, in the dashboard or with `rc packages attach <package-id> <product-id>`."
				out = append(out, fd)
			}
		}
	}
	return out
}

// entitlementProducts: an entitlement the code checks has no products, so no
// purchase can ever unlock it.
type entitlementProducts struct{}

func (entitlementProducts) ID() string { return "entitlement-products" }

func (entitlementProducts) Description() string {
	return "Entitlements the code checks have products attached"
}

func (entitlementProducts) NeedsDashboard() bool { return true }

func (r entitlementProducts) Check(ctx *Context) []model.Finding {
	d := ctx.Dashboard
	if d.Catalog == nil {
		return nil
	}
	var out []model.Finding
	for _, f := range ctx.Facts {
		for _, ref := range f.Entitlements {
			products, known := d.Catalog.EntitlementProducts[ref.ID]
			if !known || len(products) > 0 {
				continue
			}
			out = append(out, model.Finding{
				Rule:     r.ID(),
				Severity: model.Error,
				Platform: f.Platform,
				Location: ref.Location,
				Message:  fmt.Sprintf("Entitlement %q has no products attached, so no purchase can unlock it.", ref.ID),
				Fix:      "Attach the products that should grant it, in the dashboard or with `rc entitlements attach <entitlement-id> <product-id>`.",
			})
		}
	}
	return out
}
