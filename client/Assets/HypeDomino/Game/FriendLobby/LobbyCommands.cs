using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.FriendLobby
{
    public sealed class LobbyCommands
    {
        readonly LobbyIntent intent = new();

        public bool TryCreate(Session session) => TrySend(session, LobbyCommand.Create);

        public bool TryJoin(Session session, string typedCode) =>
            RoomCode.TryNormalize(typedCode, out var code) && TrySend(session, LobbyCommand.Join, code: code);

        public bool TryLeave(Session session) => TrySend(session, LobbyCommand.Leave);

        public bool TryChooseMode(Session session, Mode mode) => TrySend(session, LobbyCommand.Teams, mode: mode);

        public bool TryBack(Session session) => TrySend(session, LobbyCommand.Back);

        public bool TryPlace(Session session, string account, Seat seat) => TrySend(session, LobbyCommand.Place, account: account, seat: seat);

        public bool TryRemove(Session session, string account) => TrySend(session, LobbyCommand.Remove, account: account);

        public bool TryStart(Session session) => TrySend(session, LobbyCommand.Start);

        public bool TryInvite(Session session, string account) => TrySend(session, LobbyCommand.Invite, account: account);

        public bool TryRequestSeat(Session session, Seat seat) => TrySend(session, LobbyCommand.RequestSeat, seat: seat);

        public bool TryAdd(Session session, string account, Seat seat) => TrySend(session, LobbyCommand.Add, account: account, seat: seat);

        bool TrySend(Session session, LobbyCommand command, string code = "", Mode mode = Mode.Unspecified, string account = "", Seat seat = Seat.Unspecified)
        {
            if (!session.Accepted) return false;
            intent.Command = command;
            intent.Code = code;
            intent.Mode = mode;
            intent.AccountId = account ?? string.Empty;
            intent.Seat = seat;
            session.Send(Envelope.PayloadOneofCase.LobbyIntent, intent);
            return true;
        }
    }
}
