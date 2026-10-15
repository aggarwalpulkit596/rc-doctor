# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Common Development Commands

```bash
make build      # go build -o rc-doctor ./cmd/rc-doctor
make test       # go test -race ./..., unit + end-to-end, as CI runs it
make check      # gofmt check, go vet, tests, and golangci-lint when installed (the pre-commit hook)
make golden     # rewrite testdata/golden after an intended output change; review the diff
make snapshot   # build every release archive locally with GoReleaser

# One package's tests
go test ./internal/rules/...

# Try it on the fixtures
go run ./cmd/rc-doctor check testdata/sample --dashboard testdata/dashboard.json
go run ./cmd/rc-doctor check testdata/kmp --dashboard testdata/dashboard-kmp.json
go run ./cmd/rc-doctor check testdata/android-legacy --dashboard testdata/dashboard-legacy.json
go run ./cmd/rc-doctor facts testdata/kmp      # what the platform modules read
```

CI (`.github/workflows/ci.yml`) builds with Go 1.26, as releases do, and also runs golangci-lint v2.12.2
(`.golangci.yml`), `govulncheck`, a Windows cross-compile, the tests on macOS and on the minimum Go version in
`go.mod` (1.24), `goreleaser check`, and a parse of every JSON file. Tool versions are pinned in `mise.toml`.

## Project Architecture

rc-doctor is a single Go binary with no third-party dependencies. Every extension point is a plug-in that
registers itself from `init()` in a generic registry (`internal/registry`); a blank import in
`cmd/rc-doctor/plugins.go` links it in.

```
platform modules -> Facts --+
                            +--> rules -> Findings (+ Edits) -> suppression -> reporters (text, json, sarif)
dashboard providers --------+                    |
                                                 +--> fix engine <- review --fix, migrate --apply
```

### Package Structure
- **`internal/model`** - The shared vocabulary: `Facts`, `Dashboard`, `Catalog`, `Finding`, `Edit`, `Ref`, and the `Signal*` and `Build*` keys. No logic about any platform.
- **`internal/platform`** - The `Platform` contract and registry. `Superseder` lets an umbrella platform (KMP) replace the native ones it covers.
  - **`scan/`** - File walking (no symlinks), comment skipping, shared regexes, constant resolution (`Constants`, including `buildConfigField`), the guards around a key (`KeyGuards`: debug-only, and the platform its condition names), key platform hints (`PlatformHint`), and JVM-only Gradle modules to skip (`JVMModules`).
  - **`android/`, `ios/`, `kmp/`** - Platform modules. `android.BuildSettings` reads minSdk, desugaring, the SDK version and the manifest; the KMP module reuses it and classifies source sets by name (`sideOf`).
- **`internal/dashboard`** - Project sources: `RCCLI` (runs RevenueCat's `rc` in the checked directory and follows `next_page` through `rc api GET`), `Snapshot`, `Static`, `Offline`.
- **`internal/rules`** - Rules, one per file. They see only `rules.Context`, never the file system or a platform package.
- **`internal/config`** - `.rc-doctor.json`: rule severities, `off`, and ignore globs. Unknown fields and rule IDs are errors.
- **`internal/engine`** - detect -> collect -> load project -> run rules -> dedupe -> suppress (`rc-doctor:ignore` comments and the config) -> sort.
- **`internal/fix`** - The only code that writes user files.
- **`internal/migrate`** - The migration-source contract (no sources yet).
- **`internal/report`** - Reporters. **`internal/redact`** - Key masking for all output.
- **`internal/cli`**, **`internal/commands/*`** - Command dispatch and the commands; shared flags in `commands/shared`.
- **`docs/rules/<id>.md`** - One page per rule, embedded in the binary (`docs.go`) for `rc-doctor explain`.
- **`plugin/`** and **`.claude-plugin/`** - The agent plugin and its skill.

## Guardrails

- **Never print a full API key.** Reports, previews and `facts` go through `redact.Keys`. The end-to-end tests fail if a fixture key appears in any output. The one exception is `rc-doctor snapshot`, which writes the project's public SDK keys (they ship inside apps anyway) and never a secret key.
- **`Safe` edits must be provably correct from the data.** A best guess is `Suggested`. A key swap is `Safe` only when the key being replaced belongs to the project being checked (staging and production projects often share a bundle ID).
- **Rules stay platform-neutral.** Platform knowledge belongs in platform modules and reaches rules through `Facts` (for example `Facts.KeyPrefixes`, `Facts.Signals`, `Facts.Build`).
- **Only `internal/fix` writes files**, and only when the expected text is still on its line.
- **No third-party dependencies** without a strong reason: the tool must be trivial to install and audit.
- **Every RevenueCat fact has a source.** Key prefixes, SDK behavior and `rc` output shapes come from RevenueCat's SDK and CLI source or documentation; cite it in the code comment and in the rule's docs page. A fact you can't confirm is marked `TODO(verify)`, and a release has none.
- **Precision over recall.** A false positive costs more trust than a miss. When the code can't show whether something is wrong, lower the severity (a constant only defined, not followed to `Purchases.configure`, is info) or say in the message what rc-doctor couldn't see.

## Code Style Guidelines

- `gofmt` formatting; doc comments, written as full sentences, on every package and every exported type, function, variable and constant. Methods that only implement an interface (`Name`, `Run`, `Check`, `Write`) rely on the interface's documentation. golangci-lint must pass.
- Messages users see say what is wrong and how to fix it, in plain words. Use the RevenueCat dashboard's own terms (entitlement, offering, public SDK key).
- Errors wrap with context (`fmt.Errorf("load RevenueCat project from %s: %w", ...)`); commands print them once via `shared.Fail`.

## Testing

- Unit tests sit next to the code (`*_test.go`). Rules are tested with a hand-built `rules.Context`.
- Platform modules are tested against fixtures in `testdata/` (`sample` = native Android + iOS, `kmp` = Kotlin Multiplatform, `android-legacy` = an app that trips most rules). Each fixture includes a test source set that must be ignored.
- `cmd/rc-doctor/main_test.go` runs the real commands end to end, including the `rc` integration through a stand-in shell script (with a paginated listing) and a stand-in migration source.
- `cmd/rc-doctor/golden_test.go` pins the exact text, JSON and SARIF output in `testdata/golden`. After an intended change, run `make golden` and review the diff.
- `docs_test.go` checks that every rule has a docs page and an entry in `schema/config.schema.json`, and that the agent plugin's manifests and skill agree.
- Tests that apply fixes work on `testutil.CopyTree` copies, never on `testdata/` itself.

## Adding Things

- **Platform**: `internal/platform/<name>`, implement `platform.Platform`, register in `init()`, add to `plugins.go`, add a fixture and a `Collect` test.
- **Rule**: a file in `internal/rules` implementing `rules.Rule`, registered in `init()`, with table tests; a page in `docs/rules/<id>.md` with its sources; the ID in `schema/config.schema.json`; a fixture case when it needs code to trigger. Run it on real projects before relying on it (see `docs/survey`).
- **Migration source**: `internal/migrate/<source>` implementing `migrate.Migrator`; see the stand-in in `cmd/rc-doctor/main_test.go`.
- **Output format**: implement `report.Reporter` and register it. Document the shape in `docs/output.md`.
- **Command**: `internal/commands/<name>` implementing `cli.Command`, reusing `commands/shared`.

## Pull Requests

Commit and PR titles follow `type(scope): summary`, for example `feat(kmp): route shared keys by name` or
`fix(rules): don't suggest a tie` (the commit-msg hook in `lefthook.yml` checks it). Label each PR with one type
label as in RevenueCat's repositories (`pr:feat`, `pr:fix`, `pr:other`, `pr:dependencies`,
`pr:changelog_ignore`, plus `pr:breaking` when a flag, an exit code or the JSON or SARIF output changes
incompatibly); CI checks for it. Add user-facing changes to `CHANGELOG.latest.md`.
