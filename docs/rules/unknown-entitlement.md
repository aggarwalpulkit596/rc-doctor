# unknown-entitlement

| | |
|---|---|
| Severity | error |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | Yes |
| Automatic fix | **suggested**, when exactly one existing entitlement is within two edits |

## What it checks

Every entitlement identifier the code checks exists in the RevenueCat project: `entitlements["pro"]`, `entitlements.active["pro"]`, `entitlements.get("pro")`, `containsKey`, `keys.contains`, and RevenueCatUI's `requiredEntitlementIdentifier`. Identifiers held in constants are followed to their value.

## Why it matters

A mistyped entitlement ID compiles, passes review, and never unlocks anything, even after a successful purchase.

## Example

```swift
// Before
customerInfo.entitlements["premum"]?.isActive == true
// After
customerInfo.entitlements["premium"]?.isActive == true
```

## How to fix

Correct the identifier, or create the entitlement in the dashboard. When the identifier comes from a constant, rc-doctor's fix edits the constant's definition, not the line that uses it.

## Detection

Constants are resolved by name across Kotlin, Java and Swift string constants, `.properties` files and Gradle `getProperty("NAME", "default")` fallbacks. A name defined with different values is left alone rather than guessed.
