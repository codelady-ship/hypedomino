using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public enum MatchOutcome
    {
        None,
        Won,
        Lost,
    }

    public sealed class MatchResult
    {
        bool open;

        public MatchOutcome Outcome { get; private set; }
        public Team Winner { get; private set; }
        public Mode Mode { get; private set; }
        public uint HandsPlayed { get; private set; }
        public Team MyTeam { get; private set; }
        public uint MyScore { get; private set; }
        public uint TheirScore { get; private set; }

        public bool Read(MatchSnapshot view)
        {
            if (view.State != MatchState.Finished)
            {
                open = false;
                return false;
            }
            if (open) return false;
            open = true;
            Explain(view);
            return true;
        }

        void Explain(MatchSnapshot view)
        {
            var seated = view.Seat != Seat.Unspecified;
            MyTeam = seated ? SeatOrder.TeamOf(view.Seat) : Team.A;
            Winner = view.Winner;
            Mode = view.Mode;
            HandsPlayed = view.HandsPlayed;
            MyScore = MyTeam == Team.A ? view.Scores.TeamA : view.Scores.TeamB;
            TheirScore = MyTeam == Team.A ? view.Scores.TeamB : view.Scores.TeamA;
            Outcome = !seated || Winner == Team.Unspecified ? MatchOutcome.None : Winner == MyTeam ? MatchOutcome.Won : MatchOutcome.Lost;
        }
    }
}
