# Output formats and exit codes

`rc-doctor check` and `rc-doctor review` print text by default. `--format json` and `--format sarif` are for scripts, agents and CI. Keys are masked in every format (`goog_AbCd…`); no output ever contains a full key.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No finding at or above `--fail-on` (default `error`) |
| 1 | At least one finding at or above `--fail-on` |
| 2 | rc-doctor couldn't run: a bad flag, an unreadable config, no app found, or (with `--dashboard rc` or a snapshot) a project that couldn't be loaded |

`--fail-on` takes `error`, `warning`, `info` or `never`.

## JSON (`--format json`)

```json
{
  "schema_version": 1,
  "tool": "rc-doctor",
  "version": "v0.2.0",
  "root": "/path/to/app",
  "apps": [{ "platform": "android", "app_id": "com.example.app" }],
  "dashboard": "rc-cli",
  "summary": { "error": 1, "warning": 0, "info": 0 },
  "findings": [
    {
      "rule": "api-key",
      "severity": "error",
      "platform": "android",
      "file": "app/src/main/java/com/example/App.kt",
      "line": 11,
      "message": "appl_AbCd… is a public key for ios apps, but this is the android app. …",
      "fix": "Use the android app's public SDK key (goog_ZyXw…).",
      "fixable": "safe",
      "docs": "https://github.com/aggarwalpulkit596/rc-doctor/blob/main/docs/rules/api-key.md"
    }
  ],
  "suppressed": 0,
  "skipped": [{ "rule": "app-id", "reason": "needs your RevenueCat project: …" }],
  "notes": []
}
```

| Field | Meaning |
|---|---|
| `schema_version` | Changes only when a field is removed or changes meaning. New fields don't change it. |
| `dashboard` | Where the RevenueCat project came from: `rc-cli`, `snapshot:<file>`, or `offline`. |
| `findings[].file` | Relative to `root`. Absent for findings that aren't about a line. |
| `findings[].fixable` | `safe` (applied by `review --fix`), `suggested` (applied with `--fix --suggested`), or absent. The edits themselves aren't in the report, because they can contain full keys; `rc-doctor review` previews them masked. |
| `suppressed` | Findings dropped by `rc-doctor:ignore` comments or `.rc-doctor.json`. |
| `skipped` | Rules that didn't run, and why: turned off in `.rc-doctor.json`, or they need the project and none was loaded. |
| `notes` | Partial project loads, for example "Couldn't list products with rc". |

## SARIF (`--format sarif`)

SARIF 2.1.0, for GitHub code scanning and other CI annotations. Each rule carries its description and a `helpUri` to its page in [docs/rules](rules). Severities map to SARIF levels (`error`, `warning`, `note`). Suppressed findings are included with a `suppressions` entry (`inSource` for a comment, `external` for the config file), so code scanning shows them as dismissed. File URIs are relative to the working directory: run rc-doctor from the repository root in CI.

## Facts (`rc-doctor facts`)

What rc-doctor read from the code, before any rule runs. It's the place to look when a finding seems wrong.

```json
{
  "root": "/path/to/app",
  "platforms": ["kmp"],
  "facts": [
    {
      "platform": "android",
      "app_id": "com.example.app",
      "app_id_at": { "file": "composeApp/build.gradle.kts", "line": 12 },
      "api_keys": [
        { "value": "goog_AbCd…", "location": { "file": "secrets.properties", "line": 1 }, "configured": true }
      ],
      "configure_at": [{ "file": "composeApp/src/androidMain/kotlin/App.kt", "line": 9 }],
      "entitlements": [{ "id": "pro", "location": { "file": "…", "line": 4 }, "defined_at": { "name": "ENTITLEMENT", "location": { "file": "…", "line": 2 }, "text": "\"pro\"" } }],
      "offerings": [],
      "key_prefixes": ["goog_", "amzn_", "galx_"],
      "signals": { "restore": [{ "file": "…", "line": 30 }] },
      "build": { "android.min_sdk": { "value": "24", "location": { "file": "composeApp/build.gradle.kts", "line": 14 } } }
    }
  ]
}
```

`api_keys[].configured` means the key reaches `Purchases.configure` (written in the call, or in the constant or `buildConfigField` it passes). `debug_only` means it sits behind a debug-only guard. `signals` record things the app does anywhere in its code (`restore`, `paywall-ui`, `customer-center`, `current-offering`, `completed-by-my-app`, `finishes-purchase`, `test-store-release`).

## Snapshots (`rc-doctor snapshot`)

A snapshot saves the RevenueCat project rc-doctor loaded through `rc`, so later runs (and CI without credentials) can use `--dashboard <file>`:

```json
{
  "entitlements": ["pro"],
  "offerings": ["default"],
  "apps": [{ "id": "app1a2b3c4", "platform": "android", "app_id": "com.example.app", "public_keys": ["goog_…"] }],
  "catalog": {
    "current_offering": "default",
    "packages": { "default": [{ "id": "$rc_monthly", "products": ["pro_monthly"] }] },
    "entitlement_products": { "pro": ["pro_monthly"] },
    "product_types": { "pro_monthly": "subscription" }
  }
}
```

Public SDK keys are public by design, so a snapshot keeps them in full; it holds no secret key. `catalog` is absent when the project's offerings, packages or products couldn't be listed, and the rules that need it stay silent.
