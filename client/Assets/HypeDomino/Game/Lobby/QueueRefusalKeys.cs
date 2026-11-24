using Hypedomino.V1;

namespace HypeDomino.Game.Lobby
{
    public static class QueueRefusalKeys
    {
        public const string InMatch = "lobby.queue.refused.in_match";
        public const string MatchmakingBan = "lobby.queue.refused.matchmaking_ban";
        public const string RankedClosed = "lobby.queue.refused.ranked_closed";
        public const string Unavailable = "lobby.queue.refused.unavailable";
        public const string IntegrityFailed = "lobby.queue.refused.integrity_failed";
        public const string IntegrityUnavailable = "lobby.queue.refused.integrity_unavailable";
        public const string RankedBan = "lobby.queue.refused.ranked_ban";
        public const string Unknown = "notice.generic";

        public static string Of(QueueRefusal refusal) => refusal switch
        {
            QueueRefusal.InMatch => InMatch,
            QueueRefusal.MatchmakingBan => MatchmakingBan,
            QueueRefusal.RankedClosed => RankedClosed,
            QueueRefusal.Unavailable => Unavailable,
            QueueRefusal.IntegrityFailed => IntegrityFailed,
            QueueRefusal.IntegrityUnavailable => IntegrityUnavailable,
            QueueRefusal.RankedBan => RankedBan,
            _ => Unknown,
        };
    }
}
