// Package registry is the generic plug-in registry behind every extension point
// in rc-doctor: platforms, rules, reporters, migration sources and commands.
//
// Each plug-in registers itself from an init() function, and a blank import in
// cmd/rc-doctor/plugins.go is what links it into the binary.
package registry

import (
	"fmt"
	"sort"
	"sync"
)

// Registry holds named plug-ins of one kind.
type Registry[T any] struct {
	kind  string
	key   func(T) string
	mu    sync.RWMutex
	items map[string]T
}

// New returns an empty registry. kind names the plug-in type in error messages
// ("platform", "rule"); key returns a plug-in's unique name.
func New[T any](kind string, key func(T) string) *Registry[T] {
	return &Registry[T]{kind: kind, key: key, items: map[string]T{}}
}

// Register adds v. It panics on an empty or duplicate name, so wiring mistakes
// surface the moment the binary starts rather than in front of a user.
func (r *Registry[T]) Register(v T) {
	name := r.key(v)
	r.mu.Lock()
	defer r.mu.Unlock()
	if name == "" {
		panic(fmt.Sprintf("registry: %s registered with an empty name", r.kind))
	}
	if _, dup := r.items[name]; dup {
		panic(fmt.Sprintf("registry: %s %q registered twice", r.kind, name))
	}
	r.items[name] = v
}

// Unregister removes a plug-in. It exists so tests can add temporary plug-ins.
func (r *Registry[T]) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, name)
}

// Get returns the plug-in registered under name.
func (r *Registry[T]) Get(name string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.items[name]
	return v, ok
}

// All returns every plug-in, sorted by name so output is deterministic.
func (r *Registry[T]) All() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := r.sortedNames()
	out := make([]T, 0, len(names))
	for _, n := range names {
		out = append(out, r.items[n])
	}
	return out
}

// Names returns the registered names, sorted.
func (r *Registry[T]) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sortedNames()
}

func (r *Registry[T]) sortedNames() []string {
	names := make([]string, 0, len(r.items))
	for n := range r.items {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
