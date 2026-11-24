package adminapi_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func exampleStorePrices(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../../docs/economy/examples/store_prices.json")
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	return config
}

func storePricesWith(t *testing.T, change func(config map[string]any)) map[string]any {
	config := exampleStorePrices(t)
	change(config)
	return config
}

func targetOf(t *testing.T, body map[string]any, product, country string) (float64, bool) {
	t.Helper()
	for _, row := range body["rules"].(map[string]any)["targets"].([]any) {
		target := row.(map[string]any)
		if target["product"] == product && target["country"] == country {
			return target["cents"].(float64), true
		}
	}
	return 0, false
}

func TestStorePricesArePublishedCheckedAndRolledBack(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("vusal@example.com", rbac.Support)
	liveOps, support, content := a.signIn("leyla"), a.signIn("vusal"), a.signIn("ayla")

	status, body := a.call("GET", "/admin/config/store-prices", content, nil)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("GET", "/admin/config/store-prices", support, nil)
	expect(t, status, body, http.StatusOK, "")
	rules := body["rules"].(map[string]any)
	if body["catalog"] != nil || len(rules["products"].([]any)) != 8 || len(rules["countries"].([]any)) != 248 {
		t.Fatalf("empty %v", body)
	}
	if cents, ok := targetOf(t, body, "pass_premium", "AZ"); !ok || cents != 299 {
		t.Fatalf("default AZ target %v %v", cents, ok)
	}

	status, body = a.call("PUT", "/admin/config/store-prices", support, map[string]any{"catalog": exampleStorePrices(t), "reason": "launch"})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	for _, refused := range []map[string]any{
		storePricesWith(t, func(c map[string]any) { c["countries"].(map[string]any)["RU"] = "mid" }),
		storePricesWith(t, func(c map[string]any) { c["tiers"].(map[string]any)["low"] = 0 }),
		storePricesWith(t, func(c map[string]any) { c["countries"].(map[string]any)["AZ"] = "cheap" }),
		storePricesWith(t, func(c map[string]any) { c["overrides"] = map[string]any{"coins_tier1": map[string]any{"AZ": 0.49}} }),
		storePricesWith(t, func(c map[string]any) { c["currency"] = "AZN" }),
	} {
		status, body = a.call("PUT", "/admin/config/store-prices", liveOps, map[string]any{"catalog": refused, "reason": "bad"})
		expect(t, status, body, http.StatusBadRequest, "invalid")
	}

	status, body = a.call("PUT", "/admin/config/store-prices", liveOps, map[string]any{"catalog": exampleStorePrices(t), "reason": "launch"})
	expect(t, status, body, http.StatusOK, "")
	first := body["config_version"]
	override := storePricesWith(t, func(c map[string]any) { c["overrides"] = map[string]any{"pass_premium": map[string]any{"AZ": 2.49}} })
	status, body = a.call("PUT", "/admin/config/store-prices", liveOps, map[string]any{"catalog": override, "reason": "AZ pass test"})
	expect(t, status, body, http.StatusOK, "")

	status, body = a.call("GET", "/admin/config/store-prices", support, nil)
	expect(t, status, body, http.StatusOK, "")
	if cents, _ := targetOf(t, body, "pass_premium", "AZ"); cents != 249 {
		t.Fatalf("the published override is the target: %v", cents)
	}
	status, body = a.call("GET", "/admin/config/store-prices/history", support, nil)
	expect(t, status, body, http.StatusOK, "")
	if len(body["versions"].([]any)) != 2 {
		t.Fatalf("history %v", body)
	}

	status, body = a.call("POST", "/admin/config/store-prices/rollback", liveOps, map[string]any{"version": first, "reason": "back"})
	expect(t, status, body, http.StatusOK, "")
	status, body = a.call("GET", "/admin/config/store-prices", support, nil)
	expect(t, status, body, http.StatusOK, "")
	if cents, _ := targetOf(t, body, "pass_premium", "AZ"); cents != 299 {
		t.Fatalf("rolled back %v", cents)
	}
}
