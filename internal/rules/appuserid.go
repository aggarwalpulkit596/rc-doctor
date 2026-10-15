package rules

import (
	"fmt"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func init() { Register(hardcodedAppUserID{}) }

// hardcodedAppUserID flags fixed app user IDs passed to the SDK. RevenueCat's
// troubleshooting guide describes the result: every user who runs the build
// is logged in as the same customer and shares its purchases, and undoing it
// takes an app update plus restores. An empty ID is rejected outright
// (INVALID_APP_USER_ID).
type hardcodedAppUserID struct{}

func (hardcodedAppUserID) ID() string { return "hardcoded-app-user-id" }

func (hardcodedAppUserID) Description() string {
	return "No fixed app user IDs passed to configure or logIn"
}

func (hardcodedAppUserID) NeedsDashboard() bool { return false }

func (r hardcodedAppUserID) Check(ctx *Context) []model.Finding {
	var out []model.Finding
	for _, f := range ctx.Facts {
		for _, ref := range f.AppUserIDs {
			fd := model.Finding{
				Rule:     r.ID(),
				Severity: model.Error,
				Platform: f.Platform,
				Location: ref.Location,
				Fix:      "Pass your signed-in user's ID, or leave it out so RevenueCat creates an anonymous ID per install.",
			}
			source := ""
			if ref.Def != nil {
				source = fmt.Sprintf(" (%s, set at %s)", ref.Def.Name, ref.Def.Location)
			}
			if ref.ID == "" {
				fd.Message = "An empty app user ID" + source + " is passed to the SDK, which rejects it with INVALID_APP_USER_ID."
			} else {
				fd.Message = fmt.Sprintf("The fixed app user ID %q%s is passed to the SDK. Every user who runs this code becomes the same RevenueCat customer and shares its purchases.", ref.ID, source)
			}
			out = append(out, fd)
		}
	}
	return out
}
