# hardcoded-app-user-id

| | |
|---|---|
| Severity | error |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | No |
| Automatic fix | none |

## What it checks

Fixed app user IDs passed to the SDK: the `appUserID` argument of `Purchases.configure` or the configuration builder, and `logIn("...")` (also `logInWith`, `awaitLogIn`). It follows `SCREAMING_CASE` constants, such as `logIn(Constants.QA_USER)`, to their value. Test source sets are skipped.

## Why it matters

Every user who runs the build becomes the same RevenueCat customer and shares its purchases and entitlements. RevenueCat's troubleshooting guide describes the recovery: ship an update that removes the ID, wait for most users to update, delete the shared customer, and have users restore purchases. An empty app user ID is rejected outright with `INVALID_APP_USER_ID`.

## Example

```kotlin
// Before
PurchasesConfiguration.Builder(this, apiKey).appUserID("demo_user").build()

// After: your signed-in user's ID, or none for an anonymous ID per install
PurchasesConfiguration.Builder(this, apiKey).appUserID(session.userId).build()
```

## How to fix

Pass the signed-in user's ID from your auth system, or leave the app user ID out so RevenueCat creates an anonymous one. If a fixed ID is intentional, for example in an internal QA build, suppress the finding with `// rc-doctor:ignore hardcoded-app-user-id -- QA build only`.

## Sources

- RevenueCat, [General troubleshooting](https://www.revenuecat.com/docs/revenuecat-support/general-troubleshooting): "All users logged in with the same app user ID".
- RevenueCat, [revenuecat-troubleshoot skill](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat/skills/revenuecat-troubleshoot/SKILL.md): `INVALID_APP_USER_ID` is returned for an empty string.
