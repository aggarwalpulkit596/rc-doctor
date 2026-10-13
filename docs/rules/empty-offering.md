# empty-offering

| | |
|---|---|
| Severity | error |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | Yes, with its catalog |
| Automatic fix | none |

## What it checks

Offerings the code uses (by ID, or the current offering when the code reads it) have at least one package, and every package in them has a product attached.

## Why it matters

An offering with no packages returns an empty `availablePackages` list even though `getOfferings` succeeds, so the paywall has nothing to sell. A package with no product has nothing to buy.

## How to fix

Add packages to the offering and attach a product to each, in the dashboard or with `rc packages create <offering-id>` and `rc packages attach <package-id> <product-id>`.

## Sources

- RevenueCat, [revenuecat-troubleshoot skill](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat/skills/revenuecat-troubleshoot/SKILL.md), checklist step 4 and Phase B, Check 4.
