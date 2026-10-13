<!-- Thank you for contributing to rc-doctor! Before pressing the "Create Pull Request" button, please provide the following: -->

### Checklist
- [ ] Tests for new rules, platforms or commands, and `make check` passes
- [ ] A new rule has a page in `docs/rules/` with its sources, and an entry in `schema/config.schema.json`
- [ ] A fixture under `testdata/` for a new platform or migration source; `make golden` reviewed if output changed
- [ ] `CHANGELOG.latest.md` updated for user-facing changes
- [ ] One type label: `pr:feat`, `pr:fix`, `pr:other`, `pr:dependencies` or `pr:changelog_ignore` (plus `pr:breaking` if flags, exit codes or the JSON/SARIF output change incompatibly)

### Motivation
<!-- Why is this change required? What problem does it solve? -->
<!-- Please link to issues following this format: Resolves #999999 -->

### Description
<!-- Describe your changes in detail -->
<!-- Please describe in detail how you tested your changes -->
