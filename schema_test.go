package rcdoctor_test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/aggarwalpulkit596/rc-doctor/internal/rules"
)

// The config schema lists every rule, so editors offer exactly the rules
// that exist.
func TestConfigSchemaListsEveryRule(t *testing.T) {
	b, err := os.ReadFile("schema/config.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties struct {
			Rules struct {
				PropertyNames struct {
					Enum []string `json:"enum"`
				} `json:"propertyNames"`
			} `json:"rules"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, r := range rules.All() {
		ids = append(ids, r.ID())
	}
	if got := schema.Properties.Rules.PropertyNames.Enum; !slices.Equal(got, ids) {
		t.Errorf("schema rules = %v\nregistered    = %v", got, ids)
	}
}
