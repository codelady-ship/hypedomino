package session

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/coder/websocket"

	"github.com/Alievs-corp/hypedomino/server/internal/custom"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

var (
	errNotFriend      = errors.New("session: the invited account is not a friend")
	errUnknownCommand = errors.New("session: unknown lobby command")
)

type LobbyMembers func(ctx context.Context, account string) (custom.Member, error)

type Friends interface {
	FriendIDs(ctx context.Context, account string) ([]string, error)
}

type RankedBanCheck func(ctx context.Context, account string, now time.Time) error

type AppealFunc func(account string, sanctionID int64) *pb.AppealGrant

type RemoteLobby interface {
	Join(ctx context.Context, code, account, session string) (pb.LobbyRefusal, error)
	Invite(ctx context.Context, to string, invite *pb.LobbyInvite) error
}

type refusalError pb.LobbyRefusal

func (e refusalError) Error() string {
	return "session: lobby refused " + pb.LobbyRefusal(e).String()
}

func (m *Manager) Host(lobby *custom.Lobby, members LobbyMembers, friends Friends) {
	lobby.Guard(m)
	m.lobby, m.members, m.friends = lobby, members, friends
}

func (m *Manager) Reach(remote RemoteLobby) {
	m.remote = remote
}

func (m *Manager) Restrict(rankedBan RankedBanCheck) {
	m.rankedBan = rankedBan
}

func (m *Manager) GrantAppeals(appeal AppealFunc) {
	m.appeal = appeal
}

func (m *Manager) customRoomsOpen(account string) error {
	if m.rankedBan == nil {
		return nil
	}
	return m.rankedBan(context.Background(), account, m.clock.Now())
}

var refusals = []struct {
	err     error
	refusal pb.LobbyRefusal
}{
	{custom.ErrUnknownCode, pb.LobbyRefusal_LOBBY_REFUSAL_UNKNOWN_CODE},
	{custom.ErrNotLeader, pb.LobbyRefusal_LOBBY_REFUSAL_NOT_LEADER},
	{custom.ErrSeatTaken, pb.LobbyRefusal_LOBBY_REFUSAL_SEAT_TAKEN},
	{custom.ErrNotInRoom, pb.LobbyRefusal_LOBBY_REFUSAL_NOT_IN_LOBBY},
	{custom.ErrAlreadyInRoom, pb.LobbyRefusal_LOBBY_REFUSAL_ALREADY_IN_LOBBY},
	{custom.ErrFull, pb.LobbyRefusal_LOBBY_REFUSAL_FULL},
	{custom.ErrSeatsOpen, pb.LobbyRefusal_LOBBY_REFUSAL_SEATS_OPEN},
	{custom.ErrStarted, pb.LobbyRefusal_LOBBY_REFUSAL_STARTED},
	{custom.ErrInvalidSeat, pb.LobbyRefusal_LOBBY_REFUSAL_INVALID_SEAT},
	{custom.ErrInvalidMode, pb.LobbyRefusal_LOBBY_REFUSAL_INVALID_MODE},
	{custom.ErrInMatch, pb.LobbyRefusal_LOBBY_REFUSAL_IN_MATCH},
	{custom.ErrWrongPhase, pb.LobbyRefusal_LOBBY_REFUSAL_WRONG_PHASE},
	{custom.ErrBlocked, pb.LobbyRefusal_LOBBY_REFUSAL_BLOCKED},
	{errNotFriend, pb.LobbyRefusal_LOBBY_REFUSAL_NOT_FRIEND},
}

func refusalOf(err error) pb.LobbyRefusal {
	var remote refusalError
	if errors.As(err, &remote) {
		return pb.LobbyRefusal(remote)
	}
	var ranked *sanction.RankedBanError
	if errors.As(err, &ranked) {
		return pb.LobbyRefusal_LOBBY_REFUSAL_RANKED_BAN
	}
	for _, r := range refusals {
		if errors.Is(err, r.err) {
			return r.refusal
		}
	}
	return pb.LobbyRefusal_LOBBY_REFUSAL_UNAVAILABLE
}

func stateMessage(state custom.State) *pb.LobbyState {
	out := &pb.LobbyState{Code: state.Code, LeaderAccountId: state.Leader, MemberAccountIds: state.Members, Phase: pb.LobbyPhase(state.Phase), Mode: pb.Mode(state.Mode)}
	for _, seat := range state.Seats {
		out.Seats = append(out.Seats, &pb.LobbySeat{Seat: pb.Seat(seat.Seat) + 1, AccountId: seat.Account})
	}
	return out
}

func seatOf(seat pb.Seat) rules.Seat {
	if seat < pb.Seat_SEAT_NORTH || seat > pb.Seat_SEAT_Y {
		return rules.SeatY + 1
	}
	return rules.Seat(seat - 1)
}

func modeOf(mode pb.Mode) rules.Mode {
	if mode != pb.Mode_MODE_1V1 && mode != pb.Mode_MODE_2V2 {
		return 0
	}
	return rules.Mode(mode)
}

type lobbySink struct {
	manager *Manager
	session *Session
}

func (k lobbySink) Deliver(event custom.Event) {
	k.manager.mu.Lock()
	active, acked := k.session.active, k.session.acked
	k.manager.mu.Unlock()
	if active != nil {
		active.conn.Send(lobbyFrame(event, acked))
	}
}

func lobbyFrame(event custom.Event, acked uint64) *pb.Envelope {
	if event.State != nil {
		return gateway.LobbyStateFrame(stateMessage(*event.State), acked)
	}
	relay := &pb.LobbyRelay{AccountId: event.Account, ItemId: event.Emoji}
	if event.Request {
		relay.RequestedSeat = pb.Seat(event.Seat) + 1
	}
	return &pb.Envelope{ProtocolVersion: gateway.ProtocolVersion, AckedSequence: acked, Payload: &pb.Envelope_LobbyRelay{LobbyRelay: relay}}
}

func (m *Manager) lobbyOf(s *Session) *custom.Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	return s.lobby
}

func (m *Manager) enterLobby(s *Session, r *custom.Room) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s.lobby = r
}

func (m *Manager) exitLobby(account string, r *custom.Room) *connection {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.accounts[account] {
		if s.lobby == r {
			s.lobby = nil
			m.unroute(s)
			return s.active
		}
	}
	return nil
}

func (m *Manager) dropLobby(s *Session) {
	if s.lobby != nil {
		go s.lobby.Leave(s.account)
		s.lobby = nil
		m.unroute(s)
	}
}

func (m *Manager) lobbySession(account string, r *custom.Room) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	for token, s := range m.accounts[account] {
		if s.lobby == r {
			return token
		}
	}
	return ""
}

func (m *Manager) Invited(account string, invite *pb.LobbyInvite) {
	frame := &pb.Envelope{ProtocolVersion: gateway.ProtocolVersion, Payload: &pb.Envelope_LobbyInvite{LobbyInvite: invite}}
	m.mu.Lock()
	var active []*connection
	for _, s := range m.accounts[account] {
		if s.active != nil {
			active = append(active, s.active)
		}
	}
	m.mu.Unlock()
	for _, c := range active {
		c.conn.Send(frame)
	}
}

func (c *connection) sendLobby(state *pb.LobbyState) {
	c.conn.Send(gateway.LobbyStateFrame(state, c.Acked()))
}

func (c *connection) Lobby(intent *pb.LobbyIntent) {
	if c.manager.lobby == nil {
		c.sendLobby(&pb.LobbyState{Refusal: pb.LobbyRefusal_LOBBY_REFUSAL_UNAVAILABLE})
		return
	}
	err := c.command(intent)
	if err == nil {
		return
	}
	state := &pb.LobbyState{}
	if r := c.manager.lobbyOf(c.session); r != nil {
		state = stateMessage(r.State())
	}
	state.Refusal = refusalOf(err)
	c.explainBan(state, err)
	c.sendLobby(state)
}

func (c *connection) explainBan(state *pb.LobbyState, err error) {
	var ranked *sanction.RankedBanError
	if !errors.As(err, &ranked) {
		return
	}
	if ends := ranked.Sanction.EndsAt; ends != nil {
		state.RefusedUntilUnixMs = uint64(ends.UnixMilli())
	}
	state.BanReason = ranked.Sanction.Category.Reason()
	if c.manager.appeal != nil {
		state.Appeal = c.manager.appeal(c.session.account, ranked.Sanction.ID)
	}
}

func (c *connection) command(intent *pb.LobbyIntent) error {
	switch intent.GetCommand() {
	case pb.LobbyCommand_LOBBY_COMMAND_CREATE:
		return c.openCustomRoom(func() error { return c.enter(c.create) })
	case pb.LobbyCommand_LOBBY_COMMAND_JOIN:
		return c.openCustomRoom(func() error { return c.joinCode(intent.GetCode()) })
	}
	r := c.manager.lobbyOf(c.session)
	if r == nil {
		return custom.ErrNotInRoom
	}
	return c.inLobby(r, intent)
}

func (c *connection) openCustomRoom(open func() error) error {
	if err := c.manager.customRoomsOpen(c.session.account); err != nil {
		return err
	}
	return open()
}

func (c *connection) inLobby(r *custom.Room, intent *pb.LobbyIntent) error {
	account := c.session.account
	switch intent.GetCommand() {
	case pb.LobbyCommand_LOBBY_COMMAND_LEAVE:
		return c.leaveLobby(r)
	case pb.LobbyCommand_LOBBY_COMMAND_TEAMS:
		return r.Teams(account, modeOf(intent.GetMode()))
	case pb.LobbyCommand_LOBBY_COMMAND_BACK:
		return r.Back(account)
	case pb.LobbyCommand_LOBBY_COMMAND_PLACE:
		return r.Place(account, intent.GetAccountId(), seatOf(intent.GetSeat()))
	case pb.LobbyCommand_LOBBY_COMMAND_REMOVE:
		return c.remove(r, intent.GetAccountId())
	case pb.LobbyCommand_LOBBY_COMMAND_START:
		return c.start(r)
	case pb.LobbyCommand_LOBBY_COMMAND_INVITE:
		return c.invite(r, intent.GetAccountId())
	case pb.LobbyCommand_LOBBY_COMMAND_REQUEST_SEAT:
		return r.Request(account, seatOf(intent.GetSeat()))
	case pb.LobbyCommand_LOBBY_COMMAND_ADD:
		return c.add(r, intent.GetAccountId(), seatOf(intent.GetSeat()))
	}
	return errUnknownCommand
}

func (m *Manager) enter(s *Session, open func(custom.Member) (*custom.Room, error)) error {
	if m.lobbyOf(s) != nil {
		return custom.ErrAlreadyInRoom
	}
	member, err := m.members(context.Background(), s.account)
	if err != nil {
		return err
	}
	member.Account, member.Sink = s.account, lobbySink{manager: m, session: s}
	r, err := open(member)
	if err != nil {
		return err
	}
	m.enterLobby(s, r)
	return nil
}

func (c *connection) enter(open func(custom.Member) (*custom.Room, error)) error {
	return c.manager.enter(c.session, open)
}

func (c *connection) create(member custom.Member) (*custom.Room, error) {
	r, err := c.manager.lobby.Create(member)
	if err == nil {
		c.sendLobby(stateMessage(r.State()))
	}
	return r, err
}

func (m *Manager) joinLobby(s *Session, code string) func(custom.Member) (*custom.Room, error) {
	return func(member custom.Member) (*custom.Room, error) {
		r, err := m.lobby.Find(code)
		if err != nil {
			return nil, err
		}
		m.enterLobby(s, r)
		if err := r.Join(member); err != nil {
			m.enterLobby(s, nil)
			return nil, err
		}
		return r, nil
	}
}

func (c *connection) joinCode(code string) error {
	if _, err := c.manager.lobby.Find(code); err == nil || c.manager.remote == nil {
		return c.enter(c.manager.joinLobby(c.session, code))
	}
	switch {
	case c.manager.lobbyOf(c.session) != nil:
		return custom.ErrAlreadyInRoom
	case c.InMatch():
		return custom.ErrInMatch
	}
	refusal, err := c.manager.remote.Join(context.Background(), code, c.session.account, c.session.token)
	switch {
	case err != nil:
		return err
	case refusal != pb.LobbyRefusal_LOBBY_REFUSAL_UNSPECIFIED:
		return refusalError(refusal)
	}
	c.conn.Close(websocket.StatusGoingAway, "lobby hosted on another node")
	return nil
}

func (m *Manager) Adopt(ctx context.Context, code, account, token string) pb.LobbyRefusal {
	if m.lobby == nil {
		return pb.LobbyRefusal_LOBBY_REFUSAL_UNAVAILABLE
	}
	if err := m.claim(token); err != nil {
		return pb.LobbyRefusal_LOBBY_REFUSAL_UNAVAILABLE
	}
	m.mu.Lock()
	s := m.ensure(token, account)
	s.routed = true
	m.mu.Unlock()
	err := m.enter(s, m.joinLobby(s, code))
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.unroute(s)
		return refusalOf(err)
	}
	if s.active == nil {
		go m.expire(s, s.generation, m.clock.After(m.config.ReconnectGrace))
	}
	return pb.LobbyRefusal_LOBBY_REFUSAL_UNSPECIFIED
}

func (c *connection) leaveLobby(r *custom.Room) error {
	c.manager.exitLobby(c.session.account, r)
	err := r.Leave(c.session.account)
	c.sendLobby(&pb.LobbyState{})
	return err
}

func (c *connection) remove(r *custom.Room, account string) error {
	if err := r.Remove(c.session.account, account); err != nil {
		return err
	}
	if removed := c.manager.exitLobby(account, r); removed != nil {
		removed.sendLobby(&pb.LobbyState{})
	}
	return nil
}

func (c *connection) start(r *custom.Room) error {
	table, err := r.Start(c.session.account, room.Options{})
	if err != nil {
		return err
	}
	for _, seat := range r.State().Seats {
		c.manager.Seat(c.manager.lobbySession(seat.Account, r), table, seat.Seat)
	}
	return nil
}

func (c *connection) invite(r *custom.Room, to string) error {
	ctx, account := context.Background(), c.session.account
	if c.manager.friends != nil {
		friends, err := c.manager.friends.FriendIDs(ctx, account)
		if err != nil {
			return err
		}
		if !slices.Contains(friends, to) {
			return errNotFriend
		}
	}
	if err := r.Invite(ctx, account, to); err != nil {
		return err
	}
	invite := &pb.LobbyInvite{Code: r.Code(), FromAccountId: account}
	if c.manager.remote != nil {
		return c.manager.remote.Invite(ctx, to, invite)
	}
	c.manager.Invited(to, invite)
	return nil
}

func (c *connection) add(r *custom.Room, account string, seat rules.Seat) error {
	token := c.manager.lobbySession(account, r)
	if token == "" {
		return custom.ErrNotInRoom
	}
	if err := r.Add(c.session.account, custom.Member{Account: account}, seat); err != nil {
		return err
	}
	if table := r.Table(); table != nil {
		return c.manager.Seat(token, table, seat)
	}
	return nil
}
