# test-store-key

| | |
|---|---|
| Severity | warning when the key reaches `Purchases.configure`; info when it is only defined |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | No |
| Automatic fix | none |

## What it checks

Test Store API keys (`test_...`) in code and build configuration that release builds can reach. A key counts as debug-only when rc-doctor finds it:

- inside `#if DEBUG ... #else` (Swift, Objective-C);
- inside `if (BuildConfig.DEBUG) { ... }`, or before the `else` of `if (BuildConfig.DEBUG) "test_..." else "goog_..."` (Kotlin, Java);
- inside a Gradle `debug { ... }` build type (`buildConfigField`, `resValue`);
- in a debug source set (`src/debug/`, `src/freeDebug/`) or a debug settings file (`Debug.xcconfig`, `secrets-debug.properties`).

A key also counts as debug-only when the configure call that passes it (through a constant) is behind one of those guards.

A key written in the configure call, or in the constant or `buildConfigField` the call passes, is a warning. A key that is only defined (a constant or a settings entry rc-doctor can't follow to the call, for example one chosen at runtime) is info: rc-doctor can't tell which builds use it. The message says whether the Test Store key is the only RevenueCat key rc-doctor found for the app, and whether the app sets `forceAllowTestStoreInReleaseBuilds`.

## Why it matters

The Test Store is for development. Both SDKs stop a release build that is configured with a Test Store key:

- **Android:** the SDK logs "Test Store API key used in release build", shows an error dialog, and the app crashes when the dialog is dismissed.
- **iOS:** the SDK crashes the app, to keep the build from being uploaded to the App Store.

`forceAllowTestStoreInReleaseBuilds` turns the crash off, for internal builds that never reach a store. Apps submitted with a Test Store key are rejected during App Review either way, and Test Store purchases earn no revenue.

## Example

```swift
// Before: every build uses the Test Store
Purchases.configure(withAPIKey: "test_AbCd...")

// After
#if DEBUG
Purchases.configure(withAPIKey: "test_AbCd...")
#else
Purchases.configure(withAPIKey: "appl_ZyXw...")
#endif
```

## How to fix

Configure debug builds with the Test Store key and release builds with the store app's public SDK key (from the API keys page of the RevenueCat dashboard, or `rc apps keys <app-id>`).

rc-doctor reads code without building it, so it can miss a guard: a key chosen by a flavor, a scheme or a CI variable looks the same as a key every build uses. If only debug builds reach the line, say so on it:

```kotlin
const val RC_TEST_KEY = "test_AbCd..." // rc-doctor:ignore test-store-key -- debug builds only
```

To make this rule fail CI, raise it to an error in `.rc-doctor.json`: `{"rules": {"test-store-key": "error"}}`.

## Sources

- RevenueCat, [Test Store](https://rev.cat/sdk-test-store) (the link the SDKs print).
- purchases-android, [PurchasesFactory.kt](https://github.com/RevenueCat/purchases-android/blob/ff9d60cdba6473edf390a5aab58d12f4ac4e0a1e/purchases/src/main/kotlin/com/revenuecat/purchases/PurchasesFactory.kt#L665-L685): the release-build check, and [ConfigureStrings.kt](https://github.com/RevenueCat/purchases-android/blob/ff9d60cdba6473edf390a5aab58d12f4ac4e0a1e/purchases/src/main/kotlin/com/revenuecat/purchases/strings/ConfigureStrings.kt#L40-L44): "Apps submitted with a Test Store API key will be rejected during App Review."
- purchases-ios, [DangerousSettings.swift](https://github.com/RevenueCat/purchases-ios/blob/af36c27108850b1db77bf8fe6506613bdfc40e16/Sources/Misc/DangerousSettings.swift#L99-L107): `forceAllowTestStoreInReleaseBuilds`.
