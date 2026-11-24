using System;
using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class HandChoiceTests
    {
        readonly MatchSnapshot snapshot = new();
        readonly HandChoice choice = new();
        readonly ChainLayout board = new(MatchSnapshot.MaxTiles);
        readonly Vector2[] places = new Vector2[HandChoice.MaxEnds];

        static Tile T(uint low, uint high) => new() { Low = low, High = high };

        static LegalMove Move(uint low, uint high, BoardEnd end) => new() { Tile = T(low, high), End = end };

        static MatchView Playing(Seat seat, Seat current, params LegalMove[] legal)
        {
            var view = new MatchView { State = MatchState.Playing, Seat = seat, Mode = Mode._2V2, Current = current };
            view.Hand.Add(new[] { T(1, 2), T(3, 3), T(4, 6), T(5, 5) });
            view.LegalMoves.Add(legal);
            return view;
        }

        void Read(MatchView view)
        {
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            choice.Read(snapshot);
        }

        int IndexOf(uint low, uint high)
        {
            for (var i = 0; i < choice.HandCount; i++)
                if (choice.TileAt(i).Low == low && choice.TileAt(i).High == high) return i;
            throw new ArgumentException($"{low}-{high} is not in the hand");
        }

        [Test]
        public void OnlyTilesTheServerListsAreSelectable()
        {
            Read(Playing(Seat.South, Seat.South, Move(1, 2, BoardEnd.Left), Move(4, 6, BoardEnd.Right)));
            Assert.IsTrue(choice.IsSelectable(IndexOf(1, 2)));
            Assert.IsTrue(choice.IsSelectable(IndexOf(4, 6)));
            Assert.IsFalse(choice.IsSelectable(IndexOf(3, 3)));
            Assert.IsFalse(choice.IsSelectable(IndexOf(5, 5)));
            Assert.IsFalse(choice.IsSelectable(-1));
            Assert.IsFalse(choice.IsSelectable(4));
        }

        [Test]
        public void NothingIsSelectableOnAnotherSeatsTurn()
        {
            Read(Playing(Seat.South, Seat.West, Move(1, 2, BoardEnd.Left)));
            for (var i = 0; i < choice.HandCount; i++) Assert.IsFalse(choice.IsSelectable(i));
            Assert.AreEqual(HandTap.Ignored, choice.Tap(IndexOf(1, 2), out _));
            Assert.IsFalse(choice.CanPass);
            Assert.IsFalse(choice.CanEndTurn);
        }

        [Test]
        public void ATileWithOneLegalEndIsPlacedOnTap()
        {
            Read(Playing(Seat.South, Seat.South, Move(1, 2, BoardEnd.Left)));
            Assert.AreEqual(HandTap.Placed, choice.Tap(IndexOf(1, 2), out var placement));
            Assert.AreEqual(1u, placement.Tile.Low);
            Assert.AreEqual(2u, placement.Tile.High);
            Assert.AreEqual(BoardEnd.Left, placement.End);
            Assert.IsTrue(choice.AwaitingView);
        }

        [Test]
        public void ATileWithSeveralEndsWaitsForTheEnd()
        {
            Read(Playing(Seat.South, Seat.South, Move(4, 6, BoardEnd.Left), Move(4, 6, BoardEnd.Top), Move(1, 2, BoardEnd.Right)));
            var tile = IndexOf(4, 6);
            Span<BoardEnd> ends = stackalloc BoardEnd[HandChoice.MaxEnds];
            Assert.AreEqual(2, choice.EndsOf(tile, ends));
            Assert.AreEqual(BoardEnd.Left, ends[0]);
            Assert.AreEqual(BoardEnd.Top, ends[1]);

            Assert.AreEqual(HandTap.Selected, choice.Tap(tile, out _));
            Assert.AreEqual(tile, choice.Selected);
            Assert.IsFalse(choice.TryChooseEnd(BoardEnd.Right, out _));
            Assert.IsTrue(choice.TryChooseEnd(BoardEnd.Top, out var placement));
            Assert.AreEqual(BoardEnd.Top, placement.End);
            Assert.AreEqual(6u, placement.Tile.High);
            Assert.AreEqual(-1, choice.Selected);
        }

        [Test]
        public void TappingTheSelectedTileAgainClearsTheSelectionAndAnotherTileMovesIt()
        {
            Read(Playing(Seat.South, Seat.South, Move(4, 6, BoardEnd.Left), Move(4, 6, BoardEnd.Right), Move(5, 5, BoardEnd.Left), Move(5, 5, BoardEnd.Right)));
            Assert.AreEqual(HandTap.Selected, choice.Tap(IndexOf(4, 6), out _));
            Assert.AreEqual(HandTap.Deselected, choice.Tap(IndexOf(4, 6), out _));
            Assert.AreEqual(-1, choice.Selected);
            Assert.IsFalse(choice.TryChooseEnd(BoardEnd.Left, out _));

            choice.Tap(IndexOf(4, 6), out _);
            Assert.AreEqual(HandTap.Selected, choice.Tap(IndexOf(5, 5), out _));
            Assert.AreEqual(IndexOf(5, 5), choice.Selected);
        }

        [Test]
        public void NoInputIsTakenUntilTheNextView()
        {
            var view = Playing(Seat.South, Seat.South, Move(1, 2, BoardEnd.Left), Move(4, 6, BoardEnd.Right));
            Read(view);
            choice.Tap(IndexOf(1, 2), out _);
            Assert.AreEqual(HandTap.Ignored, choice.Tap(IndexOf(4, 6), out _));
            Assert.IsFalse(choice.IsSelectable(IndexOf(4, 6)));

            Read(view);
            Assert.IsFalse(choice.AwaitingView);
            Assert.AreEqual(HandTap.Placed, choice.Tap(IndexOf(4, 6), out _));
        }

        [Test]
        public void ASelectionSurvivesANewViewOnlyWhileTheTileStillHasSeveralEnds()
        {
            Read(Playing(Seat.South, Seat.South, Move(4, 6, BoardEnd.Left), Move(4, 6, BoardEnd.Right)));
            choice.Tap(IndexOf(4, 6), out _);

            var reordered = Playing(Seat.South, Seat.South, Move(4, 6, BoardEnd.Left), Move(4, 6, BoardEnd.Right));
            reordered.Hand.Clear();
            reordered.Hand.Add(new[] { T(4, 6), T(1, 2) });
            Read(reordered);
            Assert.AreEqual(0, choice.Selected);

            Read(Playing(Seat.South, Seat.South, Move(4, 6, BoardEnd.Left)));
            Assert.AreEqual(-1, choice.Selected);
        }

        [Test]
        public void DroppingNearAnEndPlacesTheTileThereAndElsewhereReturnsIt()
        {
            Read(Playing(Seat.South, Seat.South, Move(4, 6, BoardEnd.Left), Move(4, 6, BoardEnd.Right)));
            var tile = IndexOf(4, 6);
            ReadOnlySpan<Vector2> endPositions = new[] { new Vector2(-400, 0), new Vector2(400, 0) };

            Assert.IsFalse(choice.TryDrop(tile, endPositions, new Vector2(0, 0), out _));
            Assert.IsFalse(choice.AwaitingView);
            Assert.IsFalse(choice.TryDrop(IndexOf(3, 3), endPositions, new Vector2(400, 0), out _));

            Assert.IsTrue(choice.TryDrop(tile, endPositions, new Vector2(400 - HandChoice.SnapDistance + 1, 0), out var placement));
            Assert.AreEqual(BoardEnd.Right, placement.End);
            Assert.IsTrue(choice.AwaitingView);
        }

        [Test]
        public void EachLegalEndIsPlacedOnItsOpenHalfOfTheBoard()
        {
            Read(Playing(Seat.South, Seat.South, Move(4, 6, BoardEnd.Left), Move(4, 6, BoardEnd.Top), Move(1, 2, BoardEnd.Right)));
            ArrangeSpinnerBoard();
            var tile = IndexOf(4, 6);

            Assert.AreEqual(2, choice.EndPlacesOf(tile, board, places));
            Assert.AreEqual(new Vector2(-20f, -10f), places[0]);
            Assert.AreEqual(new Vector2(0f, 15f), places[1]);
            Assert.AreEqual(0, choice.EndPlacesOf(IndexOf(3, 3), board, places));

            Assert.IsTrue(choice.TryDrop(tile, places.AsSpan(0, 2), new Vector2(50f, 15f), out var placement));
            Assert.AreEqual(BoardEnd.Top, placement.End);
        }

        [Test]
        public void TheOpeningIsPlacedAtTheBoardCenter()
        {
            var view = new MatchView { State = MatchState.Opening, Seat = Seat.South, Mode = Mode._2V2, OpeningTeam = Team.A, HandsPlayed = 2 };
            view.Hand.Add(new[] { T(1, 2), T(6, 6) });
            Read(view);
            places[0] = Vector2.one;

            Assert.AreEqual(1, choice.EndPlacesOf(0, board, places));
            Assert.AreEqual(Vector2.zero, places[0]);
        }

        [Test]
        public void PlacingTheEndsAllocatesNothing()
        {
            Read(Playing(Seat.South, Seat.South, Move(4, 6, BoardEnd.Left), Move(4, 6, BoardEnd.Top)));
            ArrangeSpinnerBoard();
            var tile = IndexOf(4, 6);
            TestDelegate step = () => choice.EndPlacesOf(tile, board, places);
            step();
            Assert.That(step, Allocates.Nothing);
        }

        void ArrangeSpinnerBoard()
        {
            var line = new[] { new Domino(6, 3), new Domino(3, 3), new Domino(3, 1) };
            board.Arrange(line, 1, new[] { new Domino(3, 4) }, new Domino[0], new Vector2(100f, 100f), 10f);
        }

        [Test]
        public void AForcedOpeningAllowsOnlyThatTileForThatSeat()
        {
            var view = new MatchView { State = MatchState.Opening, Seat = Seat.South, Mode = Mode._2V2, OpeningTeam = Team.A, ForcedOpening = new Opening { Seat = Seat.South, Tile = T(2, 3) } };
            view.Hand.Add(new[] { T(1, 2), T(2, 3), T(6, 6) });
            Read(view);
            Assert.IsTrue(choice.IsSelectable(IndexOf(2, 3)));
            Assert.IsFalse(choice.IsSelectable(IndexOf(6, 6)));
            Assert.AreEqual(HandTap.Placed, choice.Tap(IndexOf(2, 3), out var placement));
            Assert.AreEqual(BoardEnd.Unspecified, placement.End);

            view.ForcedOpening.Seat = Seat.North;
            Read(view);
            for (var i = 0; i < choice.HandCount; i++) Assert.IsFalse(choice.IsSelectable(i));
        }

        [Test]
        public void BothPlayersOfTheOpeningTeamMayOpenWithAnyTile()
        {
            var view = new MatchView { State = MatchState.Opening, Seat = Seat.North, Mode = Mode._2V2, OpeningTeam = Team.A, HandsPlayed = 2 };
            view.Hand.Add(new[] { T(1, 2), T(6, 6) });
            Read(view);
            Assert.IsTrue(choice.IsSelectable(0));
            Assert.IsTrue(choice.IsSelectable(1));

            view.Seat = Seat.East;
            Read(view);
            Assert.IsFalse(choice.IsSelectable(0));
        }

        [Test]
        public void AfterAScoringDoubleOnlyTheListedDoublesStayAndTheTurnCanEnd()
        {
            var view = Playing(Seat.South, Seat.South, Move(5, 5, BoardEnd.Right));
            view.TurnPlacements = 1;
            Read(view);
            Assert.IsTrue(choice.IsSelectable(IndexOf(5, 5)));
            Assert.IsFalse(choice.IsSelectable(IndexOf(3, 3)));
            Assert.IsTrue(choice.CanEndTurn);
            Assert.IsFalse(choice.CanPass);
            Assert.IsTrue(choice.TryEndTurn());
            Assert.IsFalse(choice.CanEndTurn);
            Assert.IsFalse(choice.TryEndTurn());
        }

        [Test]
        public void WithNoLegalMoveTheOneVersusOnePlayerDrawsWhileTheBoneyardHoldsMoreThanOneTile()
        {
            var view = Playing(Seat.X, Seat.X);
            view.Mode = Mode._1V1;
            view.BoneyardSize = 2;
            Read(view);
            Assert.IsTrue(choice.CanDraw);
            Assert.IsFalse(choice.CanPass);

            view.BoneyardSize = 1;
            Read(view);
            Assert.IsFalse(choice.CanDraw);
            Assert.IsTrue(choice.CanPass);
            Assert.IsTrue(choice.TryPass());
            Assert.IsFalse(choice.TryPass());
        }

        [Test]
        public void WithNoLegalMoveATwoVersusTwoPlayerPasses()
        {
            Read(Playing(Seat.South, Seat.South));
            Assert.IsTrue(choice.CanPass);
            Assert.IsFalse(choice.CanDraw);
            Assert.IsFalse(choice.CanEndTurn);
        }

        [Test]
        public void PartnersSitInOneTeam()
        {
            Assert.AreEqual(Team.A, SeatOrder.TeamOf(Seat.North));
            Assert.AreEqual(Team.A, SeatOrder.TeamOf(Seat.South));
            Assert.AreEqual(Team.B, SeatOrder.TeamOf(Seat.East));
            Assert.AreEqual(Team.B, SeatOrder.TeamOf(Seat.West));
            Assert.AreEqual(Team.A, SeatOrder.TeamOf(Seat.X));
            Assert.AreEqual(Team.B, SeatOrder.TeamOf(Seat.Y));
            Assert.Throws<ArgumentOutOfRangeException>(() => SeatOrder.TeamOf(Seat.Unspecified));
        }

        [Test]
        public void ASpectatorChoosesNothing()
        {
            Read(Playing(Seat.Unspecified, Seat.Unspecified, Move(1, 2, BoardEnd.Left)));
            Assert.IsFalse(choice.IsSelectable(0));
            Assert.IsFalse(choice.CanPass);
        }

        [Test]
        public void ReadingAViewAndTappingAllocateNothing()
        {
            Assert.IsTrue(snapshot.TryRead(Playing(Seat.South, Seat.South, Move(4, 6, BoardEnd.Left), Move(4, 6, BoardEnd.Right)).ToByteArray()));
            TestDelegate step = () =>
            {
                choice.Read(snapshot);
                choice.Tap(2, out _);
                choice.TryChooseEnd(BoardEnd.Right, out _);
            };
            step();
            Assert.That(step, Allocates.Nothing);
        }
    }
}
