# entitlement-products

| | |
|---|---|
| Severity | error |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | Yes, with its catalog |
| Automatic fix | none |

## What it checks

Entitlements the code checks have at least one product attached in the project.

## Why it matters

A purchase unlocks the entitlements its product is attached to. An entitlement with no products can never be unlocked: the purchase succeeds and the feature stays locked. "User purchased but has no entitlement" is the first common issue in RevenueCat's troubleshooting playbook.

## How to fix

Attach the products that should grant it, in the dashboard or with `rc entitlements attach <entitlement-id> <product-id>`.

## Sources

- RevenueCat, [revenuecat-troubleshoot skill](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat/skills/revenuecat-troubleshoot/SKILL.md), Phase B, Check 3 and "Common issues".
