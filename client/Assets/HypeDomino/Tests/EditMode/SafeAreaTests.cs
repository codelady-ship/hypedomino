using HypeDomino.UI.Layout;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SafeAreaTests
    {
        const float Tolerance = 1e-5f;

        static readonly object[] Screens =
        {
            new object[] { "android 20:9 landscape, camera hole left", 2400, 1080, new Rect(100, 0, 2300, 1080), new Vector2(100f / 2400, 0), Vector2.one },
            new object[] { "iphone 19.5:9 landscape, side insets and home indicator", 2556, 1179, new Rect(177, 63, 2202, 1116), new Vector2(177f / 2556, 63f / 1179), new Vector2(2379f / 2556, 1) },
            new object[] { "iphone 19.5:9 portrait, island top and home indicator", 1179, 2556, new Rect(0, 102, 1179, 2277), new Vector2(0, 102f / 2556), new Vector2(1, 2379f / 2556) },
            new object[] { "tablet 4:3 without insets", 2048, 1536, new Rect(0, 0, 2048, 1536), Vector2.zero, Vector2.one },
            new object[] { "phone 16:9 without insets", 1920, 1080, new Rect(0, 0, 1920, 1080), Vector2.zero, Vector2.one },
            new object[] { "safe area reported past the screen edge", 1920, 1080, new Rect(-20, 0, 1960, 1100), Vector2.zero, Vector2.one },
        };

        [TestCaseSource(nameof(Screens))]
        public void AnchorsFollowTheSafeAreaOnEveryAspect(string device, int width, int height, Rect safeArea, Vector2 expectedMin, Vector2 expectedMax)
        {
            var anchors = SafeAreaAnchors.For(safeArea, width, height);
            AssertClose(expectedMin, anchors.Min, device);
            AssertClose(expectedMax, anchors.Max, device);
        }

        [TestCase(0, 1080)]
        [TestCase(1920, 0)]
        public void AnEmptyScreenFillsTheScreen(int width, int height)
        {
            var anchors = SafeAreaAnchors.For(new Rect(0, 0, 100, 100), width, height);
            Assert.AreEqual(SafeAreaAnchors.FullScreen.Min, anchors.Min);
            Assert.AreEqual(SafeAreaAnchors.FullScreen.Max, anchors.Max);
        }

        [Test]
        public void AnEmptySafeAreaFillsTheScreen()
        {
            var anchors = SafeAreaAnchors.For(new Rect(0, 0, 0, 0), 1920, 1080);
            Assert.AreEqual(Vector2.zero, anchors.Min);
            Assert.AreEqual(Vector2.one, anchors.Max);
        }

        [Test]
        public void ApplyingAnchorsStretchesTheRectWithoutOffsets()
        {
            var root = new GameObject("SafeArea", typeof(RectTransform));
            try
            {
                var rect = (RectTransform)root.transform;
                rect.offsetMin = new Vector2(10, 10);
                rect.offsetMax = new Vector2(-10, -10);
                SafeAreaAnchors.For(new Rect(100, 0, 2300, 1080), 2400, 1080).ApplyTo(rect);
                AssertClose(new Vector2(100f / 2400, 0), rect.anchorMin, "anchorMin");
                AssertClose(Vector2.one, rect.anchorMax, "anchorMax");
                Assert.AreEqual(Vector2.zero, rect.offsetMin);
                Assert.AreEqual(Vector2.zero, rect.offsetMax);
            }
            finally
            {
                Object.DestroyImmediate(root);
            }
        }

        static void AssertClose(Vector2 expected, Vector2 actual, string device)
        {
            Assert.AreEqual(expected.x, actual.x, Tolerance, device);
            Assert.AreEqual(expected.y, actual.y, Tolerance, device);
        }
    }
}
