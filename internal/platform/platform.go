// Package platform defines the contract every platform module implements and
// the registry the engine finds them in.
//
// To add a platform (Flutter, React Native, Kotlin Multiplatform, Unity, ...):
//  1. Create internal/platform/<name> with a type implementing Platform.
//     The helpers in internal/platform/scan do the file walking and matching.
//  2. Call platform.Register from that package's init().
//  3. Blank-import the package in cmd/rc-doctor/plugins.go.
//  4. Add a fixture app under testdata/ and a Collect test.
//
// Rules never import a platform package. They only see model.Facts, which is
// why a new platform gets every existing rule for free.
package platform

import (
	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
	"github.com/aggarwalpulkit596/rc-doctor/internal/registry"
)

// Platform turns a directory of source code into Facts.
type Platform interface {
	// Name is the stable identifier used in flags and output ("android", "ios").
	Name() string
	// Detect reports whether root contains an app of this platform that uses RevenueCat.
	Detect(root string) bool
	// Collect extracts facts. It returns one Facts per app target found under root.
	Collect(root string) ([]model.Facts, error)
	// KeyPrefixes lists the public SDK key prefixes that belong to this platform.
	KeyPrefixes() []string
}

// Superseder is implemented by a platform that covers others when detected:
// a Kotlin Multiplatform app also looks like an Android and an iOS app, and
// must be read as one KMP app, not as those two.
type Superseder interface {
	Supersedes() []string
}

var reg = registry.New[Platform]("platform", Platform.Name)

// Register adds a platform. It panics on duplicate names.
func Register(p Platform) { reg.Register(p) }

// All returns registered platforms sorted by name.
func All() []Platform { return reg.All() }

// Get returns a platform by name.
func Get(name string) (Platform, bool) { return reg.Get(name) }

// Names returns the registered platform names, sorted.
func Names() []string { return reg.Names() }
