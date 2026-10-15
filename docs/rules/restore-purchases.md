# restore-purchases

| | |
|---|---|
| Severity | warning |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | No. With it, apps that only sell consumables aren't reported. |
| Automatic fix | none |

## What it checks

That an app which configures the SDK offers a way to restore purchases: a `restorePurchases` call (or `restorePurchasesWith`, `awaitRestore`), a RevenueCatUI paywall, or the Customer Center. RevenueCat paywalls and the Customer Center include a restore button, so either counts.

Consumables (tips, coins) can't be restored. With your RevenueCat project loaded, rc-doctor skips apps whose products are all consumables; offline, the message says so.

## Why it matters

- Users who reinstall or switch devices need it to get their entitlements back. RevenueCat's troubleshooting guide recommends a restore button in the paywall or settings.
- On iOS, App Review rejects apps with restorable in-app purchases and no "Restore Purchases" option, under guideline 3.1.1.

## How to fix

Add a "Restore purchases" button that calls `Purchases.shared.restorePurchases()` (iOS) or `Purchases.sharedInstance.restorePurchasesWith(...)` (Android), on the paywall or in settings, or show a RevenueCat paywall or the Customer Center.

## Detection

rc-doctor looks for the calls anywhere in the app's code. A restore flow implemented entirely on your own backend isn't detected; suppress the finding with `rc-doctor:ignore restore-purchases` on the configure line, or turn the rule off in `.rc-doctor.json`.

## Sources

- RevenueCat, [General troubleshooting](https://www.revenuecat.com/docs/revenuecat-support/general-troubleshooting): "Customer can't access entitlements".
- RevenueCat Community, [rejected from App Store review about restore purchases](https://community.revenuecat.com/sdks-51/we-get-rejected-from-apple-store-review-about-restore-purchases-693): guideline 3.1.1.
