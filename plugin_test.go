package rcdoctor_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The agent plugin's manifests agree with each other, and the skill has the
// frontmatter agents need to find it.
func TestAgentPlugin(t *testing.T) {
	var market struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name    string `json:"name"`
			Source  string `json:"source"`
			Version string `json:"version"`
		} `json:"plugins"`
	}
	var plugin struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	for path, v := range map[string]any{".claude-plugin/marketplace.json": &market, "plugin/.claude-plugin/plugin.json": &plugin} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, v); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	if len(market.Plugins) != 1 || market.Plugins[0].Source != "./plugin" || market.Plugins[0].Name != plugin.Name || market.Plugins[0].Version != plugin.Version {
		t.Errorf("marketplace %+v doesn't match plugin %+v", market.Plugins, plugin)
	}

	skill, err := os.ReadFile("plugin/skills/rc-doctor/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	front, _, ok := strings.Cut(strings.TrimPrefix(string(skill), "---\n"), "\n---\n")
	if !ok || !strings.HasPrefix(string(skill), "---\n") {
		t.Fatal("SKILL.md must start with YAML frontmatter")
	}
	fields := map[string]string{}
	for _, line := range strings.Split(front, "\n") {
		if k, v, ok := strings.Cut(line, ": "); ok {
			fields[k] = v
		}
	}
	if fields["name"] != "rc-doctor" {
		t.Errorf("skill name = %q", fields["name"])
	}
	if d := fields["description"]; d == "" || len(d) > 1024 {
		t.Errorf("skill description is %d characters; it must be 1 to 1024", len(d))
	}
}
