using HypeDomino.Core.Diagnostics;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Rarity;
using HypeDomino.UI.Tokens;
using Unity.Profiling;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Bench
{
    public sealed class AnimationBench : MonoBehaviour
    {
        const int TargetFrameRate = 60;
        const int WindowFrames = 1200;
        const int WindowsPerPhase = 2;
        const int Cycles = 3;
        const float SettleSeconds = 5f;
        const float LoopFramesPerSecond = 12f;
        const double MissedVsyncFactor = 1.5;
        const int PanelCount = 40;
        const float PanelAlpha = 0.5f;
        const int ColumnCount = 2;
        const float AvatarSize = 300f;
        const float DiamondSize = 96f;
        const int HaloCount = 4;
        const int MarkColumns = 4;
        const int HaloCell = 7;
        const int FlowHaloCell = 8;
        const int PipOverlayCount = 6;
        const string HaloTier = "mythic";

        static readonly string[] Phases = { "still", "nested", "shared", "marks" };
        static readonly string[] DiamondTiers = { "common", "rare", "epic", "legendary", "mythic" };

        [SerializeField] Texture2D[] atlases;
        [SerializeField] int cellSize;
        [SerializeField] int framesPerAvatar;
        [SerializeField] Texture2D marksAtlas;
        [SerializeField] TextAsset rarityDefaults;

        readonly FrameTiming[] latestTiming = new FrameTiming[1];
        FrameWindow window;
        FrameWindow mainThread;
        FrameWindow renderThread;
        FrameWindow gpu;
        ProfilerRecorder gcAllocatedInFrame;
        ProfilerRecorder batchesCount;
        ProfilerRecorder setPassCount;
        ProfilerRecorder usedMemory;
        Transform screenRoot;
        Transform nestedRoot;
        GameObject marksRoot;
        AnimatedImage[] avatars;
        LoopArt[] art;
        AnimationBudget budget;
        int cycle;
        int phase;
        int windowsDone;
        float settledAt;
        bool skipReportFrame;
        bool finished;

        public void Configure(Texture2D[] avatarAtlases, int cell, int frames, Texture2D marks, TextAsset rarities)
        {
            atlases = avatarAtlases;
            cellSize = cell;
            framesPerAvatar = frames;
            marksAtlas = marks;
            rarityDefaults = rarities;
        }

        void Awake()
        {
            Application.targetFrameRate = TargetFrameRate;
            Screen.orientation = ScreenOrientation.Portrait;
            Screen.sleepTimeout = SleepTimeout.NeverSleep;
            window = new FrameWindow(WindowFrames, MissedVsyncFactor / TargetFrameRate);
            mainThread = new FrameWindow(WindowFrames, MissedVsyncFactor / TargetFrameRate);
            renderThread = new FrameWindow(WindowFrames, MissedVsyncFactor / TargetFrameRate);
            gpu = new FrameWindow(WindowFrames, MissedVsyncFactor / TargetFrameRate);
            budget = AnimationBudget.For(EffectLevel.Medium);
        }

        void OnEnable()
        {
            gcAllocatedInFrame = ProfilerRecorder.StartNew(ProfilerCategory.Memory, "GC Allocated In Frame");
            batchesCount = ProfilerRecorder.StartNew(ProfilerCategory.Render, "Batches Count");
            setPassCount = ProfilerRecorder.StartNew(ProfilerCategory.Render, "SetPass Calls Count");
            usedMemory = ProfilerRecorder.StartNew(ProfilerCategory.Memory, "System Used Memory");
        }

        void OnDisable()
        {
            gcAllocatedInFrame.Dispose();
            batchesCount.Dispose();
            setPassCount.Dispose();
            usedMemory.Dispose();
        }

        void Start()
        {
            BuildScreen();
            BeginPhase();
        }

        void Update()
        {
            FrameTimingManager.CaptureFrameTimings();
            if (finished || Time.unscaledTime < settledAt) return;
            if (FrameTimingManager.GetLatestTimings(1, latestTiming) == 0) return;

            var gcBytes = skipReportFrame ? 0 : LastValue(gcAllocatedInFrame);
            skipReportFrame = false;
            var timing = latestTiming[0];
            var work = FrameWindow.WorkMs(timing);
            window.Add(work, Time.unscaledDeltaTime, gcBytes, LastValue(batchesCount));
            mainThread.Add(timing.cpuMainThreadFrameTime, 0, 0, 0);
            renderThread.Add(timing.cpuRenderThreadFrameTime, 0, 0, 0);
            gpu.Add(timing.gpuFrameTime, 0, 0, 0);
            if (!window.IsFull) return;

            Report(window.Close());
            skipReportFrame = true;
            windowsDone++;
            if (windowsDone == WindowsPerPhase) NextPhase();
        }

        void Report(FrameWindowReport report)
        {
            var main = mainThread.Close();
            var render = renderThread.Close();
            var graphics = gpu.Close();
            Debug.Log($"AnimationBench cycle={cycle + 1} phase={Phases[phase]} window={windowsDone + 1} frames={report.Frames} " +
                      $"workP50={report.WorkP50Ms:F2} workP95={report.WorkP95Ms:F2} missed={report.MissedVsyncPercent:F2} " +
                      $"mainP50={main.WorkP50Ms:F2} mainP95={main.WorkP95Ms:F2} renderP50={render.WorkP50Ms:F2} renderP95={render.WorkP95Ms:F2} " +
                      $"gpuP50={graphics.WorkP50Ms:F2} gpuP95={graphics.WorkP95Ms:F2} " +
                      $"gcFrames={report.AllocatingFrames} gcMax={report.MaxGcBytes} batches={report.MaxBatches} " +
                      $"setPass={LastValue(setPassCount)} usedMemoryMb={LastValue(usedMemory) / (1024 * 1024)} active={budget.Active} " +
                      $"size={Screen.width}x{Screen.height} refresh={Screen.currentResolution.refreshRateRatio.value:F0}");
        }

        void NextPhase()
        {
            windowsDone = 0;
            phase++;
            if (phase == Phases.Length)
            {
                phase = 0;
                cycle++;
            }
            if (cycle == Cycles)
            {
                finished = true;
                Debug.Log("AnimationBench finished");
                return;
            }
            BeginPhase();
        }

        void BeginPhase()
        {
            var parent = Phases[phase] == "nested" ? nestedRoot : screenRoot;
            var plays = Phases[phase] == "nested" || Phases[phase] == "shared";
            var level = plays ? EffectLevel.Medium : EffectLevel.Low;
            marksRoot.SetActive(Phases[phase] == "marks");
            for (var at = 0; at < avatars.Length; at++)
            {
                avatars[at].transform.SetParent(parent, false);
                avatars[at].Show(art[at], level, false, budget, Time.unscaledTime);
            }
            settledAt = Time.unscaledTime + SettleSeconds;
        }

        void BuildScreen()
        {
            new GameObject("Camera", typeof(Camera)).GetComponent<Camera>().backgroundColor = TokenColor.BackgroundBase;
            var canvas = new GameObject("Screen", typeof(RectTransform), typeof(Canvas), typeof(CanvasScaler)).GetComponent<Canvas>();
            canvas.renderMode = RenderMode.ScreenSpaceOverlay;
            var scaler = canvas.GetComponent<CanvasScaler>();
            scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
            scaler.referenceResolution = new Vector2(1080f, 2340f);
            screenRoot = canvas.transform;
            AddStaticPanels();
            nestedRoot = new GameObject("Animated", typeof(RectTransform), typeof(Canvas)).transform;
            nestedRoot.SetParent(screenRoot, false);
            Stretch((RectTransform)nestedRoot);
            BuildAvatars();
            BuildMarks();
        }

        void BuildMarks()
        {
            marksRoot = new GameObject("Marks", typeof(RectTransform));
            marksRoot.transform.SetParent(screenRoot, false);
            Stretch((RectTransform)marksRoot.transform);
            var table = RarityTable.Parse(rarityDefaults.text);
            var overlays = new Sprite[PipOverlayCount];
            for (var pips = 0; pips < overlays.Length; pips++) overlays[pips] = MarkSprite(pips + 1);
            for (var at = 0; at < atlases.Length; at++) AddDiamond(at, table, overlays);
            for (var at = 0; at < HaloCount; at++) AddHalo(at, table);
        }

        void AddDiamond(int at, RarityTable table, Sprite[] overlays)
        {
            var fill = NewMarkImage("Diamond", marksRoot.transform, MarkSprite(0));
            Place(fill.rectTransform, at % ColumnCount, at / ColumnCount, ColumnCount, atlases.Length / ColumnCount, DiamondSize);
            var pips = NewMarkImage("Pips", fill.transform, overlays[0]);
            Stretch(pips.rectTransform);
            var diamond = fill.gameObject.AddComponent<RarityDiamond>();
            diamond.Configure(fill, pips, overlays);
            diamond.Show(table, DiamondTiers[at % DiamondTiers.Length]);
        }

        void AddHalo(int at, RarityTable table)
        {
            var image = NewMarkImage("Halo", marksRoot.transform, MarkSprite(HaloCell));
            Place(image.rectTransform, at % ColumnCount, at / ColumnCount, ColumnCount, atlases.Length / ColumnCount, AvatarSize);
            var halo = image.gameObject.AddComponent<RarityHalo>();
            halo.Configure(MarkSprite(HaloCell), MarkSprite(FlowHaloCell));
            halo.Show(table, HaloTier, EffectLevel.Medium, false, Time.unscaledTime);
        }

        static Image NewMarkImage(string name, Transform parent, Sprite sprite)
        {
            var image = new GameObject(name, typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            image.raycastTarget = false;
            image.sprite = sprite;
            image.transform.SetParent(parent, false);
            return image;
        }

        Sprite MarkSprite(int cell)
        {
            var rect = new Rect(cell % MarkColumns * cellSize, cell / MarkColumns * cellSize, cellSize, cellSize);
            return Sprite.Create(marksAtlas, rect, new Vector2(0.5f, 0.5f), 100f, 0, SpriteMeshType.FullRect);
        }

        void AddStaticPanels()
        {
            var white = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);
            Color panelColor = TokenColor.SurfaceDefault;
            panelColor.a = PanelAlpha;
            for (var at = 0; at < PanelCount; at++)
            {
                var image = new GameObject("Panel", typeof(RectTransform), typeof(Image)).GetComponent<Image>();
                image.sprite = white;
                image.raycastTarget = false;
                image.color = panelColor;
                image.transform.SetParent(screenRoot, false);
                Place(image.rectTransform, at % 4, at / 4, 4, 10, 240f);
            }
        }

        void BuildAvatars()
        {
            avatars = new AnimatedImage[atlases.Length];
            art = new LoopArt[atlases.Length];
            for (var at = 0; at < atlases.Length; at++)
            {
                art[at] = ArtOf(atlases[at]);
                var image = new GameObject("Avatar", typeof(RectTransform), typeof(Image)).GetComponent<Image>();
                image.raycastTarget = false;
                image.transform.SetParent(screenRoot, false);
                Place(image.rectTransform, at % ColumnCount, at / ColumnCount, ColumnCount, atlases.Length / ColumnCount, AvatarSize);
                avatars[at] = image.gameObject.AddComponent<AnimatedImage>();
            }
        }

        LoopArt ArtOf(Texture2D atlas)
        {
            var frames = new Sprite[framesPerAvatar];
            var columns = atlas.width / cellSize;
            for (var at = 0; at < frames.Length; at++)
            {
                var cell = new Rect(at % columns * cellSize, at / columns * cellSize, cellSize, cellSize);
                frames[at] = Sprite.Create(atlas, cell, new Vector2(0.5f, 0.5f), 100f, 0, SpriteMeshType.FullRect);
            }
            return new LoopArt(frames[0], frames, LoopFramesPerSecond);
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
