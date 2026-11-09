package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/friends"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/integrity"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
	"github.com/Alievs-corp/hypedomino/server/internal/ratelimit"
	"github.com/Alievs-corp/hypedomino/server/internal/report"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/session"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type Config struct {
	ListenAddr      string
	DebugAddr       string
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
	Friends         friends.Config
	Reports         report.Config
	Emoji           room.EmojiLimit
	Shop            shop.Config
	ShopCatalog     string
	ConfigPoll      time.Duration
	SeasonPoll      time.Duration
	OutboxPoll      time.Duration
	MatchmakingTick time.Duration
	MatchmakingTTL  time.Duration
	CountryHeader   string
	CountryCooldown time.Duration
	Reward          reward.Config
	Purchase        purchase.Config
	Play            purchase.GoogleConfig
	Apple           purchase.AppleConfig
	PushSecret      string
	Promo           promo.Config
	PromoKey        string
	Risk            risk.Config
	Integrity       integrity.NonceConfig
	AppAttest       integrity.AppAttestConfig
	AppAttestRoot   string
	IntegrityPolicy integrity.Policy
	PlayIntegrity   integrity.PlayIntegrityConfig
	RateLimit       ratelimit.Config
}

func DefaultConfig() Config {
	return Config{
		ListenAddr:      ":8080",
		LogLevel:        "info",
		ShutdownTimeout: 10 * time.Second,
		DrainTimeout:    15 * time.Minute,
		Gateway:         gateway.DefaultConfig(),
		RateLimit:       ratelimit.DefaultConfig(),
		Session:         session.DefaultConfig(),
		Cluster:         defaultCluster(),
		Relay:           cluster.DefaultRelayConfig(),
		IDToken:         idtoken.DefaultConfig(),
		Token:           token.DefaultConfig(),
		Profile:         profile.DefaultConfig(),
		Inventory:       inventory.DefaultConfig(),
		Friends:         friends.DefaultConfig(),
		Reports:         report.DefaultConfig(),
		Emoji:           room.DefaultEmojiLimit(),
		Shop:            shop.DefaultConfig(),
		ConfigPoll:      10 * time.Second,
		SeasonPoll:      time.Minute,
		OutboxPoll:      2 * time.Second,
		MatchmakingTick: time.Second,
		MatchmakingTTL:  5 * time.Second,
		CountryCooldown: 180 * 24 * time.Hour,
		Reward:          reward.DefaultConfig(),
		Purchase:        purchase.DefaultConfig(),
		Promo:           promo.DefaultConfig(),
		Rarities:        cosmetics.DefaultRarities(),
		Risk:            risk.DefaultConfig(),
		Integrity:       integrity.NonceConfig{TTL: 5 * time.Minute},
		IntegrityPolicy: integrity.Policy{IntegrityFailureAction: integrity.ActionFlag, IntegrityFailureWeight: 20, DeviceRiskAction: integrity.DeviceRiskFlag, DeviceRiskWeight: 5, MinDeviceLevel: integrity.DeviceLevelDevice},
		PlayIntegrity:   integrity.PlayIntegrityConfig{TokenMaxAge: 2 * time.Minute},
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
	r.text("HD_DEBUG_ADDR", &config.DebugAddr)
	r.text("HD_LOG_LEVEL", &config.LogLevel)
	r.list("HD_ORIGINS", &config.Origins)
	r.duration("HD_SHUTDOWN_TIMEOUT", &config.ShutdownTimeout)
	r.duration("HD_DRAIN_TIMEOUT", &config.DrainTimeout)
	r.text("HD_MIN_APP_VERSION", &config.Gateway.MinimumAppVersion)
	r.text("HD_CATALOG_VERSION", &config.Gateway.CatalogVersion)
	if config.Gateway.CatalogVersion != "" && !catalogVersionPattern.MatchString(config.Gateway.CatalogVersion) {
		r.errs = append(r.errs, fmt.Errorf("HD_CATALOG_VERSION: %q is not 1 to 64 letters, digits, '.', '_' or '-' starting with a letter or digit", config.Gateway.CatalogVersion))
	}
	r.text("HD_CONTENT_BASE_URL", &config.Gateway.ContentBaseURL)
	if err := checkContentBaseURL(config.Gateway.ContentBaseURL); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HD_CONTENT_BASE_URL: %w", err))
	}
	r.integer("HD_MAX_FRAME_BYTES", &config.Gateway.MaxFrameBytes)
	r.integer("HD_MAX_HELLO_BYTES", &config.Gateway.MaxHelloBytes)
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
	r.integer("HD_FRIENDS_MAX", &config.Friends.MaxFriends)
	r.integer("HD_FRIEND_REQUESTS_MAX", &config.Friends.MaxRequests)
	r.integer("HD_REPORT_AUTO_RENAME_THRESHOLD", &config.Reports.AutoRenameThreshold)
	r.duration("HD_REPORT_AUTO_RENAME_WINDOW", &config.Reports.AutoRenameWindow)
	r.integer("HD_REPORT_DAILY_LIMIT", &config.Reports.DailyLimit)
	r.integer("HD_REPORT_REJECTED_LIMIT", &config.Reports.RejectedLimit)
	r.duration("HD_REPORT_REJECTED_WINDOW", &config.Reports.RejectedWindow)
	r.duration("HD_REPORT_REJECTED_SILENCE", &config.Reports.RejectedSilence)
	r.integer("HD_EMOJI_BURST", &config.Emoji.Burst)
	r.duration("HD_EMOJI_REFILL", &config.Emoji.Refill)
	r.text("HD_SHOP_CATALOG", &config.ShopCatalog)
	r.integer("HD_SHOP_DAILY_SLOTS", &config.Shop.DailySlots)
	r.integer("HD_SHOP_WEEKLY_SLOTS", &config.Shop.WeeklySlots)
	r.integer("HD_SHOP_RESET_HOUR_UTC", &config.Shop.ResetHourUTC)
	r.duration("HD_CONFIG_POLL", &config.ConfigPoll)
	r.duration("HD_SEASON_POLL", &config.SeasonPoll)
	r.duration("HD_OUTBOX_POLL", &config.OutboxPoll)
	r.duration("HD_MATCHMAKING_TICK", &config.MatchmakingTick)
	r.duration("HD_MATCHMAKING_LEADER_TTL", &config.MatchmakingTTL)
	r.text("HD_COUNTRY_HEADER", &config.CountryHeader)
	r.integer("HD_RATE_IP_PER_MINUTE", &config.RateLimit.IP.PerMinute)
	r.integer("HD_RATE_IP_BURST", &config.RateLimit.IP.Burst)
	r.integer("HD_RATE_ACCOUNT_PER_MINUTE", &config.RateLimit.Account.PerMinute)
	r.integer("HD_RATE_ACCOUNT_BURST", &config.RateLimit.Account.Burst)
	r.integer("HD_RATE_SIGNIN_PER_MINUTE", &config.RateLimit.SignIn.PerMinute)
	r.integer("HD_RATE_SIGNIN_BURST", &config.RateLimit.SignIn.Burst)
	r.integer("HD_RATE_SPEND_PER_MINUTE", &config.RateLimit.Spend.PerMinute)
	r.integer("HD_RATE_SPEND_BURST", &config.RateLimit.Spend.Burst)
	r.text("HD_CLIENT_IP_HEADER", &config.RateLimit.IPHeader)
	r.duration("HD_READ_TIMEOUT", &config.RateLimit.ReadTimeout)
	r.duration("HD_IDLE_TIMEOUT", &config.RateLimit.IdleTimeout)
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
	if config.OutboxPoll < 100*time.Millisecond {
		r.errs = append(r.errs, errors.New("HD_OUTBOX_POLL: at least 100ms"))
	}
	if config.MatchmakingTick < 100*time.Millisecond || config.MatchmakingTTL < 2*config.MatchmakingTick {
		r.errs = append(r.errs, errors.New("HD_MATCHMAKING_TICK and HD_MATCHMAKING_LEADER_TTL: a tick of at least 100ms and a TTL of at least two ticks"))
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
	r.duration("HD_CLOCK_SKEW_TOLERANCE", &config.Risk.ClockSkewTolerance)
	r.integer("HD_CLOCK_SKEW_WEIGHT", &config.Risk.ClockSkewWeight)
	r.integer("HD_FAKE_RECEIPT_WEIGHT", &config.Risk.FakeReceiptWeight)
	r.integer("HD_REFUND_WEIGHT", &config.Risk.RefundWeight)
	r.duration("HD_RISK_DECAY", &config.Risk.Decay)
	r.integer("HD_RISK_REVIEW_SCORE", &config.Risk.ReviewScore)
	r.integer("HD_RISK_SHADOWBAN_SCORE", &config.Risk.ShadowbanScore)
	r.duration("HD_BAN_WAVE_INTERVAL", &config.Risk.BanWaveInterval)
	r.integer("HD_REPLAY_REJECT_THRESHOLD", &config.Risk.ReplayThreshold)
	r.duration("HD_REPLAY_REJECT_WINDOW", &config.Risk.ReplayWindow)
	r.integer("HD_REPLAY_REJECT_WEIGHT", &config.Risk.ReplayWeight)
	r.integer("HD_MAC_FAILURE_THRESHOLD", &config.Risk.MacFailureThreshold)
	r.duration("HD_MAC_FAILURE_WINDOW", &config.Risk.MacFailureWindow)
	r.integer("HD_MAC_FAILURE_WEIGHT", &config.Risk.MacFailureWeight)
	r.integer("HD_UNOWNED_EQUIP_THRESHOLD", &config.Risk.UnownedEquipThreshold)
	r.duration("HD_UNOWNED_EQUIP_WINDOW", &config.Risk.UnownedEquipWindow)
	r.integer("HD_UNOWNED_EQUIP_WEIGHT", &config.Risk.UnownedEquipWeight)
	r.integer("HD_COLLUSION_WEIGHT", &config.Risk.CollusionWeight)
	if err := config.Risk.Validate(); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HD_CLOCK_SKEW_TOLERANCE, HD_CLOCK_SKEW_WEIGHT, HD_FAKE_RECEIPT_WEIGHT, HD_REFUND_WEIGHT, HD_REPLAY_REJECT_* and HD_MAC_FAILURE_*: %w", err))
	}
	if err := config.Purchase.Validate(); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HD_REFUNDS_BEFORE_PURCHASE_BLOCK and HD_REFUND_WINDOW_DAYS: %w", err))
	}
	r.duration("HD_INTEGRITY_NONCE_TTL", &config.Integrity.TTL)
	if config.Integrity.TTL <= 0 {
		r.errs = append(r.errs, errors.New("HD_INTEGRITY_NONCE_TTL: must be positive"))
	}
	r.text("HD_APP_ATTEST_TEAM_ID", &config.AppAttest.TeamID)
	r.text("HD_APP_ATTEST_ROOT_CA", &config.AppAttestRoot)
	r.flag("HD_APP_ATTEST_DEVELOPMENT", &config.AppAttest.AllowDevelopment)
	config.AppAttest.BundleID = config.Apple.BundleID
	if config.AppAttest.TeamID != "" && (config.AppAttestRoot == "" || config.AppAttest.BundleID == "") {
		r.errs = append(r.errs, errors.New("HD_APP_ATTEST_TEAM_ID: needs HD_APP_ATTEST_ROOT_CA and HD_APPLE_BUNDLE_ID"))
	}
	r.integrity(&config)
	r.text("HD_PROMO_KEY", &config.PromoKey)
	r.integer("HD_PROMO_CODE_LENGTH", &config.Promo.CodeLength)
	r.integer("HD_PROMO_MAX_VALIDITY_DAYS", &config.Promo.MaxValidityDays)
	r.integer("HD_PROMO_ATTEMPTS_PER_HOUR", &config.Promo.AttemptsPerHour)
	r.integer("HD_PROMO_FAILED_ATTEMPTS_BEFORE_LOCK", &config.Promo.FailedAttemptsBeforeLock)
	r.integer("HD_PROMO_LOCK_MINUTES", &config.Promo.LockMinutes)
	r.integer("HD_PROMO_COINS_APPROVAL_THRESHOLD", &config.Promo.CoinsApprovalThreshold)
	r.integer("HD_PROMO_LOCK_SIGNAL_WEIGHT", &config.Promo.LockSignalWeight)
	r.integer("HD_PROMO_SPIKE_PERCENT", &config.Promo.SpikePercent)
	r.integer("HD_PROMO_SPIKE_WINDOW_MINUTES", &config.Promo.SpikeWindowMinutes)
	if config.PromoKey != "" {
		if _, err := promo.NewHasher(config.PromoKey); err != nil {
			r.errs = append(r.errs, fmt.Errorf("HD_PROMO_KEY: %w", err))
		}
	}
	if err := config.Promo.Validate(); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HD_PROMO_*: %w", err))
	}
	if err := config.Reports.Validate(); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HD_REPORT_*: %w", err))
	}
	if err := config.Friends.Validate(); err != nil {
		r.errs = append(r.errs, fmt.Errorf("HD_FRIENDS_MAX and HD_FRIEND_REQUESTS_MAX: %w", err))
	}
	if config.Inventory.EmojiSlots < 1 || config.Inventory.ActionSlots < 1 {
		r.errs = append(r.errs, errors.New("HD_EMOJI_SLOTS and HD_ACTION_SLOTS: at least 1"))
	}
	if config.Emoji.Burst < 1 || config.Emoji.Refill <= 0 {
		r.errs = append(r.errs, errors.New("HD_EMOJI_BURST and HD_EMOJI_REFILL: a burst of at least 1 and a positive refill"))
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
	if err := config.RateLimit.Validate(); err != nil {
		r.errs = append(r.errs, err)
	}
	if _, ok := parseLevel(config.LogLevel); !ok {
		r.errs = append(r.errs, fmt.Errorf("HD_LOG_LEVEL: %q is not debug, info, warn or error", config.LogLevel))
	}
	return config, errors.Join(r.errs...)
}

var catalogVersionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func checkContentBaseURL(raw string) error {
	if raw == "" {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.HasSuffix(raw, "/") || strings.Contains(raw, "#") {
		return fmt.Errorf("%q is not an https URL without a trailing slash, query or fragment", raw)
	}
	return nil
}

func (r *envReader) integrity(config *Config) {
	policy := &config.IntegrityPolicy
	r.choice("HD_INTEGRITY_FAILURE_ACTION", func(name string) bool {
		action, ok := integrity.ParseAction(name)
		policy.IntegrityFailureAction = action
		return ok
	})
	r.choice("HD_DEVICE_RISK_ACTION", func(name string) bool {
		action, ok := integrity.ParseDeviceRiskAction(name)
		policy.DeviceRiskAction = action
		return ok
	})
	r.choice("HD_INTEGRITY_MIN_DEVICE_LEVEL", func(name string) bool {
		level, ok := integrity.ParseDeviceLevel(name)
		policy.MinDeviceLevel = level
		return ok
	})
	r.flag("HD_INTEGRITY_REQUIRE_LICENSED", &policy.RequireLicensed)
	r.integer("HD_INTEGRITY_FAILURE_WEIGHT", &policy.IntegrityFailureWeight)
	r.integer("HD_DEVICE_RISK_WEIGHT", &policy.DeviceRiskWeight)
	if policy.IntegrityFailureWeight < 0 || policy.DeviceRiskWeight < 0 {
		r.errs = append(r.errs, errors.New("HD_INTEGRITY_FAILURE_WEIGHT and HD_DEVICE_RISK_WEIGHT: must not be negative"))
	}
	config.PlayIntegrity.PackageName = config.Play.PackageName
	r.list("HD_PLAY_CERT_DIGESTS", &config.PlayIntegrity.CertificateDigests)
	r.duration("HD_PLAY_INTEGRITY_MAX_AGE", &config.PlayIntegrity.TokenMaxAge)
	if len(config.PlayIntegrity.CertificateDigests) > 0 && config.Play.ServiceAccountFile == "" {
		r.errs = append(r.errs, errors.New("HD_PLAY_CERT_DIGESTS: needs HD_PLAY_PACKAGE and HD_PLAY_SERVICE_ACCOUNT"))
	}
}

func (r *envReader) choice(name string, parse func(string) bool) {
	raw, ok := r.lookup(name)
	if ok && !parse(raw) {
		r.errs = append(r.errs, fmt.Errorf("%s: %q is not a known value", name, raw))
	}
}
