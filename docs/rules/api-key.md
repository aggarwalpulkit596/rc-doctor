# api-key

| | |
|---|---|
| Severity | error; warning for a key that isn't from the project being checked; info for a placeholder in a defaults file |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | No. With it, rc-doctor can name the exact replacement key. |
| Automatic fix | **safe** when the key being replaced is one of the project's keys; otherwise **suggested** |

## What it checks

Every RevenueCat key in code and build configuration (Kotlin, Java, Swift, Objective-C, Gradle `buildConfigField` and `resValue`, `.properties`, string resources, `.xcconfig`, `Info.plist`), including keys reached through the constant passed to `Purchases.configure`:

- **A placeholder** such as `"REVENUECAT_API_KEY"` that reaches `Purchases.configure`. In a defaults or example file that a file kept out of the repository overrides (`secrets.defaults.properties`, the Secrets Gradle Plugin's convention, or `Config.example.xcconfig`), a placeholder is expected, so it's reported as info: builds made without the real file still get the placeholder.
- **A secret key** (`sk_...`) in app code.
- **A public key for another platform**: iOS apps use `appl_` keys (`mac_` for legacy Mac App Store apps), Android apps `goog_` (Google Play), `amzn_` (Amazon Appstore) or `galx_` (Galaxy Store).

Test Store keys (`test_`) work on every platform; [test-store-key](test-store-key.md) checks them.
- **A key from another project**: with the project loaded, a correctly prefixed key that isn't one of the project's public keys.

## Why it matters

- A placeholder means nothing works: the SDK can't reach your project.
- A secret key is for servers. Anyone with your code or app binary can read it, and RevenueCat's CLI authenticates with exactly this kind of key.
- A key for the wrong platform fails on the SDK's first request with an invalid credentials error (`INVALID_CREDENTIALS` on iOS, `InvalidCredentialsError` on Android).
- A key from another project sends purchases there.

## Example

```kotlin
// Before: the iOS app's key, copied into the Android app
Purchases.configure(PurchasesConfiguration.Builder(this, "appl_AbCd...").build())

// After
Purchases.configure(PurchasesConfiguration.Builder(this, "goog_ZyXw...").build())
```

## How to fix

Use the public SDK key of the app on the same platform, from the API keys page of the RevenueCat dashboard or `rc apps keys <app-id>`. After removing a secret key from an app, rotate it in the dashboard.

A key swap is applied by `rc-doctor review --fix` only when the key being replaced belongs to the project being checked, which proves the code targets that project. Otherwise it's a suggested fix (`--fix --suggested`): teams often keep staging and production in separate projects that share a bundle ID.

## Sources

- RevenueCat SDKs, the prefixes each one accepts: [APIKeyValidator.kt](https://github.com/RevenueCat/purchases-android/blob/ff9d60cdba6473edf390a5aab58d12f4ac4e0a1e/purchases/src/main/kotlin/com/revenuecat/purchases/APIKeyValidator.kt#L10-L13) in purchases-android and [Configuration.swift](https://github.com/RevenueCat/purchases-ios/blob/af36c27108850b1db77bf8fe6506613bdfc40e16/Sources/Purchasing/Configuration.swift#L519-L520) in purchases-ios.
- RevenueCat, [revenuecat-troubleshoot skill](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat/skills/revenuecat-troubleshoot/SKILL.md), checklist step 2: key prefixes and the invalid credentials error.
- RevenueCat, [revenuecat-cli skill](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat/skills/revenuecat-cli/SKILL.md): secret keys are for server-side tools only.
