using HypeDomino.UI.Cosmetics;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TableClothTests
    {
        const float Tolerance = 0.01f;
        const float Depth = 92f;
        const float Step = 224f;
        const int BottomLeft = 0;
        const int BottomRight = 1;
        const int TopLeft = 2;
        const int TopRight = 3;
        const int LeftSide = 4;
        const int RightSide = 5;
        const int FieldCentre = 10;
        static readonly Vector2 Art = new(1280f, 720f);
        static readonly Vector4 Border = new((1280f - Step) / 2f, Depth, (1280f - Step) / 2f, Depth);

        readonly TableQuad[] quads = new TableQuad[TableCloth.MaxQuads];

        int Lay(Vector2 rect, Vector4 border, out bool turned) => TableCloth.Layout(Art, border, rect, quads, out turned);

        int Lay(Vector2 rect) => Lay(rect, Border, out _);

        static void Near(float expected, float actual, string what) => Assert.AreEqual(expected, actual, Tolerance, what);

        static float Aspect(Rect rect) => rect.width / rect.height;

        [Test]
        public void AtTheAspectOfTheArtEveryQuadShowsTheArtAtOneScale()
        {
            var count = Lay(new Vector2(1920f, 1080f));

            Assert.AreEqual(TableCloth.MaxQuads, count);
            for (var at = 0; at < count; at++)
            {
                Near(quads[at].Source.xMin * 1.5f, quads[at].Target.xMin, $"quad {at} left");
                Near(quads[at].Source.yMin * 1.5f, quads[at].Target.yMin, $"quad {at} bottom");
                Near(quads[at].Source.width * 1.5f, quads[at].Target.width, $"quad {at} width");
                Near(quads[at].Source.height * 1.5f, quads[at].Target.height, $"quad {at} height");
            }
        }

        [TestCase(2400f, 1)]
        [TestCase(2520f, 2)]
        [TestCase(1440f, -1)]
        [TestCase(1080f, -2)]
        public void TheLongEdgesGainOrLoseWholeStepsAtTheirMiddle(float width, int steps)
        {
            Lay(new Vector2(width, 1080f));

            foreach (var (left, right) in new[] { (BottomLeft, BottomRight), (TopLeft, TopRight) })
            {
                Near(0f, quads[left].Source.xMin, "the left run starts at the left corner");
                Near(Art.x, quads[right].Source.xMax, "the right run ends at the right corner");
                Near(steps * Step, quads[left].Source.xMax - quads[right].Source.xMin, "the runs meet a whole number of steps apart");
                Near(quads[left].Target.xMax, quads[right].Target.xMin, "the runs touch");
                Near(width, quads[right].Target.xMax, "the edge reaches the right side");
                Near(Depth * 1.5f, quads[left].Target.height, "the band keeps its depth");
            }
        }

        [TestCase(2400f)]
        [TestCase(2520f)]
        [TestCase(1440f)]
        [TestCase(1080f)]
        public void TheRingStretchesLittleAndTheFieldKeepsItsAspect(float width)
        {
            Lay(new Vector2(width, 1080f));
            var across = quads[BottomLeft].Target.width / quads[BottomLeft].Source.width / 1.5f;

            Assert.That(across, Is.InRange(0.85f, 1.15f));
            Near(0f, quads[LeftSide].Target.xMin, "left side");
            Near(width, quads[RightSide].Target.xMax, "right side");
            Near(Depth * across * 1.5f, quads[FieldCentre].Target.xMin, "the field starts at the ring");
            Near(Aspect(quads[FieldCentre].Target), Aspect(quads[FieldCentre].Source), "the field is not stretched");
            Assert.That(quads[FieldCentre].Source.xMin, Is.GreaterThanOrEqualTo(Depth - Tolerance));
            Assert.That(quads[FieldCentre].Source.yMin, Is.GreaterThanOrEqualTo(Depth - Tolerance));
        }

        [Test]
        public void TheFieldFadesInOverTheInnerEdgeOfTheRing()
        {
            Lay(new Vector2(2400f, 1080f));

            Assert.AreEqual(0f, quads[FieldCentre - 4].AlphaAt(false, false));
            Assert.AreEqual(1f, quads[FieldCentre - 4].AlphaAt(true, true));
            Assert.AreEqual(1f, quads[FieldCentre].AlphaAt(false, false));
            Assert.AreEqual(1f, quads[FieldCentre].AlphaAt(true, true));
            Near(Art.y * TableCloth.SeamFadeShare * 1.5f, quads[FieldCentre].Target.yMin - quads[FieldCentre - 3].Target.yMin, "fade height");
        }

        [Test]
        public void ATallRectIsLaidOutAsTheSameTableTurned()
        {
            Lay(new Vector2(2400f, 1080f));
            var wide = quads[TopRight];
            Lay(new Vector2(1080f, 2400f), Border, out var turned);

            Assert.IsTrue(turned);
            Assert.AreEqual(wide.Target, quads[TopRight].Target);
            Assert.AreEqual(new Vector2(0f, 2400f), TableCloth.ToRect(Vector2.zero, new Vector2(1080f, 2400f), true));
            Assert.AreEqual(new Vector2(1080f, 0f), TableCloth.ToRect(new Vector2(2400f, 1080f), new Vector2(1080f, 2400f), true));
        }

        [Test]
        public void ArtWithoutASliceCoversTheRectAndIsCroppedEvenly()
        {
            var count = Lay(new Vector2(2400f, 1080f), Vector4.zero, out _);

            Assert.AreEqual(1, count);
            Near(2400f / 1080f, Aspect(quads[0].Source), "aspect");
            Near(Art.x / 2f, quads[0].Source.center.x, "centre x");
            Near(Art.y / 2f, quads[0].Source.center.y, "centre y");
            Assert.AreEqual(new Rect(0f, 0f, 2400f, 1080f), quads[0].Target);
        }

        [Test]
        public void AnEmptyRectHasNoQuads()
        {
            Assert.AreEqual(0, Lay(Vector2.zero));
        }

        [Test]
        public void LayingOutAllocatesNothing()
        {
            Lay(new Vector2(2400f, 1080f));

            Assert.That(() => { Lay(new Vector2(2400f, 1080f)); }, Allocates.Nothing);
        }
    }
}
