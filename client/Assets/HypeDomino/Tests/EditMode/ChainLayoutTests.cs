using System.Collections.Generic;
using HypeDomino.UI.Board;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ChainLayoutTests
    {
        const int SetSize = 28;
        static readonly Domino[] None = new Domino[0];

        readonly ChainLayout layout = new ChainLayout(SetSize);

        static Domino D(byte first, byte second) => new Domino(first, second);

        void AssertPlaced(int index, float x, float y, Vector2 facing, byte near, byte far, bool across)
        {
            var tile = layout[index];
            Assert.AreEqual(new Vector2(x, y), tile.Center, $"center of {index}");
            Assert.AreEqual(facing, tile.Facing, $"facing of {index}");
            Assert.AreEqual(near, tile.Near, $"near of {index}");
            Assert.AreEqual(far, tile.Far, $"far of {index}");
            Assert.AreEqual(across, tile.Across, $"across of {index}");
        }

        [Test]
        public void TheOpeningTileLiesAlongTheLineAtTheCenter()
        {
            layout.Arrange(new[] { D(2, 3) }, 0, None, None, new Vector2(10f, 10f), 100f);

            Assert.AreEqual(1, layout.Count);
            AssertPlaced(0, 0f, 0f, Vector2.right, 2, 3, false);
            Assert.AreEqual(new Rect(-1f, -0.5f, 2f, 1f), layout.Bounds);
            Assert.AreEqual(5f, layout.Scale);
            Assert.AreEqual(Vector2.zero, layout.Focus);
        }

        [Test]
        public void DoublesLieAcrossTheLineAndOtherTilesAlongIt()
        {
            layout.Arrange(new[] { D(1, 3), D(3, 3), D(3, 5) }, 0, None, None, new Vector2(100f, 10f), 100f);

            AssertPlaced(1, 1.5f, 0f, Vector2.right, 3, 3, true);
            AssertPlaced(2, 3f, 0f, Vector2.right, 3, 5, false);
            Assert.AreEqual(new Vector2(1f, 2f), layout[1].Size);
            Assert.AreEqual(Rect.MinMaxRect(-1f, -1f, 4f, 1f), layout.Bounds);
            Assert.AreEqual(5f, layout.Scale);
        }

        [Test]
        public void TheLeftArmGrowsWestWithTheTouchingHalfNearest()
        {
            layout.Arrange(new[] { D(4, 1), D(1, 2) }, 1, None, None, new Vector2(100f, 10f), 100f);

            AssertPlaced(1, 0f, 0f, Vector2.right, 1, 2, false);
            AssertPlaced(0, -2f, 0f, Vector2.left, 1, 4, false);
        }

        [Test]
        public void TheSpinnerSitsAtTheCenterWithFourArms()
        {
            layout.Arrange(new[] { D(1, 3), D(3, 3), D(3, 5) }, 1, new[] { D(3, 4) }, new[] { D(3, 6) }, new Vector2(100f, 100f), 10f);

            Assert.AreEqual(5, layout.Count);
            AssertPlaced(1, 0f, 0f, Vector2.right, 3, 3, true);
            AssertPlaced(2, 1.5f, 0f, Vector2.right, 3, 5, false);
            AssertPlaced(0, -1.5f, 0f, Vector2.left, 3, 1, false);
            AssertPlaced(3, 0f, 2f, Vector2.up, 3, 4, false);
            AssertPlaced(4, 0f, -2f, Vector2.down, 3, 6, false);
            Assert.AreEqual(Rect.MinMaxRect(-2.5f, -3f, 2.5f, 3f), layout.Bounds);
            Assert.AreEqual(10f, layout.Scale, "the cap keeps a short chain straight");
        }

        [Test]
        public void AnArmTurnsRightWhenItsNextTileWouldLeaveTheBox()
        {
            layout.Arrange(new[] { D(0, 1), D(1, 2), D(2, 3), D(3, 4) }, 0, None, None, new Vector2(8f, 8f), 100f);

            AssertPlaced(0, 0f, 0f, Vector2.right, 0, 1, false);
            AssertPlaced(1, 1.5f, -0.5f, Vector2.down, 1, 2, false);
            AssertPlaced(2, 1f, -2f, Vector2.left, 2, 3, false);
            AssertPlaced(3, -0.5f, -1.5f, Vector2.up, 3, 4, false);
            Assert.AreEqual(Rect.MinMaxRect(-1f, -2.5f, 2f, 0.5f), layout.Bounds);
            Assert.AreEqual(8f / 3f, layout.Scale, 1e-4f);
        }

        [Test]
        public void AnEmptyBoardPlacesNothing()
        {
            layout.Arrange(None, 0, None, None, new Vector2(10f, 10f), 100f);

            Assert.AreEqual(0, layout.Count);
            Assert.AreEqual(0f, layout.Scale);
        }

        [Test]
        public void TheOpeningTileHasItsTwoHalvesAsEnds()
        {
            layout.Arrange(new[] { D(2, 3) }, 0, None, None, new Vector2(10f, 10f), 100f);

            AssertEnd(ChainEnd.West, new Rect(-1f, -0.5f, 1f, 1f));
            AssertEnd(ChainEnd.East, new Rect(0f, -0.5f, 1f, 1f));
        }

        [Test]
        public void ALoneDoubleHasItsWestAndEastSidesAsEnds()
        {
            layout.Arrange(new[] { D(3, 3) }, 0, None, None, new Vector2(10f, 10f), 100f);

            AssertEnd(ChainEnd.West, new Rect(-0.5f, -1f, 0.5f, 2f));
            AssertEnd(ChainEnd.East, new Rect(0f, -1f, 0.5f, 2f));
        }

        [Test]
        public void AnArmEndsOnTheOpenHalfOfItsLastTile()
        {
            layout.Arrange(new[] { D(1, 3), D(3, 3), D(3, 5) }, 1, new[] { D(3, 4) }, new[] { D(3, 6) }, new Vector2(100f, 100f), 10f);

            AssertEnd(ChainEnd.West, new Rect(-2.5f, -0.5f, 1f, 1f));
            AssertEnd(ChainEnd.East, new Rect(1.5f, -0.5f, 1f, 1f));
            AssertEnd(ChainEnd.North, new Rect(-0.5f, 2f, 1f, 1f));
            AssertEnd(ChainEnd.South, new Rect(-0.5f, -3f, 1f, 1f));
        }

        [Test]
        public void AnEmptySpinnerArmEndsOnTheSpinnerHalf()
        {
            layout.Arrange(new[] { D(1, 3), D(3, 3), D(3, 5) }, 1, None, None, new Vector2(100f, 100f), 10f);

            AssertEnd(ChainEnd.North, new Rect(-0.5f, 0f, 1f, 1f));
            AssertEnd(ChainEnd.South, new Rect(-0.5f, -1f, 1f, 1f));
        }

        [Test]
        public void ADoubleAtTheEndOfTheLineEndsOnItsOuterSide()
        {
            layout.Arrange(new[] { D(1, 3), D(3, 3) }, 0, None, None, new Vector2(100f, 10f), 100f);

            AssertEnd(ChainEnd.East, new Rect(1.5f, -1f, 0.5f, 2f));
        }

        [Test]
        public void AFoldedArmEndsAlongItsLastHeading()
        {
            layout.Arrange(new[] { D(0, 1), D(1, 2), D(2, 3), D(3, 4) }, 0, None, None, new Vector2(8f, 8f), 100f);

            AssertEnd(ChainEnd.East, new Rect(-1f, -1.5f, 1f, 1f));
            AssertEnd(ChainEnd.West, new Rect(-1f, -0.5f, 1f, 1f));
        }

        [Test]
        public void BoardPointsAreScaledAroundTheFocus()
        {
            layout.Arrange(new[] { D(0, 1), D(1, 2), D(2, 3), D(3, 4) }, 0, None, None, new Vector2(8f, 8f), 100f);

            Assert.AreEqual(Vector2.zero, layout.OnBoard(layout.Focus));
            var corner = layout.OnBoard(new Vector2(2f, 0.5f));
            Assert.AreEqual(4f, corner.x, 1e-4f);
            Assert.AreEqual(4f, corner.y, 1e-4f);
        }

        [Test]
        public void AnEmptyBoardHasNoEnds()
        {
            layout.Arrange(new[] { D(2, 3) }, 0, None, None, new Vector2(10f, 10f), 100f);
            layout.Arrange(None, 0, None, None, new Vector2(10f, 10f), 100f);

            Assert.IsFalse(layout.TryGetEnd(ChainEnd.East, out _));
        }

        [Test]
        public void ReadingEndsAllocatesNothing()
        {
            layout.Arrange(new[] { D(1, 3), D(3, 3), D(3, 5) }, 1, new[] { D(3, 4) }, None, new Vector2(100f, 100f), 10f);
            TestDelegate step = () =>
            {
                for (var end = ChainEnd.West; end <= ChainEnd.South; end++)
                    if (layout.TryGetEnd(end, out var area)) layout.OnBoard(area.center);
            };
            step();
            Assert.That(step, Allocates.Nothing);
        }

        [TestCase(1080f, 1080f)]
        [TestCase(1920f, 1080f)]
        [TestCase(1080f, 1920f)]
        [TestCase(2400f, 1080f)]
        [TestCase(1080f, 600f)]
        public void EveryTileOfTheSetFitsTheBoardWithoutOverlapAndLargerThanAStraightCross(float width, float height)
        {
            var board = new Vector2(width, height);
            var line = FullSet(0, 20);
            line[10] = D(6, 6);
            var top = FullSet(20, 4);
            var bottom = FullSet(24, 4);

            layout.Arrange(line, 10, top, bottom, board, float.MaxValue);

            Assert.AreEqual(SetSize, layout.Count);
            AssertNoOverlap();
            AssertArmTouches(0, line.Length);
            Assert.LessOrEqual(layout.Bounds.width * layout.Scale, width + 0.01f);
            Assert.LessOrEqual(layout.Bounds.height * layout.Scale, height + 0.01f);
            var straight = Mathf.Min(width / StraightLength(line), height / (StraightLength(top) + StraightLength(bottom) + 2f));
            Assert.Greater(layout.Scale, straight);
        }

        [TestCase(1080f, 1920f)]
        [TestCase(1920f, 1080f)]
        public void TheLongestLineFitsTheBoard(float width, float height)
        {
            var line = FullSet(0, SetSize);

            layout.Arrange(line, 0, None, None, new Vector2(width, height), float.MaxValue);

            AssertNoOverlap();
            AssertArmTouches(0, SetSize);
            Assert.LessOrEqual(layout.Bounds.width * layout.Scale, width + 0.01f);
            Assert.LessOrEqual(layout.Bounds.height * layout.Scale, height + 0.01f);
            Assert.Greater(layout.Scale, Mathf.Min(width / StraightLength(line), height / 2f));
        }

        void AssertEnd(ChainEnd end, Rect expected)
        {
            Assert.IsTrue(layout.TryGetEnd(end, out var area), $"{end} end");
            Assert.AreEqual(expected, area, $"{end} end");
        }

        static Domino[] FullSet(int skip, int take)
        {
            var all = new List<Domino>();
            for (byte low = 0; low <= 6; low++)
            for (var high = low; high <= 6; high++)
                all.Add(D(low, high));
            return all.GetRange(skip, take).ToArray();
        }

        static float StraightLength(Domino[] tiles)
        {
            var length = 0f;
            foreach (var tile in tiles) length += tile.IsDouble ? 1f : 2f;
            return length;
        }

        void AssertNoOverlap()
        {
            for (var i = 0; i < layout.Count; i++)
            for (var j = i + 1; j < layout.Count; j++)
            {
                var a = layout[i].Area;
                var b = layout[j].Area;
                var width = Mathf.Min(a.xMax, b.xMax) - Mathf.Max(a.xMin, b.xMin);
                var height = Mathf.Min(a.yMax, b.yMax) - Mathf.Max(a.yMin, b.yMin);
                Assert.IsFalse(width > 0.01f && height > 0.01f, $"tiles {i} and {j} overlap");
                Assert.IsTrue(layout.Bounds.Contains(a.center), $"tile {i} is outside the bounds");
            }
        }

        void AssertArmTouches(int from, int to)
        {
            for (var i = from; i + 1 < to; i++)
            {
                var a = layout[i].Area;
                var b = layout[i + 1].Area;
                var width = Mathf.Min(a.xMax, b.xMax) - Mathf.Max(a.xMin, b.xMin);
                var height = Mathf.Min(a.yMax, b.yMax) - Mathf.Max(a.yMin, b.yMin);
                Assert.IsTrue(width >= -0.01f && height >= -0.01f, $"tiles {i} and {i + 1} do not touch");
            }
        }
    }
}
