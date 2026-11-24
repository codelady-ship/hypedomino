using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ScoreBoardTests
    {
        MatchSnapshot snapshot;
        ScoreBoard board;
        HandSummary summary;

        [SetUp]
        public void StartFresh()
        {
            snapshot = new MatchSnapshot();
            board = new ScoreBoard();
            summary = new HandSummary();
        }

        static MatchView View(Seat seat, uint teamA, uint teamB, params Team[] locked)
        {
            var view = new MatchView
            {
                State = MatchState.Opening,
                Seat = seat,
                Mode = Mode._2V2,
                Scores = new TeamPoints { TeamA = teamA, TeamB = teamB },
                TargetScore = 365,
                HandPointsLockThreshold = 300,
            };
            view.LockedTeams.Add(locked);
            return view;
        }

        static MatchView Ended(uint hands, HandResult result, bool withLeft = true)
        {
            var view = View(Seat.South, 0, 0);
            view.HandsPlayed = hands;
            view.LastResult = result;
            if (!withLeft) return view;
            result.Left.Add(new SeatTiles { Seat = Seat.North, Tiles = { new Tile { Low = 6, High = 6 } } });
            result.Left.Add(new SeatTiles { Seat = Seat.East });
            result.Left.Add(new SeatTiles { Seat = Seat.South, Tiles = { new Tile { Low = 1, High = 2 }, new Tile { Low = 0, High = 3 } } });
            result.Left.Add(new SeatTiles { Seat = Seat.West, Tiles = { new Tile { Low = 4, High = 5 } } });
            return view;
        }

        void Read(MatchView view) => Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));

        [Test]
        public void TheLocalTeamComesFirstAndFillsTowardTheTarget()
        {
            Read(View(Seat.East, 100, 146));
            board.Read(snapshot);
            Assert.AreEqual(Team.B, board.Mine.Team);
            Assert.AreEqual(146u, board.Mine.Score);
            Assert.AreEqual(146f / 365, board.Mine.Fill, 1e-6f);
            Assert.AreEqual(Team.A, board.Theirs.Team);
            Assert.AreEqual(300f / 365, board.LockMark, 1e-6f);
            Assert.AreEqual(365u, board.TargetScore);
        }

        [Test]
        public void AViewerWithoutASeatSeesTeamAFirst()
        {
            Read(View(Seat.Unspecified, 10, 20));
            board.Read(snapshot);
            Assert.AreEqual(Team.A, board.Mine.Team);
        }

        [Test]
        public void TheMeterNeverRunsPastFullAndAMissingTargetLeavesItEmpty()
        {
            Read(View(Seat.North, 400, 0));
            board.Read(snapshot);
            Assert.AreEqual(1f, board.Mine.Fill);

            var noTarget = View(Seat.North, 50, 0);
            noTarget.TargetScore = 0;
            Read(noTarget);
            board.Read(snapshot);
            Assert.AreEqual(0f, board.Mine.Fill);
        }

        [Test]
        public void OnlyTheTeamsTheServerListsShowTheLock()
        {
            Read(View(Seat.North, 310, 305, Team.B));
            board.Read(snapshot);
            Assert.IsFalse(board.Mine.Locked);
            Assert.IsTrue(board.Theirs.Locked);
        }

        [Test]
        public void ADominoSummaryExplainsTheRoundingAndTheCarryOver()
        {
            Read(Ended(3, new HandResult
            {
                Ending = HandEnding.Domino,
                Winner = Team.B,
                Pips = new TeamPoints { TeamA = 22, TeamB = 9 },
                Awarded = new TeamPoints { TeamB = 35 },
                CarriedIn = 10,
            }));
            Assert.IsTrue(summary.Read(snapshot));
            Assert.AreEqual(HandEnding.Domino, summary.Ending);
            Assert.IsTrue(summary.HasWinner);
            Assert.AreEqual(Team.B, summary.Winner);
            Assert.AreEqual(22u, summary.CountedPips);
            Assert.AreEqual(25u, summary.HandPoints);
            Assert.AreEqual(10u, summary.CarriedIn);
            Assert.IsFalse(summary.Locked);
            Assert.AreEqual(4, summary.LeftSeats.Length);
            Assert.AreEqual(Seat.South, summary.LeftSeats[2].Seat);
            Assert.AreEqual(2, summary.LeftTiles(summary.LeftSeats[2]).Length);
            Assert.AreEqual(3u, summary.LeftTiles(summary.LeftSeats[2])[1].High);
            Assert.AreEqual(0, summary.LeftTiles(summary.LeftSeats[1]).Length);
        }

        [Test]
        public void ALockedWinnerGetsNoHandPoints()
        {
            Read(Ended(4, new HandResult
            {
                Ending = HandEnding.Block,
                Winner = Team.A,
                Pips = new TeamPoints { TeamA = 5, TeamB = 18 },
                HandPointsLocked = true,
            }));
            Assert.IsTrue(summary.Read(snapshot));
            Assert.IsTrue(summary.Locked);
            Assert.AreEqual(18u, summary.CountedPips);
            Assert.AreEqual(0u, summary.HandPoints);
            Assert.AreEqual(HandEnding.Block, summary.Ending);
        }

        [Test]
        public void AnEqualBlockKeepsTheCarryOverWithoutAWinner()
        {
            Read(Ended(2, new HandResult
            {
                Ending = HandEnding.Block,
                Pips = new TeamPoints { TeamA = 12, TeamB = 12 },
                CarryOver = 25,
            }));
            Assert.IsTrue(summary.Read(snapshot));
            Assert.IsFalse(summary.HasWinner);
            Assert.AreEqual(25u, summary.CarryOverKept);
            Assert.AreEqual(0u, summary.HandPoints);
            Assert.AreEqual(0u, summary.CountedPips);
        }

        [Test]
        public void TheSummaryOpensOncePerHandEndAndOnlyWithTheTilesLeft()
        {
            var result = new HandResult { Ending = HandEnding.Domino, Winner = Team.A, Pips = new TeamPoints { TeamB = 7 }, Awarded = new TeamPoints { TeamA = 10 } };
            Read(Ended(1, result.Clone(), withLeft: false));
            Assert.IsFalse(summary.Read(snapshot), "a reconnect during play brings no tiles left and no summary");

            Read(Ended(1, result.Clone()));
            Assert.IsTrue(summary.Read(snapshot));
            Read(Ended(1, result.Clone(), withLeft: false));
            Assert.IsFalse(summary.Read(snapshot));
            Assert.AreEqual(4, summary.LeftSeats.Length, "the tiles stay until the next hand end");
            Read(Ended(1, result.Clone()));
            Assert.IsFalse(summary.Read(snapshot), "a new connection resends the same hand end");

            Read(Ended(2, result.Clone()));
            Assert.IsTrue(summary.Read(snapshot));
            Assert.AreEqual(2u, summary.Hand);
        }

        [Test]
        public void ReadingAllocatesNothing()
        {
            Read(Ended(1, new HandResult { Ending = HandEnding.Domino, Winner = Team.A, Awarded = new TeamPoints { TeamA = 5 } }));
            TestDelegate step = () =>
            {
                board.Read(snapshot);
                summary.Read(snapshot);
            };
            step();
            Assert.That(step, Allocates.Nothing);
        }
    }
}
