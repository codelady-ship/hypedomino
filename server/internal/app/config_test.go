package app

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/promo"
)

func lookup(values map[string]string) Lookup {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func TestConfigDefaultsWhenTheEnvironmentIsEmpty(t *testing.T) {
	config, err := ConfigFromEnv(lookup(nil))
	if err != nil {
		t.Fatal(err)
	}
	if config.ListenAddr != ":8080" || config.LogLevel != "info" || config.DrainTimeout != 15*time.Minute || config.Gateway.MaxFrameBytes != 1024 || config.Origins != nil {
		t.Fatalf("defaults %+v", config)
	}
}

func TestConfigReadsEveryVariable(t *testing.T) {
	config, err := ConfigFromEnv(lookup(map[string]string{
		"HD_LISTEN_ADDR":                       "127.0.0.1:9000",
		"HD_LOG_LEVEL":                         "DEBUG",
		"HD_ORIGINS":                           "play.example.com, admin.example.com,",
		"HD_SHUTDOWN_TIMEOUT":                  "3s",
		"HD_DRAIN_TIMEOUT":                     "2m",
		"HD_MIN_APP_VERSION":                   "1.4.0",
		"HD_CATALOG_VERSION":                   "2026.11.01-2",
		"HD_MAX_FRAME_BYTES":                   "2048",
		"HD_WRITE_QUEUE":                       "64",
		"HD_HELLO_TIMEOUT":                     "1s",
		"HD_WRITE_TIMEOUT":                     "2s",
		"HD_CLOSE_TIMEOUT":                     "500ms",
		"HD_COMPRESSION_THRESHOLD":             "512",
		"HD_INTENTS_PER_SECOND":                "20",
		"HD_INTENT_BURST":                      "40",
		"HD_PING_INTERVAL":                     "7s",
		"HD_PONG_TIMEOUT":                      "3s",
		"HD_RECONNECT_GRACE":                   "45s",
		"HD_REDIS_URL":                         "redis://cache:6379/0",
		"HD_NODE_ID":                           "server-7",
		"HD_NODE_ADDR":                         "10.0.0.7:8080",
		"HD_NODE_CAPACITY":                     "5000",
		"HD_NODE_HEARTBEAT":                    "1s",
		"HD_NODE_TTL":                          "4s",
		"HD_ROUTE_TTL":                         "3h",
		"HD_SCALE_OUT_AT":                      "0.8",
		"HD_SCALE_IN_AT":                       "0.2",
		"HD_RELAY_DIAL_TIMEOUT":                "2s",
		"HD_DATABASE_URL":                      "postgres://hd:secret@db:5432/hypedomino",
		"HD_GOOGLE_AUDIENCES":                  "1-abc.apps.googleusercontent.com, 2-def.apps.googleusercontent.com",
		"HD_APPLE_AUDIENCES":                   "com.example.hypedomino",
		"HD_NONCE_TTL":                         "4m",
		"HD_TOKEN_KEYS":                        "k2:" + testKey('b') + ",k1:" + testKey('a'),
		"HD_ACCESS_TTL":                        "5m",
		"HD_REFRESH_TTL":                       "240h",
		"HD_REFRESH_REUSE_GRACE":               "10s",
		"HD_NICKNAME_MIN_LENGTH":               "2",
		"HD_NICKNAME_MAX_LENGTH":               "12",
		"HD_NICKNAME_ALLOW_MIXED_SCRIPT":       "true",
		"HD_NICKNAME_RENAME_COOLDOWN":          "168h",
		"HD_NICKNAME_RELEASE":                  "48h",
		"HD_COSMETICS_MANIFEST":                "/etc/hypedomino/cosmetics.json",
		"HD_RARITY_ORDINALS":                   "common:1,rare:2,epic:3,legendary:4,mythic:5,cosmic:6",
		"HD_EMOJI_SLOTS":                       "8",
		"HD_ACTION_SLOTS":                      "2",
		"HD_SHOP_CATALOG":                      "/etc/hypedomino/shop.json",
		"HD_SHOP_DAILY_SLOTS":                  "5",
		"HD_SHOP_WEEKLY_SLOTS":                 "3",
		"HD_SHOP_RESET_HOUR_UTC":               "6",
		"HD_CONFIG_POLL":                       "30s",
		"HD_SEASON_POLL":                       "45s",
		"HD_DUPLICATE_COMPENSATION_PERCENT":    "25",
		"HD_PLAY_PACKAGE":                      "com.example.hypedomino",
		"HD_PLAY_SERVICE_ACCOUNT":              "/etc/hypedomino/play.json",
		"HD_PLAY_NOTIFICATION_TOKEN":           "push-secret",
		"HD_APPLE_BUNDLE_ID":                   "com.example.hypedomino",
		"HD_STORE_SANDBOX":                     "true",
		"HD_REFUNDS_BEFORE_PURCHASE_BLOCK":     "2",
		"HD_REFUND_WINDOW_DAYS":                "90",
		"HD_PROMO_KEY":                         "0123456789abcdef0123456789abcdef",
		"HD_PROMO_CODE_LENGTH":                 "10",
		"HD_PROMO_MAX_VALIDITY_DAYS":           "30",
		"HD_PROMO_ATTEMPTS_PER_HOUR":           "20",
		"HD_PROMO_FAILED_ATTEMPTS_BEFORE_LOCK": "3",
		"HD_PROMO_LOCK_MINUTES":                "15",
		"HD_PROMO_COINS_APPROVAL_THRESHOLD":    "1000",
	}))
	if err != nil {
		t.Fatal(err)
	}
	g := config.Gateway
	if config.ListenAddr != "127.0.0.1:9000" || config.LogLevel != "DEBUG" || len(config.Origins) != 2 || config.Origins[1] != "admin.example.com" {
		t.Fatalf("app config %+v", config)
	}
	if config.ShutdownTimeout != 3*time.Second || config.DrainTimeout != 2*time.Minute {
		t.Fatalf("timeouts %+v", config)
	}
	if config.Session.ReconnectGrace != 45*time.Second || g.PingInterval != 7*time.Second || g.PongTimeout != 3*time.Second {
		t.Fatalf("durations %+v %+v", config.Session, g)
	}
	if g.MinimumAppVersion != "1.4.0" || g.CatalogVersion != "2026.11.01-2" || g.MaxFrameBytes != 2048 || g.WriteQueue != 64 || g.HelloTimeout != time.Second || g.WriteTimeout != 2*time.Second || g.CloseTimeout != 500*time.Millisecond || g.CompressionThreshold != 512 || g.IntentsPerSecond != 20 || g.IntentBurst != 40 {
		t.Fatalf("gateway config %+v", g)
	}
	c := config.Cluster
	if config.Redis != "redis://cache:6379/0" || c.NodeID != "server-7" || c.Addr != "10.0.0.7:8080" || c.Capacity != 5000 || c.Heartbeat != time.Second || c.NodeTTL != 4*time.Second || c.RouteTTL != 3*time.Hour || c.ScaleOutAt != 0.8 || c.ScaleInAt != 0.2 || config.Relay.DialTimeout != 2*time.Second {
		t.Fatalf("cluster config %+v %+v", c, config.Relay)
	}
	if config.Database != "postgres://hd:secret@db:5432/hypedomino" || len(config.IDToken.Google.Audiences) != 2 || config.IDToken.Google.Audiences[1] != "2-def.apps.googleusercontent.com" || len(config.IDToken.Apple.Audiences) != 1 || config.IDToken.NonceTTL != 4*time.Minute {
		t.Fatalf("sign-in config %+v", config.IDToken)
	}
	if config.Token.AccessTTL != 5*time.Minute || config.Token.RefreshTTL != 240*time.Hour || config.Token.ReuseGrace != 10*time.Second || !strings.HasPrefix(config.Token.Keys, "k2:") {
		t.Fatalf("token config %+v", config.Token)
	}
	p := config.Profile
	if p.NicknameMinLength != 2 || p.NicknameMaxLength != 12 || !p.NicknameAllowMixedScript || p.NicknameRenameCooldown != 168*time.Hour || p.NicknameRelease != 48*time.Hour {
		t.Fatalf("profile config %+v", p)
	}
	if config.Cosmetics != "/etc/hypedomino/cosmetics.json" || config.Inventory.EmojiSlots != 8 || config.Inventory.ActionSlots != 2 {
		t.Fatalf("inventory config %q %+v", config.Cosmetics, config.Inventory)
	}
	if config.ShopCatalog != "/etc/hypedomino/shop.json" || config.Shop.DailySlots != 5 || config.Shop.WeeklySlots != 3 || config.Shop.ResetHourUTC != 6 || config.ConfigPoll != 30*time.Second || config.SeasonPoll != 45*time.Second || config.Reward.DuplicateCompensationPercent != 25 {
		t.Fatalf("shop config %q %+v %s %+v", config.ShopCatalog, config.Shop, config.ConfigPoll, config.Reward)
	}
	if config.Play.PackageName != "com.example.hypedomino" || config.Play.ServiceAccountFile != "/etc/hypedomino/play.json" || !config.Play.AllowTest || config.PushSecret != "push-secret" || config.Apple.BundleID != "com.example.hypedomino" || !config.Apple.AllowSandbox || config.Purchase.RefundsBeforePurchaseBlock != 2 || config.Purchase.RefundWindowDays != 90 {
		t.Fatalf("purchase config %+v %+v %+v", config.Play, config.Apple, config.Purchase)
	}
	if config.PromoKey != "0123456789abcdef0123456789abcdef" || config.Promo != (promo.Config{CodeLength: 10, MaxValidityDays: 30, AttemptsPerHour: 20, FailedAttemptsBeforeLock: 3, LockMinutes: 15, CoinsApprovalThreshold: 1000}) {
		t.Fatalf("promo config %q %+v", config.PromoKey, config.Promo)
	}
	if config.PromoKey != "0123456789abcdef0123456789abcdef" || config.Promo != (promo.Config{CodeLength: 10, MaxValidityDays: 30, AttemptsPerHour: 20, FailedAttemptsBeforeLock: 3, LockMinutes: 15, CoinsApprovalThreshold: 1000}) {
		t.Fatalf("promo config %q %+v", config.PromoKey, config.Promo)
	}
	if len(config.Rarities) != 6 || config.Rarities["cosmic"] != 6 {
		t.Fatalf("rarities %v", config.Rarities)
	}
}

func TestConfigReportsEveryBadValueAtOnce(t *testing.T) {
	_, err := ConfigFromEnv(lookup(map[string]string{
		"HD_LOG_LEVEL":                      "loud",
		"HD_DRAIN_TIMEOUT":                  "soon",
		"HD_WRITE_QUEUE":                    "-1",
		"HD_INTENT_BURST":                   "many",
		"HD_HELLO_TIMEOUT":                  "-5s",
		"HD_MAX_FRAME_BYTES":                "1024",
		"HD_SCALE_OUT_AT":                   "1.5",
		"HD_REDIS_URL":                      "redis://cache:6379",
		"HD_NODE_CAPACITY":                  "0",
		"HD_NICKNAME_ALLOW_MIXED_SCRIPT":    "maybe",
		"HD_NICKNAME_MAX_LENGTH":            "2",
		"HD_EMOJI_SLOTS":                    "0",
		"HD_RARITY_ORDINALS":                "common:1,rare:1",
		"HD_SHOP_RESET_HOUR_UTC":            "24",
		"HD_CONFIG_POLL":                    "100ms",
		"HD_SEASON_POLL":                    "100ms",
		"HD_DUPLICATE_COMPENSATION_PERCENT": "101",
		"HD_PLAY_PACKAGE":                   "com.example.hypedomino",
		"HD_REFUND_WINDOW_DAYS":             "0",
		"HD_PROMO_KEY":                      "short",
		"HD_PROMO_LOCK_MINUTES":             "0",
	}))
	if err == nil {
		t.Fatal("bad values were accepted")
	}
	for _, name := range []string{"HD_LOG_LEVEL", "HD_DRAIN_TIMEOUT", "HD_WRITE_QUEUE", "HD_INTENT_BURST", "HD_HELLO_TIMEOUT", "HD_SCALE_OUT_AT", "capacity", "HD_NICKNAME_ALLOW_MIXED_SCRIPT", "HD_NICKNAME_MAX_LENGTH", "HD_EMOJI_SLOTS", "HD_RARITY_ORDINALS", "HD_SHOP_RESET_HOUR_UTC", "HD_CONFIG_POLL", "HD_SEASON_POLL", "HD_DUPLICATE_COMPENSATION_PERCENT", "HD_PLAY_PACKAGE", "HD_REFUND_WINDOW_DAYS", "HD_PROMO_KEY", "HD_PROMO_*"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error does not name %s: %v", name, err)
		}
	}
	if strings.Contains(err.Error(), "HD_MAX_FRAME_BYTES") {
		t.Errorf("a valid value was reported: %v", err)
	}
}

func TestClusterConfigIsOnlyValidatedWithRedis(t *testing.T) {
	config, err := ConfigFromEnv(lookup(map[string]string{"HD_NODE_CAPACITY": "0"}))
	if err != nil {
		t.Fatalf("without HD_REDIS_URL the cluster config was validated: %v", err)
	}
	if config.Cluster.NodeID == "" {
		t.Fatal("the node id does not default to the hostname")
	}
	if _, err := ConfigFromEnv(lookup(map[string]string{"HD_REDIS_URL": "redis://cache:6379", "HD_NODE_ID": " "})); err == nil {
		t.Fatal("an empty node id was accepted with Redis configured")
	}
}

func TestADatabaseNeedsAtLeastOneSignInAudience(t *testing.T) {
	if _, err := ConfigFromEnv(lookup(map[string]string{"HD_DATABASE_URL": "postgres://db/hd"})); err == nil || !strings.Contains(err.Error(), "HD_GOOGLE_AUDIENCES") {
		t.Fatalf("a database without client ids was accepted: %v", err)
	}
	config, err := ConfigFromEnv(lookup(map[string]string{"HD_DATABASE_URL": "postgres://db/hd", "HD_APPLE_AUDIENCES": "com.example.hypedomino", "HD_TOKEN_KEYS": "k1:" + testKey('a')}))
	if err != nil || config.IDToken.Apple.Audiences[0] != "com.example.hypedomino" || config.IDToken.NonceTTL != 10*time.Minute {
		t.Fatalf("apple only: %+v %v", config.IDToken, err)
	}
}

func testKey(fill byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{fill}, 32))
}

func TestADatabaseNeedsASigningKey(t *testing.T) {
	_, err := ConfigFromEnv(lookup(map[string]string{"HD_DATABASE_URL": "postgres://db/hd", "HD_APPLE_AUDIENCES": "com.example.hypedomino"}))
	if err == nil || !strings.Contains(err.Error(), "HD_TOKEN_KEYS") {
		t.Fatalf("a database without signing keys was accepted: %v", err)
	}
	_, err = ConfigFromEnv(lookup(map[string]string{"HD_DATABASE_URL": "postgres://db/hd", "HD_APPLE_AUDIENCES": "com.example.hypedomino", "HD_TOKEN_KEYS": "k1:c2hvcnQ="}))
	if err == nil || !strings.Contains(err.Error(), "HD_TOKEN_KEYS") {
		t.Fatalf("a short signing key was accepted: %v", err)
	}
}
