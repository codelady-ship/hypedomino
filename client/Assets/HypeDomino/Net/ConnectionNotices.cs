using Hypedomino.V1;

namespace HypeDomino.Net
{
    public enum NetworkNotice
    {
        None,
        Weak,
        Reconnecting,
    }

    public enum SeatNotice
    {
        None,
        Dropped,
        BotPlaying,
    }

    public static class ConnectionNotices
    {
        public const double WeakRoundTripMs = 300;
        public const double WeakUnackedSeconds = 2;

        public static NetworkNotice Network(Connection connection, double nowSeconds)
        {
            switch (connection.State)
            {
                case ConnectionState.Waiting:
                case ConnectionState.Connecting:
                    return connection.WasOnline ? NetworkNotice.Reconnecting : NetworkNotice.None;
                case ConnectionState.Online:
                    return IsWeak(connection.RoundTrip, nowSeconds) ? NetworkNotice.Weak : NetworkNotice.None;
                default:
                    return NetworkNotice.None;
            }
        }

        public static SeatNotice Seat(MatchSnapshot view, in SeatState seat)
        {
            if (seat.Controller == Controller.Bot) return IsRosterBot(view, seat.Seat) ? SeatNotice.None : SeatNotice.BotPlaying;
            return seat.Disconnected ? SeatNotice.Dropped : SeatNotice.None;
        }

        public static bool OneTimeoutFromBot(MatchSnapshot view)
        {
            if (view.AfkConsecutiveTimeouts == 0 || view.State == MatchState.Finished) return false;
            foreach (var seat in view.Seats)
            {
                if (seat.Seat != view.Seat) continue;
                return seat.Controller != Controller.Bot && seat.ConsecutiveTimeouts + 1 == view.AfkConsecutiveTimeouts;
            }
            return false;
        }

        public static bool CanTakeBack(MatchSnapshot view)
        {
            if (view.State == MatchState.Finished) return false;
            foreach (var seat in view.Seats)
            {
                if (seat.Seat == view.Seat) return Seat(view, seat) == SeatNotice.BotPlaying;
            }
            return false;
        }

        static bool IsWeak(RoundTrip roundTrip, double nowSeconds) =>
            (roundTrip.HasSample && roundTrip.SmoothedMs > WeakRoundTripMs) ||
            (roundTrip.HasPending && nowSeconds - roundTrip.OldestPendingSentSeconds > WeakUnackedSeconds);

        static bool IsRosterBot(MatchSnapshot view, Seat seat)
        {
            var roster = view.Roster;
            for (var i = 0; i < roster.Count; i++)
            {
                if (roster[i].Seat == seat) return roster[i].Bot;
            }
            return false;
        }
    }
}
