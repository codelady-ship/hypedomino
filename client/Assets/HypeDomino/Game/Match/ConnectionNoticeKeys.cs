using HypeDomino.Net;

namespace HypeDomino.Game.Match
{
    public static class ConnectionNoticeKeys
    {
        public const string Reconnecting = "match.connection.reconnecting";
        public const string Weak = "match.connection.weak";
        public const string Dropped = "match.seat.dropped";
        public const string BotPlaying = "match.seat.bot_playing";
        public const string OneTimeoutFromBot = "match.afk.warning";
        public const string TakeBack = "match.afk.take_back";

        public static string Of(NetworkNotice notice) => notice switch
        {
            NetworkNotice.Reconnecting => Reconnecting,
            NetworkNotice.Weak => Weak,
            _ => null,
        };

        public static string Of(SeatNotice notice) => notice switch
        {
            SeatNotice.Dropped => Dropped,
            SeatNotice.BotPlaying => BotPlaying,
            _ => null,
        };
    }
}
