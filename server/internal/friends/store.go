package friends

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/profile"
)

var (
	ErrUnknownPlayer  = errors.New("friend.unknown_player")
	ErrSelf           = errors.New("friend.self")
	ErrAlreadyFriends = errors.New("friend.already_friends")
	ErrFriendsFull    = errors.New("friend.friends_full")
	ErrRequestsFull   = errors.New("friend.requests_full")
	ErrNoRequest      = errors.New("friend.no_request")
	ErrNotFriends     = errors.New("friend.not_friends")
	ErrNoTarget       = errors.New("friend.target_required")
	ErrInvalidConfig  = errors.New("friends: invalid config")
)

var accountID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type Config struct {
	MaxFriends  int
	MaxRequests int
}

func DefaultConfig() Config {
	return Config{MaxFriends: 100, MaxRequests: 50}
}

func (c Config) Validate() error {
	if c.MaxFriends < 1 || c.MaxRequests < 1 {
		return fmt.Errorf("%w: friend and request limits must be at least 1", ErrInvalidConfig)
	}
	return nil
}

type Player struct {
	AccountID string    `json:"account_id"`
	Nickname  string    `json:"nickname"`
	Since     time.Time `json:"since"`
	Presence  string    `json:"presence,omitempty"`
}

type List struct {
	Friends  []Player `json:"friends"`
	Incoming []Player `json:"incoming"`
	Outgoing []Player `json:"outgoing"`
	Blocked  []Player `json:"blocked"`
}

type Store struct {
	pool     *pgxpool.Pool
	config   Config
	presence *Presence
}

func (s *Store) ShowPresence(presence *Presence) {
	s.presence = presence
}

func NewStore(pool *pgxpool.Pool, config Config) *Store {
	return &Store{pool: pool, config: config}
}

type Target struct {
	Nickname string `json:"nickname"`
	Code     string `json:"code"`
}

func (t Target) lookup() (string, string, error) {
	switch {
	case t.Nickname != "" && t.Code == "":
		return `select p.account_id::text from profiles p join accounts a on a.id = p.account_id
			where p.nickname_key = $1 and a.state = 'active'`, profile.Key(t.Nickname), nil
	case t.Code != "" && t.Nickname == "":
		code, ok := normalizeCode(t.Code)
		if !ok {
			return "", "", ErrUnknownPlayer
		}
		return `select c.account_id::text from friend_codes c join accounts a on a.id = c.account_id
			where c.code = $1 and a.state = 'active'`, code, nil
	}
	return "", "", ErrNoTarget
}

func (s *Store) Request(ctx context.Context, sender string, target Target, now time.Time) (bool, error) {
	query, key, err := target.lookup()
	if err != nil {
		return false, err
	}
	befriended := false
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var recipient string
		err := tx.QueryRow(ctx, query, key).Scan(&recipient)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrUnknownPlayer
		case err != nil:
			return err
		case recipient == sender:
			return ErrSelf
		}
		if err := lockPair(ctx, tx, sender, recipient); err != nil {
			return err
		}
		var blocked, friends, reverse bool
		if err := tx.QueryRow(ctx, `select
			exists (select 1 from blocks where (account_id = $1 and blocked_id = $2) or (account_id = $2 and blocked_id = $1)),
			exists (select 1 from friendships where account_id = $1 and friend_id = $2),
			exists (select 1 from friend_requests where sender_id = $2 and recipient_id = $1)`, sender, recipient).Scan(&blocked, &friends, &reverse); err != nil {
			return err
		}
		switch {
		case blocked:
			return ErrUnknownPlayer
		case friends:
			return ErrAlreadyFriends
		case reverse:
			befriended = true
			return s.befriend(ctx, tx, sender, recipient, now)
		}
		return s.send(ctx, tx, sender, recipient, now)
	})
	return befriended, err
}

func (s *Store) send(ctx context.Context, tx pgx.Tx, sender, recipient string, now time.Time) error {
	if open, err := holds(ctx, tx, `select exists (select 1 from friend_requests where sender_id = $1 and recipient_id = $2)`, sender, recipient); err != nil || open {
		return err
	}
	var outgoing, incoming int
	if err := tx.QueryRow(ctx, `select (select count(*) from friend_requests where sender_id = $1), (select count(*) from friend_requests where recipient_id = $2)`,
		sender, recipient).Scan(&outgoing, &incoming); err != nil {
		return err
	}
	if outgoing >= s.config.MaxRequests || incoming >= s.config.MaxRequests {
		return ErrRequestsFull
	}
	_, err := tx.Exec(ctx, `insert into friend_requests (sender_id, recipient_id, created_at) values ($1, $2, $3)`, sender, recipient, now)
	return err
}

func (s *Store) befriend(ctx context.Context, tx pgx.Tx, a, b string, now time.Time) error {
	var full bool
	if err := tx.QueryRow(ctx, `select exists (select 1 from friendships where account_id = any($1::uuid[]) group by account_id having count(*) >= $2)`,
		[]string{a, b}, s.config.MaxFriends).Scan(&full); err != nil {
		return err
	}
	if full {
		return ErrFriendsFull
	}
	if err := dropRequests(ctx, tx, a, b); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `insert into friendships (account_id, friend_id, created_at) values ($1, $2, $3), ($2, $1, $3)`, a, b, now)
	return err
}

func (s *Store) Accept(ctx context.Context, recipient, sender string, now time.Time) error {
	if !accountID.MatchString(sender) {
		return ErrNoRequest
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockPair(ctx, tx, sender, recipient); err != nil {
			return err
		}
		open, err := holds(ctx, tx, `select exists (select 1 from friend_requests r join accounts a on a.id = r.sender_id
			where r.sender_id = $1 and r.recipient_id = $2 and a.state = 'active')`, sender, recipient)
		if err != nil {
			return err
		}
		if !open {
			return ErrNoRequest
		}
		return s.befriend(ctx, tx, recipient, sender, now)
	})
}

func (s *Store) Withdraw(ctx context.Context, sender, recipient string) error {
	if !accountID.MatchString(sender) || !accountID.MatchString(recipient) {
		return ErrNoRequest
	}
	tag, err := s.pool.Exec(ctx, `delete from friend_requests where sender_id = $1 and recipient_id = $2`, sender, recipient)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNoRequest
	}
	return err
}

func (s *Store) Remove(ctx context.Context, account, friend string) error {
	if !accountID.MatchString(friend) {
		return ErrNotFriends
	}
	tag, err := s.pool.Exec(ctx, `delete from friendships where (account_id = $1 and friend_id = $2) or (account_id = $2 and friend_id = $1)`, account, friend)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFriends
	}
	return err
}

func (s *Store) Block(ctx context.Context, account, other string, now time.Time) error {
	switch {
	case !accountID.MatchString(other):
		return ErrUnknownPlayer
	case account == other:
		return ErrSelf
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		known, err := holds(ctx, tx, `select exists (select 1 from accounts where id = $1 and state <> 'deleted')`, other)
		if err != nil {
			return err
		}
		if !known {
			return ErrUnknownPlayer
		}
		if err := lockPair(ctx, tx, account, other); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `insert into blocks (account_id, blocked_id, created_at) values ($1, $2, $3) on conflict do nothing`, account, other, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `delete from friendships where (account_id = $1 and friend_id = $2) or (account_id = $2 and friend_id = $1)`, account, other); err != nil {
			return err
		}
		return dropRequests(ctx, tx, account, other)
	})
}

func (s *Store) Unblock(ctx context.Context, account, other string) error {
	if !accountID.MatchString(other) {
		return nil
	}
	_, err := s.pool.Exec(ctx, `delete from blocks where account_id = $1 and blocked_id = $2`, account, other)
	return err
}

func (s *Store) Blocked(ctx context.Context, a, b string) (bool, error) {
	if !accountID.MatchString(a) || !accountID.MatchString(b) {
		return false, nil
	}
	return holds(ctx, s.pool, `select exists (select 1 from blocks where (account_id = $1 and blocked_id = $2) or (account_id = $2 and blocked_id = $1))`, a, b)
}

func (s *Store) List(ctx context.Context, account string) (List, error) {
	var list List
	for _, part := range []struct {
		into  *[]Player
		query string
	}{
		{&list.Friends, `select f.friend_id::text, coalesce(p.nickname, ''), f.created_at from friendships f join accounts a on a.id = f.friend_id
			left join profiles p on p.account_id = f.friend_id where f.account_id = $1 and a.state = 'active' order by lower(p.nickname)`},
		{&list.Incoming, `select r.sender_id::text, coalesce(p.nickname, ''), r.created_at from friend_requests r join accounts a on a.id = r.sender_id
			left join profiles p on p.account_id = r.sender_id where r.recipient_id = $1 and a.state = 'active' order by r.created_at desc`},
		{&list.Outgoing, `select r.recipient_id::text, coalesce(p.nickname, ''), r.created_at from friend_requests r join accounts a on a.id = r.recipient_id
			left join profiles p on p.account_id = r.recipient_id where r.sender_id = $1 and a.state = 'active' order by r.created_at desc`},
		{&list.Blocked, `select b.blocked_id::text, coalesce(p.nickname, ''), b.created_at from blocks b
			left join profiles p on p.account_id = b.blocked_id where b.account_id = $1 order by b.created_at desc`},
	} {
		rows, err := s.pool.Query(ctx, part.query, account)
		if err != nil {
			return List{}, err
		}
		players, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Player, error) {
			var p Player
			err := row.Scan(&p.AccountID, &p.Nickname, &p.Since)
			return p, err
		})
		if err != nil {
			return List{}, err
		}
		*part.into = append([]Player{}, players...)
	}
	return list, s.markPresence(ctx, list.Friends)
}

func (s *Store) FriendIDs(ctx context.Context, account string) ([]string, error) {
	if !accountID.MatchString(account) {
		return []string{}, nil
	}
	rows, err := s.pool.Query(ctx, `select f.friend_id::text from friendships f join accounts a on a.id = f.friend_id where f.account_id = $1 and a.state = 'active'`, account)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func (s *Store) markPresence(ctx context.Context, friends []Player) error {
	if s.presence == nil {
		return nil
	}
	accounts := make([]string, len(friends))
	for i, friend := range friends {
		accounts[i] = friend.AccountID
	}
	states, err := s.presence.States(ctx, accounts)
	if err != nil {
		return err
	}
	for i := range friends {
		friends[i].Presence = states[i]
	}
	return nil
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func holds(ctx context.Context, db querier, query string, args ...any) (bool, error) {
	var found bool
	err := db.QueryRow(ctx, query, args...).Scan(&found)
	return found, err
}

func lockPair(ctx context.Context, tx pgx.Tx, a, b string) error {
	rows, err := tx.Query(ctx, `select id from accounts where id = any($1::uuid[]) order by id for update`, []string{a, b})
	if err != nil {
		return err
	}
	rows.Close()
	return rows.Err()
}

func dropRequests(ctx context.Context, tx pgx.Tx, a, b string) error {
	_, err := tx.Exec(ctx, `delete from friend_requests where (sender_id = $1 and recipient_id = $2) or (sender_id = $2 and recipient_id = $1)`, a, b)
	return err
}
