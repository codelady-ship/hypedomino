package matchstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/partnerfit"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var (
	ErrUnknownMatch    = errors.New("matchstore: unknown match")
	ErrAlreadyFinished = errors.New("matchstore: match already finished")
	ErrInvalid         = errors.New("matchstore: invalid record")
)

const (
	KindRanked = "ranked"
	KindCasual = "casual"
	KindCustom = "custom"
)

const (
	ResultWin       = "win"
	ResultLoss      = "loss"
	ResultCancelled = "cancelled"
)

const (
	TopicRating    = "rating"
	TopicXp        = "xp"
	TopicMissions  = "missions"
	TopicEconomy   = "economy"
	TopicAnalytics = "analytics"
)

type Participant struct {
	Seat    rules.Seat
	Account string
	Bot     bool
	Loadout profile.Loadout
}

type Started struct {
	ID            string
	StartedAt     time.Time
	Mode          rules.Mode
	Kind          string
	Region        string
	ConfigVersion int
	Seed          uint64
	Participants  []Participant
}

type Event struct {
	Seq     int
	Kind    string
	Seat    rules.Seat
	HasSeat bool
	Payload any
	At      time.Time
}

type Message struct {
	Topic   string
	Key     string
	Payload any
}

type Finished struct {
	ID          string
	StartedAt   time.Time
	EndedAt     time.Time
	HandsPlayed int
	Scores      rules.TeamPoints
	Winner      rules.Team
	HasWinner   bool
	Cancelled   bool
	Replay      []byte
	Results     map[rules.Seat]string
	PartnerFit  map[rules.Seat]partnerfit.Count
	Events      []Event
	Outbox      []Message
}

type Pending struct {
	ID       int64
	Topic    string
	Key      string
	Payload  json.RawMessage
	Attempts int
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Begin(ctx context.Context, m Started) error {
	if m.ID == "" || m.StartedAt.IsZero() || m.Mode.Players() == 0 || len(m.Participants) != m.Mode.Players() {
		return ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `insert into matches (id, started_at, mode, kind, region, config_version, seed)
		values ($1, $2, $3, $4, nullif($5, ''), $6, $7) on conflict do nothing`,
		m.ID, m.StartedAt, m.Mode.String(), m.Kind, m.Region, m.ConfigVersion, int64(m.Seed))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	count := len(m.Participants)
	seats, teams := make([]int16, count), make([]int16, count)
	accounts, bots, loadouts := make([]*string, count), make([]bool, count), make([]string, count)
	for i, p := range m.Participants {
		if !p.Bot && p.Account != "" {
			accounts[i] = &p.Account
		}
		loadout, err := json.Marshal(p.Loadout)
		if err != nil {
			return err
		}
		seats[i], teams[i], bots[i], loadouts[i] = int16(p.Seat), int16(p.Seat.Team()), p.Bot, string(loadout)
	}
	if _, err := tx.Exec(ctx, `insert into match_participants (match_id, started_at, seat, team, account_id, bot, loadout)
		select $1, $2, p.seat, p.team, p.account::uuid, p.bot, p.loadout::jsonb
		from unnest($3::smallint[], $4::smallint[], $5::text[], $6::boolean[], $7::text[]) as p (seat, team, account, bot, loadout)`,
		m.ID, m.StartedAt, seats, teams, accounts, bots, loadouts); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Finish(ctx context.Context, f Finished) error {
	if f.ID == "" || f.StartedAt.IsZero() || f.EndedAt.IsZero() {
		return ErrInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var ended *time.Time
	err = tx.QueryRow(ctx, `select ended_at from matches where id = $1 and started_at = $2 for update`, f.ID, f.StartedAt).Scan(&ended)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ErrUnknownMatch
	case err != nil:
		return err
	case ended != nil:
		return ErrAlreadyFinished
	}
	var winner *int16
	if f.HasWinner {
		w := int16(f.Winner)
		winner = &w
	}
	if _, err := tx.Exec(ctx, `update matches set ended_at = $3, hands_played = $4, score_a = $5, score_b = $6,
		winner_team = $7, cancelled = $8, replay = $9 where id = $1 and started_at = $2`,
		f.ID, f.StartedAt, f.EndedAt, f.HandsPlayed, f.Scores[rules.TeamA], f.Scores[rules.TeamB], winner, f.Cancelled, f.Replay); err != nil {
		return err
	}
	for seat, result := range f.Results {
		tag, err := tx.Exec(ctx, `update match_participants set result = $4 where match_id = $1 and started_at = $2 and seat = $3`,
			f.ID, f.StartedAt, int16(seat), result)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: seat %s is not in match %s", ErrInvalid, seat, f.ID)
		}
	}
	for seat, fit := range f.PartnerFit {
		tag, err := tx.Exec(ctx, `update match_participants set partner_decisions = $4, partner_fits = $5 where match_id = $1 and started_at = $2 and seat = $3`,
			f.ID, f.StartedAt, int16(seat), fit.Decisions, fit.Fits)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: partner fit for seat %s outside match %s", ErrInvalid, seat, f.ID)
		}
	}
	for _, e := range f.Events {
		payload, err := json.Marshal(e.Payload)
		if err != nil {
			return err
		}
		var seat *int16
		if e.HasSeat {
			value := int16(e.Seat)
			seat = &value
		}
		at := e.At
		if at.IsZero() {
			at = f.EndedAt
		}
		if _, err := tx.Exec(ctx, `insert into match_events (match_id, started_at, seq, kind, seat, payload, at)
			values ($1, $2, $3, $4, $5, $6, $7)`, f.ID, f.StartedAt, e.Seq, e.Kind, seat, payload, at); err != nil {
			return err
		}
	}
	for _, m := range f.Outbox {
		payload, err := json.Marshal(m.Payload)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, $2, $3) on conflict (topic, key) do nothing`,
			m.Topic, m.Key, payload); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Running(ctx context.Context, account string, startedAfter time.Time) (bool, error) {
	var running bool
	err := s.pool.QueryRow(ctx, `select exists (
		select 1 from match_participants p
		join matches m on m.id = p.match_id and m.started_at = p.started_at
		where p.account_id = $1 and p.started_at > $2 and m.started_at > $2 and m.ended_at is null)`, account, startedAfter).Scan(&running)
	return running, err
}

func (s *Store) Pending(ctx context.Context, topic string, limit int) ([]Pending, error) {
	rows, err := s.pool.Query(ctx, `select id, topic, key, payload, attempts from outbox
		where published_at is null and ($1 = '' or topic = $1) order by id limit $2`, topic, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Pending
	for rows.Next() {
		var p Pending
		if err := rows.Scan(&p.ID, &p.Topic, &p.Key, &p.Payload, &p.Attempts); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) Published(ctx context.Context, id int64, now time.Time) error {
	_, err := s.pool.Exec(ctx, `update outbox set published_at = $2 where id = $1 and published_at is null`, id, now)
	return err
}

func (s *Store) Failed(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `update outbox set attempts = attempts + 1 where id = $1`, id)
	return err
}

type Publisher interface {
	Publish(ctx context.Context, message Pending) error
}

type PublisherFunc func(ctx context.Context, message Pending) error

func (f PublisherFunc) Publish(ctx context.Context, message Pending) error {
	return f(ctx, message)
}

type Relay struct {
	store     *Store
	publisher Publisher
	topic     string
	batch     int
	now       func() time.Time
}

func NewRelay(store *Store, publisher Publisher, topic string, batch int, now func() time.Time) *Relay {
	if batch <= 0 {
		batch = 100
	}
	if now == nil {
		now = time.Now
	}
	return &Relay{store: store, publisher: publisher, topic: topic, batch: batch, now: now}
}

func (r *Relay) Once(ctx context.Context) (int, error) {
	pending, err := r.store.Pending(ctx, r.topic, r.batch)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, message := range pending {
		if err := r.publisher.Publish(ctx, message); err != nil {
			if markErr := r.store.Failed(ctx, message.ID); markErr != nil {
				return published, markErr
			}
			return published, err
		}
		if err := r.store.Published(ctx, message.ID, r.now()); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

func (r *Relay) Run(ctx context.Context, every time.Duration, failed func(error)) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if _, err := r.Once(ctx); err != nil && failed != nil && ctx.Err() == nil {
			failed(err)
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func MatchMessages(kind string, key string, payload any) []Message {
	topics := []string{TopicXp, TopicMissions, TopicEconomy}
	if kind == KindRanked {
		topics = append([]string{TopicRating}, topics...)
	}
	if kind == KindCustom {
		topics = []string{TopicMissions}
	}
	topics = append(topics, TopicAnalytics)
	out := make([]Message, 0, len(topics))
	for _, topic := range topics {
		out = append(out, Message{Topic: topic, Key: key, Payload: payload})
	}
	return out
}
