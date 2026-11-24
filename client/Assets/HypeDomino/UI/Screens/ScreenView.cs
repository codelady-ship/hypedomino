using HypeDomino.UI.Tokens;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Screens
{
    [RequireComponent(typeof(Canvas), typeof(CanvasScaler))]
    public sealed class ScreenView : MonoBehaviour
    {
        public const float ReferenceShortSide = 1080f;

        [SerializeField] CanvasGroup content;
        [SerializeField] Image scrim;

        Canvas canvas;
        RectTransform body;
        GraphicRaycaster raycaster;
        ScreenFade fade;
        ScreenFade shade;

        public bool IsModal => scrim != null;

        public bool IsShown => canvas.enabled;

        public bool TakesInput => raycaster.enabled;

        public float Visibility => fade.Value;

        public int SortingOrder => canvas.sortingOrder;

        public void Configure(CanvasGroup screenContent, Image modalScrim)
        {
            content = screenContent;
            scrim = modalScrim;
            Nest();
            PaintScrim();
            Apply(false);
        }

        void Awake()
        {
            canvas = GetComponent<Canvas>();
            canvas.renderMode = RenderMode.ScreenSpaceOverlay;
            canvas.enabled = false;
            body = new GameObject("Body", typeof(RectTransform), typeof(Canvas), typeof(GraphicRaycaster)).GetComponent<RectTransform>();
            body.SetParent(transform, false);
            body.anchorMin = Vector2.zero;
            body.anchorMax = Vector2.one;
            body.sizeDelta = Vector2.zero;
            raycaster = body.GetComponent<GraphicRaycaster>();
            raycaster.enabled = false;
            var scaler = GetComponent<CanvasScaler>();
            scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
            scaler.referenceResolution = new Vector2(ReferenceShortSide, ReferenceShortSide);
            scaler.screenMatchMode = CanvasScaler.ScreenMatchMode.Expand;
            Nest();
            PaintScrim();
            Apply(false);
        }

        void Nest()
        {
            if (body == null) return;
            if (scrim != null) scrim.transform.SetParent(body, false);
            if (content != null) content.transform.SetParent(body, false);
        }

        void PaintScrim()
        {
            if (scrim != null) scrim.color = TokenColor.Scrim;
        }

        internal void Show(int sortingOrder)
        {
            canvas.sortingOrder = sortingOrder;
            canvas.enabled = true;
        }

        internal void Hide() => canvas.enabled = false;

        internal void TakeInput(bool on) => raycaster.enabled = on;

        internal void Enter()
        {
            fade.Retarget(1f, IsModal ? TokenDurationMs.Moderate : TokenDurationMs.Slow, TokenEasing.Enter);
            if (IsModal) shade.Retarget(1f, TokenDurationMs.Base, TokenEasing.Standard);
        }

        internal void Exit()
        {
            fade.Retarget(0f, TokenDurationMs.Base, TokenEasing.Exit);
            if (IsModal) shade.Retarget(0f, TokenDurationMs.Base, TokenEasing.Standard);
        }

        internal bool Advance(float deltaMs, bool reduceMotion)
        {
            var fadeDone = fade.Advance(deltaMs);
            var shadeDone = shade.Advance(deltaMs);
            Apply(reduceMotion);
            return fadeDone && shadeDone;
        }

        void Apply(bool reduceMotion)
        {
            if (content != null)
            {
                content.alpha = fade.Value;
                var scale = reduceMotion ? 1f : Mathf.LerpUnclamped(TokenMotion.PanelEnterScale, 1f, fade.Value);
                content.transform.localScale = new Vector3(scale, scale, 1f);
            }
            if (scrim != null) scrim.canvasRenderer.SetAlpha(TokenOpacity.Scrim * shade.Value);
        }
    }
}
