package app

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/session"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type Config struct {
	ListenAddr      string
	LogLevel        string
	Origins         []string
	ShutdownTimeout time.Duration
	DrainTimeout    time.Duration
	Gateway         gateway.Config
	Session         session.Config
	Redis           string
	Cluster         cluster.Config
	Relay           cluster.RelayConfig
	Database        string
	IDToken         idtoken.Config
	Token           token.Config
	Profile         profile.Config
	Cosmetics       string
	Rarities        cosmetics.Rarities
	Inventory       inventory.Config
	Shop            shop.Config
	ShopCatalog     string
	ConfigPoll      time.Duration
	SeasonPoll      time.Duration
	CountryHeader   string
	CountryCooldown time.Duration
	Reward          reward.Config
	Purchase        purchase.Config
	Play            purchase.GoogleConfig
	Apple           purchase.AppleConfig
	PushSecret      string
	Promo           promo.Config
	PromoKey        string
}

func DefaultConfig() Config {
	return Config{
		ListenAddr:      ":8080",
		LogLevel:        "info",
		ShutdownTimeout: 10 * time.Second,
		DrainTimeout:    15 * time.Minute,
		Gateway:         gateway.DefaultConfig(),
		Session:         session.DefaultConfig(),
		Cluster:         defaultCluster(),
		Relay:           cluster.DefaultRelayConfig(),
		IDToken:         idtoken.DefaultConfig(),
		Token:           token.DefaultConfig(),
		Profile:         profile.DefaultConfig(),
		Inventory:       inventory.DefaultConfig(),
		Shop:            shop.DefaultConfig(),
		ConfigPoll:      10 * time.Second,
		SeasonPoll:      time.Minute,
		CountryCooldown: 180 * 24 * time.Hour,
		Reward:          reward.DefaultConfig(),
		Purchase:        purchase.DefaultConfig(),
		Promo:           promo.DefaultConfig(),
		Rarities:        cosmetics.DefaultRarities(),
	}
}

func defaultCluster() cluster.Config {
	config := cluster.DefaultConfig()
	config.NodeID, _ = os.Hostname()
	return config
}

type Lookup func(name string) (string, bool)

type envReader struct {
	lookup Lookup
	errs   []error
}

func (r *envReader) text(name string, value *string) {
	if raw, ok := r.lookup(name); ok {
		*value = raw
	}
}

func (r *envReader) list(name string, value *[]string) {
	raw, ok := r.lookup(name)
	if !ok {
		return
	}
	*value = nil
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			*value = append(*value, item)
		}
	}
}

func (r *envReader) integer(name string, value *int) {
	raw, ok := r.lookup(name)
	if !ok {
		return
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		r.errs = append(r.errs, fmt.Errorf("%s: %q is not a non-negative integer", name, raw))
		return
	}
	*value = parsed
}

func (r *envReader) fraction(name string, value *float64) {
	raw, ok := r.lookup(name)
	if !ok {
		return
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil || parsed < 0 || parsed > 1 {
		r.errs = append(r.errs, fmt.Errorf("%s: %q is not a fraction between 0 and 1", name, raw))
		return
	}
	*value = parsed
}

func (r *envReader) flag(name string, value *bool) {
	raw, ok := r.lookup(name)
	if !ok {
		return
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("%s: %q is not true or false", name, raw))
		return
	}
	*value = parsed
}

func (r *envReader) duration(name string, value *time.Duration) {
	raw, ok := r.lookup(name)
	if !ok {
		return
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed < 0 {
		r.errs = append(r.errs, fmt.Errorf("%s: %q is not a non-negative duration", name, raw))
		return
	}
	*value = parsed
}

func ConfigFromEnv(lookup Lookup) (Config, error) {
	config := DefaultConfig()
	r := &envReader{lookup: lookup}
	r.text("HD_LISTEN_ADDR", &config.ListenAddr)
	r.text("HD_LOG_LEVEL", &config.LogLevel)
	r.list("HD_ORIGINS", &config.Origins)
	r.duration("HD_SHUTDOWN_TIMEOUT", &config.ShutdownTimeout)
	r.duration("HD_DRAIN_TIMEOUT", &config.DrainTimeout)
	r.text("HD_MIN_APP_VERSION", &config.Gateway.MinimumAppVersion)
	r.text("HD_CATALOG_VERSION", &config.Gateway.CatalogVersion)
	r.integer("HD_MAX_FRAME_BYTES", &config.Gateway.MaxFrameBytes)
	r.integer("HD_WRITE_QUEUE", &config.Gateway.WriteQueue)
	r.duration("HD_HELLO_TIMEOUT", &config.Gateway.HelloTimeout)
	r.duration("HD_WRITE_TIMEOUT", &config.Gateway.WriteTimeout)
	r.duration("HD_CLOSE_TIMEOUT", &config.Gateway.CloseTimeout)
	r.integer("HD_COMPRESSION_THRESHOLD", &config.Gateway.CompressionThreshold)
	r.integer("HD_INTENTS_PER_SECOND", &config.Gateway.IntentsPerSecond)
	r.integer("HD_INTENT_BURST", &config.Gateway.IntentBurst)
	r.duration("HD_PING_INTERVAL", &config.Gateway.PingInterval)
	r.duration("HD_PONG_TIMEOUT", &config.Gateway.PongTimeout)
	r.duration("HD_RECONNECT_GRACE", &config.Session.ReconnectGrace)
	r.text("HD_REDIS_URL", &config.Redis)
	r.text("HD_NODE_ID", &config.Cluster.NodeID)
	r.text("HD_NODE_ADDR", &config.Cluster.Addr)
	r.integer("HD_NODE_CAPACITY", &config.Cluster.Capacity)
	r.duration("HD_NODE_HEARTBEAT", &config.Cluster.Heartbeat)
	r.duration("HD_NODE_TTL", &config.Cluster.NodeTTL)
	r.duration("HD_ROUTE_TTL", &config.Cluster.RouteTTL)
	r.fraction("HD_SCALE_OUT_AT", &config.Cluster.ScaleOutAt)
	r.fraction("HD_SCALE_IN_AT", &config.Cluster.ScaleInAt)
	r.duration("HD_RELAY_DIAL_TIMEOUT", &config.Relay.DialTimeout)
	r.text("HD_DATABASE_URL", &config.Database)
	r.list("HD_GOOGLE_AUDIENCES", &config.IDToken.Google.Audiences)
	r.list("HD_APPLE_AUDIENCES", &config.IDToken.Apple.Audiences)
	r.duration("HD_NONCE_TTL", &config.IDToken.NonceTTL)
	r.text("HD_TOKEN_KEYS", &config.Token.Keys)
	r.duration("HD_ACCESS_TTL", &config.Token.AccessTTL)
	r.duration("HD_REFRESH_TTL", &config.Token.RefreshTTL)
	r.duration("HD_REFRESH_REUSE_GRACE", &config.Token.ReuseGrace)
	r.integer("HD_NICKNAME_MIN_LENGTH", &config.Profile.NicknameMinLength)
	r.integer("HD_NICKNAME_MAX_LENGTH", &config.Profile.NicknameMaxLength)
	r.flag("HD_NICKNAME_ALLOW_MIXED_SCRIPT", &config.Profile.NicknameAllowMixedScript)
	r.duration("HD_NICKNAME_RENAME_COOLDOWN", &config.Profile.NicknameRenameCooldown)
	r.duration("HD_NICKNAME_RELEASE", &config.Profile.NicknameRelease)
	r.text("HD_COSMETICS_MANIFEST", &config.Cosmetics)
	if raw, ok := lookup("HD_RARITY_ORDINALS"); ok {
		parsed, err := cosmetics.ParseRarities(raw)
		if err != nil {
			r.errs = append(r.errs, fmt.Errorf("HD_RARITY_ORDINALS: %w", err))
		}
		config.Rarities = parsed
	}
	r.integer("HD_EMOJI_SLOTS", &config.Inventory.EmojiSlots)
	r.integer("HD_ACTION_SLOTS", &config.Inventory.ActionSlots)
	r.text("HD_SHOP_CATALOG", &config.ShopCatalog)
	r.integer("HD_SHOP_DAILY_SLOTS", &config.Shop.DailySlots)
	r.integer("HD_SHOP_WEEKLY_SLOTS", &config.Shop.WeeklySlots)
	r.integer("HD_SHOP_RESET_HOUR_UTC", &config.Shop.ResetHourUTC)
	r.duration("HD_CONFIG_POLL", &config.ConfigPoll)
	r.duration("HD_SEASON_POLL", &config.SeasonPoll)
	r.text("HD_COUNTRY_HEADER", &config.CountryHeader)
	r.duration("HD_COUNTRY_CHANGE_COOLDOWN", &config.CountryCooldown)
	r.integer("HD_DUPLICATE_COMPENSATION_PERCENT", &config.Reward.DuplicateCompensationPercent)
	if err := config.Shop.Validate(); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HD_SHOP_DAILY_SLOTS, HD_SHOP_WEEKLY_SLOTS and HD_SHOP_RESET_HOUR_UTC: %w", err))
	}
	if config.ConfigPoll < time.Second {
		r.errs = append(r.errs, errors.New("HD_CONFIG_POLL: at least 1s"))
	}
	if config.SeasonPoll < time.Second {
		r.errs = append(r.errs, errors.New("HD_SEASON_POLL: at least 1s"))
	}
	if config.CountryCooldown < 0 {
		r.errs = append(r.errs, errors.New("HD_COUNTRY_CHANGE_COOLDOWN: not negative"))
	}
	if config.Reward.DuplicateCompensationPercent < 0 || config.Reward.DuplicateCompensationPercent > 100 {
		r.errs = append(r.errs, errors.New("HD_DUPLICATE_COMPENSATION_PERCENT: between 0 and 100"))
	}
	r.text("HD_PLAY_PACKAGE", &config.Play.PackageName)
	r.text("HD_PLAY_SERVICE_ACCOUNT", &config.Play.ServiceAccountFile)
	r.text("HD_PLAY_NOTIFICATION_TOKEN", &config.PushSecret)
	r.text("HD_APPLE_BUNDLE_ID", &config.Apple.BundleID)
	r.flag("HD_STORE_SANDBOX", &config.Play.AllowTest)
	config.Apple.AllowSandbox = config.Play.AllowTest
	r.integer("HD_REFUNDS_BEFORE_PURCHASE_BLOCK", &config.Purchase.RefundsBeforePurchaseBlock)
	r.integer("HD_REFUND_WINDOW_DAYS", &config.Purchase.RefundWindowDays)
	if (config.Play.PackageName == "") != (config.Play.ServiceAccountFile == "") {
		r.errs = append(r.errs, errors.New("HD_PLAY_PACKAGE and HD_PLAY_SERVICE_ACCOUNT: both or neither"))
	}
	if config.Play.PackageName != "" && config.PushSecret == "" {
		r.errs = append(r.errs, errors.New("HD_PLAY_NOTIFICATION_TOKEN: required with HD_PLAY_PACKAGE"))
	}
	if err := config.Purchase.Validate(); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HD_REFUNDS_BEFORE_PURCHASE_BLOCK and HD_REFUND_WINDOW_DAYS: %w", err))
	}
	r.text("HD_PROMO_KEY", &config.PromoKey)
	r.integer("HD_PROMO_CODE_LENGTH", &config.Promo.CodeLength)
	r.integer("HD_PROMO_MAX_VALIDITY_DAYS", &config.Promo.MaxValidityDays)
	r.integer("HD_PROMO_ATTEMPTS_PER_HOUR", &config.Promo.AttemptsPerHour)
	r.integer("HD_PROMO_FAILED_ATTEMPTS_BEFORE_LOCK", &config.Promo.FailedAttemptsBeforeLock)
	r.integer("HD_PROMO_LOCK_MINUTES", &config.Promo.LockMinutes)
	r.integer("HD_PROMO_COINS_APPROVAL_THRESHOLD", &config.Promo.CoinsApprovalThreshold)
	if config.PromoKey != "" {
		if _, err := promo.NewHasher(config.PromoKey); err != nil {
			r.errs = append(r.errs, fmt.Errorf("HD_PROMO_KEY: %w", err))
		}
	}
	if err := config.Promo.Validate(); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HD_PROMO_*: %w", err))
	}
	if config.Inventory.EmojiSlots < 1 || config.Inventory.ActionSlots < 1 {
		r.errs = append(r.errs, errors.New("HD_EMOJI_SLOTS and HD_ACTION_SLOTS: at least 1"))
	}
	if config.Profile.NicknameMinLength < 1 || config.Profile.NicknameMaxLength < config.Profile.NicknameMinLength {
		r.errs = append(r.errs, errors.New("HD_NICKNAME_MIN_LENGTH and HD_NICKNAME_MAX_LENGTH: the minimum must be at least 1 and not above the maximum"))
	}
	if config.Database != "" {
		if _, err := token.ParseKeys(config.Token.Keys); err != nil {
			r.errs = append(r.errs, fmt.Errorf("HD_TOKEN_KEYS: %w", err))
		}
	}
	if config.Database != "" && len(config.IDToken.Google.Audiences)+len(config.IDToken.Apple.Audiences) == 0 {
		r.errs = append(r.errs, errors.New("HD_GOOGLE_AUDIENCES or HD_APPLE_AUDIENCES: at least one client id is required with HD_DATABASE_URL"))
	}
	if config.Redis != "" {
		if err := config.Cluster.Validate(); err != nil {
			r.errs = append(r.errs, err)
		}
	}
	if _, ok := parseLevel(config.LogLevel); !ok {
		r.errs = append(r.errs, fmt.Errorf("HD_LOG_LEVEL: %q is not debug, info, warn or error", config.LogLevel))
	}
	return config, errors.Join(r.errs...)
}
