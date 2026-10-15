# current-offering

| | |
|---|---|
| Severity | error |
| Platforms | Android, iOS, Kotlin Multiplatform |
| Needs your RevenueCat project | Yes, with its catalog |
| Automatic fix | none |

## What it checks

When the code reads `offerings.current`, that an offering in the project is marked current.

## Why it matters

With no current offering, `offerings.current` is null and the paywall has nothing to show. RevenueCat's troubleshooting playbook checks for "at least one offering with `is_current: true`".

## How to fix

Mark an offering as current in the dashboard, or run `rc offerings set-current <offering-id>`.

## Sources

- RevenueCat, [revenuecat-troubleshoot skill](https://github.com/RevenueCat/ai-toolkit/blob/main/revenuecat/skills/revenuecat-troubleshoot/SKILL.md), Phase B, Check 4.
