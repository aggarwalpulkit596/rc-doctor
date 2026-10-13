# rc-doctor

[![ci](https://github.com/aggarwalpulkit596/rc-doctor/actions/workflows/ci.yml/badge.svg)](https://github.com/aggarwalpulkit596/rc-doctor/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

rc-doctor reads your app's code and your RevenueCat project side by side. It finds the integration mistakes that compile, pass code review and then fail in production, and it fixes the ones it can fix safely.

```
$ rc-doctor check testdata/sample --dashboard testdata/dashboard.json
rc-doctor checked android (com.example.paywall), ios (com.example.paywall)
against RevenueCat project: snapshot:testdata/dashboard.json

android
  error   api-key  testdata/sample/android/app/src/main/java/com/example/paywall/App.kt:11
          appl_AbCd… is a public key for ios apps, but this is the android app. The SDK's first request fails with an invalid credentials error.
          fix: Use the android app's public SDK key (goog_ZyXw…).  [safe fix]
  error   unknown-entitlement  testdata/sample/android/app/src/main/java/com/example/paywall/Paywall.kt:12
          Entitlement "premum" is used in code but doesn't exist in your RevenueCat project.
          fix: Did you mean "premium"?  [suggested fix]

ios
  error   unknown-offering  testdata/sample/ios/Paywall/ContentView.swift:15
          Offering "summer_sale" is used in code but doesn't exist in your RevenueCat project.
          fix: Create "summer_sale" in the RevenueCat dashboard, or use one that exists: "default", "annual_promo".
  error   api-key  testdata/sample/ios/Paywall/PaywallApp.swift:11
          Secret API key sk_AbCd… is in app code. Secret keys are for servers only and must never ship in an app.
          fix: Use the ios app's public SDK key (appl_AbCd…). Then rotate the secret key in the RevenueCat dashboard: anyone with your code or app binary can read it.  [suggested fix]

4 errors, 0 warnings.
Automatic fixes available: 1 safe, 2 suggested. Preview them by running `rc-doctor review` with the same arguments.
```

rc-doctor is an independent project. It is not made or endorsed by RevenueCat.

## Why

The worst subscription bugs don't crash anything. A mistyped entitlement ID means a user pays and the app stays locked. An offering renamed in the dashboard but not in code means an empty paywall. A key copied from the iOS app into the Android app, or a secret key pasted where the public SDK key belongs, gets past every compiler and every reviewer. A Test Store key left in a release build crashes it on launch.

Most of these are a mismatch between two places, the code and the RevenueCat project, and the rest are documented setup steps that are easy to skip. rc-doctor checks both before you ship. Run against 42 public RevenueCat integrations, it found real problems in 3 of 21 apps; see the [survey](docs/survey/2026-10-public-apps.md).

## Install

```sh
go install github.com/aggarwalpulkit596/rc-doctor/cmd/rc-doctor@latest   # Go 1.24+
```

Or download an archive for macOS, Linux or Windows from [Releases](https://github.com/aggarwalpulkit596/rc-doctor/releases). `checksums.txt` is signed with [cosign](https://github.com/sigstore/cosign) by the release workflow, so you can check a download came from this repository:

```sh
cosign verify-blob checksums.txt --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github.com/aggarwalpulkit596/rc-doctor/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
sha256sum --ignore-missing -c checksums.txt
```

rc-doctor has no third-party dependencies, so building from a checkout is `go build ./cmd/rc-doctor`.

## Quick start

```sh
rc-doctor                            # check the current directory
rc-doctor review                     # preview every fix as a diff; changes nothing
rc-doctor review --fix               # apply the safe fixes, then check again
rc-doctor review --fix --suggested   # also apply best guesses, such as typo fixes
rc-doctor explain test-store-key     # what a rule checks, why, and how to fix it
```

## Where the RevenueCat project comes from

rc-doctor never asks for or stores credentials. It reads your project through RevenueCat's own CLI, `rc`, so login, profiles and project selection work exactly as they do for `rc`: `--project-id`, then `$RC_PROJECT_ID`, then a `.revenuecat.json` in your repository, then the profile's default. rc-doctor runs `rc` from the directory it checks.

```sh
npm install -g @revenuecat/cli   # or: brew install RevenueCat/tap/rc
rc auth login                    # or set RC_API_KEY
```

| `--dashboard` | Behavior |
|---|---|
| `auto` (default) | Use `rc` if it's installed; otherwise run offline and say so. |
| `rc` | Use `rc`, and fail if it can't load the project. Add `--project-id` to pick one. |
| `offline` | Code-only checks. Rules that need the project are listed as skipped. |
| `path/to/project.json` | A snapshot written by `rc-doctor snapshot`, for CI without `rc` credentials and for reproducible bug reports. |

## Rules

| Rule | Severity | Needs the project | Finds |
|---|---|---|---|
| [`api-key`](docs/rules/api-key.md) | error | No | A placeholder passed to `Purchases.configure`, followed through constants; a secret key (`sk_`) in app code; a public key for another platform; with the project, a key from another project. |
| [`test-store-key`](docs/rules/test-store-key.md) | warning | No | A Test Store key (`test_`) that release builds can reach. The SDKs crash release builds configured with one. |
| [`hardcoded-app-user-id`](docs/rules/hardcoded-app-user-id.md) | error | No | A fixed app user ID passed to `configure` or `logIn`, which makes every user one customer. |
| [`restore-purchases`](docs/rules/restore-purchases.md) | warning | No | An app with no way to restore purchases (no `restorePurchases`, RevenueCat paywall or Customer Center). App Review rejects it on iOS. |
| [`purchases-completed-by-app`](docs/rules/purchases-completed-by-app.md) | error on Android, warning on iOS | No | `PurchasesAreCompletedBy.MY_APP` without acknowledging or finishing purchases. Google Play refunds them after 3 days. |
| [`android-application-class`](docs/rules/android-application-class.md) | error | No | `Purchases.configure` in an `Application` subclass that `AndroidManifest.xml` doesn't register, so it never runs. |
| [`play-billing-desugaring`](docs/rules/play-billing-desugaring.md) | warning | No | `minSdk` below 24 without core library desugaring, which RevenueCat's troubleshooting guide says to fix. |
| [`unknown-entitlement`](docs/rules/unknown-entitlement.md) | error | Yes | Entitlement IDs used in code that the project doesn't define, with a typo suggestion when one is close. |
| [`unknown-offering`](docs/rules/unknown-offering.md) | error | Yes | Offering IDs used in code that the project doesn't define. |
| [`app-id`](docs/rules/app-id.md) | warning | Yes | A package name or bundle ID that matches no app in the project, so offerings come back empty. |
| [`current-offering`](docs/rules/current-offering.md) | error | Yes | Code that reads the current offering when no offering is marked current. |
| [`empty-offering`](docs/rules/empty-offering.md) | error | Yes | An offering the code uses with no packages, or a package with no product. |
| [`entitlement-products`](docs/rules/entitlement-products.md) | error | Yes | An entitlement the code checks that no product grants. |

Every rule cites the RevenueCat documentation or SDK source it's based on. `rc-doctor explain <rule>` prints its page.

### Fix safety

Every fix is a single-line edit with one of two levels.

- **Safe** means the data proves it. An iOS key from this project pasted into the Android app can only be replaced by this project's Android key. `review --fix` applies these.
- **Suggested** means a best guess: a close match for a mistyped ID, or a key swap the project data can't prove. Teams often keep staging and production in separate RevenueCat projects with the same bundle ID, so rc-doctor never silently swaps in a key it can't tie to your code. These apply only with `--suggested`.

An edit applies only if the expected text is still on its line, so a file edited since the scan is skipped rather than corrupted. Files are replaced atomically and keep their permissions.

### Keys in output

Reports, fix previews and `facts` show a key's prefix and first four characters, never the whole key, in every format. CI logs are often shared more widely than the code they build. A `snapshot` file contains the public SDK keys `rc` reports for each app, which ship inside every app build anyway.

## Configuration

A `.rc-doctor.json` at the root turns rules off, changes their severity, and ignores paths. Unknown fields and rule IDs are errors, so a typo can't silently turn a check off. The [schema](schema/config.schema.json) gives editors completion.

```json
{
  "$schema": "https://raw.githubusercontent.com/aggarwalpulkit596/rc-doctor/main/schema/config.schema.json",
  "rules": { "restore-purchases": "off", "test-store-key": "error" },
  "ignore": ["examples/", "**/Previews/**"]
}
```

To silence one finding, comment on its line or the line above, naming the rules and saying why:

```kotlin
const val QA_KEY = "test_AbCd..." // rc-doctor:ignore test-store-key -- QA builds only
```

Suppressed findings are counted in the summary, and SARIF marks them as suppressed rather than dropping them.

## Commands

| Command | What it does |
|---|---|
| `check` | Run every rule and report. The default command. Never changes files. |
| `review` | The report plus a diff of every available fix. `--fix` applies safe fixes; `--suggested` adds best guesses. |
| `explain` | Print a rule's documentation: what it checks, why it matters, how to fix it. |
| `migrate` | Plan a move to RevenueCat from another SDK and apply the automatable part with `--apply`. The pipeline is in place; sources are on the roadmap. |
| `facts` | Print, as JSON, everything rc-doctor read from the code: keys, configure calls, IDs, build settings. |
| `snapshot` | Save the RevenueCat project's state as JSON through `rc`. |
| `list` | Show the platforms, rules, migration sources, formats and commands built into this binary. |
| `version` | Print the version. |

`check` flags: `--platform android,ios` to skip detection, `--dashboard`, `--project-id`, `--config`, `--format text|json|sarif`, `--fail-on error|warning|info|never`. `review` takes the same flags except `--format`. `rc-doctor help <command>` lists them all. The JSON and SARIF formats, `facts` and snapshots are documented in [docs/output.md](docs/output.md).

Exit codes: `0` nothing at or above `--fail-on`, `1` findings at or above it, `2` a usage error or rc-doctor couldn't run.

## Platforms

| Platform | Detected by | Reads |
|---|---|---|
| `android` | `com.revenuecat.purchases` in `build.gradle(.kts)` or a version catalog | Kotlin and Java sources; keys in `buildConfigField`/`resValue`, `.properties` files, string resources and the manifest; `minSdk`, desugaring and the SDK version. Skips test source sets, Apple-only Kotlin Multiplatform source sets and JVM-only modules such as a server. |
| `ios` | RevenueCat in `Package.swift`, `Package.resolved`, `Podfile(.lock)` (including the pre-4.0 `Purchases` pod) or `project.pbxproj` | Swift and Objective-C sources, `.xcconfig` files and `Info.plist`. Skips test targets. Picks the app's bundle ID over test targets and extensions. |
| `kmp` | `purchases-kmp` in a Gradle build or version catalog. Replaces `android` and `ios` for that repo. | Checks the Android and iOS apps separately. `androidMain`, other `android*` sets and the Android app module go to Android; `iosMain`, `appleMain`, `nativeMain` and the Xcode wrapper go to iOS; `commonMain` and intermediate sets such as `mobileMain` go to both, and a problem on a shared line is reported once. A key goes to the app its name, the condition around it (`if (platform is Platform.Android)`) or its prefix points at. |

Entitlement, offering and key values held in constants are followed to where they're written: Kotlin and Swift string constants, `.properties` files, Gradle `getProperty("NAME", "default")` fallbacks and `buildConfigField` values. A fix for such a value is made at the definition, not at the usage.

## In CI

Commit a snapshot so CI needs no RevenueCat credentials, and upload SARIF to get annotations on pull requests:

```yaml
jobs:
  rc-doctor:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version: "1.24"
      - run: go install github.com/aggarwalpulkit596/rc-doctor/cmd/rc-doctor@latest
      - run: rc-doctor check --dashboard rc-project.json --format sarif --fail-on never > rc-doctor.sarif
      - uses: github/codeql-action/upload-sarif@24c54180a607b1449ed407dd24f251e4e9147c8d # v4.38.3
        with:
          sarif_file: rc-doctor.sarif
      - run: rc-doctor check --dashboard rc-project.json   # fails the job on errors
```

Refresh the snapshot with `rc-doctor snapshot -o rc-project.json` when the project changes. To check against the live project instead, add `npm install -g @revenuecat/cli`, pass a RevenueCat API key as `RC_API_KEY` from your CI secrets, and use `--dashboard rc`.

## For coding agents

rc-doctor ships as a Claude Code plugin with a skill that tells an agent when to run it, how to read the JSON, and which fixes need your go-ahead:

```sh
claude plugins marketplace add aggarwalpulkit596/rc-doctor
claude plugins install rc-doctor
```

The skill is [plugin/skills/rc-doctor/SKILL.md](plugin/skills/rc-doctor/SKILL.md). It pairs with RevenueCat's own [ai-toolkit](https://github.com/RevenueCat/ai-toolkit) skills, which cover runtime troubleshooting.

## How it works

```
your code ──> platform modules ──> Facts ─┐
                                          ├──> rules ──> Findings (+ Edits) ──> reporters: text, json, sarif
RevenueCat ──> dashboard providers ───────┘                     │
project        (rc CLI, snapshot, offline)                      └──> fix engine <── review --fix, migrate --apply
```

- **Platform modules** turn source code into platform-neutral `Facts`: keys, configure calls, entitlement and offering IDs, app user IDs, the app identifier, build settings, and signals such as "calls restorePurchases".
- **Dashboard providers** load the project: the `rc` CLI, a snapshot file, or nothing.
- **Rules** compare the two and emit `Findings`. Rules only see `Facts`, never a platform package, so every rule works for every platform, including ones added later.
- **The fix engine** is the only code that writes files. Rule fixes and migrations both emit `Edits` and go through it.
- **Reporters** render the result.

Every one of these is a plug-in: a type that registers itself from `init()` in a generic registry. A blank import in `cmd/rc-doctor/plugins.go` links it into the binary.

```
cmd/rc-doctor/          main, plugins.go (the plug-in list), end-to-end and golden tests
internal/
  model/                Facts, Dashboard, Finding, Edit: the shared vocabulary
  registry/             the generic plug-in registry
  platform/             the Platform contract
    scan/               file walking, patterns, constants, guards around keys
    android/ ios/ kmp/  platform modules
  dashboard/            providers: rc CLI, snapshot, static, offline
  rules/                one file per rule
  config/               .rc-doctor.json
  engine/               detect, collect, load, run rules, suppress
  fix/                  the fix engine
  migrate/              the migration-source contract
  report/               text, json and sarif reporters
  cli/ commands/        command dispatch and the commands
docs/rules/             one page per rule, embedded in the binary for `explain`
plugin/                 the agent plugin and skill
testdata/               fixtures (native Android + iOS, KMP, a legacy Android app), snapshots, golden output
```

### Why a standalone Go binary

An app's build system can be Gradle, Xcode, SPM, CocoaPods, Bazel or a mix. rc-doctor reads files instead of hooking into any one of them, so it works the same for all of them and in any CI. Go gives a single static binary with no runtime to install, and the standard library covers everything rc-doctor needs. RevenueCat's own CLI is Go too, and this repository follows its conventions: the Makefile targets, lefthook, golangci-lint, GoReleaser with cosign-signed checksums, and `type(scope): summary` commits.

## Extending

Each extension is one new package (or file) plus, for packages, one line in `cmd/rc-doctor/plugins.go`.

**A platform** (Flutter, React Native, Unity, ...): implement `platform.Platform` (`Name`, `Detect`, `Collect`, `KeyPrefixes`) in `internal/platform/<name>`. Most platforms are a `scan.Spec` of file extensions and patterns plus an app-identifier extractor. Add a fixture under `testdata/` and a `Collect` test.

**A rule:** implement `rules.Rule` (`ID`, `Description`, `NeedsDashboard`, `Check`) and register it from `init()`. Test it with a hand-built `rules.Context`; rules never touch the file system. Add `docs/rules/<id>.md` (a test enforces it) and the ID to the config schema. Attach `model.Edit`s marked `Safe` only when the data proves the fix.

**A migration source** (Superwall, Adapty, Qonversion, raw StoreKit 2, Play Billing): implement `migrate.Migrator` (`Name`, `Description`, `Detect`, `Plan`) in `internal/migrate/<source>`. A plan is ordered steps, each with edits and/or manual instructions. `rc-doctor migrate` handles detection, preview and applying; `cmd/rc-doctor/main_test.go` shows a complete stand-in source.

**An output format:** implement `report.Reporter` and register it; it becomes `--format <name>`.

**A command:** implement `cli.Command` in `internal/commands/<name>` and reuse the flags and helpers in `internal/commands/shared`.

## Limitations

- Scanning is pattern-based, not a parser. Constants are resolved by name, not by scope; IDs built from strings at runtime are missed; a map of your own named `entitlements` can be mistaken for RevenueCat's. Guards around keys (`#if DEBUG`, `BuildConfig.DEBUG`, platform conditions) are read from the code around them, so an unusual guard can be missed; suppress the finding with a comment.
- One app per platform per repository: the first `applicationId`, and the app's bundle ID.
- Values injected only at build time (a gitignored properties file absent from CI) can't be checked there. Run rc-doctor where the file exists, or commit a template with dummy values.
- Source directories added with `srcDir(...)` outside the standard layout aren't read.
- Flutter, React Native, Unity and Capacitor aren't supported yet.

## Roadmap

- Platforms: Flutter, React Native and Expo, Unity, Capacitor. Hybrid apps outnumber native ones among public RevenueCat projects.
- Migration sources: Superwall, Adapty, Qonversion, raw StoreKit 2, Play Billing.
- Rules: Play user choice billing setup (see [RFC 0001](docs/rfcs/0001-upi-bridge.md)), a StoreKit configuration file for local testing, `Purchases.configure` called once at startup.
- India: a bridge from UPI payment gateways to RevenueCat entitlements, proposed in [RFC 0001](docs/rfcs/0001-upi-bridge.md).
- Distribution: a GitHub Action, a Gradle task and an Xcode build phase.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md), and [AGENTS.md](AGENTS.md) for the architecture and guardrails. Report security issues as described in [SECURITY.md](SECURITY.md).

## Development

```sh
make check    # gofmt, go vet, go test -race, and golangci-lint when it's installed
make golden   # refresh testdata/golden after an intended output change
make snapshot # build every release archive locally with GoReleaser
```

The end-to-end tests in `cmd/rc-doctor` run the real commands against `testdata/`, including the `rc` integration through a stand-in script, and fail if any output contains a full key. Golden tests pin the exact text, JSON and SARIF output.
