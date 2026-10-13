## Unreleased
## rc-doctor
### ✨ New Features
* Rules: `test-store-key` (release builds crash with a Test Store key), `hardcoded-app-user-id`, `restore-purchases`, `purchases-completed-by-app`, `android-application-class` (with a suggested manifest fix) and `play-billing-desugaring`
* Rules that read the RevenueCat project's catalog: `current-offering`, `empty-offering` and `entitlement-products`. `restore-purchases` skips apps whose products are all consumables
* `.rc-doctor.json` to turn rules off, change their severity and ignore paths, with a JSON schema for editors
* `rc-doctor:ignore [rules] -- reason` comments on a finding's line or the line above; suppressed findings are counted, and marked as suppressed in SARIF
* `explain <rule>` prints the rule's documentation page, which is embedded in the binary; every finding links to it
* Key checks understand debug-only guards (`#if DEBUG`, `BuildConfig.DEBUG`, Gradle `debug {}`, debug source sets and settings files) and the platform a condition around a key names
* Kotlin Multiplatform: flavor and intermediate source sets (`androidFull`, `mobileMain`, `nativeMain`) and the Android app module's `src/main` are read; JVM-only modules such as a server are skipped
* Galaxy Store (`galx_`) and legacy Mac App Store (`mac_`) keys; iOS apps on the pre-4.0 `Purchases` CocoaPod
* Values passed through `buildConfigField`, the KMP `PurchasesConfiguration.Builder(key)` and `Purchases.configure(apiKey = key)` are followed to their definition
* Agent plugin and skill: `claude plugins marketplace add aggarwalpulkit596/rc-doctor`
### 🐞 Bugfixes
* `rc` listings longer than one page are followed through `rc api GET` instead of reporting IDs on later pages as missing
* `rc` runs in the checked directory, so the repository's `.revenuecat.json` selects the project
* rc's JSON errors are shown as its message and hint, not raw JSON
* A placeholder key in a defaults file such as `secrets.defaults.properties` is info, not an error
* Hard-coded app user IDs are only reported in RevenueCat calls, not in a view model's `init(appUserId:)`
* Key literals need a prefix RevenueCat's SDKs or CLI know, so IDs such as `sci_LongIdentifier` aren't read as keys
### 🔄 Other Changes
* Every RevenueCat fact rc-doctor relies on is checked against RevenueCat's SDK or CLI source, and no `TODO(verify)` remains
* Release builds with GoReleaser for macOS, Linux and Windows, with cosign-signed checksums; CI adds golangci-lint, govulncheck, macOS tests and a Windows build, with every action pinned to a commit SHA
* Golden tests for the text, JSON and SARIF output; `make check`, lefthook hooks and `mise.toml`
* Documentation: `docs/output.md`, a survey of 42 public RevenueCat integrations, RFC 0001 on UPI payments in India, SECURITY.md and a code of conduct
