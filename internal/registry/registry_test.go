package registry

import (
	"reflect"
	"testing"
)

type plugin struct{ name string }

func newTestRegistry() *Registry[plugin] {
	return New("plugin", func(p plugin) string { return p.name })
}

func TestRegisterGetAll(t *testing.T) {
	r := newTestRegistry()
	r.Register(plugin{"zeta"})
	r.Register(plugin{"alpha"})

	if p, ok := r.Get("zeta"); !ok || p.name != "zeta" {
		t.Fatalf("Get(zeta) = %v, %v", p, ok)
	}
	if _, ok := r.Get("missing"); ok {
		t.Fatal("Get(missing) should fail")
	}
	if got, want := r.Names(), []string{"alpha", "zeta"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	if got := r.All(); len(got) != 2 || got[0].name != "alpha" {
		t.Fatalf("All() = %v, want sorted by name", got)
	}

	r.Unregister("zeta")
	if _, ok := r.Get("zeta"); ok {
		t.Fatal("Unregister did not remove zeta")
	}
}

func TestRegisterPanicsOnDuplicateOrEmpty(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first plugin
		then  plugin
	}{
		{"duplicate", plugin{"a"}, plugin{"a"}},
		{"empty", plugin{"a"}, plugin{""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRegistry()
			r.Register(tc.first)
			defer func() {
				if recover() == nil {
					t.Fatal("expected a panic")
				}
			}()
			r.Register(tc.then)
		})
	}
}
