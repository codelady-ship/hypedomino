package mission

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	SourceMission = "mission"
	Once          = "once"
)

var (
	ErrNoCatalog     = errors.New("mission.unavailable")
	ErrNotAssigned   = errors.New("mission.not_assigned")
	ErrRerollUsed    = errors.New("mission.reroll_used")
	ErrNoReplacement = errors.New("mission.no_replacement")
)

type Rewards interface {
	ApplyIn(ctx context.Context, tx pgx.Tx, grant reward.Grant, now time.Time) (reward.Report, error)
}

type Assignment struct {
	Mission   Mission    `json:"mission"`
	Period    string     `json:"period"`
	Progress  int64      `json:"progress"`
	Completed *time.Time `json:"completed_at"`
}

type Board struct {
	Version       int64        `json:"version"`
	Day           string       `json:"day"`
	Week          string       `json:"week"`
	Missions      []Assignment `json:"missions"`
	DailyRerolls  int          `json:"daily_rerolls"`
	WeeklyRerolls int          `json:"weekly_rerolls"`
}

type assignment struct {
	MissionID string
	Period    string
	Type      string
	Slot      int
	Progress  int64
	Completed *time.Time
}

type Store struct {
	pool    *pgxpool.Pool
	catalog *snapshot.Watcher[*Catalog]
	rewards Rewards
	draw    func(n int64) int64
}

func NewWatcher(pool *pgxpool.Pool, logger *slog.Logger) *snapshot.Watcher[*Catalog] {
	return snapshot.New(pool, Domain, Parse, logger)
}

func NewStore(pool *pgxpool.Pool, catalog *snapshot.Watcher[*Catalog], rewards Rewards) *Store {
	return &Store{pool: pool, catalog: catalog, rewards: rewards, draw: secureDraw}
}

func secureDraw(n int64) int64 {
	value, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		panic(err)
	}
	return value.Int64()
}

func (s *Store) current() (*Catalog, int64, error) {
	catalog, version, ok := s.catalog.Current()
	if !ok {
		return nil, 0, ErrNoCatalog
	}
	return catalog, version, nil
}

func Period(missionType string, at time.Time) string {
	utc := at.UTC()
	switch missionType {
	case Daily:
		return utc.Format(time.DateOnly)
	case Weekly:
		year, week := utc.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	}
	return Once
}

func previousPeriod(missionType string, at time.Time) string {
	if missionType == Weekly {
		return Period(Weekly, at.AddDate(0, 0, -7))
	}
	return Period(Daily, at.AddDate(0, 0, -1))
}

func (s *Store) Board(ctx context.Context, account string, now time.Time) (Board, error) {
	catalog, version, err := s.current()
	if err != nil {
		return Board{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Board{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return Board{}, err
	}
	board, err := s.boardIn(ctx, tx, catalog, version, account, now)
	if err != nil {
		return Board{}, err
	}
	return board, tx.Commit(ctx)
}

func (s *Store) boardIn(ctx context.Context, tx pgx.Tx, catalog *Catalog, version int64, account string, now time.Time) (Board, error) {
	active, err := s.ensureIn(ctx, tx, catalog, account, now)
	if err != nil {
		return Board{}, err
	}
	board := Board{Version: version, Day: Period(Daily, now), Week: Period(Weekly, now), Missions: []Assignment{}}
	for _, a := range active {
		mission, ok := catalog.Mission(a.MissionID)
		if ok {
			board.Missions = append(board.Missions, Assignment{Mission: mission, Period: a.Period, Progress: a.Progress, Completed: a.Completed})
		}
	}
	if board.DailyRerolls, err = rerollsLeft(ctx, tx, account, Daily, board.Day, catalog.DailyRerolls); err != nil {
		return Board{}, err
	}
	board.WeeklyRerolls, err = rerollsLeft(ctx, tx, account, Weekly, board.Week, catalog.WeeklyRerolls)
	return board, err
}

func rerollsLeft(ctx context.Context, tx pgx.Tx, account, missionType, period string, allowed int) (int, error) {
	var used int
	err := tx.QueryRow(ctx, `select count(*) from mission_assignments where account_id = $1 and mission_type = $2 and period = $3 and replaced_at is not null`, account, missionType, period).Scan(&used)
	return max(0, allowed-used), err
}

func (s *Store) ensureIn(ctx context.Context, tx pgx.Tx, catalog *Catalog, account string, now time.Time) ([]assignment, error) {
	for _, missionType := range []string{Daily, Weekly} {
		if err := s.fillSlotsIn(ctx, tx, catalog, account, missionType, now); err != nil {
			return nil, err
		}
	}
	for _, mission := range catalog.Missions {
		if mission.Type != Achievement || mission.Availability != Live {
			continue
		}
		if _, err := tx.Exec(ctx, `insert into mission_assignments (account_id, mission_id, period, mission_type, slot, assigned_at) values ($1, $2, $3, $4, 0, $5) on conflict do nothing`, account, mission.ID, Once, Achievement, now); err != nil {
			return nil, err
		}
	}
	return activeIn(ctx, tx, account, now)
}

func activeIn(ctx context.Context, tx pgx.Tx, account string, now time.Time) ([]assignment, error) {
	rows, err := tx.Query(ctx, `select mission_id, period, mission_type, slot, progress, completed_at from mission_assignments
		where account_id = $1 and replaced_at is null and period in ($2, $3, $4) order by mission_type, slot, mission_id`,
		account, Period(Daily, now), Period(Weekly, now), Once)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (assignment, error) {
		var a assignment
		return a, row.Scan(&a.MissionID, &a.Period, &a.Type, &a.Slot, &a.Progress, &a.Completed)
	})
}

func (s *Store) fillSlotsIn(ctx context.Context, tx pgx.Tx, catalog *Catalog, account, missionType string, now time.Time) error {
	period := Period(missionType, now)
	rows, err := tx.Query(ctx, `select slot, mission_id, period from mission_assignments where account_id = $1 and mission_type = $2 and period in ($3, $4) and replaced_at is null`, account, missionType, period, previousPeriod(missionType, now))
	if err != nil {
		return err
	}
	held, previous := map[int]bool{}, map[int]string{}
	for rows.Next() {
		var slot int
		var id, at string
		if err := rows.Scan(&slot, &id, &at); err != nil {
			rows.Close()
			return err
		}
		if at == period {
			held[slot] = true
		} else {
			previous[slot] = id
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for slot := 1; slot <= len(catalog.Slots(missionType)); slot++ {
		if held[slot] {
			continue
		}
		pool := catalog.pool(missionType, slot, nil)
		if fresh := catalog.pool(missionType, slot, []string{previous[slot]}); len(fresh) > 0 {
			pool = fresh
		}
		if len(pool) == 0 {
			continue
		}
		if _, err := tx.Exec(ctx, `insert into mission_assignments (account_id, mission_id, period, mission_type, slot, assigned_at) values ($1, $2, $3, $4, $5, $6)`, account, s.pick(pool).ID, period, missionType, slot, now); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) pool(missionType string, slot int, excluded []string) []Mission {
	var pool []Mission
	for _, mission := range c.Missions {
		if mission.Type == missionType && mission.Slot == slot && mission.Availability == Live && !slices.Contains(excluded, mission.ID) {
			pool = append(pool, mission)
		}
	}
	return pool
}

func (s *Store) pick(pool []Mission) Mission {
	var total int64
	for _, mission := range pool {
		total += int64(mission.Weight)
	}
	roll := s.draw(total)
	for _, mission := range pool {
		roll -= int64(mission.Weight)
		if roll < 0 {
			return mission
		}
	}
	return pool[len(pool)-1]
}

func (s *Store) ApplyMatch(ctx context.Context, account, matchID string, match Match, endedAt time.Time) ([]string, error) {
	catalog, version, err := s.current()
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return nil, err
	}
	tag, err := tx.Exec(ctx, `insert into mission_matches (account_id, match_id, created_at) values ($1, $2, $3) on conflict do nothing`, account, matchID, endedAt)
	if err != nil || tag.RowsAffected() == 0 {
		return nil, err
	}
	active, err := s.ensureIn(ctx, tx, catalog, account, endedAt)
	if err != nil {
		return nil, err
	}
	completed := []string{}
	for _, a := range active {
		mission, ok := catalog.Mission(a.MissionID)
		added := mission.Progress(match)
		if !ok || a.Completed != nil || added == 0 {
			continue
		}
		progress := min(a.Progress+added, mission.Target)
		done := progress >= mission.Target
		if _, err := tx.Exec(ctx, `update mission_assignments set progress = $4, completed_at = case when $5 then $6::timestamptz end where account_id = $1 and mission_id = $2 and period = $3`, account, a.MissionID, a.Period, progress, done, endedAt); err != nil {
			return nil, err
		}
		if !done {
			continue
		}
		if err := s.grantIn(ctx, tx, catalog, version, account, mission, a, endedAt); err != nil {
			return nil, err
		}
		completed = append(completed, mission.ID)
	}
	if len(completed) > 0 {
		if err := analytics.RecordServerIn(ctx, tx, analytics.MissionComplete, account, len(completed), endedAt); err != nil {
			return nil, err
		}
	}
	return completed, tx.Commit(ctx)
}

func (s *Store) grantIn(ctx context.Context, tx pgx.Tx, catalog *Catalog, version int64, account string, mission Mission, a assignment, now time.Time) error {
	bundle := mission.Bundle
	if mission.Type == Daily || mission.Type == Weekly {
		slot := catalog.Slots(mission.Type)[a.Slot-1]
		bundle = &reward.Bundle{ID: fmt.Sprintf("bundle.%s_mission_slot_%d", mission.Type, a.Slot), Revision: int(max(1, version)), Coins: slot.Coins, PassXP: slot.Xp}
	}
	if bundle == nil || (bundle.Coins == 0 && bundle.PassXP == 0 && len(bundle.Cosmetics) == 0 && len(bundle.Medals) == 0) {
		return nil
	}
	_, err := s.rewards.ApplyIn(ctx, tx, reward.Grant{
		Account:       account,
		Bundle:        *bundle,
		Source:        cosmetics.Source{Type: SourceMission, Ref: mission.ID},
		Reason:        wallet.Mission,
		Key:           "mission:" + mission.ID + ":" + a.Period,
		ConfigVersion: int(version),
	}, now)
	return err
}

func (s *Store) Reroll(ctx context.Context, account, missionID string, now time.Time) (Board, error) {
	catalog, version, err := s.current()
	if err != nil {
		return Board{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Board{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return Board{}, err
	}
	active, err := s.ensureIn(ctx, tx, catalog, account, now)
	if err != nil {
		return Board{}, err
	}
	index := slices.IndexFunc(active, func(a assignment) bool {
		return a.MissionID == missionID && a.Completed == nil && (a.Type == Daily || a.Type == Weekly)
	})
	if index < 0 {
		return Board{}, ErrNotAssigned
	}
	target := active[index]
	allowed := catalog.DailyRerolls
	if target.Type == Weekly {
		allowed = catalog.WeeklyRerolls
	}
	left, err := rerollsLeft(ctx, tx, account, target.Type, target.Period, allowed)
	if err != nil {
		return Board{}, err
	}
	if left == 0 {
		return Board{}, ErrRerollUsed
	}
	had, err := periodMissions(ctx, tx, account, target.Period)
	if err != nil {
		return Board{}, err
	}
	pool := catalog.pool(target.Type, target.Slot, had)
	if len(pool) == 0 {
		return Board{}, ErrNoReplacement
	}
	if _, err := tx.Exec(ctx, `update mission_assignments set replaced_at = $4 where account_id = $1 and mission_id = $2 and period = $3`, account, target.MissionID, target.Period, now); err != nil {
		return Board{}, err
	}
	if _, err := tx.Exec(ctx, `insert into mission_assignments (account_id, mission_id, period, mission_type, slot, assigned_at) values ($1, $2, $3, $4, $5, $6)`, account, s.pick(pool).ID, target.Period, target.Type, target.Slot, now); err != nil {
		return Board{}, err
	}
	board, err := s.boardIn(ctx, tx, catalog, version, account, now)
	if err != nil {
		return Board{}, err
	}
	return board, tx.Commit(ctx)
}

func periodMissions(ctx context.Context, tx pgx.Tx, account, period string) ([]string, error) {
	rows, err := tx.Query(ctx, `select mission_id from mission_assignments where account_id = $1 and period = $2`, account, period)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
