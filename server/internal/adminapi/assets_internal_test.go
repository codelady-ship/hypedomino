package adminapi

import (
	"encoding/json"
	"maps"
	"os"
	"testing"
)

func TestStaticTextureLimitsMatchTheExportBudgets(t *testing.T) {
	raw, err := os.ReadFile("../../../tools/asset-export/budgets.json")
	if err != nil {
		t.Fatal(err)
	}
	var budgets struct {
		AstcBlock string `json:"astcBlock"`
		Types     map[string]struct {
			Static *struct {
				MaxBytes int64 `json:"maxBytes"`
			} `json:"static"`
		} `json:"types"`
	}
	if err := json.Unmarshal(raw, &budgets); err != nil {
		t.Fatal(err)
	}
	limits := map[string]int64{}
	for kind, rule := range budgets.Types {
		if rule.Static != nil {
			limits[kind] = rule.Static.MaxBytes
		}
	}
	if budgets.AstcBlock != "6x6" || !maps.Equal(limits, staticTextureLimits) {
		t.Fatalf("budgets %s %v, server %v", budgets.AstcBlock, limits, staticTextureLimits)
	}
}
