using UnityEngine;

namespace HypeDomino.UI.Layout
{
    public readonly struct SafeAreaAnchors
    {
        public static readonly SafeAreaAnchors FullScreen = new SafeAreaAnchors(Vector2.zero, Vector2.one);

        public readonly Vector2 Min;
        public readonly Vector2 Max;

        public SafeAreaAnchors(Vector2 min, Vector2 max)
        {
            Min = min;
            Max = max;
        }

        public static SafeAreaAnchors For(Rect safeArea, int screenWidth, int screenHeight)
        {
            if (screenWidth <= 0 || screenHeight <= 0 || safeArea.width <= 0 || safeArea.height <= 0) return FullScreen;

            var min = new Vector2(Mathf.Clamp01(safeArea.xMin / screenWidth), Mathf.Clamp01(safeArea.yMin / screenHeight));
            var max = new Vector2(Mathf.Clamp01(safeArea.xMax / screenWidth), Mathf.Clamp01(safeArea.yMax / screenHeight));
            return new SafeAreaAnchors(min, max);
        }

        public void ApplyTo(RectTransform target)
        {
            target.anchorMin = Min;
            target.anchorMax = Max;
            target.offsetMin = Vector2.zero;
            target.offsetMax = Vector2.zero;
        }
    }
}
