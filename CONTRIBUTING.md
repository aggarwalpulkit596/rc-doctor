## Contributing

Thanks for helping make RevenueCat integrations easier to get right. rc-doctor is an independent project and is not
affiliated with RevenueCat.

### 1. Open an issue first

Search [existing issues](https://github.com/aggarwalpulkit596/rc-doctor/issues) before filing a new one. For a false
positive or a missed problem, the bug template asks for `rc-doctor check --format json` and `rc-doctor facts`
output; both mask keys, so they are safe to paste.

### 2. Set up

You need Go 1.24 or newer. There are no other dependencies. [mise](https://mise.jdx.dev) installs the pinned
tool versions CI uses (`mise.toml`), and `make install-hooks` adds the pre-commit and commit message checks
(`lefthook.yml`).

```bash
git clone https://github.com/aggarwalpulkit596/rc-doctor
cd rc-doctor
mise install        # optional: Go, golangci-lint and GoReleaser at CI's versions
make install-hooks  # optional: run `make check` before each commit
make test
```

### 3. Make the change

- Read [AGENTS.md](AGENTS.md) for the architecture and the guardrails, especially: never print a full key, and
  mark an edit `Safe` only when the data proves it.
- New platforms, rules, migration sources, formats and commands are plug-ins; AGENTS.md lists where each goes.
- Add tests next to the code, and a fixture under `testdata/` for a new platform or migration source.

### 4. Check it

```bash
make check    # gofmt, go vet, go test -race, and golangci-lint when it's installed
make golden   # after an intended change to the output, then review the diff in testdata/golden
```

CI also runs `govulncheck`, a Windows cross-compile, the tests on macOS and `goreleaser check`.

### 5. Open a pull request

Use a `type(scope): summary` title (`feat(rules): flag duplicate configure calls`), fill in the template, and add
user-facing changes to `CHANGELOG.latest.md`. Label the pull request with its type, as in RevenueCat's
repositories: `pr:feat`, `pr:fix`, `pr:other`, `pr:dependencies` or `pr:changelog_ignore` (CI checks for exactly
one). Add `pr:breaking` when a flag, an exit code, or the JSON or SARIF output changes incompatibly.
