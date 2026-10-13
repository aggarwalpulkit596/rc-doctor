package scan

import "regexp"

// Patterns shared across platforms. Platform modules compose these with their own.
//
// The key patterns match anything shaped like a RevenueCat key: a short
// lowercase prefix, an underscore, then a long alphanumeric token. Collectors
// only find candidates; rules decide which prefixes matter.
var (
	// KeyQuoted is a key in a string literal, including Objective-C @"..." and
	// the escaped quotes of a Gradle buildConfigField value ("\"goog_...\"").
	KeyQuoted = regexp.MustCompile(`["'](?:\\")?([a-z]{2,4}_[A-Za-z0-9]{16,})(?:\\")?["']`)

	// KeyAssigned is an unquoted key=value line: gradle.properties,
	// local.properties, .xcconfig files.
	KeyAssigned = regexp.MustCompile(`^\s*[A-Za-z0-9_.-]+\s*[=:]\s*([a-z]{2,4}_[A-Za-z0-9]{16,})\s*$`)

	// KeyXMLText is a key stored as XML element text: Android string resources,
	// Info.plist values.
	KeyXMLText = regexp.MustCompile(`>\s*([a-z]{2,4}_[A-Za-z0-9]{16,})\s*</`)

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
	regexp.MustCompile(`Configuration\.Builder\(\s*apiKey\s*=\s*` + ident + `\s*\)`),
	regexp.MustCompile(`withAPIKey:\s*` + ident + `\s*[,)]`),
}
