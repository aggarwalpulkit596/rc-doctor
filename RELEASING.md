Releasing
=========

1. In `CHANGELOG.latest.md`, replace `## Unreleased` with the new version, keeping the sections (✨ New Features,
   🐞 Bugfixes, 🔄 Other Changes). Call out any change to command flags, exit codes, or the JSON or SARIF output.
   Copy the whole entry to the top of `CHANGELOG.md`.
2. Set `.version` to the new version and open a `chore(release): X.Y.Z` pull request labeled `pr:other`.
3. After it merges, tag the merge commit `vX.Y.Z` and push the tag. The release workflow
   (`.github/workflows/release.yml`) checks that the tag matches `.version`, builds archives for macOS, Linux and
   Windows (amd64 and arm64) with GoReleaser, signs `checksums.txt` with cosign (keyless, from the workflow's
   GitHub identity), and publishes the GitHub release with `CHANGELOG.latest.md` as its notes.
   `go install github.com/aggarwalpulkit596/rc-doctor/cmd/rc-doctor@vX.Y.Z` then reports that version.
4. Reset `CHANGELOG.latest.md` to `## Unreleased` in the next pull request.

To check the archives before tagging, run `make snapshot`: it builds every target into `dist/` without
publishing anything.
