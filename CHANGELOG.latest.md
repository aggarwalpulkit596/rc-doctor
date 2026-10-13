## 0.1.0
## rc-doctor
### ✨ New Features
* `check`: report integration problems as text, JSON or SARIF, with `--fail-on` exit codes for CI
* `review`: preview every fix as a diff; `--fix` applies safe fixes, `--suggested` adds best guesses
* `migrate`: migration pipeline (detect, plan, preview, apply) ready for source plug-ins
* `facts`, `snapshot`, `list` and `version` commands
* Platforms: Android (Gradle), iOS (Xcode, SPM, CocoaPods) and Kotlin Multiplatform (purchases-kmp)
* Rules: `api-key`, `app-id`, `unknown-entitlement`, `unknown-offering`
* `api-key` follows the constant passed to `Purchases.configure` and flags placeholder keys, such as the ones RevenueCat's sample apps ship with
* Entitlement and offering IDs held in constants are followed to their definition (Kotlin and Swift constants, `.properties` files, Gradle `getProperty` defaults), and fixes are made there
* Keys in shared config files go to the app their name points at (`*_IOS_*`, `*_ANDROID_*`)
