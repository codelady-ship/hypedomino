package loadbot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/app"
	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

var ErrBadToken = errors.New("loadbot: token is not <room>/<mode>/<seat>")

type systemClock struct{}

func (systemClock) Now() time.Time                         { return time.Now() }
func (systemClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func Token(roomID uint64, mode rules.Mode, seat rules.Seat) string {
	return strconv.FormatUint(roomID, 10) + "/" + mode.String() + "/" + seat.String()
}

type claim struct {
	room uint64
	mode rules.Mode
	seat rules.Seat
}

func parseToken(token string) (claim, error) {
	parts := strings.Split(token, "/")
	if len(parts) != 3 {
		return claim{}, ErrBadToken
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return claim{}, fmt.Errorf("%w: %v", ErrBadToken, err)
	}
	mode, err := rules.ParseMode(parts[1])
	if err != nil {
		return claim{}, fmt.Errorf("%w: %v", ErrBadToken, err)
	}
	seat, err := rules.ParseSeat(parts[2])
	if err != nil {
		return claim{}, fmt.Errorf("%w: %v", ErrBadToken, err)
	}
	for _, valid := range mode.Seats() {
		if valid == seat {
			return claim{room: id, mode: mode, seat: seat}, nil
		}
	}
	return claim{}, fmt.Errorf("%w: seat %s is not in %s", ErrBadToken, seat, mode)
}

type table struct {
	room   *room.Room
	joined [6]bool
}

type Handler struct {
	rooms  *app.Rooms
	config rules.Config
	bots   bot.Config
	mu     sync.Mutex
	tables map[uint64]*table
	opened uint64
	stores *Stores
}

func NewHandler(config rules.Config, bots bot.Config) *Handler {
	return &Handler{config: config, bots: bots, tables: map[uint64]*table{}}
}

func (h *Handler) Bind(rooms *app.Rooms) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rooms = rooms
}

func (h *Handler) Store(stores *Stores) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stores = stores
}

func (h *Handler) Opened() uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.opened
}

func (h *Handler) Connect(ctx context.Context, conn *gateway.Conn, hello *pb.ClientHello) (gateway.Session, error) {
	c, err := parseToken(hello.GetSessionToken())
	if err != nil {
		return nil, gateway.ErrUnauthenticated
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms == nil {
		return nil, gateway.ErrUnauthenticated
	}
	t, ok := h.tables[c.room]
	if !ok {
		t, err = h.open(c)
		if err != nil {
			return nil, err
		}
	}
	if t.joined[c.seat] {
		return gateway.ResumeRoom(t.room, c.seat, conn, hello.GetLastAckedSequence()), nil
	}
	t.joined[c.seat] = true
	return gateway.JoinRoom(t.room, c.seat, conn), nil
}

func (h *Handler) open(c claim) (*table, error) {
	r, err := h.rooms.Open(room.Options{Mode: c.mode, Config: h.config, Bot: h.bots, BotLevel: bot.Hard, Seed: c.room, Clock: systemClock{}})
	if err != nil {
		return nil, err
	}
	t := &table{room: r}
	h.tables[c.room] = t
	h.opened++
	go h.forget(c, r, h.stores)
	return t, nil
}

func (h *Handler) forget(c claim, r *room.Room, stores *Stores) {
	if stores != nil {
		stores.track(c, r)
	}
	<-r.Done()
	h.mu.Lock()
	if t, ok := h.tables[c.room]; ok && t.room == r {
		delete(h.tables, c.room)
	}
	h.mu.Unlock()
}
