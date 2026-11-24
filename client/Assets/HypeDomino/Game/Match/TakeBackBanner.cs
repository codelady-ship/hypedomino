using HypeDomino.Net;

namespace HypeDomino.Game.Match
{
    public sealed class TakeBackBanner
    {
        bool rejoinSent;

        public bool Shown { get; private set; }

        public void Read(MatchSnapshot view)
        {
            Shown = ConnectionNotices.CanTakeBack(view);
            rejoinSent = false;
        }

        public bool Tap(Connection connection)
        {
            if (!Shown || rejoinSent) return false;
            rejoinSent = connection.Rejoin();
            return rejoinSent;
        }
    }
}
