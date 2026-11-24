using Hypedomino.V1;

namespace HypeDomino.Game.FriendLobby
{
    public static class LobbyRefusalKeys
    {
        public const string Unavailable = "lobby.room.refused.unavailable";
        public const string UnknownCode = "lobby.room.refused.unknown_code";
        public const string NotLeader = "lobby.room.refused.not_leader";
        public const string SeatTaken = "lobby.room.refused.seat_taken";
        public const string NotInLobby = "lobby.room.refused.not_in_lobby";
        public const string AlreadyInLobby = "lobby.room.refused.already_in_lobby";
        public const string Full = "lobby.room.refused.full";
        public const string SeatsOpen = "lobby.room.refused.seats_open";
        public const string Started = "lobby.room.refused.started";
        public const string InvalidSeat = "lobby.room.refused.invalid_seat";
        public const string InvalidMode = "lobby.room.refused.invalid_mode";
        public const string InMatch = "lobby.room.refused.in_match";
        public const string WrongPhase = "lobby.room.refused.wrong_phase";
        public const string Blocked = "lobby.room.refused.blocked";
        public const string NotFriend = "lobby.room.refused.not_friend";
        public const string RankedBan = "lobby.room.refused.ranked_ban";
        public const string Unknown = "notice.generic";

        public static string Of(LobbyRefusal refusal) => refusal switch
        {
            LobbyRefusal.Unspecified => null,
            LobbyRefusal.Unavailable => Unavailable,
            LobbyRefusal.UnknownCode => UnknownCode,
            LobbyRefusal.NotLeader => NotLeader,
            LobbyRefusal.SeatTaken => SeatTaken,
            LobbyRefusal.NotInLobby => NotInLobby,
            LobbyRefusal.AlreadyInLobby => AlreadyInLobby,
            LobbyRefusal.Full => Full,
            LobbyRefusal.SeatsOpen => SeatsOpen,
            LobbyRefusal.Started => Started,
            LobbyRefusal.InvalidSeat => InvalidSeat,
            LobbyRefusal.InvalidMode => InvalidMode,
            LobbyRefusal.InMatch => InMatch,
            LobbyRefusal.WrongPhase => WrongPhase,
            LobbyRefusal.Blocked => Blocked,
            LobbyRefusal.NotFriend => NotFriend,
            LobbyRefusal.RankedBan => RankedBan,
            _ => Unknown,
        };
    }
}
