# purchases-completed-by-app

| | |
|---|---|
| Severity | error on Android, warning on iOS |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | No |
| Automatic fix | none |

## What it checks

Apps configured with `PurchasesAreCompletedBy.MY_APP` (Android) or `purchasesAreCompletedBy(.myApp, ...)` (iOS) acknowledge or finish purchases themselves: `acknowledgePurchase`, `consumeAsync` or `consumePurchase` on Android, and `finish()` on a transaction (or `finishTransaction`) on iOS.

## Why it matters

In that mode the SDK no longer completes purchases.

- **Android:** Google Play refunds purchases that aren't acknowledged within 3 days.
- **iOS:** StoreKit keeps unfinished transactions and delivers them again.

## How to fix

Acknowledge or finish each purchase once access is granted, or let RevenueCat complete purchases (the default). Android's `Activity.finish()` is deliberately not counted as finishing a purchase.

## Sources

- RevenueCat, [revenuecat-troubleshoot Android notes](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat/skills/revenuecat-troubleshoot/platforms/android.md): "Entitlement not active after purchase".
