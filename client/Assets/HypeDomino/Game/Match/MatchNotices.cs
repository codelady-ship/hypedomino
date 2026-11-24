using HypeDomino.Net;

namespace HypeDomino.Game.Match
{
    public sealed class MatchNotices
    {
        public NetworkNotice Network { get; private set; }
        public bool AfkWarning { get; private set; }
        public SeatBadges Seats { get; } = new();
        public TakeBackBanner TakeBack { get; } = new();

        public bool Read(MatchSnapshot view)
        {
            var seatsChanged = Seats.Read(view);
            var takeBackWasShown = TakeBack.Shown;
            TakeBack.Read(view);
            var warning = ConnectionNotices.OneTimeoutFromBot(view);
            var warningChanged = warning != AfkWarning;
            AfkWarning = warning;
            return seatsChanged || warningChanged || takeBackWasShown != TakeBack.Shown;
        }

        public bool Tick(Connection connection, double nowSeconds)
        {
            var next = ConnectionNotices.Network(connection, nowSeconds);
            if (next == Network) return false;
            Network = next;
            return true;
        }
    }
}
