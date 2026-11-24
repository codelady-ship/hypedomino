using System;
using HypeDomino.UI.Rarity;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Cosmetics
{
    [RequireComponent(typeof(RectTransform))]
    public sealed class AvatarView : MonoBehaviour
    {
        static readonly Vector2 Centre = new Vector2(0.5f, 0.5f);

        [SerializeField] CosmeticImage avatar;
        [SerializeField] CosmeticImage frame;
        [SerializeField] RarityHalo halo;
        [SerializeField] RarityDiamond diamond;

        CosmeticStage stage;
        string markRarityId;
        Action fitArt;

        public CosmeticImage Avatar => avatar;

        public CosmeticImage Frame => frame;

        Action FitArtAction => fitArt ??= FitArt;

        public void Configure(CosmeticImage avatarImage, CosmeticImage frameImage, RarityHalo haloMark, RarityDiamond diamondMark)
        {
            StopWatchingFrame();
            avatar = avatarImage;
            frame = frameImage;
            if (isActiveAndEnabled) WatchFrame();
            halo = haloMark;
            diamond = diamondMark;
            FitArt();
        }

        public void Show(CosmeticStage onStage, AvatarLook look)
        {
            stage = onStage;
            markRarityId = look.MarkRarityId;
            avatar.Show(stage, look.Avatar, stage.BuiltInAvatar);
            frame.Show(stage, look.Frame, stage.BuiltInFrame);
            ShowMarks();
        }

        public void RefreshEffects()
        {
            if (stage == null) return;
            avatar.Refresh();
            frame.Refresh();
            ShowMarks();
        }

        void OnEnable()
        {
            WatchFrame();
            FitArt();
        }

        void OnDisable() => StopWatchingFrame();

        void OnRectTransformDimensionsChange() => FitArt();

        void WatchFrame()
        {
            if (frame != null) frame.Rested += FitArtAction;
        }

        void StopWatchingFrame()
        {
            if (frame != null) frame.Rested -= FitArtAction;
        }

        void FitArt()
        {
            var box = ((RectTransform)transform).rect;
            var side = Mathf.Clamp(Mathf.Min(box.width, box.height), 0f, LargestFrameSide());
            FitSquare(avatar, side);
            FitSquare(frame, side);
            FitSquare(halo, side);
        }

        float LargestFrameSide()
        {
            var sprite = frame != null ? frame.Shown : null;
            if (sprite == null) return float.PositiveInfinity;
            var canvas = GetComponentInParent<Canvas>();
            var scale = canvas != null && canvas.rootCanvas.scaleFactor > 0f ? canvas.rootCanvas.scaleFactor : 1f;
            return Mathf.Max(sprite.rect.width, sprite.rect.height) / scale;
        }

        static void FitSquare(Component art, float side)
        {
            if (art == null) return;
            var rect = (RectTransform)art.transform;
            rect.anchorMin = Centre;
            rect.anchorMax = Centre;
            rect.pivot = Centre;
            rect.anchoredPosition = Vector2.zero;
            rect.sizeDelta = new Vector2(side, side);
            if (art.TryGetComponent<Image>(out var image)) image.preserveAspect = true;
        }

        void ShowMarks()
        {
            if (halo != null) halo.Show(stage.Rarities, markRarityId, stage.Level, stage.ReduceMotion, Time.unscaledTime);
            if (diamond != null) diamond.Show(stage.Rarities, markRarityId);
        }
    }
}
