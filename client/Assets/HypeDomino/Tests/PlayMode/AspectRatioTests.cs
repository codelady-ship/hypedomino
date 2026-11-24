using System.Collections;
using HypeDomino.UI.Layout;
using HypeDomino.UI.Screens;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.TestTools;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class AspectRatioTests
    {
        const float ShortSideUnits = 1080f;
        const float UnitTolerance = 0.5f;

        static readonly Display[] Displays =
        {
            new Display("low, galaxy a14 20:9 portrait, camera hole", 1080, 2408, new Rect(0, 0, 1080, 2328)),
            new Display("mid, galaxy a34 19.5:9 portrait, notch", 1080, 2340, new Rect(0, 0, 1080, 2260)),
            new Display("mid, galaxy a34 19.5:9 landscape, notch", 2340, 1080, new Rect(80, 0, 2260, 1080)),
            new Display("high, galaxy s24 19.5:9 landscape, camera hole", 2340, 1080, new Rect(96, 0, 2244, 1080)),
            new Display("low, iphone xr portrait", 828, 1792, new Rect(0, 68, 828, 1636)),
            new Display("mid, iphone 12 portrait", 1170, 2532, new Rect(0, 102, 1170, 2289)),
            new Display("high, iphone 15 pro portrait, island", 1179, 2556, new Rect(0, 102, 1179, 2277)),
            new Display("high, iphone 15 pro landscape, island", 2556, 1179, new Rect(177, 63, 2202, 1116)),
            new Display("phone 16:9 portrait", 1080, 1920, new Rect(0, 0, 1080, 1920)),
            new Display("phone 21:9 portrait", 1080, 2520, new Rect(0, 0, 1080, 2440)),
            new Display("phone 21:9 landscape", 2520, 1080, new Rect(80, 0, 2440, 1080)),
            new Display("tablet 4:3 portrait", 1536, 2048, new Rect(0, 0, 1536, 2048)),
            new Display("tablet 4:3 landscape", 2048, 1536, new Rect(0, 40, 2048, 1496)),
            new Display("folding inner display, nearly square", 1812, 2176, new Rect(0, 0, 1812, 2096)),
        };

        readonly struct Display
        {
            public readonly string Name;
            public readonly int Width;
            public readonly int Height;
            public readonly Rect SafeArea;

            public Display(string name, int width, int height, Rect safeArea)
            {
                Name = name;
                Width = width;
                Height = height;
                SafeArea = safeArea;
            }
        }

        GameObject cameraObject;
        RenderTexture target;
        ScreenView screen;

        [TearDown]
        public void DestroyAll() => DestroyDisplay();

        void DestroyDisplay()
        {
            if (screen != null) Object.Destroy(screen.gameObject);
            if (cameraObject != null) Object.Destroy(cameraObject);
            if (target != null) Object.Destroy(target);
            screen = null;
            cameraObject = null;
            target = null;
        }

        IEnumerator ShowOn(int width, int height)
        {
            target = new RenderTexture(width, height, 0);
            cameraObject = new GameObject("Display", typeof(Camera));
            var displayCamera = cameraObject.GetComponent<Camera>();
            displayCamera.targetTexture = target;
            screen = new GameObject("Screen", typeof(RectTransform)).AddComponent<ScreenView>();
            var canvas = screen.GetComponent<Canvas>();
            canvas.renderMode = RenderMode.ScreenSpaceCamera;
            canvas.worldCamera = displayCamera;
            canvas.enabled = true;
            yield return null;
            yield return null;
            Canvas.ForceUpdateCanvases();
        }

        RectTransform SafeContent(Rect safeArea, int width, int height)
        {
            var safe = new GameObject("Safe", typeof(RectTransform)).GetComponent<RectTransform>();
            safe.SetParent(screen.transform, false);
            SafeAreaAnchors.For(safeArea, width, height).ApplyTo(safe);
            return safe;
        }

        [UnityTest]
        public IEnumerator TheShortSideIs1080UnitsOnEveryDisplay()
        {
            foreach (var display in Displays)
            {
                yield return ShowOn(display.Width, display.Height);
                var canvasSize = ((RectTransform)screen.transform).rect.size;
                Assert.AreEqual(Mathf.Min(display.Width, display.Height) / ShortSideUnits, screen.GetComponent<Canvas>().scaleFactor, 0.001f, display.Name);
                Assert.AreEqual(ShortSideUnits, Mathf.Min(canvasSize.x, canvasSize.y), UnitTolerance, display.Name);
                var aspect = (float)Mathf.Max(display.Width, display.Height) / Mathf.Min(display.Width, display.Height);
                Assert.AreEqual(aspect, Mathf.Max(canvasSize.x, canvasSize.y) / Mathf.Min(canvasSize.x, canvasSize.y), 0.001f, display.Name);
                DestroyDisplay();
            }
        }

        [UnityTest]
        public IEnumerator SafeContentStaysOnScreenAndKeepsItsShape()
        {
            var corners = new Vector3[4];
            foreach (var display in Displays)
            {
                yield return ShowOn(display.Width, display.Height);
                var canvasRect = ((RectTransform)screen.transform).rect;
                var safe = SafeContent(display.SafeArea, display.Width, display.Height);
                Canvas.ForceUpdateCanvases();
                var unitsPerPixel = ShortSideUnits / Mathf.Min(display.Width, display.Height);
                Assert.AreEqual(display.SafeArea.width * unitsPerPixel, safe.rect.width, UnitTolerance, display.Name);
                Assert.AreEqual(display.SafeArea.height * unitsPerPixel, safe.rect.height, UnitTolerance, display.Name);
                safe.GetLocalCorners(corners);
                foreach (var corner in corners)
                {
                    var inScreen = screen.transform.InverseTransformPoint(safe.TransformPoint(corner));
                    Assert.GreaterOrEqual(inScreen.x, canvasRect.xMin - UnitTolerance, display.Name);
                    Assert.LessOrEqual(inScreen.x, canvasRect.xMax + UnitTolerance, display.Name);
                    Assert.GreaterOrEqual(inScreen.y, canvasRect.yMin - UnitTolerance, display.Name);
                    Assert.LessOrEqual(inScreen.y, canvasRect.yMax + UnitTolerance, display.Name);
                }
                DestroyDisplay();
            }
        }
    }
}
