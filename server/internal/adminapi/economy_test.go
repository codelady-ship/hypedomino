package adminapi_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/economy"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func TestTheActiveConfigVersionIsTheNewestPublish(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Analyst)
	analyst := a.signIn("orxan")
	status, body := a.call("GET", "/admin/config/version", analyst, nil)
	if status != http.StatusOK || body["config_version"] != float64(0) || body["domain"] != nil {
		t.Fatalf("before any publish %d %v", status, body)
	}
	if _, err := snapshot.Publish(context.Background(), a.pool, "rules", []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	version, err := snapshot.Publish(context.Background(), a.pool, "shop", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	status, body = a.call("GET", "/admin/config/version", analyst, nil)
	if status != http.StatusOK || body["config_version"] != float64(version) || body["domain"] != "shop" || body["published_at"] == nil {
		t.Fatalf("active %d %v", status, body)
	}
}

func TestEconomyMonitorsAndTheirComparisonAroundAPublish(t *testing.T) {
	a := newAPIWith(t, func(c *adminapi.Config) { c.Economy = economy.DefaultConfig() })
	a.create("orxan@example.com", rbac.Analyst)
	analyst := a.signIn("orxan")
	var player string
	if err := a.pool.QueryRow(context.Background(), `insert into accounts default values returning id`).Scan(&player); err != nil {
		t.Fatal(err)
	}
	_, err := a.pool.Exec(context.Background(), `insert into ledger_entries (account_id, currency, amount, reason, source_type, source_ref, idempotency_key, config_version, balance_after, created_at)
		values ($1, 'coin', 400, 'match_reward', 'test', 'test', 'k1', 1, 400, $2), ($1, 'coin', -300, 'shop_purchase', 'test', 'test', 'k2', 1, 100, $3)`,
		player, start.Add(-30*time.Hour), start.Add(-30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	status, body := a.call("GET", "/admin/economy/monitors", analyst, nil)
	if status != http.StatusOK || body["coinsCreated"] != float64(400) || body["coinsSpent"] != float64(300) || body["sinkRatioPercent"] != float64(75) || body["medianBalance"] != float64(100) || len(body["alerts"].([]any)) != 0 {
		t.Fatalf("monitors %d %v", status, body)
	}

	version, err := snapshot.Publish(context.Background(), a.pool, "match_rewards", []byte(`{"schema":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.pool.Exec(context.Background(), `update config_snapshots set published_at = $2 where version = $1`, version, start.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	status, body = a.call("GET", fmt.Sprintf("/admin/economy/monitors/compare?version=%d", version), analyst, nil)
	before, _ := body["before"].(map[string]any)
	after, _ := body["after"].(map[string]any)
	if status != http.StatusOK || body["domain"] != "match_rewards" || before["coinsSpent"] != float64(0) || before["sinkRatioPercent"] != float64(0) || after["to"] != start.Format(time.RFC3339) {
		t.Fatalf("compare %d %v", status, body)
	}

	status, body = a.call("GET", "/admin/economy/monitors/compare?version=999", analyst, nil)
	expect(t, status, body, http.StatusNotFound, "not_found")
	status, body = a.call("GET", "/admin/economy/monitors/compare?version=x", analyst, nil)
	expect(t, status, body, http.StatusBadRequest, "invalid")
}
