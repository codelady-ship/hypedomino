using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public enum HapticCue
    {
        None,
        TilePlace,
        Score,
        Win,
    }

    public sealed class HapticCues
    {
        bool seen;
        MatchState state;
        Seat current;
        int boardTiles;
        uint teamScore;

        public void StartConnection() => seen = false;

        public HapticCue Read(MatchSnapshot view)
        {
            var tiles = view.Line.Length + view.Top.Length + view.Bottom.Length;
            var seated = view.Seat != Seat.Unspecified;
            var team = seated ? SeatOrder.TeamOf(view.Seat) : Team.Unspecified;
            var score = team == Team.A ? view.Scores.TeamA : team == Team.B ? view.Scores.TeamB : 0;
            var cue = seen && seated ? CueOf(view, team, tiles, score) : HapticCue.None;
            seen = true;
            state = view.State;
            current = view.Current;
            boardTiles = tiles;
            teamScore = score;
            return cue;
        }

        HapticCue CueOf(MatchSnapshot view, Team team, int tiles, uint score)
        {
            if (view.State == MatchState.Finished && state != MatchState.Finished && view.Winner == team) return HapticCue.Win;
            if (score > teamScore) return HapticCue.Score;
            if (tiles > boardTiles && current == view.Seat) return HapticCue.TilePlace;
            return HapticCue.None;
        }
    }
}
