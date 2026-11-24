using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchResultTests
    {
        MatchSnapshot snapshot;
        MatchResult result;

        [SetUp]
        public void StartFresh()
        {
            snapshot = new MatchSnapshot();
            result = new MatchResult();
        }

        static MatchView Playing(Seat seat, Mode mode = Mode._2V2) => new()
        {
            State = MatchState.Playing,
            Seat = seat,
            Mode = mode,
            HandsPlayed = 6,
            Scores = new TeamPoints { TeamA = 340, TeamB = 300 },
            TargetScore = 365,
        };

        static MatchView Finished(Seat seat, Team winner, uint teamA, uint teamB, Mode mode = Mode._2V2)
        {
            var view = Playing(seat, mode);
            view.State = MatchState.Finished;
            view.HandsPlayed = 7;
            view.Winner = winner;
            view.Scores = new TeamPoints { TeamA = teamA, TeamB = teamB };
            return view;
        }

        bool Read(MatchView view)
        {
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return result.Read(snapshot);
        }

        [Test]
        public void TheWinningTeamSeesTheWinWithItsOwnScoreFirst()
        {
            Assert.IsTrue(Read(Finished(Seat.East, Team.B, 310, 370)));
            Assert.AreEqual(MatchOutcome.Won, result.Outcome);
            Assert.AreEqual(Team.B, result.Winner);
            Assert.AreEqual(Team.B, result.MyTeam);
            Assert.AreEqual(370u, result.MyScore);
            Assert.AreEqual(310u, result.TheirScore);
            Assert.AreEqual(7u, result.HandsPlayed);
            Assert.AreEqual(Mode._2V2, result.Mode);
        }

        [Test]
        public void TheOtherTeamSeesTheLoss()
        {
            Assert.IsTrue(Read(Finished(Seat.X, Team.B, 300, 365, Mode._1V1)));
            Assert.AreEqual(MatchOutcome.Lost, result.Outcome);
            Assert.AreEqual(Team.A, result.MyTeam);
            Assert.AreEqual(300u, result.MyScore);
            Assert.AreEqual(365u, result.TheirScore);
            Assert.AreEqual(Mode._1V1, result.Mode);
        }

        [Test]
        public void AViewerWithoutASeatSeesTheWinnerButNoOutcome()
        {
            Assert.IsTrue(Read(Finished(Seat.Unspecified, Team.B, 200, 380)));
            Assert.AreEqual(MatchOutcome.None, result.Outcome);
            Assert.AreEqual(Team.B, result.Winner);
            Assert.AreEqual(Team.A, result.MyTeam);
            Assert.AreEqual(200u, result.MyScore);
        }

        [Test]
        public void AFinishWithoutAWinnerHasNoOutcome()
        {
            Assert.IsTrue(Read(Finished(Seat.South, Team.Unspecified, 120, 90)));
            Assert.AreEqual(MatchOutcome.None, result.Outcome);
        }

        [Test]
        public void TheResultOpensOncePerMatchEnd()
        {
            Assert.IsFalse(Read(Playing(Seat.South)));
            Assert.IsTrue(Read(Finished(Seat.South, Team.A, 370, 300)));
            Assert.IsFalse(Read(Finished(Seat.South, Team.A, 370, 300)));

            Assert.IsFalse(Read(Playing(Seat.North)));
            Assert.IsTrue(Read(Finished(Seat.North, Team.B, 330, 366)));
            Assert.AreEqual(MatchOutcome.Lost, result.Outcome);
        }

        [Test]
        public void ReadingAllocatesNothing()
        {
            Assert.IsTrue(snapshot.TryRead(Finished(Seat.South, Team.A, 370, 300).ToByteArray()));
            TestDelegate step = () => result.Read(snapshot);
            step();
            Assert.That(step, Allocates.Nothing);
        }
    }
}
