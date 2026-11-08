package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/adminaccount"
	"github.com/Alievs-corp/hypedomino/server/internal/adminsession"
	"github.com/Alievs-corp/hypedomino/server/internal/approval"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/dailylogin"
	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/livecontrol"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/medal"
	"github.com/Alievs-corp/hypedomino/server/internal/mission"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/quickmessage"
	"github.com/Alievs-corp/hypedomino/server/internal/ratelimit"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/report"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
	"github.com/Alievs-corp/hypedomino/server/internal/rulesconfig"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	MaxBodyBytes     = 16 * 1024
	ViewReplayAction = "view_replay"
)

type Config struct {
	Environment  string
	GoogleDomain string
	Session      adminsession.Config
	ApprovalTTL  time.Duration
	Shop         shop.Config

	GrantApprovalCoins int64
	TempBan            time.Duration
	Mute               time.Duration
	Limits             Limits

	PromoKey  string
	Promo     promo.Config
	Cosmetics *cosmetics.Catalog
	Assets    AssetStore
}

type Verifier interface {
	IssueNonce(ctx context.Context) (string, error)
	Verify(ctx context.Context, provider idtoken.Provider, token, rawNonce string) (idtoken.Identity, error)
}

type Handler struct {
	config    Config
	verifier  Verifier
	buckets   ratelimit.Buckets
	pool      *pgxpool.Pool
	accounts  *adminaccount.Store
	sessions  *adminsession.Store
	audit     *audit.Store
	strings   *gamestrings.Store
	approvals *approval.Store
	wallet    *wallet.Store
	promos    *promo.Store
	now       func() time.Time
	logger    *slog.Logger
	mux       *http.ServeMux
	hide      func(ctx context.Context, account string) error
}

func New(config Config, verifier Verifier, buckets ratelimit.Buckets, pool *pgxpool.Pool, now func() time.Time, logger *slog.Logger) *Handler {
	h := &Handler{
		config:    config,
		verifier:  verifier,
		buckets:   buckets,
		pool:      pool,
		accounts:  adminaccount.NewStore(pool),
		sessions:  adminsession.NewStore(pool, config.Session),
		audit:     audit.NewStore(pool),
		strings:   gamestrings.NewStore(pool),
		approvals: approval.NewStore(pool, approval.Config{TTL: config.ApprovalTTL}),
		wallet:    wallet.NewStore(pool),
		now:       now,
		logger:    logger,
		mux:       http.NewServeMux(),
	}
	if hasher, err := promo.NewHasher(config.PromoKey); err == nil && config.Cosmetics != nil {
		h.promos = promo.NewStore(pool, nil, config.Cosmetics, hasher, config.Promo)
	}
	h.mux.HandleFunc("POST /admin/nonce", h.signingIn(h.nonce))
	h.mux.HandleFunc("POST /admin/session", h.signingIn(h.signIn))
	h.mux.HandleFunc("POST /admin/session/step-up", h.signedIn(h.stepUp))
	h.mux.HandleFunc("DELETE /admin/session", h.signedIn(h.signOut))
	h.mux.HandleFunc("GET /admin/me", h.signedIn(h.me))
	h.mux.HandleFunc("PUT /admin/me/locale", h.signedIn(h.chooseLocale))
	h.mux.HandleFunc("GET /admin/audit", h.allowed(rbac.AuditLog, rbac.View, h.searchAudit))
	h.mux.HandleFunc("GET /admin/accounts", h.allowed(rbac.AdminAccounts, rbac.View, h.listAccounts))
	h.mux.HandleFunc("POST /admin/accounts", h.allowed(rbac.AdminAccounts, rbac.Create, h.createAccount))
	h.mux.HandleFunc("PUT /admin/accounts/{id}/roles", h.allowed(rbac.AdminAccounts, rbac.Edit, h.setRoles))
	h.mux.HandleFunc("POST /admin/accounts/{id}/disable", h.allowed(rbac.AdminAccounts, rbac.Retire, h.disable))
	h.mux.HandleFunc("GET /admin/approvals", h.signedIn(h.pendingApprovals))
	h.mux.HandleFunc("POST /admin/approvals/{id}/approve", h.signedIn(h.approve))
	h.mux.HandleFunc("POST /admin/approvals/{id}/reject", h.signedIn(h.reject))
	h.mux.HandleFunc("GET /admin/config/rules", h.allowed(rbac.GameConfig, rbac.View, h.rules))
	h.mux.HandleFunc("PUT /admin/config/rules", h.allowed(rbac.GameConfig, rbac.Publish, h.publishRules))
	h.mux.HandleFunc("GET /admin/config/rules/history", h.allowed(rbac.GameConfig, rbac.View, h.rulesHistory))
	h.mux.HandleFunc("POST /admin/config/rules/rollback", h.allowed(rbac.GameConfig, rbac.Publish, h.rollbackRules))
	h.mux.HandleFunc("GET /admin/config/matchmaking", h.allowed(rbac.GameConfig, rbac.View, h.domainCatalog(matchmaking.Domain)))
	h.mux.HandleFunc("PUT /admin/config/matchmaking", h.allowed(rbac.GameConfig, rbac.Publish, h.publishDomain(matchmakingDomain)))
	h.mux.HandleFunc("GET /admin/config/matchmaking/history", h.allowed(rbac.GameConfig, rbac.View, h.domainHistory(matchmaking.Domain)))
	h.mux.HandleFunc("POST /admin/config/matchmaking/rollback", h.allowed(rbac.GameConfig, rbac.Publish, h.rollbackDomain(matchmakingDomain)))
	h.mux.HandleFunc("GET /admin/assets", h.allowed(rbac.Assets, rbac.View, h.listAssets))
	h.mux.HandleFunc("POST /admin/assets/uploads", h.allowed(rbac.Assets, rbac.Create, h.startAssetUpload))
	h.mux.HandleFunc("POST /admin/assets/{id}/check", h.allowed(rbac.Assets, rbac.Create, h.checkAsset))
	h.mux.HandleFunc("GET /admin/players", h.allowed(rbac.Players, rbac.View, h.searchPlayers))
	h.mux.HandleFunc("GET /admin/players/{id}", h.allowed(rbac.Players, rbac.View, h.playerRecord))
	h.mux.HandleFunc("POST /admin/players/{id}/coins", h.allowed(rbac.Players, rbac.ActOnPlayer, h.adjustCoins))
	h.mux.HandleFunc("POST /admin/players/{id}/temporary-ban", h.allowed(rbac.Players, rbac.ActOnPlayer, h.banTemporarily))
	h.mux.HandleFunc("DELETE /admin/players/{id}/temporary-ban", h.allowed(rbac.Players, rbac.ActOnPlayer, h.liftTemporaryBan))
	h.mux.HandleFunc("POST /admin/players/{id}/shadowban", h.allowed(rbac.Players, rbac.ActOnPlayer, h.shadowban))
	h.mux.HandleFunc("DELETE /admin/players/{id}/shadowban", h.allowed(rbac.Players, rbac.ActOnPlayer, h.liftShadowban))
	h.mux.HandleFunc("POST /admin/players/{id}/mute", h.allowed(rbac.Players, rbac.ActOnPlayer, h.mute))
	h.mux.HandleFunc("DELETE /admin/players/{id}/mute", h.allowed(rbac.Players, rbac.ActOnPlayer, h.unmute))
	h.mux.HandleFunc("POST /admin/players/{id}/force-rename", h.allowed(rbac.Players, rbac.ActOnPlayer, h.forceRename))
	h.mux.HandleFunc("POST /admin/players/{id}/grant", h.allowed(rbac.Players, rbac.ActOnPlayer, h.grantReward))
	h.mux.HandleFunc("GET /admin/matches/{id}/replay", h.allowed(rbac.Players, rbac.View, h.matchReplay))
	h.mux.HandleFunc("GET /admin/promo-codes", h.allowed(rbac.PromoCodes, rbac.View, h.listPromos))
	h.mux.HandleFunc("POST /admin/promo-codes", h.allowed(rbac.PromoCodes, rbac.Create, h.createPromo))
	h.mux.HandleFunc("POST /admin/promo-codes/batch", h.allowed(rbac.PromoCodes, rbac.Create, h.createPromoBatch))
	h.mux.HandleFunc("GET /admin/promo-codes/{id}", h.allowed(rbac.PromoCodes, rbac.View, h.promoDetail))
	h.mux.HandleFunc("PUT /admin/promo-codes/{id}/accounts", h.allowed(rbac.PromoCodes, rbac.EditTargetedAccounts, h.setPromoTargets))
	h.mux.HandleFunc("POST /admin/promo-codes/{id}/disable", h.allowed(rbac.PromoCodes, rbac.DisablePromo, h.disablePromo))
	h.mux.HandleFunc("POST /admin/promo-batches/{id}/disable", h.allowed(rbac.PromoCodes, rbac.DisablePromo, h.disablePromoBatch))
	h.mux.HandleFunc("GET /admin/promo-templates", h.allowed(rbac.PromoCodes, rbac.View, h.listPromoTemplates))
	h.mux.HandleFunc("POST /admin/promo-templates", h.allowed(rbac.PromoCodes, rbac.Create, h.createPromoTemplate))
	h.mux.HandleFunc("PUT /admin/promo-templates/{id}", h.allowed(rbac.PromoCodes, rbac.Edit, h.editPromoTemplate))
	h.mux.HandleFunc("GET /admin/cosmetics/items", h.allowed(rbac.CosmeticItems, rbac.View, h.cosmeticItems))
	h.mux.HandleFunc("GET /admin/reward-bundles", h.allowed(rbac.RewardBundles, rbac.View, h.listRewardBundles))
	h.mux.HandleFunc("PUT /admin/reward-bundles/{id}", h.allowed(rbac.RewardBundles, rbac.Publish, h.editRewardBundle))
	h.mux.HandleFunc("GET /admin/config/shop", h.allowed(rbac.ShopOffers, rbac.View, h.shopCatalog))
	h.mux.HandleFunc("PUT /admin/config/shop", h.allowed(rbac.ShopOffers, rbac.Publish, h.publishShop))
	h.mux.HandleFunc("GET /admin/config/shop/history", h.allowed(rbac.ShopOffers, rbac.View, h.domainHistory(shop.Domain)))
	h.mux.HandleFunc("POST /admin/config/shop/rollback", h.allowed(rbac.ShopOffers, rbac.Publish, h.rollbackShop))
	h.mux.HandleFunc("GET /admin/config/cosmetics", h.allowed(rbac.CosmeticItems, rbac.View, h.cosmeticCatalog))
	h.mux.HandleFunc("PUT /admin/config/cosmetics", h.allowed(rbac.CosmeticItems, rbac.Publish, h.publishDomain(func() configDomain { return h.cosmeticsDomain(false) })))
	h.mux.HandleFunc("GET /admin/config/cosmetics/history", h.allowed(rbac.CosmeticItems, rbac.View, h.domainHistory(cosmetics.Domain)))
	h.mux.HandleFunc("POST /admin/config/cosmetics/rollback", h.allowed(rbac.CosmeticItems, rbac.Publish, h.rollbackDomain(func() configDomain { return h.cosmeticsDomain(true) })))
	h.mux.HandleFunc("GET /admin/config/missions", h.allowed(rbac.Missions, rbac.View, h.missionCatalog))
	h.mux.HandleFunc("PUT /admin/config/missions", h.allowed(rbac.Missions, rbac.Publish, h.publishDomain(func() configDomain { return h.missionDomain(false) })))
	h.mux.HandleFunc("GET /admin/config/missions/history", h.allowed(rbac.Missions, rbac.View, h.domainHistory(mission.Domain)))
	h.mux.HandleFunc("POST /admin/config/missions/rollback", h.allowed(rbac.Missions, rbac.Publish, h.rollbackDomain(func() configDomain { return h.missionDomain(true) })))
	h.mux.HandleFunc("GET /admin/config/daily-login", h.allowed(rbac.Missions, rbac.View, h.dailyLoginCalendar))
	h.mux.HandleFunc("PUT /admin/config/daily-login", h.allowed(rbac.Missions, rbac.Publish, h.publishDomain(dailyLoginDomain)))
	h.mux.HandleFunc("GET /admin/config/daily-login/history", h.allowed(rbac.Missions, rbac.View, h.domainHistory(dailylogin.Domain)))
	h.mux.HandleFunc("POST /admin/config/daily-login/rollback", h.allowed(rbac.Missions, rbac.Publish, h.rollbackDomain(dailyLoginDomain)))
	h.mux.HandleFunc("GET /admin/config/rating", h.allowed(rbac.RankedSeasons, rbac.View, h.ratingConfig))
	h.mux.HandleFunc("PUT /admin/config/rating", h.allowed(rbac.RankedSeasons, rbac.Publish, h.publishDomain(h.ratingDomain)))
	h.mux.HandleFunc("GET /admin/config/rating/history", h.allowed(rbac.RankedSeasons, rbac.View, h.domainHistory(rating.Domain)))
	h.mux.HandleFunc("POST /admin/config/rating/rollback", h.allowed(rbac.RankedSeasons, rbac.Publish, h.rollbackDomain(h.ratingDomain)))
	h.mux.HandleFunc("GET /admin/ranked/seasons", h.allowed(rbac.RankedSeasons, rbac.View, h.rankedSeasons))
	h.mux.HandleFunc("POST /admin/ranked/seasons", h.allowed(rbac.RankedSeasons, rbac.Create, h.createSeason))
	h.mux.HandleFunc("PATCH /admin/ranked/seasons/{id}", h.allowed(rbac.RankedSeasons, rbac.Edit, h.rescheduleSeason))
	h.mux.HandleFunc("POST /admin/ranked/seasons/{id}/extend", h.allowed(rbac.RankedSeasons, rbac.Edit, h.extendSeason))
	h.mux.HandleFunc("DELETE /admin/ranked/seasons/{id}", h.allowed(rbac.RankedSeasons, rbac.Delete, h.deleteSeason))
	h.mux.HandleFunc("GET /admin/config/pass", h.allowed(rbac.PassSeasons, rbac.View, h.passConfig))
	h.mux.HandleFunc("PUT /admin/config/pass", h.allowed(rbac.PassSeasons, rbac.Publish, h.publishDomain(func() configDomain { return h.passDomain(false) })))
	h.mux.HandleFunc("GET /admin/config/pass/history", h.allowed(rbac.PassSeasons, rbac.View, h.domainHistory(pass.Domain)))
	h.mux.HandleFunc("POST /admin/config/pass/rollback", h.allowed(rbac.PassSeasons, rbac.Publish, h.rollbackDomain(func() configDomain { return h.passDomain(true) })))
	h.mux.HandleFunc("GET /admin/config/medals", h.allowed(rbac.Medals, rbac.View, h.medalCatalog))
	h.mux.HandleFunc("PUT /admin/config/medals", h.allowed(rbac.Medals, rbac.Publish, h.publishDomain(func() configDomain { return h.medalDomain(false) })))
	h.mux.HandleFunc("GET /admin/config/medals/history", h.allowed(rbac.Medals, rbac.View, h.domainHistory(medal.Domain)))
	h.mux.HandleFunc("POST /admin/config/medals/rollback", h.allowed(rbac.Medals, rbac.Publish, h.rollbackDomain(func() configDomain { return h.medalDomain(true) })))
	h.mux.HandleFunc("GET /admin/live-controls", h.allowed(rbac.LiveControls, rbac.View, h.liveControls))
	h.mux.HandleFunc("GET /admin/calendar", h.allowed(rbac.LiveEvents, rbac.View, h.calendar))
	h.mux.HandleFunc("PUT /admin/live-controls", h.allowed(rbac.LiveControls, rbac.SwitchLiveControl, h.switchLiveControls))
	h.mux.HandleFunc("GET /admin/reviews", h.allowed(rbac.Queues, rbac.View, h.listReviews))
	h.mux.HandleFunc("POST /admin/reviews/{id}/clear", h.allowed(rbac.Queues, rbac.ActOnQueue, h.decideReview(risk.DecisionClear, ClearReviewAction)))
	h.mux.HandleFunc("POST /admin/reviews/{id}/permanent-ban", h.allowed(rbac.Queues, rbac.ActOnQueue, h.proposePermanentBan))
	h.mux.HandleFunc("POST /admin/reviews/{id}/confirm", h.allowed(rbac.Queues, rbac.ActOnQueue, h.decideReview(risk.DecisionConfirm, ConfirmReviewAction)))
	h.mux.HandleFunc("GET /admin/reports", h.allowed(rbac.Queues, rbac.View, h.listReports))
	h.mux.HandleFunc("POST /admin/players/{id}/reports/uphold", h.allowed(rbac.Queues, rbac.ActOnQueue, h.decideReports(report.StateUpheld, UpholdReportsAction)))
	h.mux.HandleFunc("POST /admin/players/{id}/reports/reject", h.allowed(rbac.Queues, rbac.ActOnQueue, h.decideReports(report.StateRejected, RejectReportsAction)))
	h.mux.HandleFunc("GET /admin/config/quick-messages", h.allowed(rbac.GameStrings, rbac.View, h.domainCatalog(quickmessage.Domain)))
	h.mux.HandleFunc("PUT /admin/config/quick-messages", h.allowed(rbac.GameStrings, rbac.Publish, h.publishDomain(quickMessagesDomain)))
	h.mux.HandleFunc("GET /admin/config/quick-messages/history", h.allowed(rbac.GameStrings, rbac.View, h.domainHistory(quickmessage.Domain)))
	h.mux.HandleFunc("POST /admin/config/quick-messages/rollback", h.allowed(rbac.GameStrings, rbac.Publish, h.rollbackDomain(quickMessagesDomain)))
	h.mux.HandleFunc("GET /admin/strings", h.allowed(rbac.GameStrings, rbac.View, h.listStrings))
	h.mux.HandleFunc("PUT /admin/strings/{key}", h.allowed(rbac.GameStrings, rbac.View, h.saveString))
	h.mux.HandleFunc("PUT /admin/strings/{key}/values/{language}", h.allowed(rbac.GameStrings, rbac.View, h.setValue))
	h.mux.HandleFunc("GET /admin/strings/export", h.allowed(rbac.GameStrings, rbac.View, h.exportStrings))
	h.mux.HandleFunc("POST /admin/strings/import", h.allowed(rbac.GameStrings, rbac.Edit, h.importStrings))
	h.mux.HandleFunc("GET /admin/strings/active", h.allowed(rbac.GameStrings, rbac.View, h.activeStrings))
	h.mux.HandleFunc("PUT /admin/strings/active", h.allowed(rbac.GameStrings, rbac.Publish, h.activateStrings))
	h.mux.HandleFunc("GET /admin/strings/bundles", h.allowed(rbac.GameStrings, rbac.View, h.listBundles))
	h.mux.HandleFunc("POST /admin/strings/bundles/{language}", h.allowed(rbac.GameStrings, rbac.Publish, h.publishBundle))
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type principal struct {
	account adminaccount.Account
	session adminsession.Session
	source  audit.Source
}

func (p principal) actor() audit.Actor {
	return audit.Actor{ID: p.account.ID, Roles: p.account.Roles}
}

func (h *Handler) audited(ctx context.Context, p principal, change func(tx pgx.Tx, now time.Time) ([]audit.Entry, error)) error {
	now := h.now()
	return pgx.BeginFunc(ctx, h.pool, func(tx pgx.Tx) error {
		entries, err := change(tx, now)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if _, err := h.audit.In(tx).Append(ctx, h.stamp(p, entry), now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (h *Handler) stamp(p principal, entry audit.Entry) audit.Entry {
	entry.Actor, entry.Environment, entry.Source = p.actor(), h.config.Environment, p.source
	return entry
}

func (h *Handler) noted(note changeNote) error {
	if strings.TrimSpace(note.Reason) == "" {
		return errReasonRequired
	}
	return nil
}

func (h *Handler) sensitive(p principal, note changeNote, now time.Time) error {
	if err := h.noted(note); err != nil {
		return err
	}
	if now.After(p.session.AuthenticatedAt.Add(h.config.Session.StepUp)) {
		return errStepUpRequired
	}
	return nil
}

type authedHandler func(w http.ResponseWriter, r *http.Request, p principal)

func (h *Handler) signedIn(next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := h.authenticate(r)
		if err != nil {
			h.refuse(w, err)
			return
		}
		if h.throttled(w, r, "account:"+p.account.ID, h.config.Limits.Account) {
			return
		}
		next(w, r, p)
	}
}

func (h *Handler) allowed(object rbac.Object, action rbac.Action, next authedHandler) http.HandlerFunc {
	return h.signedIn(func(w http.ResponseWriter, r *http.Request, p principal) {
		if err := h.allows(p, object, action); err != nil {
			h.refuse(w, err)
			return
		}
		next(w, r, p)
	})
}

func (h *Handler) authenticate(r *http.Request) (principal, error) {
	now := h.now()
	session, err := h.sessions.Authenticate(r.Context(), bearer(r), now)
	if err != nil {
		return principal{}, err
	}
	account, err := h.accounts.ByID(r.Context(), session.AccountID)
	if err != nil {
		return principal{}, err
	}
	if account.State != adminaccount.Active {
		if err := h.sessions.Revoke(r.Context(), session.ID, now); err != nil {
			return principal{}, err
		}
		return principal{}, adminsession.ErrRejected
	}
	return principal{account: account, session: session, source: source(r, session.ID)}, nil
}

func source(r *http.Request, sessionID string) audit.Source {
	return audit.Source{SessionID: sessionID, IP: clientIP(r), UserAgent: r.UserAgent()}
}

func clientIP(r *http.Request) netip.Addr {
	if forwarded := r.Header.Values("X-Forwarded-For"); len(forwarded) > 0 {
		hops := strings.Split(forwarded[len(forwarded)-1], ",")
		if addr, err := netip.ParseAddr(strings.TrimSpace(hops[len(hops)-1])); err == nil {
			return addr.Unmap()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}
	}
	addr, _ := netip.ParseAddr(host)
	return addr.Unmap()
}

func bearer(r *http.Request) string {
	scheme, value, _ := strings.Cut(r.Header.Get("Authorization"), " ")
	if !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(value)
}

type errorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}

var (
	errNotWorkspace     = errors.New("adminapi: identity is not a verified company account")
	errAccountRefused   = errors.New("adminapi: no active admin account for the identity")
	errDifferentAccount = errors.New("adminapi: step-up identity is another account")
	errGrantSeparately  = errors.New("adminapi: super_admin is granted to an existing account through an approval")
	errNoSuchVersion    = errors.New("adminapi: no such config version")
	errUnsupportedKind  = errors.New("adminapi: this kind of proposal cannot be carried out here yet")
	errOwnAccount       = errors.New("adminapi: an admin cannot change their own account")
	errMalformed        = errors.New("adminapi: malformed request")
	errForbidden        = errors.New("adminapi: the roles do not grant the action")
	errReasonRequired   = errors.New("adminapi: a reason is required")
	errStepUpRequired   = errors.New("adminapi: the action needs a recent sign-in")
)

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	return decodeUpTo(w, r, into, MaxBodyBytes)
}

func decodeUpTo(w http.ResponseWriter, r *http.Request, into any, limit int64) bool {
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit)).Decode(into)
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		reply(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "too_large"})
	case err != nil:
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
	}
	return err == nil
}

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	status, code := classify(err)
	if status >= http.StatusInternalServerError {
		h.logger.Error("admin request failed", "error", err)
	} else {
		h.logger.Info("admin request refused", "reason", code)
	}
	response := errorResponse{Error: code}
	if status == http.StatusBadRequest || code == "frozen" {
		response.Detail = err.Error()
	}
	reply(w, status, response)
}

func classify(err error) (int, string) {
	switch {
	case errors.Is(err, idtoken.ErrKeyFetch):
		return http.StatusServiceUnavailable, "provider_unavailable"
	case errors.Is(err, idtoken.ErrExpired), errors.Is(err, idtoken.ErrNotYetValid):
		return http.StatusUnauthorized, "token_expired"
	case errors.Is(err, idtoken.ErrNonce), errors.Is(err, idtoken.ErrNonceUsed):
		return http.StatusUnauthorized, "nonce_rejected"
	case errors.Is(err, idtoken.ErrMalformed), errors.Is(err, idtoken.ErrAlgorithm), errors.Is(err, idtoken.ErrUnknownKey),
		errors.Is(err, idtoken.ErrSignature), errors.Is(err, idtoken.ErrIssuer), errors.Is(err, idtoken.ErrAudience),
		errors.Is(err, idtoken.ErrSubject), errors.Is(err, idtoken.ErrUnknownProvider):
		return http.StatusUnauthorized, "token_rejected"
	case errors.Is(err, adminsession.ErrRejected):
		return http.StatusUnauthorized, "session_rejected"
	case errors.Is(err, errNotWorkspace):
		return http.StatusForbidden, "not_company_account"
	case errors.Is(err, errAccountRefused):
		return http.StatusForbidden, "account_refused"
	case errors.Is(err, errDifferentAccount):
		return http.StatusForbidden, "different_account"
	case errors.Is(err, adminaccount.ErrNotFound), errors.Is(err, playerrecord.ErrNotFound), errors.Is(err, playerrecord.ErrUnknownMatch):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, adminaccount.ErrTaken):
		return http.StatusConflict, "email_taken"
	case errors.Is(err, errGrantSeparately):
		return http.StatusConflict, "grant_by_approval"
	case errors.Is(err, errUnsupportedKind):
		return http.StatusConflict, "not_supported"
	case errors.Is(err, approval.ErrSelfDecision):
		return http.StatusForbidden, "self_decision"
	case errors.Is(err, approval.ErrNotAllowed):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, approval.ErrDecided), errors.Is(err, risk.ErrReviewClosed):
		return http.StatusConflict, "decided"
	case errors.Is(err, approval.ErrExpired):
		return http.StatusConflict, "expired"
	case errors.Is(err, approval.ErrNotFound), errors.Is(err, risk.ErrNoReview):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, approval.ErrInvalid):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, errOwnAccount):
		return http.StatusConflict, "own_account"
	case errors.Is(err, adminaccount.ErrLocale):
		return http.StatusBadRequest, "unknown_locale"
	case errors.Is(err, adminaccount.ErrInvalid), errors.Is(err, audit.ErrInvalid), errors.Is(err, errMalformed),
		errors.Is(err, playerrecord.ErrEmptyQuery):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, errBadAdjustment), errors.Is(err, wallet.ErrInvalid), errors.Is(err, wallet.ErrWrongSign):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, errBadGrant), errors.Is(err, reward.ErrInvalidBundle), errors.Is(err, reward.ErrInvalidGrant),
		errors.Is(err, inventory.ErrUnknownItem), errors.Is(err, inventory.ErrSourceNotListed), errors.Is(err, inventory.ErrDefaultItem),
		errors.Is(err, medal.ErrUnknownMedal), errors.Is(err, medal.ErrSourceNotListed), errors.Is(err, medal.ErrNoCatalog):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, wallet.ErrInsufficient):
		return http.StatusConflict, "insufficient"
	case errors.Is(err, promo.ErrUnknown):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, promo.ErrInvalid), errors.Is(err, errBadBatchCount):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, promo.ErrDuplicate):
		return http.StatusConflict, "duplicate"
	case errors.Is(err, promo.ErrApprover):
		return http.StatusForbidden, "self_decision"
	case errors.Is(err, promo.ErrNotActive):
		return http.StatusConflict, "not_pending"
	case errors.Is(err, promo.ErrDisabled):
		return http.StatusConflict, "disabled"
	case errors.Is(err, errBanNotConfigured), errors.Is(err, errPromoNotConfigured):
		return http.StatusConflict, "not_configured"
	case errors.Is(err, errNotBanned):
		return http.StatusConflict, "not_banned"
	case errors.Is(err, errShadowbanned):
		return http.StatusConflict, "shadowbanned"
	case errors.Is(err, errNotShadowbanned):
		return http.StatusConflict, "not_shadowbanned"
	case errors.Is(err, report.ErrNothingOpen):
		return http.StatusConflict, "no_open_reports"
	case errors.Is(err, errNotMuted):
		return http.StatusConflict, "not_muted"
	case errors.Is(err, errNoProfile):
		return http.StatusConflict, "no_profile"
	case errors.Is(err, rating.ErrSeason), errors.Is(err, rating.ErrSeasonGap), errors.Is(err, rating.ErrSeasonOverlap), errors.Is(err, rating.ErrSeasonEarlier):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, rating.ErrSeasonUnknown):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, rating.ErrSeasonStarted):
		return http.StatusConflict, "season_started"
	case errors.Is(err, errSeasonNotLive):
		return http.StatusConflict, "season_not_live"
	case errors.Is(err, errAssetStoreOff):
		return http.StatusServiceUnavailable, "asset_store_off"
	case errors.Is(err, objectstore.ErrUnavailable):
		return http.StatusServiceUnavailable, "asset_store_unavailable"
	case errors.Is(err, objectstore.ErrNotFound):
		return http.StatusConflict, "asset_file_missing"
	case errors.Is(err, errAssetChecked):
		return http.StatusConflict, "asset_already_checked"
	case errors.Is(err, errAssetUnknown):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, errAssetSlot):
		return http.StatusBadRequest, "asset_slot_unsupported"
	case errors.Is(err, errAssetItem):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, errBundleUnknown):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, errBundleDiffers), errors.Is(err, errBundleHeld):
		return http.StatusConflict, "bundle_not_editable"
	case errors.Is(err, errBundleStale):
		return http.StatusConflict, "stale_bundle"
	case errors.Is(err, errBundleSame):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, errReasonRequired):
		return http.StatusBadRequest, "reason_required"
	case errors.Is(err, errStepUpRequired):
		return http.StatusForbidden, "step_up_required"
	case errors.Is(err, audit.ErrForbidden), errors.Is(err, errForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, gamestrings.ErrInvalidLocalization), errors.Is(err, rulesconfig.ErrInvalid),
		errors.Is(err, shop.ErrInvalid), errors.Is(err, errOfferRemoved),
		errors.Is(err, mission.ErrInvalidCatalog), errors.Is(err, errMissionRemoved),
		errors.Is(err, dailylogin.ErrInvalidCalendar),
		errors.Is(err, medal.ErrInvalidCatalog), errors.Is(err, errMedalRemoved),
		errors.Is(err, pass.ErrInvalidConfig), errors.Is(err, errSeasonRemoved),
		errors.Is(err, rating.ErrInvalidConfig), errors.Is(err, livecontrol.ErrInvalid), errors.Is(err, quickmessage.ErrInvalid), errors.Is(err, matchmaking.ErrInvalidConfig),
		errors.Is(err, cosmetics.ErrInvalid), errors.Is(err, cosmetics.ErrSchema), errors.Is(err, cosmetics.ErrSuccessor):
		return http.StatusBadRequest, "invalid"
	case errors.Is(err, pass.ErrFrozen), errors.Is(err, rating.ErrFrozen):
		return http.StatusConflict, "frozen"
	case errors.Is(err, errStaleLiveControls):
		return http.StatusConflict, "stale_proposal"
	case errors.Is(err, errNoSuchVersion):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, gamestrings.ErrUnknownKey), errors.Is(err, gamestrings.ErrNoBundle):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, gamestrings.ErrInvalidEntry), errors.Is(err, gamestrings.ErrInvalidValue),
		errors.Is(err, gamestrings.ErrInvalidImport):
		return http.StatusBadRequest, "invalid"
	}
	return http.StatusInternalServerError, "internal"
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
