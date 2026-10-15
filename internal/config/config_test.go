package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/model"
)

func write(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefaultWhenAbsent(t *testing.T) {
	c, err := Load(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !c.Enabled("api-key") || c.Ignored("app/App.kt") {
		t.Errorf("default config must enable every rule and ignore nothing: %+v", c)
	}
	if _, set := c.Severity("api-key"); set {
		t.Error("default config sets no severities")
	}
}

func TestRulesAndIgnore(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, `{
	  "$schema": "https://example.com/config.schema.json",
	  "rules": {"restore-purchases": "off", "app-id": "error"},
	  "ignore": ["examples/**", "legacy/", "*.generated.kt", "app/src/debug/**/Keys.kt"]
	}`)
	c, err := Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Validate([]string{"restore-purchases", "app-id", "api-key"}); err != nil {
		t.Fatal(err)
	}
	if c.Enabled("restore-purchases") || !c.Enabled("api-key") {
		t.Error("restore-purchases is off; api-key is on")
	}
	if sev, ok := c.Severity("app-id"); !ok || sev != model.Error {
		t.Errorf("app-id severity = %q, %v", sev, ok)
	}
	for file, want := range map[string]bool{
		"examples/MagicWeather/App.kt":   true,
		"legacy/Old.swift":               true,
		"app/legacy/Old.swift":           true, // "legacy/" has no inner slash: any depth
		"app/src/Foo.generated.kt":       true, // no slash: any depth
		"app/src/debug/java/x/Keys.kt":   true,
		"app/src/main/java/x/Keys.kt":    false,
		"app/src/main/java/x/Paywall.kt": false,
		"examplesX/App.kt":               false,
		"app/examples/App.kt":            false, // "examples/**" is anchored to the root
	} {
		if got := c.Ignored(file); got != want {
			t.Errorf("Ignored(%q) = %v, want %v", file, got, want)
		}
	}
}

func TestErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		content, want string
	}{
		"unknown field": {`{"rule": {}}`, "unknown field"},
		"bad json":      {`{`, "parse"},
		"unknown rule":  {`{"rules": {"api-keys": "off"}}`, `unknown rule "api-keys"`},
		"bad severity":  {`{"rules": {"api-key": "fatal"}}`, `"fatal" is not off`},
		"empty ignore":  {`{"ignore": [" "]}`, "empty pattern"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, tc.content)
			c, err := Load(dir, "")
			if err == nil {
				err = c.Validate([]string{"api-key"})
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to mention %q", err, tc.want)
			}
		})
	}
	if _, err := Load(t.TempDir(), filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("an explicit --config path that doesn't exist must fail")
	}
}
