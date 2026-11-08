package matchqueue

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
)

var ErrTicket = errors.New("matchqueue: ticket needs an account, a node and a session")

type Ticket struct {
	Entry   matchmaking.Entry `json:"entry"`
	Node    string            `json:"node"`
	Session string            `json:"session"`
}

type Placement struct {
	Match   matchmaking.Match `json:"match"`
	Tickets []Ticket          `json:"tickets"`
}

type Store struct {
	client redis.UniversalClient
	prefix string
}

func NewStore(client redis.UniversalClient, prefix string) *Store {
	return &Store{client: client, prefix: prefix}
}

func (s *Store) ticketsKey() string {
	return s.prefix + "matchmaking:tickets"
}

func (s *Store) Join(ctx context.Context, ticket Ticket) error {
	if ticket.Entry.Account == "" || ticket.Node == "" || ticket.Session == "" {
		return ErrTicket
	}
	raw, err := json.Marshal(ticket)
	if err != nil {
		return err
	}
	return s.client.HSet(ctx, s.ticketsKey(), ticket.Entry.Account, raw).Err()
}

func (s *Store) Leave(ctx context.Context, account string) (bool, error) {
	removed, err := s.client.HDel(ctx, s.ticketsKey(), account).Result()
	return removed > 0, err
}

func (s *Store) Tickets(ctx context.Context) (map[string]Ticket, map[string]string, error) {
	raw, err := s.client.HGetAll(ctx, s.ticketsKey()).Result()
	if err != nil {
		return nil, nil, err
	}
	tickets := make(map[string]Ticket, len(raw))
	for account, value := range raw {
		var ticket Ticket
		if err := json.Unmarshal([]byte(value), &ticket); err != nil {
			s.client.HDel(ctx, s.ticketsKey(), account)
			continue
		}
		tickets[account] = ticket
	}
	return tickets, raw, nil
}

var takeScript = redis.NewScript(`
for i = 1, #ARGV, 2 do
  if redis.call("HGET", KEYS[1], ARGV[i]) ~= ARGV[i + 1] then
    return 0
  end
end
for i = 1, #ARGV, 2 do
  redis.call("HDEL", KEYS[1], ARGV[i])
end
return 1
`)

func (s *Store) take(ctx context.Context, raw map[string]string, accounts []string) (bool, error) {
	args := make([]any, 0, 2*len(accounts))
	for _, account := range accounts {
		args = append(args, account, raw[account])
	}
	taken, err := takeScript.Run(ctx, s.client, []string{s.ticketsKey()}, args...).Int()
	return taken == 1, err
}

type Matcher struct {
	store  *Store
	config func() matchmaking.Config
	random *rand.Rand
}

func NewMatcher(store *Store, config func() matchmaking.Config, random *rand.Rand) *Matcher {
	return &Matcher{store: store, config: config, random: random}
}

func (m *Matcher) Tick(ctx context.Context, now time.Time) ([]Placement, error) {
	tickets, raw, err := m.store.Tickets(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]matchmaking.Entry, 0, len(tickets))
	for _, ticket := range tickets {
		entries = append(entries, ticket.Entry)
	}
	matches, _ := matchmaking.Form(m.config(), entries, now, m.random)
	var placements []Placement
	for _, match := range matches {
		placement := Placement{Match: match}
		var accounts []string
		for _, seat := range match.Seats {
			if !seat.Bot {
				accounts = append(accounts, seat.Account)
				placement.Tickets = append(placement.Tickets, tickets[seat.Account])
			}
		}
		taken, err := m.store.take(ctx, raw, accounts)
		if err != nil {
			return placements, err
		}
		if taken {
			placements = append(placements, placement)
		}
	}
	return placements, nil
}

func (m *Matcher) Return(ctx context.Context, placement Placement) error {
	for _, ticket := range placement.Tickets {
		if err := m.store.Join(ctx, ticket); err != nil {
			return err
		}
	}
	return nil
}
