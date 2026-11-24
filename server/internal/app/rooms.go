package app

import (
	"context"
	"errors"
	"sync"

	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var ErrDraining = errors.New("app: not accepting new rooms")

type Rooms struct {
	mu      sync.Mutex
	live    map[*room.Room]context.CancelFunc
	closed  bool
	wg      sync.WaitGroup
	changed chan struct{}
	tune    func(rules.Config) rules.Config
	moves   room.MoveObserver
	emoji   room.EmojiLimit
	quick   func(key string) bool
}

func NewRooms() *Rooms {
	return &Rooms{live: map[*room.Room]context.CancelFunc{}, changed: make(chan struct{}, 1)}
}

func (rs *Rooms) Tune(tune func(rules.Config) rules.Config) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.tune = tune
}

func (rs *Rooms) Open(options room.Options) (*room.Room, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.closed {
		return nil, ErrDraining
	}
	if rs.tune != nil {
		options.Config = rs.tune(options.Config)
	}
	if options.Moves == nil {
		options.Moves = rs.moves
	}
	if options.Emoji == (room.EmojiLimit{}) {
		options.Emoji = rs.emoji
	}
	if options.QuickMessages == nil {
		options.QuickMessages = rs.quick
	}
	r := room.New(options)
	ctx, cancel := context.WithCancel(context.Background())
	rs.live[r] = cancel
	rs.wg.Add(1)
	go rs.run(ctx, cancel, r)
	return r, nil
}

func (rs *Rooms) run(ctx context.Context, cancel context.CancelFunc, r *room.Room) {
	defer rs.wg.Done()
	go r.Run(ctx)
	select {
	case <-r.Finished():
	case <-ctx.Done():
	}
	cancel()
	<-r.Done()
	rs.mu.Lock()
	delete(rs.live, r)
	rs.mu.Unlock()
	select {
	case rs.changed <- struct{}{}:
	default:
	}
}

func (rs *Rooms) Len() int {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return len(rs.live)
}

func (rs *Rooms) Close() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.closed = true
}

func (rs *Rooms) Drain(ctx context.Context) error {
	for rs.Len() > 0 {
		select {
		case <-rs.changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (rs *Rooms) Stop() {
	rs.mu.Lock()
	for _, cancel := range rs.live {
		cancel()
	}
	rs.mu.Unlock()
	rs.wg.Wait()
}

func (rs *Rooms) LimitEmoji(limit room.EmojiLimit) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.emoji = limit
}

func (rs *Rooms) AllowQuickMessages(allowed func(key string) bool) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.quick = allowed
}

func (rs *Rooms) ObserveMoves(moves room.MoveObserver) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.moves = moves
}
