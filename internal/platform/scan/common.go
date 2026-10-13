package scan

import (
	"regexp"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

// Patterns shared across platforms. Platform modules compose these with their own.
//
// keyToken is a RevenueCat API key: a prefix RevenueCat's SDKs or CLI know,
// then a long alphanumeric token. The public SDK key prefixes are appl_ and
// mac_ (Apple), goog_, amzn_ and galx_ (Google Play, Amazon, Galaxy Store),
// test_ (Test Store) and strp_ (Stripe); sk_ marks a secret key. Collectors
// only find candidates; rules decide which prefixes matter. Placeholders that
// look nothing like a key are found through the configure call instead.
const keyToken = `((?:appl|mac|goog|amzn|galx|test|strp|sk)_[A-Za-z0-9]{16,})`

var (
	// KeyQuoted is a key in a string literal, including Objective-C @"..." and
	// the escaped quotes of a Gradle buildConfigField value ("\"goog_...\"").
	KeyQuoted = regexp.MustCompile(`["'](?:\\")?` + keyToken + `(?:\\")?["']`)

	// KeyAssigned is an unquoted key=value line: gradle.properties,
	// local.properties, .xcconfig files.
	KeyAssigned = regexp.MustCompile(`^\s*[A-Za-z0-9_.-]+\s*[=:]\s*` + keyToken + `\s*$`)

	// KeyXMLText is a key stored as XML element text: Android string resources,
	// Info.plist values.
	KeyXMLText = regexp.MustCompile(`>\s*` + keyToken + `\s*</`)

	// ConfigureCall is Purchases.configure( in Kotlin, Java, Swift, Dart and TypeScript.
	ConfigureCall = regexp.MustCompile(`\bPurchases\.configure\s*\(`)

	// EntitlementSubscript is entitlements["pro"], entitlements.active["pro"],
	// entitlements.all["pro"], and the Objective-C @"pro" spelling.
	EntitlementSubscript = regexp.MustCompile(`entitlements(?:\??\.(?:active|all))?\s*\[\s*@?"([^"]+)"\s*\]`)

	// EntitlementCall is entitlements.get("pro"), entitlements.active.containsKey("pro")
	// and entitlements.active.keys.contains("pro"), with or without ?. calls.
	EntitlementCall = regexp.MustCompile(`entitlements(?:\??\.(?:active|all))?(?:\??\.keys)?\s*\??\.(?:get|containsKey|contains)\(\s*"([^"]+)"`)

	// PaywallEntitlement is the requiredEntitlementIdentifier argument of
	// RevenueCatUI paywalls: a Swift label, a Kotlin named argument or the
	// Kotlin builder setter.
	PaywallEntitlement = regexp.MustCompile(`[rR]equiredEntitlementIdentifier\s*[:=(]\s*"([^"]+)"`)

	// OfferingSubscript is offerings["main"], offerings.all["main"], and the
	// Objective-C @"main" spelling.
	OfferingSubscript = regexp.MustCompile(`offerings(?:\??\.all)?\s*\[\s*@?"([^"]+)"\s*\]`)

	// OfferingCall is offerings.get("main"), offerings.all.get("main") and getOffering("main").
	OfferingCall = regexp.MustCompile(`(?:offerings(?:\??\.all)?\??\.get|getOffering)\(\s*"([^"]+)"`)
)

// Identifier forms of the patterns above: entitlements[Config.ENTITLEMENT_ID]
// and friends. Each captures a (possibly qualified) identifier, which
// ConstTable.Lookup resolves to a value.
const ident = `([A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)`

var (
	EntitlementIdentSubscript = regexp.MustCompile(`entitlements(?:\??\.(?:active|all))?\s*\[\s*` + ident + `\s*\]`)
	EntitlementIdentCall      = regexp.MustCompile(`entitlements(?:\??\.(?:active|all))?(?:\??\.keys)?\s*\??\.(?:get|containsKey|contains)\(\s*` + ident + `\s*\)`)
	PaywallEntitlementIdent   = regexp.MustCompile(`[rR]equiredEntitlementIdentifier\s*[:=(]\s*` + ident + `\s*[,)]`)
	OfferingIdentSubscript    = regexp.MustCompile(`offerings(?:\??\.all)?\s*\[\s*` + ident + `\s*\]`)
	OfferingIdentCall         = regexp.MustCompile(`(?:offerings(?:\??\.all)?\??\.get|getOffering)\(\s*` + ident + `\s*\)`)

	// EntitlementIdents and OfferingIdents bundle the identifier forms.
	EntitlementIdents = []*regexp.Regexp{EntitlementIdentSubscript, EntitlementIdentCall, PaywallEntitlementIdent}
	OfferingIdents    = []*regexp.Regexp{OfferingIdentSubscript, OfferingIdentCall}
)

// KeyIdents capture the identifier passed as the API key when configuring the
// SDK: PurchasesConfiguration.Builder(this, Constants.GOOGLE_API_KEY),
// AmazonConfiguration.Builder(context, key), the KMP Builder(apiKey = KEY) and
// Swift's withAPIKey: Constants.apiKey. Resolved through the Constants table,
// they reveal keys defined away from the call, including placeholders.
var KeyIdents = []*regexp.Regexp{
	regexp.MustCompile(`Configuration\.Builder\(\s*[^,()]+,\s*` + ident + `\s*\)`),
	regexp.MustCompile(`Configuration\.Builder\(\s*(?:apiKey\s*=\s*)?` + ident + `\s*\)`), // KMP: Builder(apiKey)
	regexp.MustCompile(`Purchases\.configure\(\s*apiKey\s*=\s*` + ident + `\s*[,)]`),      // KMP: configure(apiKey = key) { ... }
	regexp.MustCompile(`withAPIKey:\s*` + ident + `\s*[,)]`),
}

// App user IDs passed to the SDK: the Android builder's appUserID("..."),
// an appUserID argument to Purchases.configure or a configuration builder
// (Swift's appUserID: "...", Kotlin's appUserId = "..."), Swift's
// .with(appUserID: "..."), and logIn("...") on Purchases with RevenueCat's
// spellings (logIn, logInWith, awaitLogIn). The call has to be RevenueCat's:
// a view model's init(appUserId: "...") or an auth service's logIn("...")
// are not. Only appUserID-style labels are accepted inside logIn(...), so
// logIn(withUsername: ...) doesn't match.
const (
	appUserIDValue = `(?:"([^"]*)")`
	configureCall  = `(?:\bPurchases\.configure|Configuration\.Builder)\(.*[(,]\s*appUser(?:ID|Id)\s*[:=]\s*`
	withAppUserID  = `\.with\(\s*appUserID\s*:\s*`
	logInCall      = `\b[Pp]urchases(?:\.shared(?:Instance)?)?\.(?:logIn|logInWith|awaitLogIn)\(\s*(?:(?:new)?[aA]ppUser(?:ID|Id)\s*[:=]\s*)?`
)

var (
	AppUserIDLiterals = []*regexp.Regexp{
		regexp.MustCompile(`\.appUser(?:ID|Id)\(\s*` + appUserIDValue + `\s*\)`),
		regexp.MustCompile(configureCall + appUserIDValue),
		regexp.MustCompile(withAppUserID + appUserIDValue),
		regexp.MustCompile(logInCall + appUserIDValue),
	}
	AppUserIDIdents = []*regexp.Regexp{
		regexp.MustCompile(`\.appUser(?:ID|Id)\(\s*` + ident + `\s*\)`),
		regexp.MustCompile(configureCall + ident + `\s*[,)]`),
		regexp.MustCompile(withAppUserID + ident + `\s*[,)]`),
		regexp.MustCompile(logInCall + ident + `\s*[,)]`),
	}
)

// Signals shared by every language. Platform modules add their own (for
// example Swift's transaction.finish(), which on Android would match
// Activity.finish()).
var (
	SignalRestore         = regexp.MustCompile(`\brestorePurchases\w*\b|\bawaitRestore\b`)
	SignalPaywallUI       = regexp.MustCompile(`\b(?:Paywall|PaywallDialog|PaywallView|PaywallFooter|PaywallViewController|OriginalTemplatePaywallFooter)\s*\(|\bPaywallActivityLauncher\b|\.presentPaywall(?:IfNeeded)?\s*\(|\.paywallFooter\s*\(`)
	SignalCustomerCenter  = regexp.MustCompile(`\bCustomerCenter(?:View|ViewController)?\s*\(|\.presentCustomerCenter\s*\(|\bShowCustomerCenter\b|\bCustomerCenterActivity\b`)
	SignalCurrentOffering = regexp.MustCompile(`\bofferings[?!]?\.(?:current\b|getCurrent\(\))`)
	SignalCompletedByApp  = regexp.MustCompile(`PurchasesAreCompletedBy\.(?:MY_APP|MyApp)\b|purchasesAreCompletedBy\(\s*\.myApp\b`)
	// SignalTestStoreRelease is the SDK's opt-out from stopping release builds
	// that use a Test Store key (DangerousSettings on both platforms).
	SignalTestStoreRelease = regexp.MustCompile(`\bforceAllowTestStoreInReleaseBuilds\b`)
)

// CommonSignals are the signals every platform looks for.
func CommonSignals() map[string][]*regexp.Regexp {
	return map[string][]*regexp.Regexp{
		model.SignalRestore:          {SignalRestore},
		model.SignalPaywallUI:        {SignalPaywallUI},
		model.SignalCustomerCenter:   {SignalCustomerCenter},
		model.SignalCurrentOffering:  {SignalCurrentOffering},
		model.SignalCompletedByMyApp: {SignalCompletedByApp},
		model.SignalTestStoreRelease: {SignalTestStoreRelease},
	}
}

// WithSignals returns CommonSignals plus extra patterns for one signal.
func WithSignals(name string, extra ...*regexp.Regexp) map[string][]*regexp.Regexp {
	m := CommonSignals()
	m[name] = append(m[name], extra...)
	return m
}
