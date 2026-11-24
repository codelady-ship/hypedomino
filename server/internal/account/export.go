package account

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const ExportSchemaVersion = 1

type exportSection struct {
	name  string
	table string
	query string
}

var exportSections = []exportSection{
	{"account", "accounts", `select id, state, country, language, created_at, country_changed_at, deletion_requested_at, banned_until, shadowbanned_at, muted_until from accounts where id = $1`},
	{"email", "auth_identities", `select email, created_at from auth_identities where account_id = $1 and email is not null order by created_at`},
	{"profile", "profiles", `select nickname, nickname_changed_at, rename_required, loadout, stats, updated_at from profiles where account_id = $1`},
	{"nickname_history", "nickname_history", `select nickname, replaced_at from nickname_history where account_id = $1 order by replaced_at`},
	{"notification_settings", "push_settings", `select gameplay, promotional, updated_at from push_settings where account_id = $1`},
	{"matches", "match_participants", `select match_id, started_at, seat, team, result, loadout from match_participants where account_id = $1 order by started_at`},
	{"ratings", "ratings", `select mode, season_id, mmr, matches, wins, placement_played, placement_penalty, placed, rank_index, lp, apex_tier, reset_index, board_country, updated_at from ratings where account_id = $1 order by mode`},
	{"rating_changes", "rating_changes", `select match_id, won, offender, placement, k, expected, mmr_before, mmr_after, lp_delta, rank_before, lp_before, rank_after, lp_after from rating_changes where account_id = $1`},
	{"season_results", "season_results", `select season_id, mode, position, rank_index, apex_tier, lp, wins, mmr, reset_index, country from season_results where account_id = $1 order by season_id, mode`},
	{"season_rewards", "season_rewards", `select season_id, mode, tier, rank_index, coins, bundle_id, emblems, created_at, seen_at from season_rewards where account_id = $1 order by created_at`},
	{"leaderboard_emblems", "leaderboard_emblems", `select mode, season_id, board, bracket, position, awarded_at, revoked_at, revoke_reason from leaderboard_emblems where account_id = $1 order by awarded_at`},
	{"leaderboard_removals", "leaderboard_removals", `select season_id, mode, removed_at from leaderboard_removals where account_id = $1 order by removed_at`},
	{"emblem_showcase", "emblem_showcases", `select emblems, updated_at from emblem_showcases where account_id = $1`},
	{"wallets", "wallets", `select currency, balance, updated_at from wallets where account_id = $1 order by currency`},
	{"ledger", "ledger_entries", `select currency, amount, reason, source_type, source_ref, balance_after, created_at from ledger_entries where account_id = $1 order by id`},
	{"inventory", "inventory_items", `select item_id, source_type, source_ref, granted_at, revoked_at from inventory_items where account_id = $1 order by granted_at`},
	{"reward_grants", "reward_grants", `select bundle_id, source_type, source_ref, report, created_at, revoked_at from reward_grants where account_id = $1 order by id`},
	{"chest_opens", "chest_opens", `select offer_id, roll, pity_before, pity_after, rarity, item_id, created_at from chest_opens where account_id = $1 order by id`},
	{"promo_targets", "promo_targets", `select promo_id from promo_targets where account_id = $1 order by promo_id`},
	{"promo_redemptions", "promo_redemptions", `select promo_id, redeemed_at from promo_redemptions where account_id = $1 order by redeemed_at`},
	{"promo_attempts", "promo_attempts", `select ok, attempted_at from promo_attempts where account_id = $1 order by id`},
	{"pass_progress", "pass_progress", `select season_id, xp_total, premium_at, entered_at from pass_progress where account_id = $1 order by entered_at`},
	{"pass_xp", "pass_xp_events", `select season_id, amount, source_type, source_ref, economy_day, created_at from pass_xp_events where account_id = $1 order by created_at`},
	{"medals", "medal_ownership", `select medal_id, source_type, source_ref, granted_at, revoked_at from medal_ownership where account_id = $1 order by granted_at`},
	{"medal_showcase", "medal_showcases", `select medals, updated_at from medal_showcases where account_id = $1`},
	{"missions", "mission_assignments", `select mission_id, period, mission_type, slot, progress, assigned_at, completed_at, replaced_at from mission_assignments where account_id = $1 order by assigned_at`},
	{"mission_matches", "mission_matches", `select match_id, created_at from mission_matches where account_id = $1 order by created_at`},
	{"daily_login", "daily_logins", `select claims, last_day, updated_at from daily_logins where account_id = $1`},
	{"purchases", "purchases", `select platform, transaction_id, order_id, offer_id, product_id, state, purchased_at, granted_at, refunded_at from purchases where account_id = $1 order by purchased_at`},
	{"penalties", "penalty_records", `select record, updated_at from penalty_records where account_id = $1`},
	{"account_sanctions", "account_sanctions", `select id, kind, category, starts_at, ends_at, lifted_at from account_sanctions where account_id = $1 order by starts_at, id`},
	{"appeals", "appeals", `select sanction_id, text, state, created_at, decided_at from appeals where account_id = $1 order by created_at, id`},
	{"sanctions", "risk_actions", `select action, created_at from risk_actions where account_id = $1 order by created_at`},
	{"sanction_signals", "risk_signals", `select kind as category, created_at from risk_signals where account_id = $1
		and created_at <= greatest((select max(created_at) from risk_actions where account_id = $1), (select shadowbanned_at from accounts where id = $1))
		order by created_at`},
	{"friends", "friendships", `select p.nickname, f.created_at from friendships f left join profiles p on p.account_id = f.friend_id where f.account_id = $1 order by f.created_at`},
	{"friend_requests", "friend_requests", `select case when r.sender_id = $1 then 'sent' else 'received' end as direction, p.nickname, r.created_at from friend_requests r
		left join profiles p on p.account_id = case when r.sender_id = $1 then r.recipient_id else r.sender_id end
		where $1 in (r.sender_id, r.recipient_id) order by r.created_at`},
	{"blocks", "blocks", `select p.nickname, b.created_at from blocks b left join profiles p on p.account_id = b.blocked_id where b.account_id = $1 order by b.created_at`},
	{"friend_code", "friend_codes", `select code from friend_codes where account_id = $1`},
	{"consents", "consent_records", `select purpose, granted, policy_version, source, app_version, recorded_at from consent_records where account_id = $1 order by recorded_at, id`},
	{"analytics", "analytics_ids", `select i.analytics_id, e.name, e.label, e.value, e.client_time_ms, e.received_at from analytics_ids i join analytics_events e using (analytics_id) where i.account_id = $1 order by e.received_at, e.id`},
}

var withheldFromExport = map[string]bool{
	"auth_sessions":   true,
	"app_attest_keys": true,
	"push_devices":    true,
	"rating_signals":  true,
	"risk_reviews":    true,
	"player_reports":  true,
	"data_exports":    true,
}

type exportDocument struct {
	SchemaVersion int                        `json:"schema_version"`
	GeneratedAt   time.Time                  `json:"generated_at"`
	Sections      map[string]json.RawMessage `json:"sections"`
}

func (s *Store) Export(ctx context.Context, id string, now time.Time) ([]byte, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	document := exportDocument{SchemaVersion: ExportSchemaVersion, GeneratedAt: now.UTC(), Sections: make(map[string]json.RawMessage, len(exportSections))}
	for _, section := range exportSections {
		var rows json.RawMessage
		if err := tx.QueryRow(ctx, `select coalesce(jsonb_agg(to_jsonb(r)), '[]'::jsonb) from (`+section.query+`) r`, id).Scan(&rows); err != nil {
			return nil, fmt.Errorf("export %s: %w", section.name, err)
		}
		document.Sections[section.name] = rows
	}
	return json.Marshal(document)
}
