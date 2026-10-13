# app-id

| | |
|---|---|
| Severity | warning |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | Yes, with its apps |
| Automatic fix | none |

## What it checks

That the app's identifier (Android `applicationId`, iOS `PRODUCT_BUNDLE_IDENTIFIER`) matches an app of the same platform in the RevenueCat project, and that the project has an app for each platform the repository ships.

## Why it matters

RevenueCat matches the running app to an app in your project by bundle ID or package name, exactly, including capitalization. On a mismatch, offerings come back empty because the app isn't recognized.

## How to fix

Point rc-doctor (and the app) at the right project, or correct the bundle ID or package name in the dashboard. Build flavors that change the identifier (`com.example.app.dev`) each need their own app in the dashboard or their own project, which is why this is a warning rather than an error.

## Detection

rc-doctor reads the first `applicationId` in the Gradle build files, and for iOS the shortest non-test bundle ID in `project.pbxproj` (app extensions must use IDs that start with the app's). `applicationIdSuffix` and identifiers set through build variables aren't resolved.

## Sources

- RevenueCat, [revenuecat-troubleshoot skill](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat/skills/revenuecat-troubleshoot/SKILL.md), checklist step 3, and its Android notes on `applicationId` and flavors.
