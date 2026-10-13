# android-application-class

| | |
|---|---|
| Severity | error when the class isn't registered; warning when another class is |
| Platforms | Android |
| Needs your RevenueCat project | No |
| Automatic fix | **suggested**: add `android:name` to `<application>` |

## What it checks

When `Purchases.configure` runs in an `Application` subclass, that `AndroidManifest.xml` registers that class as `android:name` on `<application>`.

## Why it matters

Android only creates the `Application` class the manifest names. If it isn't registered, `onCreate` never runs, the SDK is never configured, and every RevenueCat call fails. RevenueCat's Android troubleshooting guide names this as the cause when no `Purchases` log output appears at all.

## Example

```xml
<!-- Before -->
<application android:label="@string/app_name">
<!-- After -->
<application android:name="com.example.app.App" android:label="@string/app_name">
```

## Detection

rc-doctor looks for an `Application` (or `MultiDexApplication`) subclass in each file that calls `Purchases.configure`, and reads `<application android:name>` from the app module's `src/main` (or KMP `src/androidMain`) manifest. If configure runs elsewhere, or the repository has no main manifest, the rule stays silent. A different class registered in the manifest is a warning, since it may be a subclass registered on purpose.

## Sources

- RevenueCat, [revenuecat-troubleshoot Android notes](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat/skills/revenuecat-troubleshoot/platforms/android.md): "Turn on debug logging".
