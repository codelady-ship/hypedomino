using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SoundCuesTests
    {
        MatchSnapshot snapshot;
        SoundCues cues;

        [SetUp]
        public void StartFresh()
        {
            snapshot = new MatchSnapshot();
            cues = new SoundCues();
        }

        static MatchView View(Seat seat, Seat current, int boardTiles, uint teamA = 0, uint teamB = 0)
        {
            var view = new MatchView
            {
                State = MatchState.Playing,
                Seat = seat,
                Mode = Mode._2V2,
                Current = current,
                HandsPlayed = 1,
                Scores = new TeamPoints { TeamA = teamA, TeamB = teamB },
                Board = new BoardView(),
            };
            for (uint tile = 0; tile < boardTiles; tile++) view.Board.Line.Add(new OrientedTile { First = tile, Second = tile });
            return view;
        }

        static MatchView OneOnOne(Seat current, int boardTiles, uint boneyard, uint placements = 0)
        {
            var view = View(Seat.X, current, boardTiles);
            view.Mode = Mode._1V1;
            view.BoneyardSize = boneyard;
            view.TurnPlacements = placements;
            return view;
        }

        static MatchView HandEnded(Seat seat, uint handsPlayed, uint teamA)
        {
            var view = View(seat, Seat.West, 0, teamA);
            view.State = MatchState.Opening;
            view.HandsPlayed = handsPlayed;
            view.LastResult = new HandResult { Winner = Team.A, Ending = HandEnding.Domino };
            view.LastResult.Left.Add(new SeatTiles { Seat = Seat.West, Tiles = { new Tile { Low = 2, High = 3 } } });
            return view;
        }

        static MatchView Finished(Seat seat, Team winner)
        {
            var view = HandEnded(seat, 9, 370);
            view.State = MatchState.Finished;
            view.Winner = winner;
            return view;
        }

        MatchSounds Read(MatchView view)
        {
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return cues.Read(snapshot);
        }

        [Test]
        public void TheFirstViewOfAConnectionPlaysNothing()
        {
            Assert.AreEqual(MatchSounds.None, Read(View(Seat.South, Seat.South, 5, 100, 0)));
            cues.StartConnection();
            Assert.AreEqual(MatchSounds.None, Read(Finished(Seat.South, Team.A)));
        }

        [Test]
        public void EverySeatsPlacementPlaysTheTileSound()
        {
            Read(View(Seat.South, Seat.South, 1));
            Assert.AreEqual(MatchSounds.TilePlace, Read(View(Seat.South, Seat.West, 2)));
            Assert.AreEqual(MatchSounds.TilePlace, Read(View(Seat.South, Seat.North, 3)), "West placed");
            Assert.AreEqual(MatchSounds.None, Read(View(Seat.South, Seat.North, 3)), "a view without a new tile");
        }

        [Test]
        public void ADrawFromTheBoneyardPlaysTheDrawSound()
        {
            Read(OneOnOne(Seat.Y, 4, 10));
            Assert.AreEqual(MatchSounds.TileDraw, Read(OneOnOne(Seat.Y, 4, 9)));
            Assert.AreEqual(MatchSounds.TilePlace, Read(OneOnOne(Seat.X, 5, 9)));
        }

        [Test]
        public void ATurnThatMovesOnWithoutAPlacementOrDrawIsAPass()
        {
            Read(View(Seat.South, Seat.West, 6));
            Assert.AreEqual(MatchSounds.Pass, Read(View(Seat.South, Seat.North, 6)));
            Read(OneOnOne(Seat.Y, 6, 1));
            Assert.AreEqual(MatchSounds.Pass, Read(OneOnOne(Seat.X, 6, 1)), "the last boneyard tile is never drawn");
        }

        [Test]
        public void EndingATurnOfSeveralDoublesIsNoPass()
        {
            Read(OneOnOne(Seat.X, 6, 5, 1));
            Assert.AreEqual(MatchSounds.TilePlace, Read(OneOnOne(Seat.X, 7, 5, 2)));
            Assert.AreEqual(MatchSounds.None, Read(OneOnOne(Seat.Y, 7, 5)));
        }

        [Test]
        public void OnlyTheLocalTeamsPointsPlayAComboSound()
        {
            Read(View(Seat.East, Seat.East, 3, 10, 20));
            Assert.AreEqual(MatchSounds.TilePlace | MatchSounds.Combo2, Read(View(Seat.East, Seat.South, 4, 10, 30)));
            Assert.AreEqual(MatchSounds.TilePlace, Read(View(Seat.East, Seat.West, 5, 25, 30)), "the other team scored");
        }

        [TestCase(5u, MatchSounds.Combo1)]
        [TestCase(10u, MatchSounds.Combo2)]
        [TestCase(15u, MatchSounds.Combo3)]
        [TestCase(20u, MatchSounds.Combo4)]
        [TestCase(25u, MatchSounds.Combo5)]
        [TestCase(30u, MatchSounds.Combo5)]
        [TestCase(45u, MatchSounds.Combo5)]
        public void TheComboSoundFollowsTheScoreCascadeStep(uint points, MatchSounds combo)
        {
            Read(View(Seat.South, Seat.South, 3, 100, 0));
            Assert.AreEqual(MatchSounds.TilePlace | combo, Read(View(Seat.South, Seat.West, 4, 100 + points, 0)));
        }

        [Test]
        public void AChainOfPlacementsRaisesTheComboStepAsTheCascadeDoes()
        {
            Read(OneOnOne(Seat.X, 6, 5, 1));
            var view = OneOnOne(Seat.X, 7, 5, 2);
            view.Scores.TeamA = 10;
            Assert.AreEqual(MatchSounds.TilePlace | MatchSounds.Combo3, Read(view));
        }

        [Test]
        public void ARiseBelowTheFirstStepPlaysTheScoreSound()
        {
            Assert.AreEqual(MatchSounds.Score, SoundCues.ScoreSoundOf(0));
            Assert.AreEqual(MatchSounds.Combo5, SoundCues.ScoreSoundOf(6));
        }

        [Test]
        public void OneScoringViewPlaysOneComboAndNoScoreSound()
        {
            Read(View(Seat.South, Seat.South, 3, 0, 0));
            var sounds = Read(View(Seat.South, Seat.West, 4, 20, 0));
            Assert.AreEqual(MatchSounds.None, sounds & MatchSounds.Score);
            Assert.AreEqual(MatchSounds.Combo4, sounds & ~MatchSounds.TilePlace);
        }

        [Test]
        public void AHandEndCarriesNoComboSound()
        {
            Read(View(Seat.South, Seat.South, 9, 0, 0));
            Assert.AreEqual(MatchSounds.HandEnd, Read(HandEnded(Seat.South, 2, 40)));
        }

        [Test]
        public void AHandEndPlaysOnceAndAloneAsTheBoardClears()
        {
            Read(View(Seat.South, Seat.South, 9));
            Assert.AreEqual(MatchSounds.HandEnd, Read(HandEnded(Seat.South, 2, 40)));
            Assert.AreEqual(MatchSounds.None, Read(HandEnded(Seat.South, 2, 40)));
        }

        [Test]
        public void TheMatchEndPlaysTheOutcomeOfTheLocalTeamOnce()
        {
            Read(View(Seat.South, Seat.South, 8));
            Assert.AreEqual(MatchSounds.MatchWin, Read(Finished(Seat.South, Team.A)));
            Assert.AreEqual(MatchSounds.None, Read(Finished(Seat.South, Team.A)));

            StartFresh();
            Read(View(Seat.East, Seat.South, 8));
            Assert.AreEqual(MatchSounds.MatchLose, Read(Finished(Seat.East, Team.A)));
        }

        [Test]
        public void AViewerHearsTheBoardButNoOutcomeOrScore()
        {
            Read(View(Seat.Unspecified, Seat.South, 1));
            Assert.AreEqual(MatchSounds.TilePlace, Read(View(Seat.Unspecified, Seat.West, 2, 10, 0)));
            Assert.AreEqual(MatchSounds.HandEnd, Read(Finished(Seat.Unspecified, Team.A)));
        }

        [Test]
        public void TheTurnWarningPlaysOncePerLocalTurn()
        {
            Assert.IsFalse(cues.ReadTurnWarning(false));
            Assert.IsTrue(cues.ReadTurnWarning(true));
            Assert.IsFalse(cues.ReadTurnWarning(true));
            Assert.IsFalse(cues.ReadTurnWarning(false));
            Assert.IsTrue(cues.ReadTurnWarning(true), "the next local turn");
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
