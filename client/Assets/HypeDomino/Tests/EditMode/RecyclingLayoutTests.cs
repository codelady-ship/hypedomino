using HypeDomino.UI.Lists;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class RecyclingLayoutTests
    {
        readonly RecyclingLayout layout = new RecyclingLayout(100f, 20f, 1000f);

        [Test]
        public void RowsCoverTheViewportWithOneSpareRowAboveAndBelow()
        {
            Assert.AreEqual(120f, layout.Stride);
            Assert.AreEqual(12, layout.RowCount);
        }

        [Test]
        public void ContentHeightHasNoGapAfterTheLastRow()
        {
            Assert.AreEqual(0f, layout.ContentHeight(0));
            Assert.AreEqual(100f, layout.ContentHeight(1));
            Assert.AreEqual(340f, layout.ContentHeight(3));
        }

        [Test]
        public void RowsArePlacedOneStrideApart()
        {
            Assert.AreEqual(0f, layout.RowTop(0));
            Assert.AreEqual(600f, layout.RowTop(5));
        }

        [Test]
        public void TheFirstRowKeepsOneRowAboveTheVisibleOne()
        {
            Assert.AreEqual(0, layout.FirstRow(0f, 1000));
            Assert.AreEqual(0, layout.FirstRow(119f, 1000));
            Assert.AreEqual(0, layout.FirstRow(120f, 1000));
            Assert.AreEqual(1, layout.FirstRow(240f, 1000));
            Assert.AreEqual(39, layout.FirstRow(4900f, 1000));
        }

        [Test]
        public void TheFirstRowNeverRunsPastTheEndOrBeforeTheStart()
        {
            Assert.AreEqual(1000 - 12, layout.FirstRow(1_000_000f, 1000));
            Assert.AreEqual(0, layout.FirstRow(-300f, 1000));
            Assert.AreEqual(0, layout.FirstRow(5000f, 5));
        }

        [Test]
        public void EveryVisibleRowIsInsideTheRecycledRange()
        {
            for (var offset = 0f; offset < 120_000f; offset += 7f)
            {
                var first = layout.FirstRow(offset, 1000);
                var lastVisible = (int)((offset + 1000f) / layout.Stride);
                Assert.LessOrEqual(first, (int)(offset / layout.Stride), $"offset {offset}");
                Assert.Less(System.Math.Min(lastVisible, 999), first + layout.RowCount, $"offset {offset}");
            }
        }
    }
}
