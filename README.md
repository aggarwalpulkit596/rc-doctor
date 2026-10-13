# rc-doctor

rc-doctor reads your app's code and your RevenueCat project side by side. It finds the integration mistakes that compile, pass code review and then fail in production, and it fixes the ones it can fix safely.

```
$ rc-doctor check testdata/sample --dashboard testdata/dashboard.json
rc-doctor checked android (com.example.paywall), ios (com.example.paywall)
against RevenueCat project: snapshot:testdata/dashboard.json

android
  error   api-key  testdata/sample/android/app/src/main/java/com/example/paywall/App.kt:11
          appl_AbCd… is a public key for ios apps, but this is the android app. Each RevenueCat app has its own public SDK key.
          fix: Use the android app's public SDK key (goog_ZyXw…).  [safe fix]
  error   unknown-entitlement  testdata/sample/android/app/src/main/java/com/example/paywall/Paywall.kt:12
          Entitlement "premum" is used in code but doesn't exist in your RevenueCat project.
          fix: Did you mean "premium"?  [suggested fix]

ios
  error   unknown-offering  testdata/sample/ios/Paywall/ContentView.swift:15
          Offering "summer_sale" is used in code but doesn't exist in your RevenueCat project.
          fix: Create "summer_sale" in the RevenueCat dashboard, or use one that exists: "default", "annual_promo".
  error   api-key  testdata/sample/ios/Paywall/PaywallApp.swift:8
          Secret API key sk_AbCd… is in app code. Secret keys are for servers only and must never ship in an app.
          fix: Use the ios app's public SDK key (appl_AbCd…). Then rotate the secret key in the RevenueCat dashboard: anyone with your code or app binary can read it.  [suggested fix]

4 errors, 0 warnings.
Automatic fixes available: 1 safe, 2 suggested. Preview them by running `rc-doctor review` with the same arguments.
```

rc-doctor is an independent project. It is not made or endorsed by RevenueCat.

## Why

The worst subscription bugs don't crash anything. A mistyped entitlement ID means a user pays and the app stays locked. An offering renamed in the dashboard but not in code means an empty paywall. A key copied from the iOS app into the Android app, or a secret key pasted where the public SDK key belongs, gets past every compiler and every reviewer.

Each of these is a mismatch between two places: the code and the RevenueCat project. rc-doctor checks one against the other before you ship.

## Quick start

```sh
# From a checkout (Go 1.24+; no third-party dependencies):
go build -o rc-doctor ./cmd/rc-doctor

# Or, once the repository is published:
go install github.com/aggarwalpulkit596/rc-doctor/cmd/rc-doctor@latest
```

```sh
rc-doctor                        # check the current directory
rc-doctor review                 # preview every fix as a diff; changes nothing
rc-doctor review --fix           # apply the safe fixes, then check again
rc-doctor review --fix --suggested   # also apply best guesses, such as typo fixes
```

## Where the RevenueCat project comes from

rc-doctor never asks for or stores credentials. It reads your project through RevenueCat's own `rc` CLI, so login, profiles and project selection work exactly as they do for `rc`.

| `--dashboard` | Behaviour |
|---|---|
| `auto` (default) | Use `rc` if it's installed; otherwise run offline and say so. |
| `rc` | Use `rc`, and fail if it can't load the project. Add `--project-id` to pick one. |
| `offline` | Code-only checks. Rules that need the project are listed as skipped. |
| `path/to/project.json` | A snapshot written by `rc-doctor snapshot`, for CI without `rc` credentials and for reproducible bug reports. |

## Commands

| Command | What it does |
|---|---|
| `check` | Run every rule and report. The default command. Never changes files. |
| `review` | The report plus a diff of every available fix. `--fix` applies safe fixes; `--suggested` adds best guesses. |
| `migrate` | Plan a move to RevenueCat from another SDK and apply the automatable part with `--apply`. The pipeline is in place; sources are on the roadmap. |
| `facts` | Print, as JSON, everything rc-doctor found in the code: keys, configure calls, entitlement and offering IDs. |
| `snapshot` | Save the RevenueCat project's state as JSON through `rc`. |
| `list` | Show the platforms, rules, migration sources, formats and commands built into this binary. |
| `version` | Print the version. |

`check` flags: `--platform android,ios` to skip detection, `--dashboard` (below), `--format text|json|sarif`, `--fail-on error|warning|info|never`. `review` takes the same flags except `--format`. `rc-doctor help <command>` lists them all.

Exit codes: `0` nothing at or above `--fail-on`, `1` findings at or above it, `2` usage error or rc-doctor itself failed.

## Rules

| Rule | Needs the project | Finds |
|---|---|---|
| `api-key` | No | A placeholder passed to `Purchases.configure` (`"REVENUECAT_API_KEY"`), followed through constants; secret keys (`sk_`) in app code; a public key for another platform; with the project loaded, a key that isn't from the project being checked. |
| `unknown-entitlement` | Yes | Entitlement IDs used in code that the project doesn't define, with a typo suggestion when one is close. |
| `unknown-offering` | Yes | Offering IDs used in code that the project doesn't define. |
| `app-id` | Yes | A package name or bundle ID that matches no app in the project, or a platform the project has no app for. |

### Fix safety

Every fix is a single-line edit with one of two levels.

- **Safe** means the data proves it. An iOS key from this project pasted into the Android app can only be replaced by this project's Android key. `review --fix` applies these.
- **Suggested** means a best guess: a close match for a mistyped ID, or a key swap the project data can't prove. Teams often keep staging and production in separate RevenueCat projects with the same bundle ID, so rc-doctor never silently swaps in a key it can't tie to your code. These apply only with `--suggested`.

An edit applies only if the expected text is still on its line, so a file edited since the scan is skipped rather than corrupted. Files are replaced atomically and keep their permissions.

### Keys in output

Reports, fix previews and `facts` show a key's prefix and first four characters, never the whole key, in every format. CI logs are often shared more widely than the code they build. A `snapshot` file contains the public SDK keys `rc` reports for each app, which ship inside every app build anyway.

## Platforms

| Platform | Detected by | Reads |
|---|---|---|
| `android` | `com.revenuecat.purchases` in `build.gradle(.kts)` or a version catalog | Kotlin and Java sources; keys in `buildConfigField`/`resValue`, `.properties` files, string resources and the manifest. Skips test source sets and Apple-only Kotlin Multiplatform source sets. |
| `ios` | RevenueCat in `Package.swift`, `Package.resolved`, `Podfile(.lock)` or `project.pbxproj` | Swift and Objective-C sources, `.xcconfig` files and `Info.plist`. Skips test targets. Picks the app's bundle ID over test targets and extensions. |
| `kmp` | `purchases-kmp` in a Gradle build or version catalog. Replaces `android` and `ios` for that repo. | Checks the Android and iOS apps separately: `androidMain` belongs to Android, `iosMain`/`appleMain` and the Xcode wrapper to iOS, and `commonMain` to both (a problem on a shared line is reported once). A key in a shared file such as `secrets.properties` goes to the app its name points at (`REVENUECAT_IOS_KEY`), then to the app its prefix belongs to. |

Entitlement and offering IDs held in constants are followed to where the value is written: Kotlin and Swift
string constants, `.properties` files and Gradle `getProperty("NAME", "default")` fallbacks, in that order of
precedence when a name is defined twice. A fix for such an ID is made at the definition, not at the usage.

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
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
      - run: go install github.com/aggarwalpulkit596/rc-doctor/cmd/rc-doctor@latest
      - run: rc-doctor check --dashboard rc-project.json --format sarif --fail-on never > rc-doctor.sarif
      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: rc-doctor.sarif
      - run: rc-doctor check --dashboard rc-project.json   # fails the job on errors
```

Refresh the snapshot with `rc-doctor snapshot -o rc-project.json` when the project changes.

## How it works

```
your code ──> platform modules ──> Facts ─┐
                                          ├──> rules ──> Findings (+ Edits) ──> reporters: text, json, sarif
RevenueCat ──> dashboard providers ───────┘                     │
project        (rc CLI, snapshot, offline)                      └──> fix engine <── review --fix, migrate --apply
```

- **Platform modules** turn source code into platform-neutral `Facts`: keys, configure calls, entitlement and offering IDs, the app identifier.
- **Dashboard providers** load the project: the `rc` CLI, a snapshot file, or nothing.
- **Rules** compare the two and emit `Findings`. Rules only see `Facts`, never a platform package, so every rule works for every platform, including ones added later.
- **The fix engine** is the only code that writes files. Rule fixes and migrations both emit `Edits` and go through it.
- **Reporters** render the result.

Every one of these is a plug-in: a type that registers itself from `init()` in a generic registry. A blank import in `cmd/rc-doctor/plugins.go` links it into the binary.

```
cmd/rc-doctor/          main, plugins.go (the plug-in list), end-to-end tests
internal/
  model/                Facts, Dashboard, Finding, Edit: the shared vocabulary
  registry/             the generic plug-in registry
  platform/             the Platform contract
    scan/               file walking and the patterns platforms share
    android/ ios/ kmp/  platform modules
  dashboard/            providers: rc CLI, snapshot, static, offline
  rules/                rules: api-key, app-id, unknown-entitlement, unknown-offering
  engine/               detect, collect, load, run rules
  fix/                  the fix engine
  migrate/              the migration-source contract
  report/               text, json and sarif reporters
  cli/                  command dispatch
  commands/             check, review, migrate, facts, snapshot, list, version
testdata/               fixtures: a native Android + iOS app, a KMP app, and project snapshots
```

### Why a standalone Go binary

An app's build system can be Gradle, Xcode, SPM, CocoaPods, Bazel or a mix. rc-doctor reads files instead of hooking into any one of them, so it works the same for all of them and in any CI. Go gives a single static binary with no runtime to install, and the standard library covers everything rc-doctor needs.

## Extending

Each extension is one new package (or file) plus, for packages, one line in `cmd/rc-doctor/plugins.go`.

**A platform** (Flutter, React Native, Kotlin Multiplatform, Unity, ...): implement `platform.Platform` (`Name`, `Detect`, `Collect`, `KeyPrefixes`) in `internal/platform/<name>`. Most platforms are a `scan.Spec` of file extensions and patterns plus an app-identifier extractor. Add a fixture under `testdata/` and a `Collect` test.

**A rule:** implement `rules.Rule` (`ID`, `Description`, `NeedsDashboard`, `Check`) and register it from `init()`. Test it with a hand-built `rules.Context`; rules never touch the file system. Attach `model.Edit`s marked `Safe` only when the data proves the fix.

**A migration source** (Superwall, Adapty, Qonversion, raw StoreKit 2, Play Billing): implement `migrate.Migrator` (`Name`, `Description`, `Detect`, `Plan`) in `internal/migrate/<source>`. A plan is ordered steps, each with edits and/or manual instructions. `rc-doctor migrate` handles detection, preview and applying; `cmd/rc-doctor/main_test.go` shows a complete stand-in source.

**An output format:** implement `report.Reporter` and register it; it becomes `--format <name>`.

**A command:** implement `cli.Command` in `internal/commands/<name>` and reuse the flags and helpers in `internal/commands/shared`.

## Limitations

- Scanning is pattern-based, not a parser. Constants are resolved by name, not by scope; IDs built from strings at runtime are missed; and a map of your own named `entitlements` can be mistaken for RevenueCat's.
- One app per platform per repository: the first `applicationId`, and the app's bundle ID.
- Values injected only at build time (a gitignored properties file absent from CI) can't be checked there. Run rc-doctor where the file exists, or commit a template with dummy values.
- Flutter, React Native and Unity aren't supported yet.
- The JSON field names rc-doctor reads from `rc`, and the public key prefixes for each platform, are marked `TODO(verify)` in the code and should be confirmed against `rc schema` and RevenueCat's documentation.

## Roadmap

- Platforms: Flutter, React Native and Expo, Unity, Capacitor.
- Migration sources: Superwall, Adapty, Qonversion, raw StoreKit 2, Play Billing.
- Rules: products in each offering resolve (via `rc offerings verify`), a StoreKit configuration file for local testing, `Purchases.configure` called once at startup.
- Distribution: a GitHub Action, a Gradle task, an Xcode build phase, and a skill for coding agents.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md), and [AGENTS.md](AGENTS.md) for the architecture and guardrails.

## Development

```sh
go test ./...
go vet ./...
gofmt -l .
```

The end-to-end tests in `cmd/rc-doctor` run the real commands against `testdata/`, including the `rc` integration through a stand-in script, and fail if any output contains a full key.
