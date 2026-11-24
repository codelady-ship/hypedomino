package room

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	pending chan time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Unix(1_700_000_000, 0)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pending = make(chan time.Time)
	return c.pending
}

func (c *fakeClock) fire() {
	c.mu.Lock()
	pending := c.pending
	c.now = c.now.Add(time.Hour)
	c.mu.Unlock()
	if pending == nil {
		panic("no turn timer armed")
	}
	pending <- c.now
}

type chanSink chan Message

func (s chanSink) Deliver(m Message) { s <- m }

type table struct {
	t     *testing.T
	room  *Room
	clock *fakeClock
	sinks map[rules.Seat]chanSink
	mode  rules.Mode
	stop  context.CancelFunc
}

func newTable(t *testing.T, mode rules.Mode, seed uint64) *table {
	t.Helper()
	clock := newFakeClock()
	config := rules.DefaultConfig()
	room := New(Options{Mode: mode, Config: config, Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: seed, Clock: clock})
	ctx, cancel := context.WithCancel(context.Background())
	go room.Run(ctx)
	tab := &table{t: t, room: room, clock: clock, sinks: map[rules.Seat]chanSink{}, mode: mode, stop: cancel}
	t.Cleanup(func() {
		cancel()
		<-room.Done()
	})
	for _, seat := range mode.Seats() {
		tab.sinks[seat] = make(chanSink, 4096)
		room.Join(seat, tab.sinks[seat])
	}
	for _, seat := range mode.Seats() {
		room.Loading(seat, 100, true)
	}
	return tab
}

func (tab *table) snapshot(seat rules.Seat) View {
	reply := make(chan View, 1)
	tab.room.Snapshot(seat, reply)
	return <-reply
}

func (tab *table) record() replay.Record {
	reply := make(chan replay.Record, 1)
	tab.room.Record(reply)
	return <-reply
}

func (tab *table) inbox(seat rules.Seat) []Message {
	tab.snapshot(tab.mode.Seats()[0])
	var messages []Message
	for {
		select {
		case m := <-tab.sinks[seat]:
			messages = append(messages, m)
		default:
			return messages
		}
	}
}

func (tab *table) settle() {
	for _, seat := range tab.mode.Seats() {
		tab.inbox(seat)
	}
}

func (tab *table) view(seat rules.Seat) View {
	tab.t.Helper()
	messages := tab.inbox(seat)
	if len(messages) == 0 {
		tab.t.Fatalf("%s received nothing", seat)
	}
	return messages[len(messages)-1].View
}

func (tab *table) rejection(seat rules.Seat) (Message, bool) {
	for _, m := range tab.inbox(seat) {
		if m.Rejected {
			return m, true
		}
	}
	return Message{}, false
}
func (tab *table) humanStep(sequences map[rules.Seat]uint64) bool {
	tab.t.Helper()
	tab.settle()
	var seat rules.Seat
	var view View
	for _, candidate := range tab.mode.Seats() {
		v := tab.snapshot(candidate)
		if v.State == StateFinished {
			return false
		}
		if v.Controllers[candidate] == rules.ControllerBot {
			continue
		}
		if (v.State == StatePlaying && v.Current == candidate) || (v.State == StateOpening && (v.HasForced && v.ForcedOpening.Seat == candidate || !v.HasForced && candidate.Team() == v.OpeningTeam)) {
			seat, view = candidate, v
			break
		}
	}
	if view.State == StateWaiting || view.State == StateLoading {
		return false
	}
	sequences[seat]++
	intent := replay.Intent{Seat: seat}
	switch {
	case view.State == StateOpening && view.HasForced:
		intent.Kind, intent.Tile = replay.KindOpen, view.ForcedOpening.Tile
	case view.State == StateOpening:
		intent.Kind, intent.Tile = replay.KindOpen, view.Hand[0]
	case len(view.LegalMoves) > 0 && view.TurnPlacements > 0 && sequences[seat]%2 == 0:
		intent.Kind = replay.KindEndTurn
	case len(view.LegalMoves) > 0:
		intent.Kind, intent.Tile, intent.End = replay.KindPlace, view.LegalMoves[0].Tile, view.LegalMoves[0].End
	case view.Mode == rules.Mode1v1 && view.BoneyardSize > 1:
		intent.Kind = replay.KindDraw
	default:
		intent.Kind = replay.KindPass
	}
	tab.room.Intent(seat, sequences[seat], intent)
	if m, rejected := tab.rejection(seat); rejected {
		tab.t.Fatalf("%s: %v (view %+v)", seat, m.Reason, view)
	}
	return true
}

func TestViewsHideOtherHands(t *testing.T) {
	tab := newTable(t, rules.Mode2v2, 1)
	for _, seat := range tab.mode.Seats() {
		v := tab.view(seat)
		if v.State != StateOpening || len(v.Hand) != 7 || v.Seat != seat || !v.HasForced {
			t.Fatalf("%s: %+v", seat, v)
		}
		for _, other := range tab.mode.Seats() {
			if v.HandSizes[other] != 7 {
				t.Fatalf("%s sees hand size %d for %s", seat, v.HandSizes[other], other)
			}
		}
		if v.TurnDeadline != tab.clock.Now().Add(20*time.Second) {
			t.Fatalf("%s deadline %v", seat, v.TurnDeadline)
		}
	}
	own := map[rules.Seat][]rules.Tile{}
	for _, seat := range tab.mode.Seats() {
		own[seat] = tab.snapshot(seat).Hand
	}
	for _, seat := range tab.mode.Seats() {
		for _, other := range tab.mode.Seats() {
			if seat == other {
				continue
			}
			for _, tile := range own[other] {
				for _, mine := range own[seat] {
					if tile == mine {
						t.Fatalf("%s and %s both see %s", seat, other, tile)
					}
				}
			}
		}
	}
}

func TestIllegalIntentsAreRejectedWithTheirSequence(t *testing.T) {
	tab := newTable(t, rules.Mode2v2, 2)
	tab.settle()
	v := tab.snapshot(rules.SeatNorth)
	wrong := v.ForcedOpening.Seat.Next()
	tab.room.Intent(wrong, 1, replay.Intent{Kind: replay.KindOpen, Tile: v.ForcedOpening.Tile})
	m, _ := tab.rejection(wrong)
	if !m.Rejected || m.Sequence != 1 || !errors.Is(m.Reason, rules.ErrWrongOpener) || m.View.Acked != 0 {
		t.Fatalf("wrong opener: %+v", m)
	}
	tab.room.Intent(v.ForcedOpening.Seat, 1, replay.Intent{Kind: replay.KindOpen, Tile: v.ForcedOpening.Tile})
	if _, rejected := tab.rejection(v.ForcedOpening.Seat); rejected || tab.snapshot(v.ForcedOpening.Seat).Acked != 1 || tab.snapshot(v.ForcedOpening.Seat).State != StatePlaying {
		t.Fatalf("forced opening: %+v", tab.snapshot(v.ForcedOpening.Seat))
	}
	tab.settle()
	tab.room.Intent(v.ForcedOpening.Seat, 1, replay.Intent{Kind: replay.KindPass})
	tab.room.Intent(v.ForcedOpening.Seat, 2, replay.Intent{Kind: replay.KindPass})
	m, _ = tab.rejection(v.ForcedOpening.Seat)
	if !m.Rejected || m.Sequence != 2 || !errors.Is(m.Reason, rules.ErrNotYourTurn) {
		t.Fatalf("stale sequence was answered or pass out of turn accepted: %+v", m)
	}
	tab.room.Intent(rules.SeatY, 1, replay.Intent{Kind: replay.KindPass})
	if len(tab.sinks[rules.SeatY]) != 0 {
		t.Fatal("a seat outside the mode received a message")
	}
	current := tab.snapshot(rules.SeatNorth).Current
	tab.room.Intent(current, 1, replay.Intent{Kind: replay.KindPlace, Tile: rules.MustTile(6, 6), End: rules.End(9)})
	if m, _ := tab.rejection(current); !m.Rejected || !(errors.Is(m.Reason, rules.ErrTileNotHeld) || errors.Is(m.Reason, rules.ErrIllegalPlacement)) {
		t.Fatalf("bad placement: %+v", m)
	}
	if record := tab.record(); len(record.Intents) != 1 {
		t.Fatalf("record holds %d intents, want the opening only", len(record.Intents))
	}
}

func TestHumansPlayAWholeMatchAndTheRecordVerifies(t *testing.T) {
	for _, mode := range []rules.Mode{rules.Mode1v1, rules.Mode2v2} {
		tab := newTable(t, mode, 3)
		sequences := map[rules.Seat]uint64{}
		for steps := 0; tab.humanStep(sequences); steps++ {
			if steps > 20000 {
				t.Fatalf("%s: match does not end", mode)
			}
		}
		tab.settle()
		v := tab.snapshot(tab.mode.Seats()[0])
		if v.State != StateFinished || !v.HasWinner || v.Scores[v.Winner] < rules.DefaultConfig().TargetScore {
			t.Fatalf("%s: final view %+v", mode, v)
		}
		record := tab.record()
		if !record.HasResult || record.Result.Winner != v.Winner {
			t.Fatalf("%s: record %+v", mode, record.Result)
		}
		if err := replay.Verify(record); err != nil {
			t.Fatalf("%s: %v", mode, err)
		}
		tab.room.Intent(v.Current, 999, replay.Intent{Kind: replay.KindPass})
		if m, _ := tab.rejection(v.Current); !m.Rejected || !errors.Is(m.Reason, ErrFinished) {
			t.Fatalf("%s: intent after the end: %+v", mode, m)
		}
	}
}

func TestViewsShowTheTilesLeftOnlyBetweenHandsAndTheLockedTeams(t *testing.T) {
	config := rules.DefaultConfig()
	tab := newTable(t, rules.Mode2v2, 3)
	sequences := map[rules.Seat]uint64{}
	seat := tab.mode.Seats()[0]
	between, playing := 0, 0
	for steps := 0; tab.humanStep(sequences); steps++ {
		if steps > 20000 {
			t.Fatal("match does not end")
		}
		v := tab.snapshot(seat)
		if v.TargetScore != config.TargetScore || v.LockThreshold != config.HandPointsLockThreshold {
			t.Fatalf("view marks %d and %d, config says %d and %d", v.TargetScore, v.LockThreshold, config.TargetScore, config.HandPointsLockThreshold)
		}
		if v.State == StateOpening {
			for _, team := range []rules.Team{rules.TeamA, rules.TeamB} {
				if v.Locked[team] != (v.Scores[team] >= config.HandPointsLockThreshold) {
					t.Fatalf("team %s locked %v with %v at the start of the hand", team, v.Locked[team], v.Scores)
				}
			}
		}
		if !v.HasLastResult {
			continue
		}
		if v.State == StatePlaying {
			playing++
			if v.LastEnd.Left != nil {
				t.Fatalf("a view during play shows the tiles left of the last hand: %+v", v.LastEnd)
			}
			continue
		}
		between++
		pips := rules.TeamPoints{}
		for _, s := range tab.mode.Seats() {
			pips[s.Team()] += v.LastEnd.Left[s].Pips()
		}
		if pips != v.LastResult.Pips || v.LastEnd.Status == rules.HandContinues {
			t.Fatalf("tiles left count %v, result says %v, ending %s", pips, v.LastResult.Pips, v.LastEnd.Status)
		}
	}
	if between == 0 || playing == 0 {
		t.Fatalf("saw %d views between hands and %d during play after a hand", between, playing)
	}
}

func TestTimeoutsHandTheSeatToTheBotAndReconnectTakesItBack(t *testing.T) {
	tab := newTable(t, rules.Mode2v2, 4)
	tab.settle()
	opener := tab.snapshot(rules.SeatNorth).ForcedOpening.Seat
	tab.clock.fire()
	v := tab.view(opener)
	if v.State != StatePlaying || v.Controllers[opener] != rules.ControllerPlayer || v.HandSizes[opener] != 6 {
		t.Fatalf("after the opening timeout: %+v", v)
	}
	timeouts := map[rules.Seat]int{opener: 1}
	sequences := map[rules.Seat]uint64{}
	for tab.snapshot(opener).Controllers[opener] == rules.ControllerPlayer {
		tab.settle()
		v := tab.snapshot(opener)
		if v.State == StateFinished {
			t.Fatal("match ended before the AFK mark")
		}
		if v.State == StatePlaying && v.Current == opener {
			tab.clock.fire()
			tab.settle()
			timeouts[opener]++
			continue
		}
		if !tab.humanStep(sequences) {
			t.Fatal("nobody could move")
		}
	}
	if timeouts[opener] != 3 {
		t.Fatalf("seat went to the bot after %d timeouts", timeouts[opener])
	}
	tab.settle()
	tab.room.Intent(opener, 100, replay.Intent{Kind: replay.KindPass})
	if m, _ := tab.rejection(opener); !m.Rejected || !errors.Is(m.Reason, ErrSeatByBot) {
		t.Fatalf("an AFK seat moved itself: %+v", m)
	}
	tab.room.Reconnect(opener, tab.sinks[opener])
	if v := tab.view(opener); v.Controllers[opener] != rules.ControllerPlayer || v.Presence[opener] != (rules.PresenceReport{Timeouts: 3, MarkedAfk: true}) || v.Timeouts[opener] != 0 {
		t.Fatalf("reconnect did not take the seat back: %+v", v)
	}
}

func TestALeaverIsPlayedByTheBotUntilTheyComeBack(t *testing.T) {
	tab := newTable(t, rules.Mode1v1, 5)
	tab.settle()
	v := tab.snapshot(rules.SeatX)
	leaver, other := v.ForcedOpening.Seat, v.ForcedOpening.Seat.Next()
	tab.room.Leave(leaver)
	v = tab.view(other)
	if v.Controllers[leaver] != rules.ControllerBot || v.State != StatePlaying || v.Current != other {
		t.Fatalf("after leaving: %+v", v)
	}
	tab.room.Reconnect(leaver, tab.sinks[leaver])
	if v := tab.view(leaver); v.Controllers[leaver] != rules.ControllerPlayer {
		t.Fatal("a returning leaver did not take the seat back")
	}
	sequences := map[rules.Seat]uint64{}
	for steps := 0; tab.humanStep(sequences); steps++ {
		if steps > 20000 {
			t.Fatal("match does not end")
		}
	}
	tab.settle()
	final := tab.snapshot(other)
	if final.State != StateFinished || !final.HasWinner {
		t.Fatalf("final view %+v", final)
	}
	if err := replay.Verify(tab.record()); err != nil {
		t.Fatal(err)
	}
	recorded := tab.record()
	tab.stop()
	if kept, ok := tab.room.FinalRecord(); !ok || !kept.HasResult || len(kept.Intents) != len(recorded.Intents) {
		t.Fatalf("final record %v %+v", ok, kept.Result)
	}
	if final.Presence[leaver] != (rules.PresenceReport{Left: true}) || final.Presence[other] != (rules.PresenceReport{}) {
		t.Fatalf("presence reports %+v", final.Presence)
	}
}

func TestConcurrentIntentsAreSerialised(t *testing.T) {
	tab := newTable(t, rules.Mode2v2, 6)
	tab.settle()
	opener := tab.snapshot(rules.SeatNorth).ForcedOpening
	tab.room.Intent(opener.Seat, 1, replay.Intent{Kind: replay.KindOpen, Tile: opener.Tile})
	tab.settle()
	var wg sync.WaitGroup
	for _, seat := range tab.mode.Seats() {
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(seat rules.Seat, sequence uint64) {
				defer wg.Done()
				tab.room.Intent(seat, sequence, replay.Intent{Kind: replay.KindPass})
				reply := make(chan View, 1)
				tab.room.Snapshot(seat, reply)
				<-reply
			}(seat, uint64(i+2))
		}
	}
	wg.Wait()
	tab.settle()
	v := tab.snapshot(rules.SeatNorth)
	if v.State != StatePlaying || v.Board.Line == nil || len(v.Board.Line) != 1 {
		t.Fatalf("passes changed the board: %+v", v)
	}
	if record := tab.record(); len(record.Intents) != 1 {
		t.Fatalf("record holds %d intents", len(record.Intents))
	}
}

func TestJoiningATakenSeatIsRefused(t *testing.T) {
	tab := newTable(t, rules.Mode1v1, 7)
	tab.settle()
	extra := make(chanSink, 1)
	tab.room.Join(rules.SeatX, extra)
	if m := <-extra; !m.Rejected || !errors.Is(m.Reason, ErrSeatTaken) {
		t.Fatalf("second join: %+v", m)
	}
	tab.room.Join(rules.SeatNorth, extra)
	if m := <-extra; !m.Rejected || !errors.Is(m.Reason, ErrSeatTaken) {
		t.Fatalf("join outside the mode: %+v", m)
	}
}

func TestIntentsBeforeTheMatchStartsAreRefused(t *testing.T) {
	clock := newFakeClock()
	room := New(Options{Mode: rules.Mode2v2, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), Seed: 8, Clock: clock})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	sink := make(chanSink, 16)
	room.Join(rules.SeatNorth, sink)
	if m := <-sink; m.View.State != StateLoading {
		t.Fatalf("after the first join: %+v", m.View)
	}
	room.Intent(rules.SeatNorth, 1, replay.Intent{Kind: replay.KindPass})
	if m := <-sink; !m.Rejected || !errors.Is(m.Reason, ErrNotStarted) {
		t.Fatalf("intent while waiting: %+v", m)
	}
	reply := make(chan replay.Record, 1)
	room.Record(reply)
	if record := <-reply; record.Intents != nil || record.HasResult {
		t.Fatalf("record before start: %+v", record)
	}
}

func TestADisconnectKeepsTheSeatUntilTheSameSinkIsReplaced(t *testing.T) {
	tab := newTable(t, rules.Mode1v1, 6)
	tab.settle()
	v := tab.snapshot(rules.SeatX)
	gone, other := v.ForcedOpening.Seat, v.ForcedOpening.Seat.Next()
	tab.room.Disconnect(gone, tab.sinks[gone])
	tab.snapshot(other)
	tab.clock.fire()
	v = tab.view(other)
	if v.Controllers[gone] != rules.ControllerPlayer || v.State != StatePlaying || v.Current != other {
		t.Fatalf("after a disconnect and one timeout: %+v", v)
	}
	if !v.Disconnected[gone] || v.Disconnected[other] || v.Timeouts[gone] != 1 || v.Timeouts[other] != 0 || v.AfkTimeouts != rules.DefaultConfig().AfkConsecutiveTimeouts {
		t.Fatalf("the view does not show the disconnect and its timeout: disconnected %v, timeouts %v, afk at %d", v.Disconnected, v.Timeouts, v.AfkTimeouts)
	}
	if messages := tab.inbox(gone); len(messages) != 0 {
		t.Fatalf("a disconnected seat received %d messages", len(messages))
	}
	stale := make(chanSink, 1)
	tab.room.Disconnect(other, stale)
	tab.room.Intent(other, 1, replay.Intent{Kind: replay.KindPass})
	if m, rejected := tab.rejection(other); !rejected || !errors.Is(m.Reason, rules.ErrMustPlace) {
		t.Fatalf("a stale sink dropped the live one: %+v", m)
	}
	fresh := make(chanSink, 64)
	tab.sinks[gone] = fresh
	tab.room.Join(gone, fresh)
	if v := tab.view(gone); v.Controllers[gone] != rules.ControllerPlayer || len(v.Hand) != 6 || v.Disconnected[gone] {
		t.Fatalf("joining the seat again: %+v", v)
	}
	tab.room.Leave(gone)
	if v := tab.view(other); v.Controllers[gone] != rules.ControllerBot {
		t.Fatalf("after leaving: %+v", v)
	}
	tab.room.Join(gone, fresh)
	if v := tab.view(gone); v.Controllers[gone] != rules.ControllerPlayer {
		t.Fatal("a returning leaver did not take the seat back by joining")
	}
}

func TestTheCreatorKicksASeatAndAdmitsItAgain(t *testing.T) {
	tab := newTable(t, rules.Mode2v2, 8)
	tab.settle()
	v := tab.snapshot(rules.SeatNorth)
	kicked, other := v.ForcedOpening.Seat, v.ForcedOpening.Seat.Next()
	tab.room.Kick(kicked)
	v = tab.view(other)
	if v.Controllers[kicked] != rules.ControllerBot || v.State != StatePlaying {
		t.Fatalf("after the kick: %+v", v)
	}
	back := make(chanSink, 16)
	tab.room.Reconnect(kicked, back)
	if m := <-back; !m.Rejected || !errors.Is(m.Reason, ErrBarred) {
		t.Fatalf("a kicked player reconnected: %+v", m)
	}
	tab.room.Join(kicked, back)
	if m := <-back; !m.Rejected || !errors.Is(m.Reason, ErrBarred) {
		t.Fatalf("a kicked player joined: %+v", m)
	}
	tab.room.Admit(kicked)
	tab.room.Join(kicked, back)
	if m := <-back; m.Rejected || m.View.Controllers[kicked] != rules.ControllerPlayer {
		t.Fatalf("after admit: %+v", m)
	}
	tab.room.Kick(rules.SeatX)
	if tab.snapshot(other).Controllers[rules.SeatX] != rules.ControllerPlayer {
		t.Fatal("kicking a seat outside the mode changed something")
	}
	if tab.room.Kind() != KindRanked {
		t.Fatalf("default kind %s", tab.room.Kind())
	}
}

func snapshotOf(r *Room, seat rules.Seat) View {
	reply := make(chan View, 1)
	r.Snapshot(seat, reply)
	return <-reply
}

func TestTheViewCarriesTheTurnLengthAndWarningOfTheMatchConfig(t *testing.T) {
	clock := newFakeClock()
	config := rules.DefaultConfig()
	config.TurnSeconds = 30
	config.TurnWarningSeconds = 7
	room := New(Options{Mode: rules.Mode1v1, Config: config, Bot: bot.DefaultConfig(), Seed: 9, Clock: clock})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	for _, seat := range rules.Mode1v1.Seats() {
		room.Join(seat, make(chanSink, 64))
		room.Loading(seat, 100, true)
	}
	v := snapshotOf(room, rules.SeatX)
	if v.State != StateOpening || v.TurnSeconds != 30 || v.TurnWarning != 7 {
		t.Fatalf("view turn length %d and warning %d in state %v", v.TurnSeconds, v.TurnWarning, v.State)
	}
	if !v.TurnDeadline.Equal(clock.Now().Add(30 * time.Second)) {
		t.Fatalf("deadline %v is not one turn length after %v", v.TurnDeadline, clock.Now())
	}
}

func TestLoadingShowsTheRosterProgressAndDeadline(t *testing.T) {
	clock := newFakeClock()
	roster := []Player{{Seat: rules.SeatX, Nickname: "ayla", Avatar: "avatar.nar", RankLabel: "Silver II"}, {Seat: rules.SeatY, Nickname: "bot", Bot: true}}
	room := New(Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), Seed: 9, Clock: clock, Roster: roster})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	sink := make(chanSink, 16)
	room.Join(rules.SeatX, sink)
	m := <-sink
	if m.View.Disconnected[rules.SeatX] || m.View.Disconnected[rules.SeatY] {
		t.Fatalf("a joined human or a bot seat shows as disconnected: %v", m.View.Disconnected)
	}
	if m.View.State != StateLoading || !m.View.LoadingDeadline.Equal(clock.Now().Add(rules.LoadingLimit(rules.DefaultConfig()))) || !reflect.DeepEqual(m.View.Roster, roster) {
		t.Fatalf("loading view %+v", m.View)
	}
	room.Loading(rules.SeatX, 37, false)
	if m := <-sink; m.View.Loading[rules.SeatX] != 30 || m.View.Ready[rules.SeatX] {
		t.Fatalf("after reporting progress: %+v", m.View)
	}
	room.Loading(rules.SeatX, 100, true)
	if m := <-sink; m.View.State != StateLoading || m.View.Loading[rules.SeatX] != 100 || !m.View.Ready[rules.SeatX] {
		t.Fatalf("after reporting ready: %+v", m.View)
	}
	room.Loading(rules.SeatX, 50, false)
	room.Intent(rules.SeatX, 1, replay.Intent{Kind: replay.KindPass})
	if m := <-sink; !m.Rejected || !errors.Is(m.Reason, ErrNotStarted) {
		t.Fatalf("ready was withdrawn or a hand was dealt early: %+v", m)
	}
}

func TestEveryoneReadyDealsBeforeTheDeadline(t *testing.T) {
	tab := newTable(t, rules.Mode2v2, 12)
	tab.settle()
	v := tab.snapshot(rules.SeatNorth)
	if v.State != StateOpening || v.LoadingDeadline != (time.Time{}) {
		t.Fatalf("after everyone reported ready: %+v", v)
	}
	for _, seat := range rules.Mode2v2.Seats() {
		if v.Controllers[seat] != rules.ControllerPlayer || !v.Ready[seat] || v.Loading[seat] != 100 {
			t.Fatalf("%s after the deal: %+v", seat, v)
		}
	}
}

func TestTheDeadlineDealsAndTheBotPlaysAbsentSeatsUntilTheyArrive(t *testing.T) {
	clock := newFakeClock()
	room := New(Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 9, Clock: clock, ReconnectGrace: 2 * time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	x, y := make(chanSink, 4096), make(chanSink, 4096)
	room.Join(rules.SeatX, x)
	room.Loading(rules.SeatX, 100, true)
	if v := snapshotOf(room, rules.SeatX); v.State != StateLoading {
		t.Fatalf("one ready seat dealt the hand: %+v", v)
	}
	clock.fire()
	v := snapshotOf(room, rules.SeatX)
	if v.State == StateLoading || v.State == StateFinished || v.Controllers[rules.SeatY] != rules.ControllerBot || v.Presence[rules.SeatY] != (rules.PresenceReport{ByBot: true}) {
		t.Fatalf("after the loading deadline: %+v", v)
	}
	room.Join(rules.SeatY, y)
	if v := (<-y).View; v.Controllers[rules.SeatY] != rules.ControllerPlayer || v.Presence[rules.SeatY] != (rules.PresenceReport{}) {
		t.Fatalf("an arriving player did not take the seat: %+v", v)
	}
}

func TestASeatStillAbsentAfterTheGraceIsLeft(t *testing.T) {
	clock := newFakeClock()
	room := New(Options{Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 9, Clock: clock, ReconnectGrace: 2 * time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go room.Run(ctx)
	x := make(chanSink, 4096)
	room.Join(rules.SeatX, x)
	room.Loading(rules.SeatX, 100, true)
	snapshotOf(room, rules.SeatX)
	clock.fire()
	snapshotOf(room, rules.SeatX)
	clock.fire()
	if v := snapshotOf(room, rules.SeatX); v.Presence[rules.SeatY].Left {
		t.Fatalf("left before the grace: %+v", v)
	}
	clock.fire()
	if v := snapshotOf(room, rules.SeatX); v.Presence[rules.SeatY] != (rules.PresenceReport{ByBot: true, Left: true}) {
		t.Fatalf("after the grace: %+v", v.Presence[rules.SeatY])
	}
}
