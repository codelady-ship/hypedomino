package custom

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type counter struct{ next uint64 }

func (c *counter) Uint64() uint64 {
	c.next++
	return c.next * 0x9e3779b97f4a7c15
}

type sameCode struct{}

func (sameCode) Uint64() uint64 { return 7 }

type realClock struct{}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type chanSink chan room.Message

func (s chanSink) Deliver(m room.Message) { s <- m }

type eventSink chan Event

func (s eventSink) Deliver(e Event) { s <- e }

type opened struct {
	table  *room.Room
	cancel context.CancelFunc
}

func opener(t *testing.T) (Opener, *[]opened) {
	t.Helper()
	var tables []opened
	return func(options room.Options, _ []SeatState) (*room.Room, error) {
		options.Config, options.Bot, options.BotLevel, options.Clock = rules.DefaultConfig(), bot.DefaultConfig(), bot.Hard, realClock{}
		r := room.New(options)
		ctx, cancel := context.WithCancel(context.Background())
		go r.Run(ctx)
		t.Cleanup(func() { cancel(); <-r.Done() })
		tables = append(tables, opened{r, cancel})
		return r, nil
	}, &tables
}

func player(account string) Member {
	return Member{Account: account}
}

func accounts(state State) []string {
	var out []string
	for _, seat := range state.Seats {
		out = append(out, seat.Account)
	}
	return out
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func refused(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestCodesAreShortUnambiguousAndUnique(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	seen := map[string]bool{}
	for index := 0; index < 500; index++ {
		r, err := lobby.Create(player("creator"))
		must(t, err)
		code := r.Code()
		if len(code) != CodeLength || strings.ContainsAny(code, "01IO") || strings.ToUpper(code) != code || seen[code] {
			t.Fatalf("code %q", code)
		}
		seen[code] = true
		if found, err := lobby.Find(code); err != nil || found != r {
			t.Fatalf("find %q: %v", code, err)
		}
	}
	_, err := lobby.Find("NOPE42")
	refused(t, err, ErrUnknownCode)
	if lobby.Len() != 500 {
		t.Fatalf("lobby holds %d rooms", lobby.Len())
	}
}

func TestCreateRetriesACollidingCode(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(sameCode{}, open)
	first, _ := lobby.Create(player("a"))
	first.Close()
	second, err := lobby.Create(player("b"))
	if err != nil || second.Code() != first.Code() || lobby.Len() != 1 {
		t.Fatalf("after closing, the code was not reused: %v", err)
	}
}

func TestLeadershipPassesToTheEarliestJoinerAndTheLastLeaverClosesTheRoom(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	r, _ := lobby.Create(player("ana"))
	for _, friend := range []string{"ben", "cem", "dan"} {
		must(t, r.Join(player(friend)))
	}
	refused(t, r.Join(player("eve")), ErrFull)
	refused(t, r.Join(player("ben")), ErrAlreadyInRoom)
	refused(t, r.Teams("ben", rules.Mode2v2), ErrNotLeader)
	refused(t, r.Leave("zed"), ErrNotInRoom)
	must(t, r.Leave("ana"))
	if state := r.State(); state.Leader != "ben" || !slices.Equal(state.Members, []string{"ben", "cem", "dan"}) {
		t.Fatalf("after the creator left: %+v", state)
	}
	refused(t, r.Teams("ana", rules.Mode2v2), ErrNotLeader)
	must(t, r.Leave("cem"))
	must(t, r.Leave("ben"))
	if leader := r.State().Leader; leader != "dan" {
		t.Fatalf("leader %q", leader)
	}
	must(t, r.Teams("dan", rules.Mode1v1))
	must(t, r.Leave("dan"))
	if !r.State().Closed || lobby.Len() != 0 {
		t.Fatalf("room still open: %+v", r.State())
	}
	refused(t, r.Join(player("eve")), ErrUnknownCode)
	_, err := lobby.Find(r.Code())
	refused(t, err, ErrUnknownCode)
}

func TestTheLeaderSplitsTeamsStartsAndEveryoneReturnsToTheLobby(t *testing.T) {
	open, tables := opener(t)
	lobby := NewLobby(&counter{}, open)
	r, _ := lobby.Create(player("ana"))
	must(t, r.Join(player("ben")))
	must(t, r.Join(player("cem")))
	if phase := r.State().Phase; phase != PhaseLobby {
		t.Fatalf("a new room is in phase %d", phase)
	}
	_, err := r.Start("ana", room.Options{})
	refused(t, err, ErrWrongPhase)
	refused(t, r.Place("ana", "ben", rules.SeatX), ErrWrongPhase)
	refused(t, r.Teams("ana", rules.Mode(9)), ErrInvalidMode)
	must(t, r.Teams("ana", rules.Mode1v1))
	refused(t, r.Place("ben", "ben", rules.SeatX), ErrNotLeader)
	refused(t, r.Place("ana", "ben", rules.SeatNorth), ErrInvalidSeat)
	refused(t, r.Place("ana", "zed", rules.SeatX), ErrNotInRoom)
	must(t, r.Place("ana", "ana", rules.SeatX))
	_, err = r.Start("ana", room.Options{})
	refused(t, err, ErrSeatsOpen)
	must(t, r.Place("ana", "ben", rules.SeatX))
	must(t, r.Place("ana", "cem", rules.SeatY))
	must(t, r.Place("ana", "ana", rules.SeatY))
	if got := strings.Join(accounts(r.State()), ","); got != "ben,ana" {
		t.Fatalf("seating %s", got)
	}
	must(t, r.Place("ana", "ana", rules.SeatX))
	if got := strings.Join(accounts(r.State()), ","); got != "ana,ben" {
		t.Fatalf("a placed member did not swap seats: %s", got)
	}
	must(t, r.Back("ana"))
	refused(t, r.Back("ana"), ErrWrongPhase)
	must(t, r.Teams("ana", rules.Mode1v1))
	if got := strings.Join(accounts(r.State()), ","); got != "ana,ben" {
		t.Fatalf("seats not kept across the lobby: %s", got)
	}
	_, err = r.Start("ben", room.Options{})
	refused(t, err, ErrNotLeader)
	table, err := r.Start("ana", room.Options{Seed: 3})
	must(t, err)
	if table.Kind() != room.KindCustom || len(*tables) != 1 || r.Table() != table || r.State().Phase != PhaseMatch {
		t.Fatalf("started table kind %s, state %+v", table.Kind(), r.State())
	}
	_, err = r.Start("ana", room.Options{})
	refused(t, err, ErrStarted)
	refused(t, r.Join(player("dan")), ErrStarted)
	refused(t, r.Teams("ana", rules.Mode1v1), ErrStarted)
	refused(t, r.Emote("cem", "emoji.laugh", time.Now()), ErrWrongPhase)
	must(t, r.Leave("ben"))
	(*tables)[0].cancel()
	waitPhase(t, r, PhaseLobby)
	state := r.State()
	if r.Table() != nil || !slices.Equal(state.Members, []string{"ana", "cem"}) || strings.Join(accounts(state), ",") != "ana," {
		t.Fatalf("after the match: %+v", state)
	}
}

func waitPhase(t *testing.T, r *Room, phase Phase) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for r.State().Phase != phase {
		if time.Now().After(deadline) {
			t.Fatalf("phase %d never reached, state %+v", phase, r.State())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestChangingTheModeClearsTheSeats(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	r, _ := lobby.Create(player("ana"))
	must(t, r.Teams("ana", rules.Mode2v2))
	must(t, r.Place("ana", "ana", rules.SeatSouth))
	must(t, r.Teams("ana", rules.Mode1v1))
	if got := strings.Join(accounts(r.State()), ","); got != "," {
		t.Fatalf("seats after the mode change: %s", got)
	}
}

func TestTheLeaderRemovesAndAddsPlayersDuringTheMatch(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	r, _ := lobby.Create(player("ana"))
	must(t, r.Join(player("ben")))
	must(t, r.Teams("ana", rules.Mode1v1))
	must(t, r.Place("ana", "ana", rules.SeatX))
	must(t, r.Place("ana", "ben", rules.SeatY))
	table, err := r.Start("ana", room.Options{Seed: 4})
	must(t, err)
	sinks := map[string]chanSink{"ana": make(chanSink, 64), "ben": make(chanSink, 64)}
	table.Join(rules.SeatX, sinks["ana"])
	table.Join(rules.SeatY, sinks["ben"])
	for _, sink := range sinks {
		if m := <-sink; m.Rejected {
			t.Fatalf("join rejected: %v", m.Reason)
		}
	}
	refused(t, r.Remove("ben", "ana"), ErrNotLeader)
	refused(t, r.Remove("ana", "ana"), ErrNotLeader)
	refused(t, r.Remove("ana", "zed"), ErrNotInRoom)
	must(t, r.Remove("ana", "ben"))
	if got := accounts(r.State()); got[1] != "" || slices.Contains(r.State().Members, "ben") {
		t.Fatalf("ben still in the room: %+v", r.State())
	}
	reply := make(chan room.View, 1)
	table.Snapshot(rules.SeatX, reply)
	if v := <-reply; v.Controllers[rules.SeatY] != rules.ControllerBot {
		t.Fatalf("the removed seat is not played by the bot: %+v", v)
	}
	back := make(chanSink, 16)
	table.Reconnect(rules.SeatY, back)
	if m := <-back; !m.Rejected || !errors.Is(m.Reason, room.ErrBarred) {
		t.Fatalf("the removed player came back on their own: %+v", m)
	}
	refused(t, r.Add("ana", player("cem"), rules.SeatX), ErrSeatTaken)
	refused(t, r.Add("ana", player("ana"), rules.SeatY), ErrAlreadyInRoom)
	refused(t, r.Add("ben", player("cem"), rules.SeatY), ErrNotLeader)
	must(t, r.Add("ana", player("cem"), rules.SeatY))
	table.Join(rules.SeatY, back)
	if m := <-back; m.Rejected || m.View.Controllers[rules.SeatY] != rules.ControllerPlayer {
		t.Fatalf("the added player could not take the seat: %+v", m)
	}
	must(t, r.Leave("cem"))
	table.Snapshot(rules.SeatX, reply)
	if v := <-reply; v.Controllers[rules.SeatY] != rules.ControllerBot || accounts(r.State())[1] != "cem" {
		t.Fatalf("leaving during the match: %+v seats %v", v, accounts(r.State()))
	}
}

func TestASeatRequestReachesOnlyTheLeaderDuringTeamSetup(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	anaSink, benSink := make(eventSink, 16), make(eventSink, 16)
	r, _ := lobby.Create(Member{Account: "ana", Sink: anaSink})
	must(t, r.Join(Member{Account: "ben", Sink: benSink}))
	refused(t, r.Request("ben", rules.SeatY), ErrWrongPhase)
	must(t, r.Teams("ana", rules.Mode1v1))
	refused(t, r.Request("ben", rules.SeatNorth), ErrInvalidSeat)
	refused(t, r.Request("zed", rules.SeatY), ErrNotInRoom)
	must(t, r.Request("ben", rules.SeatY))
	seat := func(e Event) string {
		if !e.Request {
			return ""
		}
		return e.Seat.String()
	}
	if got := relayed(anaSink, seat); !slices.Equal(got, []string{"ben:" + rules.SeatY.String()}) {
		t.Fatalf("leader got %v", got)
	}
	if got := relayed(benSink, seat); len(got) != 0 {
		t.Fatalf("member got %v", got)
	}
}

func relayed(sink eventSink, field func(Event) string) []string {
	var out []string
	for {
		select {
		case e := <-sink:
			if field(e) != "" {
				out = append(out, e.Account+":"+field(e))
			}
		default:
			return out
		}
	}
}

func TestLobbyEmojiNeedsTheEmojiEquippedAndKeepsTheMatchSpamLimit(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	lobby.LimitEmoji(room.EmojiLimit{Burst: 2, Refill: time.Minute})
	sink := make(eventSink, 16)
	r, _ := lobby.Create(Member{Account: "ana", Emoji: []string{"emoji.laugh"}, Sink: sink})
	must(t, r.Join(Member{Account: "ben", Emoji: []string{"emoji.laugh"}, Muted: true}))
	now := time.Unix(1_800_000_000, 0)
	refused(t, r.Emote("ana", "emoji.cry", now), ErrUnknownEmoji)
	refused(t, r.Emote("ben", "emoji.laugh", now), ErrMuted)
	must(t, r.Emote("ana", "emoji.laugh", now))
	must(t, r.Emote("ana", "emoji.laugh", now))
	refused(t, r.Emote("ana", "emoji.laugh", now), ErrEmojiThrottled)
	must(t, r.Emote("ana", "emoji.laugh", now.Add(time.Minute)))
	if got := relayed(sink, func(e Event) string { return e.Emoji }); len(got) != 3 {
		t.Fatalf("relayed emoji %v", got)
	}
	must(t, r.Teams("ana", rules.Mode1v1))
	refused(t, r.Emote("ana", "emoji.laugh", now.Add(time.Hour)), ErrWrongPhase)
}

type blocks map[[2]string]bool

func (b blocks) Blocked(_ context.Context, a, c string) (bool, error) {
	if a == "broken" {
		return false, errors.New("store down")
	}
	return b[[2]string{a, c}] || b[[2]string{c, a}], nil
}

func TestInvitesSkipBlockedPairsFullRoomsAndBusyAccounts(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	lobby.Guard(busy{"playing": true})
	lobby.Screen(blocks{{"zed", "ana"}: true})
	ctx := context.Background()
	r, _ := lobby.Create(player("ana"))
	must(t, r.Invite(ctx, "ana", "ben"))
	refused(t, r.Invite(ctx, "ana", "zed"), ErrBlocked)
	refused(t, r.Invite(ctx, "eve", "ben"), ErrNotInRoom)
	refused(t, r.Invite(ctx, "ana", "ana"), ErrAlreadyInRoom)
	refused(t, r.Invite(ctx, "ana", "playing"), ErrInMatch)
	must(t, r.Join(player("broken")))
	if err := r.Invite(ctx, "broken", "ben"); err == nil {
		t.Fatal("a failed block lookup let the invite through")
	}
	must(t, r.Join(player("cem")))
	must(t, r.Join(player("dan")))
	refused(t, r.Invite(ctx, "ana", "ben"), ErrFull)
}

func TestMembersSeeEveryStateChange(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	sink := make(eventSink, 16)
	r, _ := lobby.Create(Member{Account: "ana", Sink: sink})
	must(t, r.Join(player("ben")))
	must(t, r.Teams("ana", rules.Mode1v1))
	var phases []Phase
	for len(sink) > 0 {
		if e := <-sink; e.State != nil {
			phases = append(phases, e.State.Phase)
		}
	}
	if !slices.Equal(phases, []Phase{PhaseLobby, PhaseTeams}) {
		t.Fatalf("state pushes %v", phases)
	}
}

type busy map[string]bool

func (b busy) InMatch(account string) bool { return b[account] }

func TestAnAccountInARunningMatchCannotEnterALobby(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	lobby.Guard(busy{"playing": true})
	_, err := lobby.Create(player("playing"))
	refused(t, err, ErrInMatch)
	r, err := lobby.Create(player("ana"))
	must(t, err)
	refused(t, r.Join(player("playing")), ErrInMatch)
	must(t, r.Teams("ana", rules.Mode2v2))
	refused(t, r.Add("ana", player("playing"), rules.SeatEast), ErrInMatch)
	must(t, r.Join(player("ben")))
}

type directory struct {
	claimed map[string]bool
	fail    error
}

func (d *directory) Claim(ctx context.Context, code string) error {
	if d.fail != nil {
		return d.fail
	}
	d.claimed[code] = true
	return nil
}

func (d *directory) Release(ctx context.Context, code string) error {
	delete(d.claimed, code)
	return nil
}

func TestARoomCodeIsClaimedInTheDirectoryUntilTheRoomCloses(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	codes := &directory{claimed: map[string]bool{}}
	lobby.Route(codes)
	r, err := lobby.Create(player("ana"))
	must(t, err)
	if !codes.claimed[r.Code()] {
		t.Fatalf("code %s not claimed", r.Code())
	}
	must(t, r.Leave("ana"))
	if len(codes.claimed) != 0 || lobby.Len() != 0 {
		t.Fatalf("closed room left claims %v and %d rooms", codes.claimed, lobby.Len())
	}
	codes.fail = errors.New("directory down")
	_, err = lobby.Create(player("ana"))
	refused(t, err, codes.fail)
	if lobby.Len() != 0 {
		t.Fatal("a room whose code was not claimed was kept")
	}
}

func TestRenewReclaimsTheCodesOfOpenRoomsOnly(t *testing.T) {
	open, _ := opener(t)
	lobby := NewLobby(&counter{}, open)
	must(t, lobby.Renew(context.Background()))
	codes := &directory{claimed: map[string]bool{}}
	lobby.Route(codes)
	kept, err := lobby.Create(player("ana"))
	must(t, err)
	closed, err := lobby.Create(player("ben"))
	must(t, err)
	must(t, closed.Leave("ben"))
	clear(codes.claimed)
	must(t, lobby.Renew(context.Background()))
	if !codes.claimed[kept.Code()] || len(codes.claimed) != 1 {
		t.Fatalf("renewed %v, want only %s", codes.claimed, kept.Code())
	}
	codes.fail = errors.New("directory down")
	refused(t, lobby.Renew(context.Background()), codes.fail)
}
