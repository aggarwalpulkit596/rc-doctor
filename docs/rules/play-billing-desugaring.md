# play-billing-desugaring

| | |
|---|---|
| Severity | warning |
| Platforms | Android, the Android app of a Kotlin Multiplatform project |
| Needs your RevenueCat project | No |
| Automatic fix | none (it's a build configuration choice) |

## What it checks

Android apps with `minSdk` below 24 and core library desugaring off, using RevenueCat's Android SDK 8.10.8 or later (or a version rc-doctor can't read).

## Why it matters

RevenueCat's Android SDK 8.10.8 moved to Play Billing Library 7.1.1 (see the SDK's CHANGELOG). That library uses Java 8 types some devices below Android 7.0 (API 24) lack, and the app can crash there with `NoCoreLibraryDesugaringException` or `NoClassDefFoundError` in the billing client.

## How to fix

Either enable core library desugaring:

```kotlin
android {
    compileOptions {
        isCoreLibraryDesugaringEnabled = true
    }
}
dependencies {
    coreLibraryDesugaring("com.android.tools:desugar_jdk_libs:<version>")
}
```

or raise `minSdk` to 24.

## Detection

rc-doctor reads `minSdk`/`minSdkVersion` from the app module's build file (following `libs.versions.x.get()` into the version catalog), the desugaring flag from any Gradle build file, and the SDK version from a direct dependency or the version catalog. A `minSdk` set only in a convention plugin isn't read, and the rule then stays silent.

## Sources

- RevenueCat, [General troubleshooting](https://www.revenuecat.com/docs/revenuecat-support/general-troubleshooting): "Android crashes (NoCoreLibraryDesugaringException, NoClassDefFoundError in the billing client)".
- RevenueCat, [purchases-android CHANGELOG](https://github.com/RevenueCat/purchases-android/blob/main/CHANGELOG.md): 8.10.8 updates the Play Billing Library to 7.1.1.
