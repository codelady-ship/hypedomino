using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class BoardChainTests
    {
        static readonly Vector2 WideBoard = new(100f, 100f);

        readonly MatchSnapshot snapshot = new();
        readonly BoardChain chain = new();

        static OrientedTile T(uint first, uint second) => new() { First = first, Second = second };

        MatchSnapshot Read(BoardView board)
        {
            var view = new MatchView { State = MatchState.Playing, Board = board };
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        static BoardView SpinnerBoard() => new()
        {
            Line = { T(2, 3), T(3, 3), T(3, 1) },
            OpeningIndex = 0,
            Spinner = new Tile { Low = 3, High = 3 },
            Top = { T(3, 5) },
            Bottom = { T(3, 0) },
        };

        void AssertCenter(int index, float x, float y)
        {
            Assert.AreEqual(new Vector2(x, y), chain.Layout[index].Center, $"center of {index}");
        }

        [Test]
        public void WithoutASpinnerTheOpeningTileIsTheAnchor()
        {
            chain.Arrange(Read(new BoardView { Line = { T(1, 5), T(5, 2), T(2, 3), T(3, 3) }, OpeningIndex = 2 }), WideBoard, 1f);

            Assert.AreEqual(4, chain.Layout.Count);
            AssertCenter(2, 0f, 0f);
            AssertCenter(1, -2f, 0f);
            AssertCenter(0, -4f, 0f);
            AssertCenter(3, 1.5f, 0f);
            Assert.AreEqual(2, chain.Layout[1].Near);
            Assert.IsTrue(chain.Layout[3].Across);
        }

        [Test]
        public void TheSpinnerIsTheAnchorOnceItExists()
        {
            var view = Read(SpinnerBoard());

            Assert.AreEqual(1, BoardChain.AnchorIndex(view));
            chain.Arrange(view, WideBoard, 1f);

            Assert.AreEqual(5, chain.Layout.Count);
            AssertCenter(1, 0f, 0f);
            AssertCenter(0, -1.5f, 0f);
            AssertCenter(2, 1.5f, 0f);
            AssertCenter(3, 0f, 2f);
            AssertCenter(4, 0f, -2f);
            Assert.AreEqual(5, chain.Layout[3].Far);
            Assert.AreEqual(0, chain.Layout[4].Far);
        }

        [Test]
        public void ANewHandLaysOnlyItsOwnTiles()
        {
            chain.Arrange(Read(SpinnerBoard()), WideBoard, 1f);
            chain.Arrange(Read(new BoardView { Line = { T(4, 6) } }), WideBoard, 1f);

            Assert.AreEqual(1, chain.Layout.Count);
            AssertCenter(0, 0f, 0f);
            Assert.AreEqual(4, chain.Layout[0].Near);
        }

        [Test]
        public void AnEmptyBoardLaysNothing()
        {
            chain.Arrange(Read(new BoardView()), WideBoard, 1f);

            Assert.AreEqual(0, chain.Layout.Count);
            Assert.AreEqual(0f, chain.Layout.Scale);
        }

        [Test]
        public void ArrangingAViewAllocatesNothing()
        {
            var view = Read(SpinnerBoard());
            TestDelegate arrange = () => chain.Arrange(view, WideBoard, 1f);
            arrange();

            Assert.That(arrange, Allocates.Nothing);
        }
    }
}
