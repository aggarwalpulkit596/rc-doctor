# rc-doctor on 42 public RevenueCat integrations

October 2026. This is what rc-doctor found in public GitHub repositories that use RevenueCat's native or Kotlin Multiplatform SDKs, what turned out to be wrong in rc-doctor itself, and what was fixed before the final run. Repositories are not named. No issue or pull request was opened on any of them.

## Summary

- **3 of 21 apps** had an error or warning, and every one held up on review: a purchase flow with no way to restore purchases (2) and a Test Store API key passed to `Purchases.configure` in every build (1). One of those apps also ships a placeholder key in place of its real one.
- **Test Store keys are a live concern.** 3 of 21 apps commit a Test Store key, and one configures every build with it, which crashes release builds. Two apps (one of those three, and one that keeps its key out of the repository) wrote their own Gradle check that fails a release build configured with a Test Store key: the mistake `test-store-key` now catches for everyone.
- **Demos and templates** account for most findings: 6 of 12 sell something with no restore path, and 3 ship placeholder keys by design.
- **The survey found 9 problems in rc-doctor**, mostly in Kotlin Multiplatform projects. All are fixed and covered by tests; one false positive class remains when no project is loaded (below).

## Method

1. **Sample.** 51 repositories from GitHub's `revenuecat` and `in-app-purchase` topics and web searches for RevenueCat imports, cloned at their default branch on 13 October 2026. rc-doctor detected a RevenueCat integration in 42; the other 9 only mention RevenueCat in docs or don't use it.
2. **Run.** `rc-doctor check --dashboard offline` on each. No RevenueCat project was loaded, so the 6 rules that compare code with a project (`app-id`, `unknown-entitlement`, `unknown-offering`, `current-offering`, `empty-offering`, `entitlement-products`) didn't run.
3. **Triage.** Every finding was checked by hand against the code. Each false positive or miss became an rc-doctor fix with a test, and the run was repeated until the remaining findings held up.
4. **Classify.** Each repository was sorted by what it is, from its README: an app (21), a demo, tutorial or template (12), a library or wrapper (6), or a RevenueCat sample (3).

| | apps | demos and templates | RevenueCat samples | libraries |
|---|---|---|---|---|
| Repositories | 21 | 12 | 3 | 6 |
| Native iOS | 9 | 12 | 1 | 3 |
| Native Android | 3 | 0 | 1 | 0 |
| Native iOS and Android in one repository | 1 | 0 | 0 | 1 |
| Kotlin Multiplatform | 8 | 0 | 1 | 2 |
| `Purchases.configure` found | 20 | 12 | 3 | 6 |
| A real public SDK key committed | 12 | 2 | 0 | 0 |
| A placeholder key committed | 1 | 3 | 2 | 0 |

The repository with no configure call found adds RevenueCat code through a custom source directory, which rc-doctor doesn't read (see limitations).

## Findings in the final run

| Rule | Severity | Apps | Demos and templates | RevenueCat samples | Verdict |
|---|---|---|---|---|---|
| `restore-purchases` | warning | 2 | 6 | 0 | 7 correct; 1 a tip jar, which sells consumables only |
| `api-key` (placeholder) | error | 1 | 3 | 1 | correct; placeholders by design in the 4 templates and samples |
| `api-key` (placeholder in a defaults file) | info | 0 | 0 | 1 | correct; the Secrets Gradle Plugin's defaults file |
| `test-store-key` | warning | 1 | 0 | 0 | correct: configured in `Application.onCreate` for every build |
| `test-store-key` | info | 2 | 0 | 0 | correct: both pick the Test Store key at runtime for debug builds |

No app, demo or sample had a secret key, a key for the wrong platform, a hard-coded app user ID, purchases completed by the app without acknowledging them, a missing `Application` registration, or a `minSdk` below 24 without core library desugaring.

The tip jar is the false positive that remains: consumables can't be restored, and rc-doctor can't see product types without the project. Offline, the message now says apps that only sell consumables don't need a restore button. With the project loaded, `restore-purchases` reads product types and skips apps that only sell consumables.

## What the survey changed in rc-doctor

Each item was a false positive (FP) or a missed finding (miss) on at least one repository.

1. **FP: dead branches in Kotlin Multiplatform platform code.** `if (platform is Platform.Android) "goog_..." else "appl_..."`, copied into `iosMain`, read as an Android key in the iOS app. rc-doctor now reads the condition around a key (and the condition an `else` branch excludes) and drops keys the source set can never use.
2. **Miss: KMP source sets.** Flavor source sets (`androidFull`), intermediate sets (`mobileMain`) and the Android app module's `src/main` weren't scanned, so configure calls there were missed. Source sets are now classified by name.
3. **FP: JVM modules.** A Ktor server next to the app was scanned as app code, where a secret key is where it belongs. Gradle modules that only build for the JVM are now skipped.
4. **FP: app user IDs outside RevenueCat calls.** A SwiftUI preview's `init(appUserId: "...")` read as a hard-coded app user ID. The patterns now require a RevenueCat call (`Purchases.configure`, a configuration builder, `.with(appUserID:)`, `Purchases...logIn`).
5. **FP: placeholders in defaults files.** `secrets.defaults.properties` holds a placeholder on purpose; a file kept out of the repository overrides it. That's info now, not an error.
6. **FP: Test Store keys picked at runtime.** A Test Store constant chosen with `takeIf { isDebug }` was a warning. A key only defined, and not followed to the configure call, is info now; a key that reaches `Purchases.configure` (directly, or through a constant or `buildConfigField`) stays a warning.
7. **Miss: SDK 3.x on iOS.** Apps on the pre-4.0 `Purchases` CocoaPod weren't detected.
8. **Miss: KMP configuration forms.** Keys passed as `PurchasesConfiguration.Builder(KEY)` or `Purchases.configure(apiKey = KEY) { }` weren't followed to their constant.
9. **Noise: key-shaped IDs.** A quiz question ID shaped like `xxx_LongIdentifier` showed up as a key in `rc-doctor facts`. Key literals now need a prefix RevenueCat's SDKs or CLI know.

## RevenueCat's own sample apps

Run separately on the examples in `purchases-android` and `purchases-ios`:

- The MagicWeather apps, SampleCat and VanillaAdTrackingSample hold placeholder keys, by design.
- `purchase-tester` offers `PurchasesAreCompletedBy.MY_APP` as a runtime option without acknowledging purchases. That's expected in a test tool, and correct for the rule.
- MagicWeatherCompose and CustomEntitlementComputationSample set `minSdk = 23` without core library desugaring. RevenueCat's troubleshooting guide says to enable desugaring or raise `minSdk` to 24 to avoid crashes on launch, so `play-billing-desugaring` warns. Either the samples or the guidance may want a second look.
- `rc apps keys` labels a key as Amazon Appstore when it starts with `amaz_`, while the Android SDK's key validator expects `amzn_`, so Amazon keys would show up as a generic public SDK key.

## Limitations

- **Offline only.** The rules that need a project weren't exercised. Running rc-doctor against real projects with `rc` is the next step, and it needs the app's owner.
- **Selection.** Public repositories lean toward demos, templates and indie apps. Large production apps rarely publish their source.
- **Not read:** custom source directories added with `srcDir(...)`, keys in files kept out of the repository (on purpose: rc-doctor checks what is checked in, or runs where the secrets are), and Flutter, React Native, Unity and other hybrid SDKs. Hybrid projects outnumber native ones on GitHub's `revenuecat` topic, so they are the clearest platform gap.

## Reproducing it

```sh
git clone --depth 1 <repository> app
rc-doctor check --dashboard offline --format json --fail-on never app > app.check.json
rc-doctor facts app > app.facts.json
```

The JSON output is documented in [output.md](../output.md). The list of repositories and the per-repository notes are kept out of this report on purpose.
