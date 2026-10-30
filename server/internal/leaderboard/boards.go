package leaderboard

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const (
	BoardGlobal = "global"

	lpCap    = 999_999_999
	winsCap  = 999_999_999
	lpScale  = 1e9
	rebuildB = 1000
)

var (
	ErrUnknownBoard = errors.New("leaderboard.unknown_board")
	ErrUnavailable  = errors.New("leaderboard.unavailable")
)

type Source interface {
	Placed(ctx context.Context, season string, mode rules.Mode, visit func(rating.Entry) error) error
}

type Config struct {
	Prefix string
	TTL    time.Duration
}

func DefaultConfig() Config {
	return Config{Prefix: "lb", TTL: 30 * 24 * time.Hour}
}

type Row struct {
	Position int         `json:"position"`
	Account  string      `json:"account_id"`
	Rank     rating.Rank `json:"rank"`
	Tier     string      `json:"tier"`
	Wins     int         `json:"wins"`
}

type Boards struct {
	client     redis.UniversalClient
	source     Source
	config     Config
	mu         sync.Mutex
	rebuilding map[string]bool
}

func New(client redis.UniversalClient, source Source, config Config) *Boards {
	return &Boards{client: client, source: source, config: config, rebuilding: map[string]bool{}}
}

func (b *Boards) keys(season string, mode rules.Mode, board string) (set, members, versions string) {
	base := b.config.Prefix + ":" + season + ":" + mode.String() + ":" + board
	return base, base + ":m", base + ":v"
}

func apexOrder(tier string) int {
	switch tier {
	case rating.TierChallenger:
		return 3
	case rating.TierGrandmaster:
		return 2
	case rating.TierMaster:
		return 1
	}
	return 0
}

func score(e rating.Entry) float64 {
	return -(float64(e.Rank.Index*4+apexOrder(e.Tier))*lpScale + float64(min(max(e.Rank.LP, 0), lpCap)))
}

func member(e rating.Entry) string {
	at := e.LPAt.Unix()
	if e.LPAt.IsZero() {
		at = 0
	}
	return fmt.Sprintf("%010d:%09d:%s", max(at, 0), winsCap-min(max(e.Wins, 0), winsCap), e.Account)
}

func decode(m string, s float64, position int) (Row, error) {
	parts := strings.SplitN(m, ":", 3)
	if len(parts) != 3 {
		return Row{}, fmt.Errorf("leaderboard: malformed member %q", m)
	}
	wins, err := strconv.Atoi(parts[1])
	if err != nil {
		return Row{}, fmt.Errorf("leaderboard: malformed member %q", m)
	}
	value := int64(-s)
	primary := int(value / int64(lpScale))
	row := Row{Position: position, Account: parts[2], Wins: winsCap - wins}
	row.Rank = rating.Rank{Index: primary / 4, LP: int(value % int64(lpScale))}
	row.Tier = row.Rank.Tier()
	switch primary % 4 {
	case 3:
		row.Tier = rating.TierChallenger
	case 2:
		row.Tier = rating.TierGrandmaster
	}
	return row, nil
}

var updateScript = redis.NewScript(`
local old = redis.call('HGET', KEYS[2], ARGV[1])
local version = redis.call('HGET', KEYS[3], ARGV[1])
if version and tonumber(version) > tonumber(ARGV[4]) then return 0 end
if old and old ~= ARGV[2] then redis.call('ZREM', KEYS[1], old) end
redis.call('ZADD', KEYS[1], ARGV[3], ARGV[2])
redis.call('HSET', KEYS[2], ARGV[1], ARGV[2])
redis.call('HSET', KEYS[3], ARGV[1], ARGV[4])
redis.call('EXPIRE', KEYS[1], ARGV[5])
redis.call('EXPIRE', KEYS[2], ARGV[5])
redis.call('EXPIRE', KEYS[3], ARGV[5])
return 1
`)

var removeScript = redis.NewScript(`
local old = redis.call('HGET', KEYS[2], ARGV[1])
if old then redis.call('ZREM', KEYS[1], old) end
redis.call('HDEL', KEYS[2], ARGV[1])
redis.call('HDEL', KEYS[3], ARGV[1])
return old and 1 or 0
`)

func (b *Boards) Update(ctx context.Context, season string, mode rules.Mode, entries []rating.Entry) error {
	set, members, versions := b.keys(season, mode, BoardGlobal)
	ttl := int(b.config.TTL.Seconds())
	for _, e := range entries {
		if err := updateScript.Run(ctx, b.client, []string{set, members, versions}, e.Account, member(e), score(e), e.Version, ttl).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (b *Boards) Remove(ctx context.Context, season string, mode rules.Mode, accounts []string) error {
	set, members, versions := b.keys(season, mode, BoardGlobal)
	for _, account := range accounts {
		if err := removeScript.Run(ctx, b.client, []string{set, members, versions}, account).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (b *Boards) Rebuild(ctx context.Context, season string, mode rules.Mode) (int, error) {
	set, members, versions := b.keys(season, mode, BoardGlobal)
	b.mu.Lock()
	if b.rebuilding[set] {
		b.mu.Unlock()
		return 0, nil
	}
	b.rebuilding[set] = true
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.rebuilding, set)
		b.mu.Unlock()
	}()
	stamp := strconv.FormatInt(time.Now().UnixNano(), 36)
	tempSet, tempMembers, tempVersions := set+":tmp:"+stamp, members+":tmp:"+stamp, versions+":tmp:"+stamp
	pipe := b.client.Pipeline()
	count, queued := 0, 0
	flush := func() error {
		if queued == 0 {
			return nil
		}
		_, err := pipe.Exec(ctx)
		queued = 0
		return err
	}
	err := b.source.Placed(ctx, season, mode, func(e rating.Entry) error {
		m := member(e)
		pipe.ZAdd(ctx, tempSet, redis.Z{Score: score(e), Member: m})
		pipe.HSet(ctx, tempMembers, e.Account, m)
		pipe.HSet(ctx, tempVersions, e.Account, e.Version)
		count++
		queued++
		if queued >= rebuildB {
			return flush()
		}
		return nil
	})
	if err == nil {
		err = flush()
	}
	if err != nil {
		b.client.Del(ctx, tempSet, tempMembers, tempVersions)
		return 0, err
	}
	if count == 0 {
		return 0, b.client.Del(ctx, set, members, versions).Err()
	}
	swap := b.client.TxPipeline()
	swap.Rename(ctx, tempSet, set)
	swap.Rename(ctx, tempMembers, members)
	swap.Rename(ctx, tempVersions, versions)
	swap.Expire(ctx, set, b.config.TTL)
	swap.Expire(ctx, members, b.config.TTL)
	swap.Expire(ctx, versions, b.config.TTL)
	if _, err := swap.Exec(ctx); err != nil {
		return 0, err
	}
	return count, nil
}

func (b *Boards) ensure(ctx context.Context, season string, mode rules.Mode, board string) (string, string, error) {
	if board != BoardGlobal {
		return "", "", fmt.Errorf("%w: %s", ErrUnknownBoard, board)
	}
	set, members, _ := b.keys(season, mode, board)
	exists, err := b.client.Exists(ctx, set).Result()
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if exists == 0 {
		if _, err := b.Rebuild(ctx, season, mode); err != nil {
			return "", "", err
		}
	}
	return set, members, nil
}

func (b *Boards) Top(ctx context.Context, season string, mode rules.Mode, board string, size int) ([]Row, error) {
	set, _, err := b.ensure(ctx, season, mode, board)
	if err != nil {
		return nil, err
	}
	return b.rows(ctx, set, 0, int64(size)-1)
}

func (b *Boards) rows(ctx context.Context, set string, start, stop int64) ([]Row, error) {
	if stop < start {
		return []Row{}, nil
	}
	members, err := b.client.ZRangeWithScores(ctx, set, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	rows := make([]Row, 0, len(members))
	for i, z := range members {
		row, err := decode(fmt.Sprint(z.Member), z.Score, int(start)+i+1)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (b *Boards) Around(ctx context.Context, season string, mode rules.Mode, board, account string, radius int) ([]Row, bool, error) {
	set, members, err := b.ensure(ctx, season, mode, board)
	if err != nil {
		return nil, false, err
	}
	m, err := b.client.HGet(ctx, members, account).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	position, err := b.client.ZRank(ctx, set, m).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	rows, err := b.rows(ctx, set, max(position-int64(radius), 0), position+int64(radius))
	return rows, err == nil, err
}

func (b *Boards) Size(ctx context.Context, season string, mode rules.Mode, board string) (int64, error) {
	set, _, err := b.ensure(ctx, season, mode, board)
	if err != nil {
		return 0, err
	}
	return b.client.ZCard(ctx, set).Result()
}
