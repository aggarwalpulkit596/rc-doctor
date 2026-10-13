package scan

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/testutil"
)

func TestAppUserIDPatterns(t *testing.T) {
	for _, tc := range []struct {
		line string
		want []string
	}{
		{`PurchasesConfiguration.Builder(this, key).appUserID("demo_user").build()`, []string{"demo_user"}},
		{`Purchases.configure(withAPIKey: key, appUserID: "qa")`, []string{"qa"}},
		{`Configuration.Builder(withAPIKey: key).with(appUserID: "qa")`, []string{"qa"}},
		{`PurchasesConfiguration.Builder(apiKey = key, appUserId = "kmp_user")`, []string{"kmp_user"}},
		{`Purchases.shared.logIn("tester")`, []string{"tester"}},
		{`Purchases.sharedInstance.logInWith("tester", onError = {}) {}`, []string{"tester"}},
		{`Purchases.sharedInstance.awaitLogIn(newAppUserID = "tester")`, []string{"tester"}},
		{`.appUserID("")`, []string{""}},
		// Not RevenueCat app user IDs:
		{`PFUser.logIn(withUsername: "bob", password: pw)`, nil},
		{`var appUserId = ""`, nil},
		{`val appUserID: String = "placeholder"`, nil},
		{`Purchases.shared.logIn(user.id)`, nil},
		{`TransactionDetailHistoryItemView(item: .init(data: .stub, appUserId: "app_user001"))`, nil},
		{`authService.logIn("demo@example.com")`, nil},
		{`let (customerInfo, _) = try await purchases.logIn("tester")`, []string{"tester"}},
	} {
		if got := literalCaptures(AppUserIDLiterals, tc.line); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s\n  got %q, want %q", tc.line, got, tc.want)
		}
	}
	if got := captures(AppUserIDIdents, `Purchases.sharedInstance.logIn(Constants.QA_USER)`); !reflect.DeepEqual(got, []string{"Constants.QA_USER"}) {
		t.Errorf("idents = %q", got)
	}
}

func TestSignalPatterns(t *testing.T) {
	for _, tc := range []struct {
		re    *regexp.Regexp
		line  string
		match bool
	}{
		{SignalRestore, `Purchases.sharedInstance.restorePurchasesWith(onError = {}) {}`, true},
		{SignalRestore, `try await Purchases.shared.restorePurchases()`, true},
		{SignalRestore, `Purchases.sharedInstance.awaitRestore()`, true},
		{SignalRestore, `Purchases.sharedInstance.syncPurchases()`, false},
		{SignalPaywallUI, `Paywall(options = PaywallOptions.Builder(dismissRequest).build())`, true},
		{SignalPaywallUI, `.presentPaywallIfNeeded(requiredEntitlementIdentifier: "pro")`, true},
		{SignalPaywallUI, `private val launcher = PaywallActivityLauncher(this, this)`, true},
		{SignalPaywallUI, `PaywallScreen(onClose = {})`, false},
		{SignalCustomerCenter, `.presentCustomerCenter(isPresented: $show)`, true},
		{SignalCustomerCenter, `CustomerCenter(modifier = Modifier.fillMaxSize())`, true},
		{SignalCurrentOffering, `val offering = offerings.current ?: return`, true},
		{SignalCurrentOffering, `offerings?.current?.availablePackages`, true},
		{SignalCurrentOffering, `Offering current = offerings.getCurrent();`, true},
		{SignalCurrentOffering, `offerings["default"]`, false},
		{SignalCompletedByApp, `.purchasesAreCompletedBy(PurchasesAreCompletedBy.MY_APP)`, true},
		{SignalCompletedByApp, `.purchasesAreCompletedBy(.myApp, storeKitVersion: .storeKit2)`, true},
		{SignalCompletedByApp, `.purchasesAreCompletedBy(PurchasesAreCompletedBy.REVENUECAT)`, false},
	} {
		if got := tc.re.MatchString(tc.line); got != tc.match {
			t.Errorf("%s on %q = %v, want %v", tc.re, tc.line, got, tc.match)
		}
	}
}

// An app user ID passed as a constant is followed to its value only when the
// name reads as a constant (SCREAMING_CASE): user.id is runtime data.
func TestAppUserIDConstants(t *testing.T) {
	root := testutil.WriteFiles(t, map[string]string{
		"app/Consts.kt": "object Consts {\n    const val QA_USER = \"qa_account\"\n    val id = \"not_an_id\"\n}\n",
		"app/Login.kt":  "Purchases.sharedInstance.logIn(Consts.QA_USER)\nPurchases.sharedInstance.logIn(Consts.id)\n",
	})
	set := Patterns{Extensions: []string{".kt"}, AppUserIDIdents: AppUserIDIdents}
	res, err := Source(root, Spec{Sets: []Patterns{set}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.AppUserIDs) != 1 || res.AppUserIDs[0].ID != "qa_account" || res.AppUserIDs[0].Def == nil || res.AppUserIDs[0].Def.Name != "QA_USER" {
		t.Errorf("app user IDs = %+v", res.AppUserIDs)
	}
}
