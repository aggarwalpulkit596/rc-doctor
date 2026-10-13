## Contributing

Thanks for helping make RevenueCat integrations easier to get right. rc-doctor is an independent project and is not
affiliated with RevenueCat.

### 1. Open an issue first

Search [existing issues](https://github.com/aggarwalpulkit596/rc-doctor/issues) before filing a new one. For a false
positive or a missed problem, the bug template asks for `rc-doctor check --format json` and `rc-doctor facts`
output; both mask keys, so they are safe to paste.

### 2. Set up

You need Go 1.24 or newer. There are no other dependencies.

```bash
git clone https://github.com/aggarwalpulkit596/rc-doctor
cd rc-doctor
go test ./...
```

### 3. Make the change

- Read [AGENTS.md](AGENTS.md) for the architecture and the guardrails, especially: never print a full key, and
  mark an edit `Safe` only when the data proves it.
- New platforms, rules, migration sources, formats and commands are plug-ins; AGENTS.md lists where each goes.
- Add tests next to the code, and a fixture under `testdata/` for a new platform or migration source.

### 4. Check it

```bash
gofmt -l .        # prints nothing
go vet ./...
go test -race ./...
```

### 5. Open a pull request

Use a `type(scope): summary` title (`feat(rules): flag duplicate configure calls`), fill in the template, and add
user-facing changes to `CHANGELOG.latest.md`.
