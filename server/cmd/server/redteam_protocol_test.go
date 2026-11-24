package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type stillClock struct{ now time.Time }

func (c stillClock) Now() time.Time                       { return c.now }
func (c stillClock) After(time.Duration) <-chan time.Time { return nil }

func foreignToken(t *testing.T, a *attacker) string {
	t.Helper()
	keys, err := token.ParseKeys("attacker:" + randomSecret(t))
	if err != nil {
		t.Fatal(err)
	}
	signer, err := token.NewSigner(keys, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	forged, err := signer.Sign(token.Claims{Issuer: token.Issuer, Account: a.account, Session: a.pair.Session, Device: redTeamDevice, IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	return forged
}

func unsignedToken(t *testing.T, a *attacker) string {
	t.Helper()
	claims, err := json.Marshal(map[string]any{"iss": token.Issuer, "sub": a.account, "sid": a.pair.Session, "dev": redTeamDevice, "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`)) + "." + base64.RawURLEncoding.EncodeToString(claims) + "."
}

func (h *redTeam) expiredToken(t *testing.T, a *attacker) string {
	t.Helper()
	keys, err := token.ParseKeys(h.tokenKeys)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := token.NewSigner(keys, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	issued := time.Now().Add(-2 * time.Hour)
	expired, err := signer.Sign(token.Claims{Issuer: token.Issuer, Account: a.account, Session: a.pair.Session, Device: redTeamDevice, IssuedAt: issued.Unix(), ExpiresAt: issued.Add(15 * time.Minute).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	return expired
}

func TestRedTeamHelloRefusesEveryForgedIdentityAndVersion(t *testing.T) {
	h := startRedTeam(t)
	a := h.attacker(t)
	h.connect(t, a).Drop()
	oversized := clientHello(a.pair.AccessToken)
	oversized.BuildHash = strings.Repeat("f", 9000)
	cases := []struct {
		name      string
		frame     *pb.Envelope
		wantError pb.ErrorCode
		wantHello pb.HelloStatus
		wantClose websocket.StatusCode
	}{
		{"missing protocol version", hello(0, clientHello(a.pair.AccessToken)), pb.ErrorCode_ERROR_CODE_UNSUPPORTED_PROTOCOL, 0, websocket.StatusProtocolError},
		{"future protocol version", hello(gateway.ProtocolVersion+1, clientHello(a.pair.AccessToken)), pb.ErrorCode_ERROR_CODE_UNSUPPORTED_PROTOCOL, 0, websocket.StatusProtocolError},
		{"app below the minimum version", hello(gateway.ProtocolVersion, &pb.ClientHello{AppVersion: "0.9.9", SessionToken: a.pair.AccessToken}), 0, pb.HelloStatus_HELLO_STATUS_UPDATE_REQUIRED, websocket.StatusNormalClosure},
		{"no session token", protocolHello(""), pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, 0, websocket.StatusPolicyViolation},
		{"access token signed with another key", protocolHello(foreignToken(t, a)), pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, 0, websocket.StatusPolicyViolation},
		{"unsigned access token", protocolHello(unsignedToken(t, a)), pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, 0, websocket.StatusPolicyViolation},
		{"expired access token", protocolHello(h.expiredToken(t, a)), pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, 0, websocket.StatusPolicyViolation},
		{"bare session id of a live session", protocolHello(a.pair.Session), pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, 0, websocket.StatusPolicyViolation},
		{"refresh token", protocolHello(a.pair.RefreshToken), pb.ErrorCode_ERROR_CODE_UNAUTHENTICATED, 0, websocket.StatusPolicyViolation},
		{"intent before the hello", &pb.Envelope{ProtocolVersion: gateway.ProtocolVersion, Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}}, 0, 0, websocket.StatusPolicyViolation},
		{"hello above the hello size limit", hello(gateway.ProtocolVersion, oversized), 0, 0, websocket.StatusMessageTooBig},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			conn := h.dialRaw(t)
			conn.writeFrame(c.frame)
			if c.wantError != 0 || c.wantHello != 0 {
				reply, err := conn.read()
				if err != nil {
					t.Fatalf("no answer before the close: %v", err)
				}
				if reply.GetError().GetCode() != c.wantError || reply.GetServerHello().GetStatus() != c.wantHello {
					t.Fatalf("answered %v", reply)
				}
			}
			if status := conn.closed(); status != c.wantClose {
				t.Fatalf("closed with %v, want %v", status, c.wantClose)
			}
		})
	}

	t.Run("claimed sequence and device clock", func(t *testing.T) {
		b := h.attacker(t)
		claim := clientHello(b.pair.AccessToken)
		claim.LastAckedSequence, claim.ClientTimeUnixMs = 1_000_000, uint64(time.Now().Add(48*time.Hour).UnixMilli())
		frame := hello(gateway.ProtocolVersion, claim)
		frame.Sequence, frame.AckedSequence = 1_000_000, 1_000_000
		conn := h.dialRaw(t)
		conn.writeFrame(frame)
		reply, err := conn.read()
		if err != nil {
			t.Fatal(err)
		}
		server := time.UnixMilli(int64(reply.GetServerHello().GetServerTimeUnixMs()))
		if reply.GetServerHello().GetStatus() != pb.HelloStatus_HELLO_STATUS_ACCEPTED || reply.GetAckedSequence() != 0 || time.Since(server).Abs() > time.Minute {
			t.Fatalf("a claimed sequence and clock answered %v", reply)
		}
		h.awaitSignal(t, b, "clock_skew")
	})
}

func (h *redTeam) frame(c *botclient.Client, frame *pb.Envelope) []byte {
	c.Sequence++
	frame.ProtocolVersion, frame.Sequence, frame.SessionNonce = gateway.ProtocolVersion, c.Sequence, c.Nonce()
	data, _ := proto.Marshal(frame)
	return data
}

func expectError(t *testing.T, c *botclient.Client, want pb.ErrorCode) *pb.Error {
	t.Helper()
	for {
		frame, err := c.Read()
		if err != nil {
			t.Fatalf("waiting for %v: %v", want, err)
		}
		if refusal := frame.GetError(); refusal != nil {
			if refusal.GetCode() != want {
				t.Fatalf("answered %v, want %v", refusal.GetCode(), want)
			}
			return refusal
		}
	}
}

func send(t *testing.T, c *botclient.Client, data []byte) {
	t.Helper()
	if err := c.WriteRaw(websocket.MessageBinary, data); err != nil {
		t.Fatal(err)
	}
}

func TestRedTeamFramesAreBoundToTheirSessionAndShape(t *testing.T) {
	h := startRedTeam(t)
	pass := func() *pb.Envelope {
		return &pb.Envelope{Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}}
	}

	t.Run("missing and foreign session nonces", func(t *testing.T) {
		a, other := h.attacker(t), h.attacker(t)
		c, foreign := h.connect(t, a), h.connect(t, other)
		for attempt := range 5 {
			data := h.frame(c, pass())
			frame := &pb.Envelope{}
			proto.Unmarshal(data, frame)
			frame.SessionNonce = []uint64{0, foreign.Nonce(), c.Nonce() + 1, c.Nonce() - 1, foreign.Nonce() ^ c.Nonce()}[attempt]
			data, _ = proto.Marshal(frame)
			send(t, c, data)
			expectError(t, c, pb.ErrorCode_ERROR_CODE_REPLAY_REJECTED)
		}
		h.awaitSignal(t, a, "replay_reject")
		send(t, c, h.frame(c, pass()))
		expectError(t, c, pb.ErrorCode_ERROR_CODE_ILLEGAL_INTENT)
	})

	t.Run("queue and lobby intents with forged fields", func(t *testing.T) {
		leader, a := h.attacker(t), h.attacker(t)
		c := h.connect(t, a)
		send(t, c, h.frame(c, &pb.Envelope{Payload: &pb.Envelope_JoinQueueIntent{JoinQueueIntent: &pb.JoinQueueIntent{Mode: pb.Mode(99), Queue: pb.QueueKind(99)}}}))
		if state := nextQueueState(t, c); state.GetStatus() != pb.QueueStatus_QUEUE_STATUS_REFUSED {
			t.Fatalf("an unknown mode and queue answered %v", state)
		}
		owner := h.connect(t, leader)
		send(t, owner, h.frame(owner, lobby(pb.LobbyCommand_LOBBY_COMMAND_CREATE)))
		code := nextLobbyState(t, owner).GetCode()
		join := lobby(pb.LobbyCommand_LOBBY_COMMAND_JOIN)
		join.GetLobbyIntent().Code = code
		send(t, c, h.frame(c, join))
		if state := nextLobbyState(t, c); state.GetRefusal() != pb.LobbyRefusal_LOBBY_REFUSAL_UNSPECIFIED || state.GetCode() != code {
			t.Fatalf("joining the lobby answered %v", state)
		}
		for _, command := range []pb.LobbyCommand{pb.LobbyCommand_LOBBY_COMMAND_TEAMS, pb.LobbyCommand_LOBBY_COMMAND_START, pb.LobbyCommand_LOBBY_COMMAND_REMOVE} {
			intent := lobby(command)
			intent.GetLobbyIntent().AccountId, intent.GetLobbyIntent().Mode = leader.account, pb.Mode_MODE_1V1
			send(t, c, h.frame(c, intent))
			if state := nextLobbyState(t, c); state.GetRefusal() == pb.LobbyRefusal_LOBBY_REFUSAL_UNSPECIFIED || state.GetLeaderAccountId() != leader.account {
				t.Fatalf("%v from a member answered %v", command, state)
			}
		}
	})

	t.Run("malformed, text and oversized frames", func(t *testing.T) {
		cases := []struct {
			name string
			kind websocket.MessageType
			data []byte
			want websocket.StatusCode
		}{
			{"garbage bytes", websocket.MessageBinary, []byte{0xff, 0xff, 0xff, 0xff, 0x0f, 0x80}, websocket.StatusPolicyViolation},
			{"text frame", websocket.MessageText, []byte(`{"place_tile_intent":{}}`), websocket.StatusUnsupportedData},
			{"frame above the frame limit", websocket.MessageBinary, oversizedFrame(t, 2048), websocket.StatusMessageTooBig},
			{"frame above the read limit", websocket.MessageBinary, oversizedFrame(t, 64*1024), websocket.StatusMessageTooBig},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				conn := h.dialRaw(t)
				conn.writeFrame(protocolHello(h.attacker(t).pair.AccessToken))
				if reply, err := conn.read(); err != nil || reply.GetServerHello().GetStatus() != pb.HelloStatus_HELLO_STATUS_ACCEPTED {
					t.Fatalf("hello answered %v, %v", reply, err)
				}
				conn.write(c.kind, c.data)
				if status := conn.closed(); status != c.want {
					t.Fatalf("closed with %v, want %v", status, c.want)
				}
			})
		}
	})

	t.Run("intent flood", func(t *testing.T) {
		c := h.connect(t, h.attacker(t))
		var flood [][]byte
		for range 60 {
			flood = append(flood, h.frame(c, pass()))
		}
		answers := make(chan floodAnswer, 1)
		go func() { answers <- countErrors(c, len(flood)) }()
		for _, data := range flood {
			if err := c.WriteRaw(websocket.MessageBinary, data); err != nil {
				break
			}
		}
		answer := <-answers
		burst := gateway.DefaultConfig().IntentBurst
		limited := answer.counts[pb.ErrorCode_ERROR_CODE_RATE_LIMITED] > 0 || answer.closed == websocket.StatusPolicyViolation
		if answer.err != nil && answer.closed != websocket.StatusPolicyViolation || !limited || answer.counts[pb.ErrorCode_ERROR_CODE_ILLEGAL_INTENT] > 2*burst {
			t.Fatalf("a flood of 60 intents was answered %v, closed %v (%v)", answer.counts, answer.closed, answer.err)
		}
	})
}

func oversizedFrame(t *testing.T, size int) []byte {
	t.Helper()
	padding := make([]byte, size)
	rand.Read(padding)
	frame := &pb.Envelope{ProtocolVersion: gateway.ProtocolVersion, Payload: &pb.Envelope_EmojiIntent{EmojiIntent: &pb.EmojiIntent{ItemId: base64.RawURLEncoding.EncodeToString(padding)}}}
	data, err := proto.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type floodAnswer struct {
	counts map[pb.ErrorCode]int
	closed websocket.StatusCode
	err    error
}

func countErrors(c *botclient.Client, want int) floodAnswer {
	answer := floodAnswer{counts: map[pb.ErrorCode]int{}, closed: -1}
	for seen := 0; seen < want; {
		frame, err := c.Receive()
		if err != nil {
			answer.closed, answer.err = websocket.CloseStatus(err), err
			return answer
		}
		if refusal := frame.GetError(); refusal != nil {
			answer.counts[refusal.GetCode()]++
			seen++
		}
	}
	return answer
}

func lobby(command pb.LobbyCommand) *pb.Envelope {
	return &pb.Envelope{Payload: &pb.Envelope_LobbyIntent{LobbyIntent: &pb.LobbyIntent{Command: command}}}
}

func nextQueueState(t *testing.T, c *botclient.Client) *pb.QueueState {
	t.Helper()
	for {
		frame, err := c.Read()
		if err != nil {
			t.Fatal(err)
		}
		if state := frame.GetQueueState(); state != nil {
			return state
		}
	}
}

func nextLobbyState(t *testing.T, c *botclient.Client) *pb.LobbyState {
	t.Helper()
	for {
		frame, err := c.Read()
		if err != nil {
			t.Fatal(err)
		}
		if state := frame.GetLobbyState(); state != nil {
			return state
		}
	}
}

type seatedAttacker struct {
	attacker *attacker
	client   *botclient.Client
	view     *pb.MatchView
}

func (h *redTeam) seatedPair(t *testing.T) (*room.Room, []*seatedAttacker) {
	t.Helper()
	r := room.New(room.Options{
		Kind: room.KindCasual, Mode: rules.Mode1v1, Config: rules.DefaultConfig(), Bot: bot.DefaultConfig(), BotLevel: bot.Hard, Seed: 7,
		Clock: stillClock{now: time.Now()}, Table: "table.night_market_deluxe",
		Roster: []room.Player{{Seat: rules.SeatX, Nickname: "RaiderX"}, {Seat: rules.SeatY, Nickname: "RaiderY"}},
	})
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	go r.Run(ctx)
	var seated []*seatedAttacker
	for _, seat := range []rules.Seat{rules.SeatX, rules.SeatY} {
		a := h.attacker(t)
		if err := h.sessions.Place(a.pair.Session, a.account, r, seat); err != nil {
			t.Fatal(err)
		}
		seated = append(seated, &seatedAttacker{attacker: a, client: h.connect(t, a)})
	}
	for _, s := range seated {
		s.view = awaitView(t, s.client, botclient.NeedsReady)
		if err := s.client.Ready(); err != nil {
			t.Fatal(err)
		}
	}
	for _, s := range seated {
		s.view = awaitView(t, s.client, playing)
	}
	return r, seated
}

func playing(view *pb.MatchView) bool {
	return view.GetState() == pb.MatchState_MATCH_STATE_OPENING || view.GetState() == pb.MatchState_MATCH_STATE_PLAYING
}

func awaitView(t *testing.T, c *botclient.Client, wanted func(*pb.MatchView) bool) *pb.MatchView {
	t.Helper()
	for {
		frame, err := c.Read()
		if err != nil {
			t.Fatal(err)
		}
		if view := frame.GetMatchView(); view != nil && wanted(view) {
			return view
		}
	}
}

func nextView(t *testing.T, c *botclient.Client) *pb.MatchView {
	t.Helper()
	for {
		frame, err := c.Read()
		if err != nil {
			t.Fatal(err)
		}
		if refusal := frame.GetError(); refusal != nil {
			t.Fatalf("a legal intent was refused: %v", refusal)
		}
		if view := frame.GetMatchView(); view != nil {
			return view
		}
	}
}

func sameTable(before, after *pb.MatchView) bool {
	return after.GetState() == before.GetState() && after.GetCurrent() == before.GetCurrent() && after.GetTurnPlacements() == before.GetTurnPlacements() &&
		after.GetBoneyardSize() == before.GetBoneyardSize() && proto.Equal(after.GetBoard(), before.GetBoard()) &&
		proto.Equal(after.GetScores(), before.GetScores()) && tilesEqual(after.GetHand(), before.GetHand())
}

func tilesEqual(a, b []*pb.Tile) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !proto.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func (h *redTeam) refused(t *testing.T, s *seatedAttacker, attack string, frame *pb.Envelope) {
	t.Helper()
	send(t, s.client, h.frame(s.client, frame))
	refusal := expectError(t, s.client, pb.ErrorCode_ERROR_CODE_ILLEGAL_INTENT)
	if refusal.GetRejectedSequence() != s.client.Sequence {
		t.Fatalf("%s: refused sequence %d, sent %d", attack, refusal.GetRejectedSequence(), s.client.Sequence)
	}
	view := nextView(t, s.client)
	if !sameTable(s.view, view) {
		t.Fatalf("%s changed the table:\nbefore %v\nafter  %v", attack, s.view, view)
	}
}

func serverOnlyPayloads(a *attacker) []*pb.Envelope {
	return []*pb.Envelope{
		{Payload: &pb.Envelope_MatchView{MatchView: &pb.MatchView{State: pb.MatchState_MATCH_STATE_FINISHED, Scores: &pb.TeamPoints{TeamA: 999}, Winner: pb.Team_TEAM_A}}},
		{Payload: &pb.Envelope_ServerHello{ServerHello: &pb.ServerHello{Status: pb.HelloStatus_HELLO_STATUS_ACCEPTED}}},
		{Payload: &pb.Envelope_QueueState{QueueState: &pb.QueueState{Status: pb.QueueStatus_QUEUE_STATUS_MATCH_FOUND}}},
		{Payload: &pb.Envelope_EmojiShown{EmojiShown: &pb.EmojiShown{ItemId: "emoji.forged"}}},
		{Payload: &pb.Envelope_ClientHello{ClientHello: clientHello(a.pair.AccessToken)}},
		{},
	}
}

func tilesOutsideTheSet() []*pb.Envelope {
	return []*pb.Envelope{
		placement(&pb.Tile{Low: 7, High: 250}, pb.BoardEnd_BOARD_END_UNSPECIFIED),
		placement(&pb.Tile{Low: 5, High: 2}, pb.BoardEnd_BOARD_END_LEFT),
		placement(&pb.Tile{Low: 1, High: 2}, pb.BoardEnd(99)),
		placement(nil, pb.BoardEnd_BOARD_END_RIGHT),
	}
}

func (h *redTeam) undecodable(t *testing.T, s *seatedAttacker, frame *pb.Envelope) {
	t.Helper()
	send(t, s.client, h.frame(s.client, frame))
	if refusal := expectError(t, s.client, pb.ErrorCode_ERROR_CODE_ILLEGAL_INTENT); refusal.GetRejectedSequence() != s.client.Sequence {
		t.Fatalf("%v: refused sequence %d, sent %d", frame, refusal.GetRejectedSequence(), s.client.Sequence)
	}
}

func placement(tile *pb.Tile, end pb.BoardEnd) *pb.Envelope {
	return &pb.Envelope{Payload: &pb.Envelope_PlaceTileIntent{PlaceTileIntent: &pb.PlaceTileIntent{Tile: tile, End: end}}}
}

func notHeld(hand []*pb.Tile) []*pb.Tile {
	var tiles []*pb.Tile
	for high := uint32(0); high <= 6; high++ {
		for low := uint32(0); low <= high; low++ {
			candidate := &pb.Tile{Low: low, High: high}
			if !holds(hand, candidate) {
				tiles = append(tiles, candidate)
			}
		}
	}
	return tiles
}

func holds(hand []*pb.Tile, tile *pb.Tile) bool {
	for _, held := range hand {
		if proto.Equal(held, tile) {
			return true
		}
	}
	return false
}

func openingTile(view *pb.MatchView) *pb.Tile {
	if forced := view.GetForcedOpening(); forced != nil {
		return forced.GetTile()
	}
	return view.GetHand()[0]
}

func withUnknownField(data []byte) []byte {
	data = protowire.AppendTag(data, 1000, protowire.VarintType)
	return protowire.AppendVarint(data, 999_999)
}

func TestRedTeamMatchRefusesIllegalIntentsAndReplays(t *testing.T) {
	h := startRedTeam(t)
	r, seated := h.seatedPair(t)
	mover, idle := seated[0], seated[1]
	if idle.view.GetCurrent() == idle.view.GetSeat() {
		mover, idle = idle, mover
	}
	draw := &pb.Envelope{Payload: &pb.Envelope_DrawIntent{DrawIntent: &pb.DrawIntent{}}}
	pass := &pb.Envelope{Payload: &pb.Envelope_PassIntent{PassIntent: &pb.PassIntent{}}}
	endTurn := &pb.Envelope{Payload: &pb.Envelope_EndTurnIntent{EndTurnIntent: &pb.EndTurnIntent{}}}

	h.refused(t, idle, "opening out of turn", placement(idle.view.GetHand()[0], pb.BoardEnd_BOARD_END_UNSPECIFIED))
	h.refused(t, mover, "opening with a tile not held", placement(notHeld(mover.view.GetHand())[0], pb.BoardEnd_BOARD_END_UNSPECIFIED))
	h.refused(t, mover, "opening at an end of an empty board", placement(openingTile(mover.view), pb.BoardEnd_BOARD_END_LEFT))
	for name, intent := range map[string]*pb.Envelope{"pass at the opening": pass, "draw at the opening": draw, "end turn at the opening": endTurn} {
		h.refused(t, mover, name, intent)
	}

	opening := withUnknownField(h.frame(mover.client, placement(openingTile(mover.view), pb.BoardEnd_BOARD_END_UNSPECIFIED)))
	send(t, mover.client, opening)
	mover.view = nextView(t, mover.client)
	idle.view = nextView(t, idle.client)
	if len(mover.view.GetBoard().GetLine()) != 1 {
		t.Fatalf("the opening with an unknown field was not applied: %v", mover.view)
	}

	send(t, mover.client, opening)
	toMove, waiting := mover, idle
	if idle.view.GetCurrent() == idle.view.GetSeat() {
		toMove, waiting = idle, mover
	}
	ends := toMove.view.GetBoard().GetOpenEnds()
	h.refused(t, toMove, "a tile not held that fits the board", placement(fitting(notHeld(toMove.view.GetHand()), ends[0].GetValue()), ends[0].GetEnd()))
	h.refused(t, toMove, "a held tile at an end it does not fit", placement(misfit(toMove.view, ends[0].GetValue()), ends[0].GetEnd()))
	h.refused(t, toMove, "a held tile at an end that does not exist yet", placement(toMove.view.GetHand()[0], pb.BoardEnd_BOARD_END_TOP))
	if len(toMove.view.GetLegalMoves()) > 0 {
		h.refused(t, toMove, "pass while holding a legal placement", pass)
		h.refused(t, toMove, "draw while holding a legal placement", draw)
	}
	h.refused(t, waiting, "a placement out of turn", placement(waiting.view.GetHand()[0], ends[0].GetEnd()))
	for _, frame := range append(tilesOutsideTheSet(), serverOnlyPayloads(toMove.attacker)...) {
		h.undecodable(t, toMove, frame)
	}

	mover.client.Drop()
	resumed := h.connect(t, mover.attacker)
	nextView(t, resumed)
	send(t, resumed, opening)
	expectError(t, resumed, pb.ErrorCode_ERROR_CODE_REPLAY_REJECTED)

	records := make(chan replay.Record, 1)
	r.Record(records)
	if record := <-records; len(record.Intents) != 1 {
		t.Fatalf("the match recorded %d intents after one legal opening, want 1: %+v", len(record.Intents), record.Intents)
	}
}

func fitting(tiles []*pb.Tile, value uint32) *pb.Tile {
	for _, tile := range tiles {
		if tile.GetLow() == value || tile.GetHigh() == value {
			return tile
		}
	}
	return tiles[0]
}

func misfit(view *pb.MatchView, value uint32) *pb.Tile {
	for _, tile := range view.GetHand() {
		if tile.GetLow() != value && tile.GetHigh() != value {
			return tile
		}
	}
	return notHeld(view.GetHand())[0]
}
