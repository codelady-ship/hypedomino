using HypeDomino.Net;
using Hypedomino.V1;
using UnityEngine;

namespace HypeDomino.Game.Match
{
    public readonly struct TeamMeter
    {
        public readonly Team Team;
        public readonly uint Score;
        public readonly float Fill;
        public readonly bool Locked;

        public TeamMeter(Team team, uint score, float fill, bool locked)
        {
            Team = team;
            Score = score;
            Fill = fill;
            Locked = locked;
        }
    }

    public sealed class ScoreBoard
    {
        public uint TargetScore { get; private set; }
        public uint LockThreshold { get; private set; }
        public float LockMark { get; private set; }
        public TeamMeter Mine { get; private set; }
        public TeamMeter Theirs { get; private set; }

        public void Read(MatchSnapshot view)
        {
            TargetScore = view.TargetScore;
            LockThreshold = view.HandPointsLockThreshold;
            LockMark = FillOf(LockThreshold);
            var mine = view.Seat == Seat.Unspecified ? Team.A : SeatOrder.TeamOf(view.Seat);
            var theirs = mine == Team.A ? Team.B : Team.A;
            Mine = Meter(view, mine);
            Theirs = Meter(view, theirs);
        }

        TeamMeter Meter(MatchSnapshot view, Team team)
        {
            var score = team == Team.A ? view.Scores.TeamA : view.Scores.TeamB;
            return new TeamMeter(team, score, FillOf(score), view.IsHandPointsLocked(team));
        }

        float FillOf(uint points) => TargetScore == 0 ? 0f : Mathf.Clamp01((float)points / TargetScore);
    }
}
