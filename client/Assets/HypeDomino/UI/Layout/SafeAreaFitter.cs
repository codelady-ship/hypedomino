using UnityEngine;

namespace HypeDomino.UI.Layout
{
    [RequireComponent(typeof(RectTransform))]
    public sealed class SafeAreaFitter : MonoBehaviour
    {
        RectTransform target;
        Rect fittedSafeArea;
        int fittedWidth;
        int fittedHeight;

        void Awake() => target = (RectTransform)transform;

        void OnEnable() => Fit();

        void Update()
        {
            if (Screen.safeArea == fittedSafeArea && Screen.width == fittedWidth && Screen.height == fittedHeight) return;
            Fit();
        }

        void Fit()
        {
            fittedSafeArea = Screen.safeArea;
            fittedWidth = Screen.width;
            fittedHeight = Screen.height;
            SafeAreaAnchors.For(fittedSafeArea, fittedWidth, fittedHeight).ApplyTo(target);
        }
    }
}
