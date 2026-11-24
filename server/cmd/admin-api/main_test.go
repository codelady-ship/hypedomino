package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/adminsession"
	"github.com/Alievs-corp/hypedomino/server/internal/economy"
	"github.com/Alievs-corp/hypedomino/server/internal/googleauth/googleauthtest"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/ratelimit"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
)

func env(values map[string]string) lookup {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func TestConfigReadsTheAdminEnvironment(t *testing.T) {
	got, err := configFromEnv(env(map[string]string{
		"HD_DATABASE_URL":                 "postgres://admin",
		"HD_ADMIN_ENVIRONMENT":            "staging",
		"HD_ADMIN_GOOGLE_DOMAIN":          "example.com",
		"HD_ADMIN_GOOGLE_AUDIENCES":       " web.apps.example , ,other ",
		"HD_ADMIN_SESSION_IDLE":           "20m",
		"HD_ADMIN_STEP_UP":                "5m",
		"HD_ADMIN_APPROVAL_TTL":           "48h",
		"HD_SHOP_DAILY_SLOTS":             "6",
		"HD_SHOP_RESET_HOUR_UTC":          "4",
		"HD_SHOP_REFERENCE_PRICE_DAYS":    "14",
		"HD_ADMIN_GRANT_APPROVAL_COINS":   "5000",
		"HD_TEMP_BAN_HOURS":               "72",
		"HD_RANKED_BAN_HOURS":             "48",
		"HD_READ_TIMEOUT":                 "30s",
		"HD_ADMIN_RATE_ACCOUNT_BURST":     "80",
		"HD_ADMIN_RATE_SIGNIN_PER_MINUTE": "4",
		"HD_BALANCE_MEDIAN_ALERT_COINS":   "40000",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := config{
		listenAddr: ":8090", logLevel: "info", database: "postgres://admin", audiences: []string{"web.apps.example", "other"}, readLimit: 30 * time.Second, idleLimit: time.Minute, configPoll: 10 * time.Second,
		api: adminapi.Config{
			Environment: "staging", GoogleDomain: "example.com",
			Session:            adminsession.Config{Idle: 20 * time.Minute, Max: 12 * time.Hour, StepUp: 5 * time.Minute},
			ApprovalTTL:        48 * time.Hour,
			Shop:               shop.Config{DailySlots: 6, WeeklySlots: 2, ResetHourUTC: 4, ReferencePriceDays: 14},
			GrantApprovalCoins: 5000,
			TempBan:            72 * time.Hour,
			Mute:               adminapi.DefaultMuteHours * time.Hour,
			RankedBan:          48 * time.Hour,
			Limits:             adminapi.Limits{Account: ratelimit.Rule{PerMinute: 120, Burst: 80}, SignIn: ratelimit.Rule{PerMinute: 4, Burst: 10}},
			Economy:            economy.Config{SinkRatioLowPercent: 60, SinkRatioHighPercent: 120, AccountDailyEarnAlertCoins: 1500, BalanceMedianAlertCoins: 40000, FarmWeight: 10},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("config %+v, want %+v", got, want)
	}
}

func TestTheAssetStoreIsOptionalButACompleteOneIsRequiredOnceNamed(t *testing.T) {
	base := map[string]string{"HD_DATABASE_URL": "postgres://admin", "HD_ADMIN_ENVIRONMENT": "staging", "HD_ADMIN_GOOGLE_DOMAIN": "example.com", "HD_ADMIN_GOOGLE_AUDIENCES": "web.apps.example"}
	without, err := configFromEnv(env(base))
	if err != nil || without.api.Assets != nil {
		t.Fatalf("without a store: %v %v", without.api.Assets, err)
	}

	base["HD_ASSET_STORE_URL"] = "https://store.example/bucket"
	if _, err := configFromEnv(env(base)); err == nil || !strings.Contains(err.Error(), "HD_ASSET_STORE_") {
		t.Fatalf("incomplete store: %v", err)
	}

	base["HD_ASSET_STORE_REGION"], base["HD_ASSET_STORE_ACCESS_KEY"], base["HD_ASSET_STORE_SECRET_KEY"] = "auto", "<access>", "<secret>"
	with, err := configFromEnv(env(base))
	if err != nil || with.api.Assets == nil {
		t.Fatalf("complete store: %v %v", with.api.Assets, err)
	}
}

func TestTheContentStoreIsOptionalButMustBeCompleteAndOfThisEnvironment(t *testing.T) {
	complete := map[string]string{"HD_CONTENT_STORE_REGION": "auto", "HD_CONTENT_STORE_ACCESS_KEY": "<access>", "HD_CONTENT_STORE_SECRET_KEY": "<secret>"}
	cases := []struct {
		name    string
		extra   map[string]string
		store   bool
		refused string
	}{
		{"none", nil, false, ""},
		{"complete", merged(complete, map[string]string{"HD_CONTENT_STORE_URL": "https://store.example/hypedomino-content-staging"}), true, ""},
		{"incomplete", map[string]string{"HD_CONTENT_STORE_URL": "https://store.example/hypedomino-content-staging"}, false, "HD_CONTENT_STORE_*"},
		{"another environment", merged(complete, map[string]string{"HD_CONTENT_STORE_URL": "https://store.example/hypedomino-content-production"}), false, "HD_CONTENT_STORE_URL"},
		{"no environment", merged(complete, map[string]string{"HD_CONTENT_STORE_URL": "https://store.example/hypedomino-content"}), false, "HD_CONTENT_STORE_URL"},
	}
	for _, c := range cases {
		base := merged(map[string]string{"HD_DATABASE_URL": "postgres://admin", "HD_ADMIN_ENVIRONMENT": "staging", "HD_ADMIN_GOOGLE_DOMAIN": "example.com", "HD_ADMIN_GOOGLE_AUDIENCES": "web"}, c.extra)
		got, err := configFromEnv(env(base))
		if c.refused != "" {
			if err == nil || !strings.Contains(err.Error(), c.refused) {
				t.Errorf("%s: %v, want an error naming %s", c.name, err, c.refused)
			}
			continue
		}
		if err != nil || (got.api.Content != nil) != c.store {
			t.Errorf("%s: store %v, error %v", c.name, got.api.Content, err)
		}
	}
}

func merged(maps ...map[string]string) map[string]string {
	all := map[string]string{}
	for _, m := range maps {
		for key, value := range m {
			all[key] = value
		}
	}
	return all
}

func TestConfigNamesEveryMissingOrInvalidValue(t *testing.T) {
	_, err := configFromEnv(env(map[string]string{"HD_ADMIN_ENVIRONMENT": "prod", "HD_ADMIN_SESSION_MAX": "0s", "HD_SHOP_WEEKLY_SLOTS": "two", "HD_SHOP_RESET_HOUR_UTC": "24", "HD_SHOP_REFERENCE_PRICE_DAYS": "0", "HD_ADMIN_GRANT_APPROVAL_COINS": "0", "HD_TEMP_BAN_HOURS": "-1", "HD_MUTE_HOURS": "0", "HD_RANKED_BAN_HOURS": "0", "HD_ADMIN_RATE_SIGNIN_BURST": "0"}))
	if err == nil {
		t.Fatal("empty environment accepted")
	}
	for _, name := range []string{"HD_DATABASE_URL", "HD_ADMIN_ENVIRONMENT", "HD_ADMIN_GOOGLE_DOMAIN", "HD_ADMIN_GOOGLE_AUDIENCES", "HD_ADMIN_SESSION_MAX", "HD_SHOP_WEEKLY_SLOTS", "HD_SHOP_RESET_HOUR_UTC", "HD_SHOP_REFERENCE_PRICE_DAYS", "HD_ADMIN_GRANT_APPROVAL_COINS", "HD_TEMP_BAN_HOURS", "HD_MUTE_HOURS", "HD_RANKED_BAN_HOURS", "HD_ADMIN_RATE_*"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestConfigTakesPromoCodesOnlyWithTheirKey(t *testing.T) {
	base := map[string]string{"HD_DATABASE_URL": "postgres://admin", "HD_ADMIN_ENVIRONMENT": "staging", "HD_ADMIN_GOOGLE_DOMAIN": "example.com", "HD_ADMIN_GOOGLE_AUDIENCES": "web"}
	with := func(extra map[string]string) map[string]string {
		merged := map[string]string{}
		for key, value := range base {
			merged[key] = value
		}
		for key, value := range extra {
			merged[key] = value
		}
		return merged
	}
	got, err := configFromEnv(env(with(map[string]string{"HD_PROMO_KEY": strings.Repeat("k", 32), "HD_PROMO_COINS_APPROVAL_THRESHOLD": "500"})))
	if err != nil || got.api.Promo.CoinsApprovalThreshold != 500 || got.api.Promo.CodeLength != promo.DefaultConfig().CodeLength || got.api.Cosmetics == nil {
		t.Fatalf("config %+v, %v", got.api, err)
	}
	_, err = configFromEnv(env(with(map[string]string{"HD_PROMO_KEY": "short", "HD_PROMO_CODE_LENGTH": "3", "HD_COSMETICS_MANIFEST": "missing.json"})))
	for _, name := range []string{"HD_PROMO_KEY", "HD_PROMO_*", "HD_COSMETICS_MANIFEST"} {
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("error %v does not name %s", err, name)
		}
	}
}

func TestStoreCredentialsAreOptionalButCompleteOnceNamed(t *testing.T) {
	base := func(extra map[string]string) map[string]string {
		values := map[string]string{"HD_DATABASE_URL": "postgres://admin", "HD_ADMIN_ENVIRONMENT": "staging", "HD_ADMIN_GOOGLE_DOMAIN": "example.com", "HD_ADMIN_GOOGLE_AUDIENCES": "web"}
		for key, value := range extra {
			values[key] = value
		}
		return values
	}
	without, err := configFromEnv(env(base(nil)))
	if err != nil || without.api.Stores.GooglePlay != nil || without.api.Stores.AppStore != nil {
		t.Fatalf("without credentials: %+v %v", without.api.Stores, err)
	}

	_, err = configFromEnv(env(base(map[string]string{"HD_PLAY_PACKAGE": "com.example.game", "HD_ASC_ISSUER_ID": "issuer", "HD_ASC_APP_ID": "123"})))
	for _, name := range []string{"HD_PLAY_SERVICE_ACCOUNT", "HD_ASC_PRIVATE_KEY"} {
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("error %v does not name %s", err, name)
		}
	}

	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	p8 := filepath.Join(t.TempDir(), "AuthKey.p8")
	os.WriteFile(p8, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600)
	play := googleauthtest.New(t, "scope")
	with, err := configFromEnv(env(base(map[string]string{
		"HD_PLAY_PACKAGE": "com.example.game", "HD_PLAY_SERVICE_ACCOUNT": play.File,
		"HD_ASC_ISSUER_ID": "issuer", "HD_ASC_KEY_ID": "KEY", "HD_ASC_PRIVATE_KEY": p8, "HD_ASC_APP_ID": "123",
	})))
	if err != nil || with.api.Stores.GooglePlay == nil || with.api.Stores.AppStore == nil {
		t.Fatalf("with credentials: %+v %v", with.api.Stores, err)
	}
}

func TestSentryDSNMustBeAnEURegionDSNAndIsNeverEchoed(t *testing.T) {
	base := map[string]string{"HD_DATABASE_URL": "postgres://admin", "HD_ADMIN_ENVIRONMENT": "staging", "HD_ADMIN_GOOGLE_DOMAIN": "example.com", "HD_ADMIN_GOOGLE_AUDIENCES": "web"}
	got, err := configFromEnv(env(merged(base, map[string]string{"HD_SENTRY_DSN": "https://public@o1.ingest.de.sentry.io/1", "HD_SENTRY_ENVIRONMENT": "staging"})))
	if err != nil || got.sentryDSN != "https://public@o1.ingest.de.sentry.io/1" || got.sentryEnv != "staging" {
		t.Fatalf("config %q %q err %v", got.sentryDSN, got.sentryEnv, err)
	}
	for _, dsn := range []string{"https://hidden-key@o1.ingest.us.sentry.io/1", "hidden-key %zz"} {
		_, err := configFromEnv(env(merged(base, map[string]string{"HD_SENTRY_DSN": dsn})))
		if err == nil || !strings.Contains(err.Error(), "HD_SENTRY_DSN") || strings.Contains(err.Error(), "hidden-key") {
			t.Fatalf("%s: %v", dsn, err)
		}
	}
}
