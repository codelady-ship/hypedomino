using System;
using System.Globalization;
using HypeDomino.Core.Diagnostics;
using HypeDomino.UI.Layout;
using HypeDomino.UI.Screens;
using HypeDomino.UI.Tokens;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Diagnostics
{
    [RequireComponent(typeof(FrameTimingSampler))]
    public sealed class FrameStatsOverlay : MonoBehaviour
    {
        public const string WorkP95Row = "p95 ms";
        public const string MissedVsyncRow = "missed %";
        public const string GcFramesRow = "gc frames";
        public const string BatchesRow = "batches";

        const int TableSize = 1000;
        const int LabelWidth = TokenSpacing.Xxxl * 4;
        const int ValueWidth = TokenSpacing.Xxxl * 2;
        const int RowHeight = TokenTextSize.Caption + TokenSpacing.Xs;
        const int Padding = TokenSpacing.Sm;

        static readonly string[] Rows = { WorkP95Row, MissedVsyncRow, GcFramesRow, BatchesRow };
        static string[] wholeNumbers;
        static string[] tenths;

        readonly Text[] values = new Text[Rows.Length];
        FrameTimingSampler sampler;
        Font font;
        int shownVersion;

        void Awake()
        {
            if (!Debug.isDebugBuild)
            {
                Destroy(this);
                return;
            }
            sampler = GetComponent<FrameTimingSampler>();
            FillTables();
            Build();
            Show(default);
        }

        void Update()
        {
            if (sampler.LiveVersion == shownVersion) return;
            shownVersion = sampler.LiveVersion;
            Show(sampler.Live);
        }

        public void Show(in FrameWindowReport report)
        {
            values[0].text = Tenths(report.WorkP95Ms);
            values[1].text = Tenths(report.MissedVsyncPercent);
            values[2].text = Whole(report.AllocatingFrames);
            values[2].color = report.AllocatingFrames > 0 ? TokenColor.RoleDangerText : TokenColor.TextPrimary;
            values[3].text = Whole(report.MaxBatches);
        }

        public Text ValueOf(string row) => values[Array.IndexOf(Rows, row)];

        static string Whole(long value) => wholeNumbers[(int)Math.Clamp(value, 0, TableSize - 1)];

        static string Tenths(double value) => tenths[(int)Math.Clamp(Math.Round(value * 10), 0, TableSize - 1)];

        static void FillTables()
        {
            if (wholeNumbers != null) return;
            wholeNumbers = new string[TableSize];
            tenths = new string[TableSize];
            for (var at = 0; at < TableSize; at++)
            {
                wholeNumbers[at] = at.ToString(CultureInfo.InvariantCulture);
                tenths[at] = (at / 10.0).ToString("0.0", CultureInfo.InvariantCulture);
            }
        }

        void Build()
        {
            font = Resources.GetBuiltinResource<Font>("LegacyRuntime.ttf");
            var root = new GameObject("Frame stats", typeof(RectTransform), typeof(Canvas), typeof(CanvasScaler));
            root.transform.SetParent(transform, false);
            var canvas = root.GetComponent<Canvas>();
            canvas.renderMode = RenderMode.ScreenSpaceOverlay;
            canvas.sortingOrder = short.MaxValue;
            var scaler = root.GetComponent<CanvasScaler>();
            scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
            scaler.referenceResolution = new Vector2(ScreenView.ReferenceShortSide, ScreenView.ReferenceShortSide);
            scaler.screenMatchMode = CanvasScaler.ScreenMatchMode.Expand;

            var safe = Child("Safe area", root.transform);
            Stretch(safe);
            safe.gameObject.AddComponent<SafeAreaFitter>();

            var panel = Child("Panel", safe);
            panel.anchorMin = panel.anchorMax = panel.pivot = new Vector2(0f, 1f);
            panel.sizeDelta = new Vector2(LabelWidth + ValueWidth + 2 * Padding, Rows.Length * RowHeight + 2 * Padding);
            var background = panel.gameObject.AddComponent<Image>();
            Color shade = TokenColor.Scrim;
            shade.a = TokenOpacity.Scrim;
            background.color = shade;
            background.raycastTarget = false;

            for (var row = 0; row < Rows.Length; row++)
            {
                var top = -Padding - row * RowHeight;
                Label(Rows[row], panel, Padding, top, LabelWidth, TextAnchor.MiddleLeft, TokenColor.TextSecondary).text = Rows[row];
                values[row] = Label(Rows[row] + " value", panel, Padding + LabelWidth, top, ValueWidth, TextAnchor.MiddleRight, TokenColor.TextPrimary);
            }
        }

        Text Label(string name, Transform parent, float left, float top, float width, TextAnchor alignment, Color color)
        {
            var rect = Child(name, parent);
            rect.anchorMin = rect.anchorMax = rect.pivot = new Vector2(0f, 1f);
            rect.anchoredPosition = new Vector2(left, top);
            rect.sizeDelta = new Vector2(width, RowHeight);
            var text = rect.gameObject.AddComponent<Text>();
            text.font = font;
            text.fontSize = TokenTextSize.Caption;
            text.alignment = alignment;
            text.color = color;
            text.raycastTarget = false;
            return text;
        }

        static RectTransform Child(string name, Transform parent)
        {
            var rect = new GameObject(name, typeof(RectTransform)).GetComponent<RectTransform>();
            rect.SetParent(parent, false);
            return rect;
        }

        static void Stretch(RectTransform rect)
        {
            rect.anchorMin = Vector2.zero;
            rect.anchorMax = Vector2.one;
            rect.sizeDelta = Vector2.zero;
        }
    }
}
