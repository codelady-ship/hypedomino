using System;
using System.Collections;
using System.Collections.Generic;
using HypeDomino.Core.Diagnostics;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Screens;
using HypeDomino.UI.Tokens;
using Unity.Profiling;
using Unity.Profiling.LowLevel.Unsafe;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Bench
{
    public sealed class CanvasBench : MonoBehaviour, IRowBinder
    {
        const int TargetFrameRate = 60;
        const int WindowFrames = 1200;
        const int Cycles = 3;
        const float SettleSeconds = 3f;
        const double MissedVsyncFactor = 1.5;
        const int LobbyPanels = 40;
        const int ItemCount = 1000;
        const float RowHeight = 180f;
        const float IconSize = 140f;
        const float LabelHeight = 60f;
        const int LabelFontSize = 40;
        const float ScrollUnitsPerSecond = 1500f;
        const float PushPopSeconds = 0.6f;
        const int MaxMarkers = 16;
        const double NanosecondsPerMillisecond = 1e6;

        static readonly string[] Phases = { "static", "scroll", "rebind", "pushpop" };
        static readonly string[] MarkerPrefixes = { "Canvas.", "CanvasUpdate.", "UGUI.", "PostLateUpdate.PlayerUpdateCanvases", "PostLateUpdate.PlayerEmitCanvasGeometry", "ClipperRegistry.Cull" };

        readonly FrameTiming[] latestTiming = new FrameTiming[1];
        readonly List<string> markerNames = new(MaxMarkers);
        readonly List<ProfilerRecorder> markers = new(MaxMarkers);
        readonly List<double[]> markerSamples = new(MaxMarkers);
        readonly Dictionary<RectTransform, Text[]> rowLabels = new();
        FrameWindow window;
        ProfilerRecorder gcAllocatedInFrame;
        ProfilerRecorder batchesCount;
        ScreenStack stack;
        ScreenView lobby;
        ScreenView shop;
        RecyclingList list;
        ScrollRect scroll;
        string[] names;
        string[] prices;
        string[] pricesAlternate;
        Font font;
        Sprite white;
        int sampled;
        int cycle;
        int phase;
        int rebindTick;
        float scrollDirection = 1f;
        float nextPushPopAt;
        float settledAt;
        bool skipReportFrame;
        bool finished;
        bool started;

        void Awake()
        {
            Application.targetFrameRate = TargetFrameRate;
            Screen.orientation = ScreenOrientation.Portrait;
            Screen.sleepTimeout = SleepTimeout.NeverSleep;
            window = new FrameWindow(WindowFrames, MissedVsyncFactor / TargetFrameRate);
            font = Resources.GetBuiltinResource<Font>("LegacyRuntime.ttf");
            white = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);
            names = new string[ItemCount];
            prices = new string[ItemCount];
            pricesAlternate = new string[ItemCount];
            for (var at = 0; at < ItemCount; at++)
            {
                names[at] = $"Item {at + 1:D4}";
                prices[at] = $"{(at * 37) % 990 + 10}";
                pricesAlternate[at] = $"{(at * 53) % 990 + 10}";
            }
        }

        void OnEnable()
        {
            gcAllocatedInFrame = ProfilerRecorder.StartNew(ProfilerCategory.Memory, "GC Allocated In Frame");
            batchesCount = ProfilerRecorder.StartNew(ProfilerCategory.Render, "Batches Count");
        }

        void OnDisable()
        {
            gcAllocatedInFrame.Dispose();
            batchesCount.Dispose();
            foreach (var marker in markers) marker.Dispose();
        }

        IEnumerator Start()
        {
            new GameObject("Camera", typeof(Camera)).GetComponent<Camera>().backgroundColor = TokenColor.BackgroundBase;
            stack = new GameObject("Screens").AddComponent<ScreenStack>();
            lobby = BuildLobby();
            shop = BuildShop(out var template);
            stack.Push(lobby);
            stack.Push(shop);
            yield return null;
            yield return null;
            list.Build(template, RowHeight, TokenSpacing.Sm);
            list.Show(ItemCount, this);
            started = true;
            BeginPhase();
        }

        public void Bind(RectTransform row, int index)
        {
            if (!rowLabels.TryGetValue(row, out var labels))
            {
                labels = row.GetComponentsInChildren<Text>(true);
                rowLabels.Add(row, labels);
            }
            labels[0].text = names[index];
            labels[1].text = rebindTick % 2 == 0 ? prices[index] : pricesAlternate[index];
        }

        void Update()
        {
            if (!started) return;
            Drive();
            FrameTimingManager.CaptureFrameTimings();
            if (finished || Time.unscaledTime < settledAt) return;
            if (markers.Count == 0) FindMarkers();
            if (FrameTimingManager.GetLatestTimings(1, latestTiming) == 0) return;

            var gcBytes = skipReportFrame ? 0 : LastValue(gcAllocatedInFrame);
            skipReportFrame = false;
            var timing = latestTiming[0];
            var work = FrameWindow.WorkMs(timing);
            window.Add(work, Time.unscaledDeltaTime, gcBytes, LastValue(batchesCount));
            for (var at = 0; at < markers.Count; at++) markerSamples[at][sampled] = LastValue(markers[at]) / NanosecondsPerMillisecond;
            sampled++;
            if (!window.IsFull) return;

            Report(window.Close());
            sampled = 0;
            skipReportFrame = true;
            NextPhase();
        }

        void Drive()
        {
            switch (Phases[phase])
            {
                case "scroll":
                    var content = scroll.content;
                    var limit = Mathf.Max(0f, content.rect.height - scroll.viewport.rect.height);
                    var y = content.anchoredPosition.y + scrollDirection * ScrollUnitsPerSecond * Time.unscaledDeltaTime;
                    if (y >= limit || y <= 0f) scrollDirection = -scrollDirection;
                    content.anchoredPosition = new Vector2(0f, Mathf.Clamp(y, 0f, limit));
                    break;
                case "rebind":
                    rebindTick++;
                    var first = Mathf.FloorToInt(scroll.content.anchoredPosition.y / (RowHeight + TokenSpacing.Sm));
                    for (var index = Mathf.Max(0, first - 1); index < Mathf.Min(ItemCount, first + list.RowCount); index++) list.Refresh(index);
                    break;
                case "pushpop":
                    if (Time.unscaledTime < nextPushPopAt) break;
                    nextPushPopAt = Time.unscaledTime + PushPopSeconds;
                    if (stack.Top == shop) stack.Pop();
                    else stack.Push(shop);
                    break;
            }
        }

        void BeginPhase()
        {
            var showShop = Phases[phase] != "static";
            if (showShop && stack.Top != shop) stack.Push(shop);
            if (!showShop && stack.Top == shop) stack.Pop();
            settledAt = Time.unscaledTime + SettleSeconds;
            nextPushPopAt = settledAt;
        }

        void NextPhase()
        {
            phase++;
            if (phase == Phases.Length)
            {
                phase = 0;
                cycle++;
            }
            if (cycle == Cycles)
            {
                finished = true;
                Debug.Log("CanvasBench finished");
                return;
            }
            BeginPhase();
        }

        void Report(FrameWindowReport report)
        {
            var line = new System.Text.StringBuilder();
            line.Append($"CanvasBench cycle={cycle + 1} phase={Phases[phase]} frames={report.Frames} workP50={report.WorkP50Ms:F2} workP95={report.WorkP95Ms:F2} missed={report.MissedVsyncPercent:F2} gcFrames={report.AllocatingFrames} gcMax={report.MaxGcBytes} batches={report.MaxBatches}");
            for (var at = 0; at < markers.Count; at++)
            {
                var samples = markerSamples[at];
                Array.Sort(samples);
                line.Append($" {markerNames[at]}=p50:{samples[WindowFrames / 2]:F3}/p95:{samples[WindowFrames * 95 / 100]:F3}/max:{samples[WindowFrames - 1]:F3}");
            }
            line.Append($" size={Screen.width}x{Screen.height}");
            Debug.Log(line.ToString());
        }

        void FindMarkers()
        {
            var available = new List<ProfilerRecorderHandle>();
            ProfilerRecorderHandle.GetAvailable(available);
            foreach (var handle in available)
            {
                if (markers.Count == MaxMarkers) break;
                var name = ProfilerRecorderHandle.GetDescription(handle).Name;
                if (!IsCanvasMarker(name) || markerNames.Contains(name)) continue;
                markerNames.Add(name);
                markers.Add(new ProfilerRecorder(handle, 1, ProfilerRecorderOptions.Default | ProfilerRecorderOptions.StartImmediately));
                markerSamples.Add(new double[WindowFrames]);
            }
            Debug.Log($"CanvasBench markers={string.Join(",", markerNames)}");
        }

        static bool IsCanvasMarker(string name)
        {
            foreach (var prefix in MarkerPrefixes)
                if (name.StartsWith(prefix, StringComparison.Ordinal)) return true;
            return false;
        }

        ScreenView BuildLobby()
        {
            var view = NewScreen("Lobby", out var content);
            for (var at = 0; at < LobbyPanels; at++)
            {
                var panel = NewImage("Panel", content, TokenColor.SurfaceDefault);
                Place(panel.rectTransform, at % 4, at / 4, 4, LobbyPanels / 4, 240f);
                var label = NewText("Label", panel.transform, TokenColor.TextPrimary);
                label.text = names[at];
                Stretch(label.rectTransform);
            }
            return view;
        }

        ScreenView BuildShop(out RectTransform template)
        {
            var view = NewScreen("Shop", out var content);
            var background = NewImage("Background", content, TokenColor.BackgroundRaised);
            Stretch(background.rectTransform);
            var holder = new GameObject("List", typeof(RectTransform), typeof(ScrollRect));
            holder.transform.SetParent(content, false);
            Stretch((RectTransform)holder.transform);
            var viewport = new GameObject("Viewport", typeof(RectTransform), typeof(RectMask2D)).GetComponent<RectTransform>();
            viewport.SetParent(holder.transform, false);
            Stretch(viewport);
            var rows = new GameObject("Rows", typeof(RectTransform)).GetComponent<RectTransform>();
            rows.SetParent(viewport, false);
            Stretch(rows);
            scroll = holder.GetComponent<ScrollRect>();
            scroll.viewport = viewport;
            scroll.content = rows;
            scroll.horizontal = false;
            list = holder.AddComponent<RecyclingList>();
            template = RowTemplate(content);
            return view;
        }

        RectTransform RowTemplate(Transform parent)
        {
            var row = NewImage("Row", parent, TokenColor.SurfaceDefault);
            var icon = NewImage("Icon", row.transform, TokenColor.RolePrimaryFill);
            icon.rectTransform.anchorMin = new Vector2(0f, 0.5f);
            icon.rectTransform.anchorMax = new Vector2(0f, 0.5f);
            icon.rectTransform.sizeDelta = new Vector2(IconSize, IconSize);
            icon.rectTransform.anchoredPosition = new Vector2(TokenSpacing.Lg + IconSize / 2f, 0f);
            Label(row.transform, "Name", TextAnchor.MiddleLeft, TokenColor.TextPrimary);
            Label(row.transform, "Price", TextAnchor.MiddleRight, TokenColor.RoleWarningFill);
            row.gameObject.SetActive(false);
            return row.rectTransform;
        }

        void Label(Transform row, string name, TextAnchor alignment, Color32 color)
        {
            var label = NewText(name, row, color);
            label.alignment = alignment;
            var rect = label.rectTransform;
            rect.anchorMin = new Vector2(0f, 0.5f);
            rect.anchorMax = new Vector2(1f, 0.5f);
            rect.sizeDelta = new Vector2(-(IconSize + TokenSpacing.Lg * 3), LabelHeight);
            rect.anchoredPosition = new Vector2((IconSize + TokenSpacing.Lg) / 2f, 0f);
        }

        ScreenView NewScreen(string name, out Transform content)
        {
            var view = new GameObject(name, typeof(RectTransform)).AddComponent<ScreenView>();
            var root = new GameObject("Content", typeof(RectTransform), typeof(CanvasGroup));
            root.transform.SetParent(view.transform, false);
            Stretch((RectTransform)root.transform);
            view.Configure(root.GetComponent<CanvasGroup>(), null);
            content = root.transform;
            return view;
        }

        Image NewImage(string name, Transform parent, Color32 color)
        {
            var image = new GameObject(name, typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            image.sprite = white;
            image.raycastTarget = false;
            image.color = color;
            image.transform.SetParent(parent, false);
            return image;
        }

        Text NewText(string name, Transform parent, Color32 color)
        {
            var text = new GameObject(name, typeof(RectTransform), typeof(Text)).GetComponent<Text>();
            text.font = font;
            text.fontSize = LabelFontSize;
            text.alignment = TextAnchor.MiddleCenter;
            text.raycastTarget = false;
            text.color = color;
            text.transform.SetParent(parent, false);
            return text;
        }

        static void Place(RectTransform rect, int column, int row, int columns, int rows, float size)
        {
            var anchor = new Vector2((column + 0.5f) / columns, (row + 0.5f) / rows);
            rect.anchorMin = anchor;
            rect.anchorMax = anchor;
            rect.sizeDelta = new Vector2(size, size);
        }

        static void Stretch(RectTransform rect)
        {
            rect.anchorMin = Vector2.zero;
            rect.anchorMax = Vector2.one;
            rect.offsetMin = Vector2.zero;
            rect.offsetMax = Vector2.zero;
        }

        static long LastValue(ProfilerRecorder recorder) => recorder.Valid ? recorder.LastValue : 0;
    }
}
