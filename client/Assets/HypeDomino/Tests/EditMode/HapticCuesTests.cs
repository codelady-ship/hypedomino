using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class HapticCuesTests
    {
        MatchSnapshot snapshot;
        HapticCues cues;

        [SetUp]
        public void StartFresh()
        {
            snapshot = new MatchSnapshot();
            cues = new HapticCues();
        }

        static MatchView View(Seat seat, Seat current, int boardTiles, uint teamA = 0, uint teamB = 0)
        {
            var view = new MatchView
            {
                State = MatchState.Playing,
                Seat = seat,
                Mode = Mode._2V2,
                Current = current,
                Scores = new TeamPoints { TeamA = teamA, TeamB = teamB },
                Board = new BoardView(),
            };
            for (uint tile = 0; tile < boardTiles; tile++) view.Board.Line.Add(new OrientedTile { First = tile, Second = tile });
            return view;
        }

        static MatchView Finished(Seat seat, Team winner, uint teamA, uint teamB)
        {
            var view = View(seat, Seat.Unspecified, 0, teamA, teamB);
            view.State = MatchState.Finished;
            view.Winner = winner;
            return view;
        }

        HapticCue Read(MatchView view)
        {
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return cues.Read(snapshot);
        }

        [Test]
        public void TheFirstViewOfAConnectionPlaysNothing()
        {
            Assert.AreEqual(HapticCue.None, Read(View(Seat.South, Seat.South, 5, 100, 0)));
        }

        [Test]
        public void ANewConnectionPlaysNothingOnItsFirstView()
        {
            Read(View(Seat.South, Seat.South, 1));
            cues.StartConnection();
            Assert.AreEqual(HapticCue.None, Read(Finished(Seat.South, Team.A, 370, 0)));
            Assert.AreEqual(HapticCue.None, Read(Finished(Seat.South, Team.A, 370, 0)));
        }

        [Test]
        public void OnlyTheLocalPlayersOwnPlacementTaps()
        {
            Read(View(Seat.South, Seat.South, 1));
            Assert.AreEqual(HapticCue.TilePlace, Read(View(Seat.South, Seat.West, 2)));
            Assert.AreEqual(HapticCue.None, Read(View(Seat.South, Seat.North, 3)), "West placed");
            Assert.AreEqual(HapticCue.None, Read(View(Seat.South, Seat.North, 4)), "the partner placed");
        }

        [Test]
        public void ALocalPlacementInTheMiddleOfAChainTapsEachTime()
        {
            Read(View(Seat.South, Seat.South, 1));
            Assert.AreEqual(HapticCue.TilePlace, Read(View(Seat.South, Seat.South, 2)));
            Assert.AreEqual(HapticCue.TilePlace, Read(View(Seat.South, Seat.West, 3)));
        }

        [Test]
        public void PointsForTheLocalTeamOutrankThePlacement()
        {
            Read(View(Seat.East, Seat.East, 3, 10, 20));
            Assert.AreEqual(HapticCue.Score, Read(View(Seat.East, Seat.South, 4, 10, 30)));
            Assert.AreEqual(HapticCue.Score, Read(View(Seat.East, Seat.North, 5, 10, 35)), "the partner scored for the team");
            Assert.AreEqual(HapticCue.None, Read(View(Seat.East, Seat.West, 6, 25, 35)), "the other team scored");
        }

        [Test]
        public void HandPointsAtTheEndOfAHandScoreEvenAsTheBoardClears()
        {
            Read(View(Seat.X, Seat.X, 9));
            var next = View(Seat.X, Seat.Y, 0, 40, 0);
            next.Mode = Mode._1V1;
            next.State = MatchState.Opening;
            Assert.AreEqual(HapticCue.Score, Read(next));
        }

        [Test]
        public void OnlyTheWinningTeamFeelsTheWinOnce()
        {
            Read(View(Seat.South, Seat.South, 8, 360, 300));
            Assert.AreEqual(HapticCue.Win, Read(Finished(Seat.South, Team.A, 370, 300)));
            Assert.AreEqual(HapticCue.None, Read(Finished(Seat.South, Team.A, 370, 300)));

            StartFresh();
            Read(View(Seat.East, Seat.South, 8, 360, 300));
            Assert.AreEqual(HapticCue.None, Read(Finished(Seat.East, Team.A, 370, 300)));
        }

        [Test]
        public void AViewerWithoutASeatFeelsNothing()
        {
            Read(View(Seat.Unspecified, Seat.South, 1));
            Assert.AreEqual(HapticCue.None, Read(View(Seat.Unspecified, Seat.West, 2, 10, 0)));
            Assert.AreEqual(HapticCue.None, Read(Finished(Seat.Unspecified, Team.A, 370, 0)));
        }

        [Test]
        public void ReadingAllocatesNothing()
        {
            Assert.IsTrue(snapshot.TryRead(View(Seat.South, Seat.South, 3, 10, 0).ToByteArray()));
            TestDelegate step = () => cues.Read(snapshot);
            step();
            Assert.That(step, Allocates.Nothing);
        }
    }
}
