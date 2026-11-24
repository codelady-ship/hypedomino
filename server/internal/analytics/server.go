package analytics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	MissionComplete  = "mission_complete"
	PassLevel        = "pass_level"
	RewardGrant      = "reward_grant"
	PurchaseComplete = "purchase_complete"
)

const consentedAnalyticsID = `select i.analytics_id from analytics_ids i
	where i.account_id = $2
		and (select c.granted from consent_records c where c.account_id = $2 and c.purpose = 'analytics' order by c.recorded_at desc, c.id desc limit 1)
		and i.first_seen_at > coalesce((select max(c.recorded_at) from consent_records c where c.account_id = $2 and c.purpose = 'analytics' and not c.granted), '-infinity')
	order by i.first_seen_at desc, i.analytics_id
	limit 1`

func RecordServerIn(ctx context.Context, tx pgx.Tx, name, account string, count int, at time.Time) error {
	return RecordServerLabeledIn(ctx, tx, name, account, "", count, at)
}

func RecordServerLabeledIn(ctx context.Context, tx pgx.Tx, name, account, label string, count int, at time.Time) error {
	_, err := tx.Exec(ctx, `insert into analytics_events (analytics_id, name, label, value, received_at)
		select (`+consentedAnalyticsID+`), $1, $5, 0, $3 from generate_series(1, $4)`, name, account, at, count, label)
	return err
}
