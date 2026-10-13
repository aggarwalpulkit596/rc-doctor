// Package buildinfo reports which rc-doctor build is running.
package buildinfo

import "runtime/debug"

// Version is stamped at release time:
//
//	go build -ldflags "-X github.com/aggarwalpulkit596/rc-doctor/internal/buildinfo.Version=v0.1.0" ./cmd/rc-doctor
var Version = "dev"

// String returns Version, or the module version when the binary was installed
// with `go install .../cmd/rc-doctor@vX.Y.Z`.
func String() string {
	if Version != "dev" {
		return Version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return Version
}
