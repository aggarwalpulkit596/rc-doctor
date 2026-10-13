Releasing
=========

1. Move the entries in `CHANGELOG.latest.md` to the top of `CHANGELOG.md` under the new version, using the
   existing sections (✨ New Features, 🐞 Bugfixes, 🔄 Other Changes). Call out any change to command flags or
   JSON/SARIF output.
2. Set `.version` to the new version and open a `chore(release): X.Y.Z` pull request.
3. After it merges, tag the commit `vX.Y.Z` and push the tag. `go install github.com/aggarwalpulkit596/rc-doctor/cmd/rc-doctor@vX.Y.Z`
   then reports that version.
4. To attach binaries to the GitHub release, build each target with the version stamped in:

   ```bash
   V=$(cat .version)
   for t in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64; do
     CGO_ENABLED=0 GOOS=${t%/*} GOARCH=${t#*/} go build -trimpath \
       -ldflags "-s -w -X github.com/aggarwalpulkit596/rc-doctor/internal/buildinfo.Version=v$V" \
       -o dist/rc-doctor-$V-${t%/*}-${t#*/} ./cmd/rc-doctor
   done
   ```
