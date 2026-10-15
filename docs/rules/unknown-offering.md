# unknown-offering

| | |
|---|---|
| Severity | error |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | Yes |
| Automatic fix | **suggested**, when exactly one existing offering is within two edits |

## What it checks

Every offering identifier the code looks up exists in the RevenueCat project: `offerings["main"]`, `offerings.all["main"]`, `offerings.get("main")`, `getOffering("main")`, and Swift's `offerings.offering(identifier: "main")`. Identifiers held in constants are followed to their value.

## Why it matters

An offering renamed in the dashboard but not in code returns nothing, and the paywall shows no products.

## How to fix

Correct the identifier, create the offering in the dashboard, or use the current offering (`offerings.current`), which you can change remotely without an app release.
