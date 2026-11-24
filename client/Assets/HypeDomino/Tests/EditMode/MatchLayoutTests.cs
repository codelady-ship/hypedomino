using System;
using HypeDomino.Net;
using HypeDomino.UI.Match;
using HypeDomino.UI.Tokens;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchLayoutTests
    {
        const float Tolerance = 1e-3f;

        static readonly object[] Shapes =
        {
            new object[] { "16:9 landscape", new Vector2(1920, 1080), new Vector4(0, 0, 0, 0) },
            new object[] { "19.5:9 landscape, side insets and home indicator", new Vector2(2340, 1080), new Vector4(150, 150, 0, 58) },
            new object[] { "20:9 landscape, camera hole left", new Vector2(2400, 1080), new Vector4(100, 0, 0, 0) },
            new object[] { "21:9 landscape", new Vector2(2520, 1080), new Vector4(0, 0, 0, 0) },
            new object[] { "4:3 landscape tablet", new Vector2(1440, 1080), new Vector4(0, 0, 0, 0) },
            new object[] { "folding inner display landscape", new Vector2(1190, 1080), new Vector4(0, 0, 0, 0) },
            new object[] { "16:9 portrait", new Vector2(1080, 1920), new Vector4(0, 0, 0, 0) },
            new object[] { "19.5:9 portrait, island and home indicator", new Vector2(1080, 2340), new Vector4(0, 0, 94, 58) },
            new object[] { "4:3 portrait tablet", new Vector2(1080, 1440), new Vector4(0, 0, 0, 0) },
            new object[] { "folding inner display portrait", new Vector2(1080, 1190), new Vector4(0, 0, 0, 0) },
        };

        static Rect SafeArea(Vector2 canvas, Vector4 insets) =>
            Rect.MinMaxRect(insets.x, insets.w, canvas.x - insets.y, canvas.y - insets.z);

        [Test]
        public void SeatsGoClockwiseFromNorthInTwoVersusTwo()
        {
            Assert.AreEqual(0, SeatOrder.Clockwise(Seat.North));
            Assert.AreEqual(1, SeatOrder.Clockwise(Seat.East));
            Assert.AreEqual(2, SeatOrder.Clockwise(Seat.South));
            Assert.AreEqual(3, SeatOrder.Clockwise(Seat.West));
            Assert.AreEqual(4, SeatOrder.Size(Mode._2V2));
        }

        [Test]
        public void SeatsGoFromXToYInOneVersusOne()
        {
            Assert.AreEqual(0, SeatOrder.Clockwise(Seat.X));
            Assert.AreEqual(1, SeatOrder.Clockwise(Seat.Y));
            Assert.AreEqual(2, SeatOrder.Size(Mode._1V1));
        }

        [Test]
        public void AViewerWithoutASeatSitsSouthOrX()
        {
            Assert.AreEqual(SeatOrder.Clockwise(Seat.South), SeatOrder.Viewer(Seat.Unspecified, Mode._2V2));
            Assert.AreEqual(SeatOrder.Clockwise(Seat.X), SeatOrder.Viewer(Seat.Unspecified, Mode._1V1));
            Assert.AreEqual(SeatOrder.Clockwise(Seat.East), SeatOrder.Viewer(Seat.East, Mode._2V2));
        }

        [Test]
        public void AnUnknownSeatOrModeIsRejected()
        {
            Assert.Throws<ArgumentOutOfRangeException>(() => SeatOrder.Clockwise(Seat.Unspecified));
            Assert.Throws<ArgumentOutOfRangeException>(() => SeatOrder.Size(Mode.Unspecified));
            Assert.Throws<ArgumentOutOfRangeException>(() => TableSides.Of(0, 0, 3));
            Assert.Throws<ArgumentOutOfRangeException>(() => TableSides.Of(0, 4, 4));
            Assert.Throws<ArgumentOutOfRangeException>(() => new MatchLayout().Arrange(new Rect(0, 0, 1920, 1080), 3));
        }

        [Test]
        public void SouthSeesNorthOnTopEastOnTheRightAndWestOnTheLeft()
        {
            var south = SeatOrder.Clockwise(Seat.South);
            Assert.AreEqual(TableSide.Bottom, TableSides.Of(south, south, 4));
            Assert.AreEqual(TableSide.Left, TableSides.Of(south, SeatOrder.Clockwise(Seat.West), 4));
            Assert.AreEqual(TableSide.Top, TableSides.Of(south, SeatOrder.Clockwise(Seat.North), 4));
            Assert.AreEqual(TableSide.Right, TableSides.Of(south, SeatOrder.Clockwise(Seat.East), 4));
        }

        [Test]
        public void EverySeatSitsAtTheBottomWithItsPartnerOnTopAndTheNextTurnOnTheLeft()
        {
            for (var viewer = 0; viewer < 4; viewer++)
            {
                Assert.AreEqual(TableSide.Bottom, TableSides.Of(viewer, viewer, 4), $"viewer {viewer}");
                Assert.AreEqual(TableSide.Left, TableSides.Of(viewer, (viewer + 1) % 4, 4), $"viewer {viewer}");
                Assert.AreEqual(TableSide.Top, TableSides.Of(viewer, (viewer + 2) % 4, 4), $"viewer {viewer}");
                Assert.AreEqual(TableSide.Right, TableSides.Of(viewer, (viewer + 3) % 4, 4), $"viewer {viewer}");
            }
        }

        [Test]
        public void TheOpponentSitsOnTopInOneVersusOne()
        {
            int x = SeatOrder.Clockwise(Seat.X), y = SeatOrder.Clockwise(Seat.Y);
            Assert.AreEqual(TableSide.Bottom, TableSides.Of(x, x, 2));
            Assert.AreEqual(TableSide.Top, TableSides.Of(x, y, 2));
            Assert.AreEqual(TableSide.Bottom, TableSides.Of(y, y, 2));
            Assert.AreEqual(TableSide.Top, TableSides.Of(y, x, 2));
            Assert.IsFalse(TableSides.IsUsed(TableSide.Left, 2));
            Assert.IsFalse(TableSides.IsUsed(TableSide.Right, 2));
            Assert.IsTrue(TableSides.IsUsed(TableSide.Left, 4));
        }

        [TestCaseSource(nameof(Shapes))]
        public void TwoVersusTwoFitsTheSafeAreaWithoutOverlap(string shape, Vector2 canvas, Vector4 insets)
        {
            var safe = SafeArea(canvas, insets);
            var layout = new MatchLayout();
            layout.Arrange(safe, 4);
            var regions = new[] { layout.Board, layout.Hand, layout.Seat(TableSide.Bottom), layout.Seat(TableSide.Left), layout.Seat(TableSide.Top), layout.Seat(TableSide.Right) };
            AssertFitsWithoutOverlap(shape, safe, regions);
            AssertSeatPanels(shape, layout, 4);
            Assert.AreEqual(layout.Seat(TableSide.Left).center.y, layout.Seat(TableSide.Right).center.y, Tolerance, shape);
            Assert.Greater(layout.Board.xMin, layout.Seat(TableSide.Left).xMax, shape);
            Assert.Less(layout.Board.xMax, layout.Seat(TableSide.Right).xMin, shape);
        }

        [TestCaseSource(nameof(Shapes))]
        public void OneVersusOneFitsTheSafeAreaWithTheBoardAcrossTheWidth(string shape, Vector2 canvas, Vector4 insets)
        {
            var safe = SafeArea(canvas, insets);
            var layout = new MatchLayout();
            layout.Arrange(safe, 2);
            var regions = new[] { layout.Board, layout.Hand, layout.Seat(TableSide.Bottom), layout.Seat(TableSide.Top) };
            AssertFitsWithoutOverlap(shape, safe, regions);
            AssertSeatPanels(shape, layout, 2);
            Assert.AreEqual(Rect.zero, layout.Seat(TableSide.Left), shape);
            Assert.AreEqual(Rect.zero, layout.Seat(TableSide.Right), shape);
            Assert.AreEqual(safe.xMin + MatchLayout.Margin, layout.Board.xMin, Tolerance, shape);
            Assert.AreEqual(safe.xMax - MatchLayout.Margin, layout.Board.xMax, Tolerance, shape);
        }

        [TestCaseSource(nameof(Shapes))]
        public void TheHandRunsAlongTheBottomFromTheLocalSeatToTheRightEdge(string shape, Vector2 canvas, Vector4 insets)
        {
            var safe = SafeArea(canvas, insets);
            var layout = new MatchLayout();
            layout.Arrange(safe, 4);
            var mine = layout.Seat(TableSide.Bottom);
            Assert.AreEqual(safe.xMin + MatchLayout.Margin, mine.xMin, Tolerance, shape);
            Assert.AreEqual(safe.yMin + MatchLayout.Margin, mine.yMin, Tolerance, shape);
            Assert.AreEqual(mine.yMin, layout.Hand.yMin, Tolerance, shape);
            Assert.AreEqual(mine.xMax + MatchLayout.Margin, layout.Hand.xMin, Tolerance, shape);
            Assert.AreEqual(safe.xMax - MatchLayout.Margin, layout.Hand.xMax, Tolerance, shape);
            Assert.AreEqual(TokenSize.HandHeight, layout.Hand.height, Tolerance, shape);
            Assert.AreEqual(safe.center.x, layout.Seat(TableSide.Top).center.x, Tolerance, shape);
            Assert.AreEqual(safe.yMax - MatchLayout.Margin, layout.Seat(TableSide.Top).yMax, Tolerance, shape);
        }

        [Test]
        public void TheBoardShrinksToNothingWhenTheSafeAreaIsTooSmall()
        {
            var layout = new MatchLayout();
            layout.Arrange(new Rect(0, 0, 400, 400), 4);
            Assert.GreaterOrEqual(layout.Board.width, 0f);
            Assert.GreaterOrEqual(layout.Board.height, 0f);
            Assert.GreaterOrEqual(layout.Hand.width, 0f);
        }

        [Test]
        public void ArrangingAgainAllocatesNothing()
        {
            var layout = new MatchLayout();
            var safe = new Rect(0, 0, 2400, 1080);
            TestDelegate step = () => layout.Arrange(safe, 4);
            step();
            Assert.That(step, Allocates.Nothing);
        }

        [Test]
        public void TheViewPlacesEachRegionAndHidesTheSidesInOneVersusOne()
        {
            var root = new GameObject("Match", typeof(RectTransform));
            try
            {
                var rootRect = (RectTransform)root.transform;
                rootRect.sizeDelta = new Vector2(2400, 1080);
                var board = Child(rootRect, "Board");
                var hand = Child(rootRect, "Hand");
                var seats = new RectTransform[TableSides.Count];
                for (var i = 0; i < seats.Length; i++) seats[i] = Child(rootRect, ((TableSide)i).ToString());
                var view = root.AddComponent<MatchLayoutView>();
                view.Bind(board, hand, seats);

                AssertPlaced(rootRect, board, view.Layout.Board);
                AssertPlaced(rootRect, hand, view.Layout.Hand);
                for (var i = 0; i < seats.Length; i++)
                {
                    Assert.IsTrue(seats[i].gameObject.activeSelf);
                    AssertPlaced(rootRect, seats[i], view.Layout.Seat((TableSide)i));
                }

                view.SetSeatCount(2);
                Assert.IsTrue(seats[(int)TableSide.Bottom].gameObject.activeSelf);
                Assert.IsTrue(seats[(int)TableSide.Top].gameObject.activeSelf);
                Assert.IsFalse(seats[(int)TableSide.Left].gameObject.activeSelf);
                Assert.IsFalse(seats[(int)TableSide.Right].gameObject.activeSelf);
                AssertPlaced(rootRect, board, view.Layout.Board);
            }
            finally
            {
                UnityEngine.Object.DestroyImmediate(root);
            }
        }

        static RectTransform Child(RectTransform parent, string name)
        {
            var child = new GameObject(name, typeof(RectTransform));
            child.transform.SetParent(parent, false);
            return (RectTransform)child.transform;
        }

        static void AssertPlaced(RectTransform root, RectTransform target, Rect expected)
        {
            var min = (Vector2)root.InverseTransformPoint(target.TransformPoint(target.rect.min));
            var max = (Vector2)root.InverseTransformPoint(target.TransformPoint(target.rect.max));
            Assert.AreEqual(expected.xMin, min.x, Tolerance, target.name);
            Assert.AreEqual(expected.yMin, min.y, Tolerance, target.name);
            Assert.AreEqual(expected.xMax, max.x, Tolerance, target.name);
            Assert.AreEqual(expected.yMax, max.y, Tolerance, target.name);
        }

        static void AssertSeatPanels(string shape, MatchLayout layout, int seatCount)
        {
            for (var i = 0; i < TableSides.Count; i++)
            {
                var side = (TableSide)i;
                if (!TableSides.IsUsed(side, seatCount)) continue;
                var panel = layout.Seat(side);
                Assert.AreEqual(MatchLayout.SeatPanelSize, panel.size, $"{shape} {side}");
                Assert.GreaterOrEqual(panel.width, TokenSize.TouchTargetMin, $"{shape} {side}");
                Assert.GreaterOrEqual(panel.height, TokenSize.TouchTargetMin, $"{shape} {side}");
                Assert.Greater(layout.Board.width * layout.Board.height, panel.width * panel.height, $"{shape}: the board is larger than the {side} panel");
            }
        }

        static void AssertFitsWithoutOverlap(string shape, Rect safe, Rect[] regions)
        {
            for (var i = 0; i < regions.Length; i++)
            {
                var region = regions[i];
                Assert.Greater(region.width, 0f, $"{shape}: region {i} has no width");
                Assert.Greater(region.height, 0f, $"{shape}: region {i} has no height");
                Assert.GreaterOrEqual(region.xMin, safe.xMin + MatchLayout.Margin - Tolerance, $"{shape}: region {i} left");
                Assert.GreaterOrEqual(region.yMin, safe.yMin + MatchLayout.Margin - Tolerance, $"{shape}: region {i} bottom");
                Assert.LessOrEqual(region.xMax, safe.xMax - MatchLayout.Margin + Tolerance, $"{shape}: region {i} right");
                Assert.LessOrEqual(region.yMax, safe.yMax - MatchLayout.Margin + Tolerance, $"{shape}: region {i} top");
                for (var j = i + 1; j < regions.Length; j++)
                    Assert.IsFalse(region.Overlaps(regions[j]), $"{shape}: regions {i} and {j} overlap");
            }
        }
    }
}
