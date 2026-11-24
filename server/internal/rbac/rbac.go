package rbac

import "slices"

type Role string

const (
	SuperAdmin Role = "super_admin"
	Content    Role = "content"
	LiveOps    Role = "live_ops"
	Support    Role = "support"
	Analyst    Role = "analyst"
)

type Object string

const (
	CosmeticItems      Object = "cosmetic_items"
	Assets             Object = "assets"
	RarityTable        Object = "rarity_table"
	RewardBundles      Object = "reward_bundles"
	ShopOffers         Object = "shop_offers"
	StoreProducts      Object = "store_products"
	PassSeasons        Object = "pass_seasons"
	Missions           Object = "missions"
	Medals             Object = "medals"
	LeaderboardEmblems Object = "leaderboard_emblems"
	PromoCodes         Object = "promo_codes"
	RankedSeasons      Object = "ranked_seasons"
	LiveEvents         Object = "live_events"
	GameConfig         Object = "game_config"
	AntiCheatConfig    Object = "anti_cheat_config"
	PrivacyConfig      Object = "privacy_config"
	ModerationConfig   Object = "moderation_config"
	GameStrings        Object = "game_strings"
	WordFilter         Object = "word_filter"
	LiveControls       Object = "live_controls"
	Snapshots          Object = "snapshots"
	Players            Object = "players"
	Queues             Object = "queues"
	RightsRequests     Object = "rights_requests"
	ContentGateReports Object = "content_gate_reports"
	Dashboards         Object = "dashboards"
	AnalyticsEvents    Object = "analytics_events"
	AuditLog           Object = "audit_log"
	AdminAccounts      Object = "admin_accounts"
)

type Action string

const (
	View                 Action = "view"
	Create               Action = "create"
	Edit                 Action = "edit"
	Publish              Action = "publish"
	Retire               Action = "retire"
	Delete               Action = "delete"
	ViewCounts           Action = "view_counts"
	CulturalReview       Action = "cultural_review"
	SwitchOffForIncident Action = "switch_off_for_incident"
	ApprovePromo         Action = "approve_promo"
	DisablePromo         Action = "disable_promo"
	EditTargetedAccounts Action = "edit_targeted_accounts"
	SwitchLiveControl    Action = "switch_live_control"
	Rollback             Action = "rollback"
	RollbackOwnDomains   Action = "rollback_own_domains"
	ActOnPlayer          Action = "act_on_player"
	ActOnQueue           Action = "act_on_queue"
	GrantSuperAdmin      Action = "grant_super_admin"
	ExportAudit          Action = "export_audit"
	PushStorePrices      Action = "push_store_prices"
)

func IsRole(role Role) bool {
	switch role {
	case SuperAdmin, Content, LiveOps, Support, Analyst:
		return true
	}
	return false
}

func IsObject(object Object) bool {
	_, known := matrix[object]
	return known
}

func Objects() []Object {
	objects := make([]Object, 0, len(matrix))
	for object := range matrix {
		objects = append(objects, object)
	}
	slices.Sort(objects)
	return objects
}

func Allows(roles []Role, object Object, action Action) bool {
	grants := matrix[object]
	for _, role := range roles {
		for _, granted := range grants[role] {
			if granted == action {
				return true
			}
		}
	}
	return false
}

func Granted(roles []Role, object Object) []Action {
	var granted []Action
	for _, role := range roles {
		for _, action := range matrix[object][role] {
			if !slices.Contains(granted, action) {
				granted = append(granted, action)
			}
		}
	}
	return granted
}
