---
name: rc-doctor
description: Check a RevenueCat integration for mistakes that compile but fail in production, by reading the app's code next to its RevenueCat project, and apply the fixes that are safe. Use when the user asks to check, audit or review a RevenueCat integration, before a release that touches purchases, after adding or upgrading the RevenueCat SDK, or when they report a wrong-platform or invalid-credentials API key error, an entitlement that never unlocks, an empty paywall or offering, a crash on launch after configuring Purchases, or purchases refunded after three days. Works on native Android, native iOS and Kotlin Multiplatform apps.
---

# rc-doctor: check a RevenueCat integration

rc-doctor reads the app's code (Kotlin, Java, Swift, Objective-C, Gradle, `.properties`, `.xcconfig`, `Info.plist`) and, through RevenueCat's own `rc` CLI, the RevenueCat project. It reports mismatches between the two with a file and line, and it can apply the fixes the data proves.

It checks code, not runtime behavior. For a problem it doesn't cover (a sandbox purchase that never shows up, a store misconfiguration), use RevenueCat's `revenuecat-troubleshoot` skill.

## 1. Make sure rc-doctor is installed

```sh
rc-doctor version
```

If it's missing, ask before installing, then either:

```sh
go install github.com/aggarwalpulkit596/rc-doctor/cmd/rc-doctor@latest   # Go 1.24 or newer
```

or download the archive for the platform from https://github.com/aggarwalpulkit596/rc-doctor/releases and put `rc-doctor` on the PATH.

## 2. Decide where the RevenueCat project comes from

Six of the thirteen rules compare the code with the project: unknown entitlement or offering IDs, the app ID, the current offering, empty offerings, and entitlements without products. They need RevenueCat's CLI, `rc`:

```sh
rc auth status
```

- **`rc` is installed and logged in:** use `--dashboard rc`. Add `--project-id <id>` if the repository has no `.revenuecat.json` and the user has several projects.
- **`rc` is missing or logged out:** run offline (`--dashboard offline`) and tell the user which rules were skipped. Offer `npm install -g @revenuecat/cli` (or `brew install RevenueCat/tap/rc`) and `rc auth login`, which the user runs themselves.
- **A snapshot file is in the repo** (from `rc-doctor snapshot`): pass its path with `--dashboard`.

Never ask the user to paste a RevenueCat API key into the conversation. `rc` keeps its own login, and `RC_API_KEY` belongs in their shell or CI secrets.

## 3. Run the check

From the app's root:

```sh
rc-doctor check --format json --fail-on never --dashboard rc .
```

The output is documented at https://github.com/aggarwalpulkit596/rc-doctor/blob/main/docs/output.md. For each finding, report the severity, the rule, `file:line`, the message and the fix, in plain words. For more on a rule:

```sh
rc-doctor explain <rule-id>
```

Read `skipped` and `notes` as well: they say which checks didn't run and why.

## 4. Fix

Preview every available fix first. This changes nothing:

```sh
rc-doctor review --dashboard rc .
```

- **Safe fixes** are proven by the project data, for example replacing an iOS key in the Android app with this project's Android key. Apply them with `rc-doctor review --fix` once the user agrees.
- **Suggested fixes** are best guesses: a close match for a mistyped entitlement ID, or a key swap rc-doctor can't tie to this project, since staging and production projects often share a bundle ID. Show each one and apply `--fix --suggested` only when the user confirms them.

Some findings are fixed in the RevenueCat project, not the code: no current offering, an offering without packages, an entitlement without products. The message names the `rc` command, for example `rc offerings set-current <offering-id>`. These change the live project the app serves. Propose the exact command and run it only on the user's explicit go-ahead, never as part of a batch.

Run `rc-doctor check` again after fixing, and report what's left.

## 5. When a finding is wrong

rc-doctor reads code without compiling it, so it can miss a guard (a key chosen by a build flavor, a scheme or a CI variable). Confirm with the user before suppressing anything, then mark the line with the reason:

```kotlin
const val RC_TEST_KEY = "test_..." // rc-doctor:ignore test-store-key -- debug builds only
```

Suppress for a whole path, or turn a rule off, in `.rc-doctor.json` at the root:

```json
{
  "$schema": "https://raw.githubusercontent.com/aggarwalpulkit596/rc-doctor/main/schema/config.schema.json",
  "rules": { "restore-purchases": "off" },
  "ignore": ["examples/"]
}
```

If rc-doctor is plainly wrong, suggest opening an issue at https://github.com/aggarwalpulkit596/rc-doctor/issues with the output of `rc-doctor check --format json` and `rc-doctor facts`. Both mask keys, so they are safe to share.

## Guardrails

- Never print, log or commit a full API key. rc-doctor masks keys as `goog_AbCd…`; keep them masked in anything you write.
- A secret key (`sk_...`) found in app code has to be rotated in the RevenueCat dashboard after it's removed, because anyone with the code or the app binary can read it. Say so.
- Don't change the RevenueCat project, push code, or rotate keys without the user's explicit approval.

## Keep it in CI

Suggest a CI step once the integration is clean. Either give CI a RevenueCat API key and install `rc`:

```sh
npm install -g @revenuecat/cli
RC_API_KEY=$RC_API_KEY rc-doctor check --dashboard rc --format sarif --fail-on never > rc-doctor.sarif
RC_API_KEY=$RC_API_KEY rc-doctor check --dashboard rc
```

or commit a snapshot so CI needs no credentials: `rc-doctor snapshot -o rc-project.json`, then `rc-doctor check --dashboard rc-project.json`. The README has a complete GitHub Actions job with SARIF upload.
