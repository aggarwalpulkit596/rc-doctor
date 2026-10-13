# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## Common Development Commands

```bash
# Build the binary
go build -o rc-doctor ./cmd/rc-doctor

# Run all tests (unit + end-to-end), with the race detector as CI does
go test -race ./...

# Run one package's tests
go test ./internal/rules/...

# Static checks (CI fails on either)
gofmt -l .      # must print nothing; fix with gofmt -w .
go vet ./...

# Try it on the fixtures
go run ./cmd/rc-doctor check testdata/sample --dashboard testdata/dashboard.json
go run ./cmd/rc-doctor check testdata/kmp --dashboard testdata/dashboard-kmp.json
```

## Project Architecture

rc-doctor is a single Go binary with no third-party dependencies. Every extension point is a plug-in that
registers itself from `init()` in a generic registry (`internal/registry`); a blank import in
`cmd/rc-doctor/plugins.go` links it in.

```
platform modules -> Facts --+
                            +--> rules -> Findings (+ Edits) -> reporters (text, json, sarif)
dashboard providers --------+                    |
                                                 +--> fix engine <- review --fix, migrate --apply
```

### Package Structure
- **`internal/model`** - The shared vocabulary: `Facts`, `Dashboard`, `Finding`, `Edit`, `Ref`. No logic about any platform.
- **`internal/platform`** - The `Platform` contract and registry. `Superseder` lets an umbrella platform (KMP) replace the native ones it covers.
  - **`scan/`** - File walking, comment skipping, shared regexes, constant resolution (`Constants`), key platform hints (`PlatformHint`).
  - **`android/`, `ios/`, `kmp/`** - Platform modules.
- **`internal/dashboard`** - Project sources: `RCCLI` (shells out to RevenueCat's `rc`), `Snapshot`, `Static`, `Offline`.
- **`internal/rules`** - Rules. They see only `rules.Context`, never the file system or a platform package.
- **`internal/engine`** - detect -> collect -> load project -> run rules -> dedupe -> sort.
- **`internal/fix`** - The only code that writes user files.
- **`internal/migrate`** - The migration-source contract (no sources yet).
- **`internal/report`** - Reporters. **`internal/redact`** - Key masking for all output.
- **`internal/cli`**, **`internal/commands/*`** - Command dispatch and the commands; shared flags in `commands/shared`.

## Guardrails

- **Never print a full API key.** Reports, previews and `facts` go through `redact.Keys`. The end-to-end tests fail if a fixture key appears in any output.
- **`Safe` edits must be provably correct from the data.** A best guess is `Suggested`. A key swap is `Safe` only when the key being replaced belongs to the project being checked (staging and production projects often share a bundle ID).
- **Rules stay platform-neutral.** Platform knowledge belongs in platform modules and reaches rules through `Facts` (for example `Facts.KeyPrefixes`).
- **Only `internal/fix` writes files**, and only when the expected text is still on its line.
- **No third-party dependencies** without a strong reason: the tool must be trivial to install and audit.
- Facts about RevenueCat that are not yet confirmed (rc CLI JSON field names, key prefixes) are marked `TODO(verify)`. Don't remove the marker without checking the source.

## Code Style Guidelines

- `gofmt` formatting; doc comments on every exported identifier, written as full sentences.
- Messages users see say what is wrong and how to fix it, in plain words. Use the RevenueCat dashboard's own terms (entitlement, offering, public SDK key).
- Errors wrap with context (`fmt.Errorf("load RevenueCat project from %s: %w", ...)`); commands print them once via `shared.Fail`.

## Testing

- Unit tests sit next to the code (`*_test.go`). Rules are tested with a hand-built `rules.Context`.
- Platform modules are tested against fixtures in `testdata/` (`sample` = native Android + iOS, `kmp` = Kotlin Multiplatform). Each fixture includes a test source set that must be ignored.
- `cmd/rc-doctor/main_test.go` runs the real commands end to end, including the `rc` integration through a stand-in shell script and a stand-in migration source.
- Tests that apply fixes work on `testutil.CopyTree` copies, never on `testdata/` itself.

## Adding Things

- **Platform**: `internal/platform/<name>`, implement `platform.Platform`, register in `init()`, add to `plugins.go`, add a fixture and a `Collect` test.
- **Rule**: a file in `internal/rules` implementing `rules.Rule`, registered in `init()`, with table tests.
- **Migration source**: `internal/migrate/<source>` implementing `migrate.Migrator`; see the stand-in in `cmd/rc-doctor/main_test.go`.
- **Output format**: implement `report.Reporter` and register it.
- **Command**: `internal/commands/<name>` implementing `cli.Command`, reusing `commands/shared`.

## Pull Requests

Commit and PR titles follow `type(scope): summary`, for example `feat(kmp): route shared keys by name` or
`fix(rules): don't suggest a tie`. Add user-facing changes to `CHANGELOG.latest.md`.
