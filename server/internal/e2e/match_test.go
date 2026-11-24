package e2e

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/app"
	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/session"
)

var fullMatchSeeds = []uint64{1, 2, 3}

func TestS1OneVersusOneFullMatch(t *testing.T) {
	for _, seed := range fullMatchSeeds {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) { playFullMatch(t, rules.Mode1v1, seed) })
	}
}

func TestS2TwoVersusTwoFullMatch(t *testing.T) {
	for _, seed := range fullMatchSeeds {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) { playFullMatch(t, rules.Mode2v2, seed) })
	}
}

type frozenClock struct{ now time.Time }

func (c frozenClock) Now() time.Time                       { return c.now }
func (c frozenClock) After(time.Duration) <-chan time.Time { return nil }

type expiredClock struct{ frozenClock }

func (c expiredClock) After(time.Duration) <-chan time.Time {
	due := make(chan time.Time, 1)
	due <- c.now
	return due
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

type server struct {
	url      string
	sessions *session.Manager
	logs     *lockedBuffer
	stop     func()
}

func startServer(t *testing.T, clock frozenClock, seed uint64) *server {
	t.Helper()
	config := app.DefaultConfig()
	config.ListenAddr = "127.0.0.1:0"
	config.DrainTimeout = time.Second
	config.Gateway.IntentsPerSecond = 1000
	config.Gateway.IntentBurst = 1000
	logs := &lockedBuffer{}
	sessions := session.New(config.Session, expiredClock{clock}, rules.NewSeededRandom(seed))
	a := app.New(config, app.NewLogger(logs, "info"), sessions)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()
	s := &server{sessions: sessions, logs: logs}
	s.stop = func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("server stopped with %v", err)
		}
	}
	select {
	case addr := <-a.Addr():
		s.url = "ws://" + addr + "/ws"
	case err := <-done:
		t.Fatalf("server did not start: %v", err)
	}
	return s
}

type seatLog struct {
	seat    rules.Seat
	final   *pb.MatchView
	hands   map[uint32]map[rules.Tile]bool
	sent    int
	traffic int
	err     error
}

func fullRoster(mode rules.Mode) []room.Player {
	var roster []room.Player
	for _, seat := range mode.Seats() {
		roster = append(roster, room.Player{
			Seat: seat, Nickname: strings.Repeat("ş", 16), Frame: "frame.golden_dragon_mythic",
			Avatar: "avatar.neon_tiger_legendary", Tile: "tile.storm_sapphire_legendary", RankLabel: "Grandmaster 1000 LP",
			Emblems: []string{"emblem.season_1_challenger", "emblem.season_2_challenger", "emblem.season_3_challenger"},
		})
	}
	return roster
}

func playFullMatch(t *testing.T, mode rules.Mode, seed uint64) {
	baseline := runtime.NumGoroutine()
	clock := frozenClock{now: time.UnixMilli(1761480000000)}
	config := rules.DefaultConfig()
	srv := startServer(t, clock, seed)
	r := room.New(room.Options{Kind: room.KindCasual, Mode: mode, Config: config, Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: seed, Clock: clock, Table: "table.night_market_deluxe", Roster: fullRoster(mode)})
	roomCtx, stopRoom := context.WithCancel(context.Background())
	go r.Run(roomCtx)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	logs := make([]*seatLog, len(mode.Seats()))
	var players sync.WaitGroup
	for index, seat := range mode.Seats() {
		token := srv.sessions.Create(fmt.Sprintf("e2e-%d-%s", seed, seat))
		if err := srv.sessions.Seat(token, r, seat); err != nil {
			t.Fatalf("seed %d: seating %s: %v", seed, seat, err)
		}
		logs[index] = &seatLog{seat: seat, hands: map[uint32]map[rules.Tile]bool{}}
		brain := botclient.NewBrain(bot.Hard, bot.DefaultConfig(), config, rules.NewSeededRandom(seed<<8|uint64(index)))
		players.Add(1)
		go func() {
			defer players.Done()
			if logs[index].err = playSeat(ctx, srv.url, token, brain, logs[index]); logs[index].err != nil {
				cancel()
			}
		}()
	}
	players.Wait()
	for _, log := range logs {
		if log.err != nil {
			t.Errorf("seed %d: %s: %v", seed, log.seat, log.err)
		}
	}
	if t.Failed() {
		stopRoom()
		srv.stop()
		t.FailNow()
	}

	reply := make(chan replay.Record, 1)
	r.Record(reply)
	record := <-reply
	stopRoom()
	<-r.Done()
	srv.stop()

	assertNoForeignTiles(t, seed, logs)
	assertReplayMatches(t, seed, record, logs)
	assertDominoFinish(t, seed, record, logs[0].final)
	assertTrafficBudget(t, seed, mode, logs)
	assertCleanShutdown(t, seed, srv.logs.String(), baseline)
}

func playSeat(ctx context.Context, url, token string, brain *botclient.Brain, log *seatLog) error {
	c, err := botclient.Dial(ctx, url, botclient.Options{Token: token, AppVersion: "1.0.0", Platform: pb.Platform_PLATFORM_ANDROID})
	if err != nil {
		return err
	}
	defer c.Close()
	for {
		frame, err := c.Read()
		if err != nil {
			return err
		}
		if rejected := frame.GetError(); rejected != nil {
			return fmt.Errorf("intent %d refused: %s", rejected.GetRejectedSequence(), rejected.GetCode())
		}
		view := frame.GetMatchView()
		if view == nil {
			continue
		}
		log.observe(view)
		if view.GetState() == pb.MatchState_MATCH_STATE_FINISHED {
			log.final = view
			log.sent = c.Sent
			log.traffic = c.SentBytes + c.ReceivedBytes
			return nil
		}
		if botclient.NeedsReady(view) {
			if err := c.Ready(); err != nil {
				return err
			}
			continue
		}
		if botclient.Idle(view) || frame.GetAckedSequence() != c.Sequence {
			continue
		}
		intent, err := brain.Decide(view)
		if err != nil {
			return fmt.Errorf("hand %d: %w", view.GetHandsPlayed(), err)
		}
		if err := c.Send(intent); err != nil {
			return err
		}
	}
}

func (log *seatLog) observe(view *pb.MatchView) {
	seen := log.hands[view.GetHandsPlayed()]
	if seen == nil {
		seen = map[rules.Tile]bool{}
		log.hands[view.GetHandsPlayed()] = seen
	}
	for _, tile := range view.GetHand() {
		seen[rules.Tile{Low: uint8(tile.GetLow()), High: uint8(tile.GetHigh())}] = true
	}
}

func assertNoForeignTiles(t *testing.T, seed uint64, logs []*seatLog) {
	t.Helper()
	for i, first := range logs {
		for _, second := range logs[i+1:] {
			for hand, tiles := range first.hands {
				for tile := range tiles {
					if second.hands[hand][tile] {
						t.Errorf("seed %d E-1: hand %d tile %s was shown to both %s and %s", seed, hand, tile, first.seat, second.seat)
					}
				}
			}
		}
	}
}

func assertReplayMatches(t *testing.T, seed uint64, record replay.Record, logs []*seatLog) {
	t.Helper()
	if err := replay.Verify(record); err != nil {
		t.Fatalf("seed %d E-2: %v", seed, err)
	}
	sent := 0
	for _, log := range logs {
		sent += log.sent
		for _, seat := range log.final.GetSeats() {
			if seat.GetController() != pb.Controller_CONTROLLER_PLAYER {
				t.Errorf("seed %d: %s ended controlled by %v, every intent must come from the bot clients", seed, seat.GetSeat(), seat.GetController())
			}
		}
		scores := log.final.GetScores()
		got := rules.TeamPoints{int(scores.GetTeamA()), int(scores.GetTeamB())}
		if got != record.Result.Scores || rules.Team(log.final.GetWinner()-1) != record.Result.Winner {
			t.Errorf("seed %d E-2: %s saw scores %v winner %v, replay has %v winner %v", seed, log.seat, got, log.final.GetWinner(), record.Result.Scores, record.Result.Winner)
		}
	}
	if len(record.Intents) != sent {
		t.Errorf("seed %d E-2: record holds %d intents, clients sent %d", seed, len(record.Intents), sent)
	}
}

func assertDominoFinish(t *testing.T, seed uint64, record replay.Record, final *pb.MatchView) {
	t.Helper()
	m, err := replay.Replay(record)
	if err != nil {
		t.Fatalf("seed %d E-3: %v", seed, err)
	}
	winner, finished := m.Winner()
	if !finished || !record.Result.HasWinner {
		t.Fatalf("seed %d E-3: replayed match is not finished", seed)
	}
	scores, target := m.Scores(), m.Config().TargetScore
	if scores[winner] < target || scores[winner] <= scores[winner.Other()] {
		t.Errorf("seed %d E-3: winner %s has %v against target %d", seed, winner, scores, target)
	}
	domino := false
	for _, seat := range m.Mode().Seats() {
		domino = domino || len(m.Hand(seat)) == 0
	}
	if !domino {
		t.Errorf("seed %d E-3: the last hand did not end in a domino", seed)
	}
	t.Logf("seed %d: %s won %v after %d hands, %d intents", seed, winner, scores, m.HandsPlayed(), len(record.Intents))
}

const matchTrafficBudgetBytes = 200_000

func assertTrafficBudget(t *testing.T, seed uint64, mode rules.Mode, logs []*seatLog) {
	t.Helper()
	for _, log := range logs {
		if log.traffic > matchTrafficBudgetBytes {
			t.Errorf("seed %d E-7: %s moved %d bytes in a %s match, budget %d", seed, log.seat, log.traffic, mode, matchTrafficBudgetBytes)
		}
		t.Logf("seed %d: %s %s match traffic %d bytes", seed, log.seat, mode, log.traffic)
	}
}

func assertCleanShutdown(t *testing.T, seed uint64, logs string, baseline int) {
	t.Helper()
	if strings.Contains(logs, `"level":"ERROR"`) {
		t.Errorf("seed %d E-6: server logged an error:\n%s", seed, logs)
	}
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > baseline && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if left := runtime.NumGoroutine() - baseline; left > 0 {
		buffer := make([]byte, 1<<16)
		t.Errorf("seed %d E-6: %d goroutines left after the room closed:\n%s", seed, left, buffer[:runtime.Stack(buffer, true)])
	}
}
