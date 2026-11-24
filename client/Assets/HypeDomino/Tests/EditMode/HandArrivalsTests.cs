using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class HandArrivalsTests
    {
        readonly MatchSnapshot snapshot = new();
        HandArrivals arrivals;

        [SetUp]
        public void Build() => arrivals = new HandArrivals();

        static Tile T(uint low, uint high) => new() { Low = low, High = high };

        static MatchView View(MatchState state, uint hand, uint opponentSize, bool placed, params Tile[] tiles)
        {
            var match = new MatchView { State = state, Mode = Mode._1V1, Seat = Seat.X, HandsPlayed = hand, Board = new BoardView() };
            match.Hand.Add(tiles);
            match.Seats.Add(new SeatView { Seat = Seat.X, HandSize = (uint)tiles.Length });
            match.Seats.Add(new SeatView { Seat = Seat.Y, HandSize = opponentSize });
            if (placed) match.Board.Line.Add(new OrientedTile { First = 6, Second = 6 });
            return match;
        }

        HandArrival Read(MatchView match)
        {
            Assert.IsTrue(snapshot.TryRead(match.ToByteArray()));
            return arrivals.Read(snapshot);
        }

        [Test]
        public void AFreshHandOnAnEmptyBoardIsDealtInHandOrder()
        {
            Assert.AreEqual(HandArrival.Deal, Read(View(MatchState.Opening, 0, 7, false, T(0, 1), T(2, 3), T(4, 4))));

            CollectionAssert.AreEqual(new[] { 0, 1, 2 }, arrivals.ArrivedTiles.ToArray());
            Assert.AreEqual(3u, arrivals.ArrivedAt(Seat.X));
            Assert.AreEqual(7u, arrivals.ArrivedAt(Seat.Y));
        }

        [Test]
        public void JoiningAHandInProgressAnimatesNothing()
        {
            Assert.AreEqual(HandArrival.None, Read(View(MatchState.Playing, 0, 6, true, T(0, 1), T(2, 3))));

            Assert.AreEqual(0, arrivals.ArrivedTiles.Length);
            Assert.AreEqual(0u, arrivals.ArrivedAt(Seat.Y));
        }

        [Test]
        public void ADrawnTileArrivesAlone()
        {
            Read(View(MatchState.Playing, 0, 6, true, T(0, 1), T(2, 3)));

            Assert.AreEqual(HandArrival.Draw, Read(View(MatchState.Playing, 0, 6, true, T(0, 1), T(5, 6), T(2, 3))));

            CollectionAssert.AreEqual(new[] { 1 }, arrivals.ArrivedTiles.ToArray());
            Assert.AreEqual(1u, arrivals.ArrivedAt(Seat.X));
            Assert.AreEqual(0u, arrivals.ArrivedAt(Seat.Y));
        }

        [Test]
        public void TheOpponentDrawingShowsOnItsSeat()
        {
            Read(View(MatchState.Playing, 0, 6, true, T(0, 1)));

            Assert.AreEqual(HandArrival.Draw, Read(View(MatchState.Playing, 0, 8, true, T(0, 1))));

            Assert.AreEqual(0, arrivals.ArrivedTiles.Length);
            Assert.AreEqual(2u, arrivals.ArrivedAt(Seat.Y));
        }

        [Test]
        public void PlacingATileIsNoArrival()
        {
            Read(View(MatchState.Playing, 0, 6, true, T(0, 1), T(2, 3)));

            Assert.AreEqual(HandArrival.None, Read(View(MatchState.Playing, 0, 5, true, T(2, 3))));
        }

        [Test]
        public void TheNextHandIsDealtAgain()
        {
            Read(View(MatchState.Playing, 0, 1, true, T(0, 1)));

            Assert.AreEqual(HandArrival.Deal, Read(View(MatchState.Opening, 1, 7, false, T(0, 1), T(3, 3))));

            CollectionAssert.AreEqual(new[] { 0, 1 }, arrivals.ArrivedTiles.ToArray());
        }

        [Test]
        public void TheEndOfTheMatchClearsTheHand()
        {
            Read(View(MatchState.Playing, 0, 6, true, T(0, 1)));

            Assert.AreEqual(HandArrival.None, Read(View(MatchState.Finished, 1, 0, true)));
            Assert.AreEqual(0u, arrivals.ArrivedAt(Seat.Y));
        }

        [Test]
        public void ReadingAViewAllocatesNothing()
        {
            var first = View(MatchState.Playing, 0, 6, true, T(0, 1), T(2, 3)).ToByteArray();
            var second = View(MatchState.Playing, 0, 7, true, T(0, 1), T(2, 3), T(5, 5)).ToByteArray();
            var other = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(first));
            Assert.IsTrue(other.TryRead(second));
            arrivals.Read(snapshot);
            arrivals.Read(other);

            Assert.That(() =>
            {
                arrivals.Read(snapshot);
                arrivals.Read(other);
            }, Allocates.Nothing);
        }
    }
}
