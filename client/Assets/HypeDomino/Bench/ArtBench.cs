using System;
using System.Collections.Generic;
using HypeDomino.Core.Diagnostics;
using HypeDomino.UI.Board;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Effects;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using UnityEngine;
using UnityEngine.EventSystems;
using UnityEngine.UI;

namespace HypeDomino.Bench
{
    public sealed class ArtBench : MonoBehaviour
    {
        const float ReferenceShortSide = 1080f;
        const float ReferenceLongSide = 2340f;
        const float HudHeight = 132f;
        const float Margin = 24f;
        const float Gap = 12f;
        const float LabelHeight = 30f;
        const int LabelFontSize = 22;
        const int HudFontSize = 30;
        const int HudTextFontSize = 26;
        const float NavButtonWidth = 124f;
        const float ToggleWidth = 232f;
        const float StatsIntervalSeconds = 0.5f;
        const int StatsLogEvery = 4;
        const float SwipeMinUnits = 200f;
        const float SmallestTileShortSide = 41.7f;
        const float MedianTileShortSide = 65f;
        const float LargestFrameSide = 256f;
        const int AvatarColumns = 5;
        const int AvatarRows = 3;
        const int FrameColumns = 5;
        const int EmojiColumns = 10;
        const int EmojiRows = 5;
        const int MedalColumns = 10;
        const int MedalRows = 4;
        const int KitColumns = 20;
        const int KitRows = 8;
        const int SoundColumns = 5;
        const int SoundRows = 4;
        const int RateIndexDefault = 1;
        const int TableChainAnchor = 3;
        const float CaptionWidth = 240f;

        static readonly float[] LoopRates = { 8f, 12f };
        static readonly (string name, float aspect, bool belowHud)[] TableShapes = { ("screen", 0f, false), ("under HUD", 0f, true), ("16:9", 16f / 9f, true), ("4:3", 4f / 3f, true) };
        static readonly float[] FrameSides = { TokenSize.TouchTargetMin, TokenSize.SeatPanelHeight, LargestFrameSide };
        static readonly Domino[] TableChain = { new Domino(5, 1), new Domino(1, 3), new Domino(3, 6), new Domino(6, 6), new Domino(6, 4), new Domino(4, 4), new Domino(4, 2) };
        static readonly Vector2 ChainBoard = new Vector2(ReferenceLongSide, ReferenceShortSide / 3f);
        static readonly int AnimatedSlots = Math.Max(TokenEffectLevel.MediumMaxAnimatedCosmetics, TokenEffectLevel.HighMaxAnimatedCosmetics);
        static float AreaHeight => ReferenceShortSide - HudHeight - 2f * Margin;

        sealed class LoopSlot
        {
            public AnimatedImage Image;
            public ArtBenchLoop Source;
            public LoopArt Art;
        }

        sealed class TableSlot
        {
            public RectTransform Area;
            public RectTransform Surface;
            public AspectRatioFitter Fitter;
            public Image Still;
            public readonly List<LoopSlot> Layers = new();
        }

        sealed class Sheet
        {
            public string Section;
            public string Title;
            public RectTransform Root;
            public readonly List<LoopSlot> Loops = new();
            public readonly List<TableSlot> Tables = new();
            public ArtBenchEffect Effect;
        }

        [SerializeField] ArtBenchContent content;

        readonly List<Sheet> sheets = new();
        readonly FrameTiming[] latestTiming = new FrameTiming[1];
        readonly Dictionary<string, Sprite> effectSprites = new();
        readonly Dictionary<string, Material> effectMaterials = new();
        EffectLibrary library;
        Font font;
        Sprite white;
        RectTransform screen;
        RectTransform sheetsRoot;
        Text title;
        Text stats;
        Text levelLabel;
        Text motionLabel;
        Text rateLabel;
        Text shapeLabel;
        int shapeIndex;
        AudioSource sfxSource;
        AudioSource musicSource;
        AnimationBudget budget;
        EffectLevel level = EffectLevel.Medium;
        bool reduceMotion;
        int rateIndex = RateIndexDefault;
        int current;
        int statFrames;
        float statSeconds;
        float statWorst;
        double workSum;
        double workWorst;
        double mainSum;
        double gpuSum;
        int timedFrames;
        int statReports;
        Vector2 touchStart;

        public void Configure(ArtBenchContent art) => content = art;

        void Awake()
        {
            Screen.orientation = ScreenOrientation.LandscapeLeft;
            Screen.sleepTimeout = SleepTimeout.NeverSleep;
            Application.SetStackTraceLogType(LogType.Log, StackTraceLogType.None);
            QualitySettings.vSyncCount = 0;
            Application.targetFrameRate = TokenGraphics.TargetFpsOfEffectLevel[(int)level];
            font = Resources.GetBuiltinResource<Font>("LegacyRuntime.ttf");
            white = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);
            budget = AnimationBudget.For(level);
            sfxSource = gameObject.AddComponent<AudioSource>();
            musicSource = gameObject.AddComponent<AudioSource>();
            musicSource.loop = true;
        }

        void Start()
        {
            var camera = new GameObject("Camera", typeof(Camera)).GetComponent<Camera>();
            camera.clearFlags = CameraClearFlags.SolidColor;
            camera.backgroundColor = TokenColor.BackgroundBase;
            new GameObject("EventSystem", typeof(EventSystem), typeof(StandaloneInputModule));
            BuildScreen();
            IndexEffects();
            BuildAvatarSheets();
            BuildFrameSheets();
            BuildTableSheets();
            BuildEmojiSheets();
            BuildMedalSheets();
            BuildTileSheets();
            BuildEffectSpriteSheet();
            BuildEffectSheets();
            BuildKitSheet();
            BuildSoundSheet();
            BuildHud();
            LogClips();
            Open(0);
        }

        void Update()
        {
            TrackStats();
            TrackSwipe();
        }

        void BuildScreen()
        {
            var canvas = new GameObject("Screen", typeof(RectTransform), typeof(Canvas), typeof(CanvasScaler), typeof(GraphicRaycaster)).GetComponent<Canvas>();
            canvas.renderMode = RenderMode.ScreenSpaceOverlay;
            var scaler = canvas.GetComponent<CanvasScaler>();
            scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
            scaler.referenceResolution = new Vector2(ReferenceLongSide, ReferenceShortSide);
            scaler.matchWidthOrHeight = 1f;
            screen = (RectTransform)canvas.transform;
            sheetsRoot = NewRect("Sheets", screen);
            Stretch(sheetsRoot);
        }

        void BuildAvatarSheets()
        {
            var side = CellArtSide(AvatarRows);
            foreach (var chunk in Chunks(content.avatars, AvatarColumns * AvatarRows, 1))
            {
                var sheet = NewSheet("avatars", $"avatars {chunk[0].name}..");
                var area = Area(sheet.Root);
                for (var at = 0; at < chunk.Count; at++)
                {
                    var cell = Cell(area, at, AvatarColumns, AvatarRows);
                    AddLoop(sheet, cell, chunk[at], side);
                    var frame = content.frames.Length > 0 ? content.frames[at % content.frames.Length] : null;
                    if (frame != null) NewImage("Frame", cell, frame.still, side, CellCentre);
                    Label(cell, chunk[at].IsAnimated ? $"{chunk[at].name} (anim)" : chunk[at].name);
                }
            }
        }

        void BuildFrameSheets()
        {
            var avatar = FirstStill(content.avatars);
            foreach (var chunk in Chunks(content.frames, FrameColumns, FrameSides.Length))
            {
                var sheet = NewSheet("frames", $"frames {chunk[0].name}..");
                var area = Area(sheet.Root);
                for (var at = 0; at < chunk.Count; at++)
                {
                    var column = Cell(area, at, FrameColumns, 1);
                    var top = 0f;
                    foreach (var side in FrameSides)
                    {
                        var anchor = new Vector2(0.5f, 1f);
                        var offset = new Vector2(0f, -top - side / 2f);
                        if (avatar != null) NewImage("Avatar", column, avatar.still, side, anchor).rectTransform.anchoredPosition = offset;
                        AddLoop(sheet, column, chunk[at], side, anchor).anchoredPosition = offset;
                        top += side + Gap;
                    }
                    Label(column, chunk[at].IsAnimated ? $"{chunk[at].name} (anim)" : chunk[at].name);
                }
            }
        }

        void BuildTableSheets()
        {
            foreach (var table in content.tables)
            {
                var sheet = NewSheet("tables", table.layers.Length > 0 ? $"table {table.name} ({table.layers.Length} layers)" : $"table {table.name}");
                var area = NewRect("TableArea", sheet.Root);
                Stretch(area);
                var surface = NewRect("Table", area);
                Stretch(surface);
                var fitter = surface.gameObject.AddComponent<AspectRatioFitter>();
                fitter.aspectMode = AspectRatioFitter.AspectMode.FitInParent;
                fitter.enabled = false;
                var slot = new TableSlot { Area = area, Surface = surface, Fitter = fitter, Still = NewTable("Static", surface, table.still) };
                foreach (var layer in table.layers)
                {
                    var image = NewTable(layer.name, surface, layer.still);
                    image.gameObject.SetActive(false);
                    slot.Layers.Add(new LoopSlot { Image = image.gameObject.AddComponent<AnimatedImage>(), Source = layer, Art = ArtOf(layer) });
                }
                sheet.Tables.Add(slot);
                AddTileChain(sheet.Root, SmallestTileShortSide, -60f, "smallest 41.7");
                AddTileChain(sheet.Root, MedianTileShortSide, 120f, "median 65");
            }
        }

        void AddTileChain(RectTransform parent, float shortSide, float y, string caption)
        {
            if (content.tileFaceSet == null) return;
            var layout = new ChainLayout(TableChain.Length);
            layout.Arrange(TableChain, TableChainAnchor, Array.Empty<Domino>(), Array.Empty<Domino>(), ChainBoard, shortSide);
            var row = NewRect(caption, parent);
            row.anchoredPosition = new Vector2(0f, y);
            var shadows = new Image[TableChain.Length];
            for (var at = 0; at < shadows.Length; at++) shadows[at] = NewImage("TileShadow", row, null, 0f, CellCentre);
            var tiles = new Image[TableChain.Length];
            for (var at = 0; at < tiles.Length; at++) tiles[at] = NewImage("Tile", row, null, 0f, CellCentre);
            var view = row.gameObject.AddComponent<BoardTilesView>();
            view.Configure(content.tileFaceSet, tiles, TileShadowSprite(), shadows);
            view.Show(layout);
            var label = NewText(caption, row, TokenColor.TextPrimary, LabelFontSize);
            label.alignment = TextAnchor.MiddleRight;
            label.text = caption;
            label.rectTransform.sizeDelta = new Vector2(CaptionWidth, LabelHeight);
            label.rectTransform.anchoredPosition = new Vector2(-(layout.Bounds.width * layout.Scale + CaptionWidth) / 2f - Gap, 0f);
        }

        Sprite TileShadowSprite() => Array.Find(content.effectSprites, sprite => sprite != null && sprite.name == "tile_shadow");

        void BuildEmojiSheets()
        {
            var side = CellArtSide(EmojiRows);
            foreach (var chunk in Chunks(content.emoji, EmojiColumns * EmojiRows, 1))
            {
                var sheet = NewSheet("emoji", $"emoji {chunk[0].name}..");
                var area = Area(sheet.Root);
                for (var at = 0; at < chunk.Count; at++)
                {
                    var cell = Cell(area, at, EmojiColumns, EmojiRows);
                    AddLoop(sheet, cell, chunk[at], side);
                    Label(cell, chunk[at].name);
                }
            }
        }

        void BuildMedalSheets()
        {
            AddMedalSheet("medals on background.base at 120", TokenColor.BackgroundBase);
            AddMedalSheet("medals on surface.default at 120", TokenColor.SurfaceDefault);
        }

        void AddMedalSheet(string caption, Color32 background)
        {
            var sheet = NewSheet("medals", caption);
            var backdrop = NewImage("Backdrop", sheet.Root, white, 0f, CellCentre);
            Stretch(backdrop.rectTransform);
            backdrop.preserveAspect = false;
            backdrop.color = background;
            var area = Area(sheet.Root);
            for (var at = 0; at < content.medals.Length && at < MedalColumns * MedalRows; at++)
            {
                var cell = Cell(area, at, MedalColumns, MedalRows);
                NewImage(content.medals[at].name, cell, content.medals[at].still, TokenSize.TouchTargetMin, CellCentre);
                Label(cell, content.medals[at].name);
            }
        }

        void BuildTileSheets()
        {
            var sheet = NewSheet("tiles", "tile faces at canvas size (128 x 256 units)");
            var area = Area(sheet.Root);
            var columns = (content.tileFaces.Length + 1) / 2;
            for (var at = 0; at < content.tileFaces.Length; at++)
            {
                var cell = Cell(area, at, columns, 2);
                var face = content.tileFaces[at];
                NewImage(face.name, cell, face, 0f, CellCentre).rectTransform.sizeDelta = face.rect.size;
                Label(cell, face.name);
            }
            if (content.tileBacks.Length == 0) return;
            var backs = NewSheet("tiles", "tile backs at 41.7, 65 and 128 units");
            var backArea = Area(backs.Root);
            var sides = new[] { SmallestTileShortSide, MedianTileShortSide, content.tileBacks[0].still.rect.width };
            for (var at = 0; at < content.tileBacks.Length; at++)
            for (var size = 0; size < sides.Length; size++)
            {
                var back = content.tileBacks[at];
                var cell = Cell(backArea, at * sides.Length + size, sides.Length, content.tileBacks.Length);
                var scale = sides[size] / back.still.rect.width;
                AddLoop(backs, cell, back, 0f).sizeDelta = back.still.rect.size * scale;
                Label(cell, back.IsAnimated ? $"{back.name} {sides[size]} (anim)" : $"{back.name} {sides[size]}");
            }
        }

        void BuildEffectSpriteSheet()
        {
            var sheet = NewSheet("effects", "effect sprites with their material");
            var area = Area(sheet.Root);
            var columns = (content.effectSprites.Length + 1) / 2;
            for (var at = 0; at < content.effectSprites.Length; at++)
            {
                var sprite = content.effectSprites[at];
                var cell = Cell(area, at, columns, 2);
                var image = NewImage(sprite.name, cell, sprite, 0f, CellCentre);
                image.rectTransform.sizeDelta = sprite.rect.size;
                image.material = MaterialOf(sprite.name);
                Label(cell, sprite.name);
            }
        }

        void BuildEffectSheets()
        {
            if (library == null) return;
            var stageSize = new Vector2(ReferenceLongSide / 2f, AreaHeight * 0.75f);
            for (var kind = 0; kind < TokenEffect.NameOfKind.Length; kind++)
            {
                var sheet = NewSheet("effects", $"effect {TokenEffect.NameOfKind[kind]} (EffectPlayer, team A)");
                var stage = NewRect(TokenEffect.NameOfKind[kind], Area(sheet.Root));
                stage.sizeDelta = stageSize;
                sheet.Effect = stage.gameObject.AddComponent<ArtBenchEffect>();
                sheet.Effect.Configure((TokenEffectKind)kind, library, content);
            }
        }

        void BuildKitSheet()
        {
            var sheet = NewSheet("kit", "ui kit sprites");
            var area = Area(sheet.Root);
            var side = AreaHeight / KitRows - Gap;
            for (var at = 0; at < content.kit.Length && at < KitColumns * KitRows; at++)
            {
                var sprite = content.kit[at];
                var image = NewImage(sprite.name, Cell(area, at, KitColumns, KitRows), sprite, 0f, CellCentre);
                var sliced = sprite.border != Vector4.zero;
                image.type = sliced ? Image.Type.Sliced : Image.Type.Simple;
                image.preserveAspect = !sliced;
                image.rectTransform.sizeDelta = sliced ? new Vector2(side, side) : Vector2.Min(sprite.rect.size, new Vector2(side, side));
            }
        }

        void BuildSoundSheet()
        {
            var sheet = NewSheet("sound", "sound: tap to play, music toggles its loop");
            var area = Area(sheet.Root);
            var at = 0;
            foreach (var clip in content.sfx) AddSoundButton(area, at++, clip, false);
            foreach (var clip in content.music) AddSoundButton(area, at++, clip, true);
        }

        void AddSoundButton(RectTransform area, int at, AudioClip clip, bool music)
        {
            var cell = Cell(area, at, SoundColumns, SoundRows);
            var label = music ? $"music {clip.name}" : clip.name;
            var button = NewButton(label, cell, label, () => PlayClip(clip, music));
            Stretch((RectTransform)button.transform);
            ((RectTransform)button.transform).offsetMin = new Vector2(Gap, Gap);
            ((RectTransform)button.transform).offsetMax = new Vector2(-Gap, -Gap);
        }

        void PlayClip(AudioClip clip, bool music)
        {
            if (clip.loadState == AudioDataLoadState.Unloaded) clip.LoadAudioData();
            Debug.Log($"ArtBench play clip={clip.name} music={music} state={clip.loadState} length={clip.length:F2}");
            if (!music)
            {
                sfxSource.PlayOneShot(clip);
                return;
            }
            if (musicSource.isPlaying && musicSource.clip == clip)
            {
                musicSource.Stop();
                return;
            }
            musicSource.clip = clip;
            musicSource.Play();
        }

        void LogClips()
        {
            foreach (var clip in content.sfx) LogClip(clip);
            foreach (var clip in content.music) LogClip(clip);
        }

        static void LogClip(AudioClip clip) =>
            Debug.Log($"ArtBench clip={clip.name} loadType={clip.loadType} state={clip.loadState} length={clip.length:F2} channels={clip.channels} hz={clip.frequency}");

        void BuildHud()
        {
            var hud = NewImage("Hud", screen, white, 0f, CellCentre);
            hud.raycastTarget = true;
            hud.preserveAspect = false;
            Color backdrop = TokenColor.Scrim;
            backdrop.a = TokenOpacity.Scrim;
            hud.color = backdrop;
            var rect = hud.rectTransform;
            rect.anchorMin = new Vector2(0f, 1f);
            rect.anchorMax = Vector2.one;
            rect.pivot = new Vector2(0.5f, 1f);
            rect.sizeDelta = new Vector2(0f, HudHeight);
            rect.anchoredPosition = Vector2.zero;
            AddNav(rect, 0, "<<", () => Open(PreviousSection(current)));
            AddNav(rect, 1, "<", () => Open(current - 1));
            AddNav(rect, 2, ">", () => Open(current + 1));
            AddNav(rect, 3, ">>", () => Open(NextSection(current)));
            levelLabel = AddToggle(rect, 2, NextLevel);
            motionLabel = AddToggle(rect, 1, ToggleMotion);
            rateLabel = AddToggle(rect, 0, NextRate);
            shapeLabel = AddToggle(rect, 3, NextShape);
            var left = 4f * (NavButtonWidth + Gap) + Gap;
            var right = 4f * (ToggleWidth + Gap) + Gap;
            title = HudText(rect, left, right, 0.5f, 1f);
            stats = HudText(rect, left, right, 0f, 0.5f);
            RefreshLabels();
        }

        void AddNav(RectTransform hud, int at, string text, Action press)
        {
            var button = (RectTransform)NewButton(text, hud, text, press).transform;
            button.anchorMin = new Vector2(0f, 0.5f);
            button.anchorMax = new Vector2(0f, 0.5f);
            button.sizeDelta = new Vector2(NavButtonWidth, HudHeight - 2f * Gap);
            button.anchoredPosition = new Vector2(Gap + at * (NavButtonWidth + Gap) + NavButtonWidth / 2f, 0f);
        }

        Text AddToggle(RectTransform hud, int fromRight, Action press)
        {
            var button = NewButton("Toggle", hud, string.Empty, press);
            var rect = (RectTransform)button.transform;
            rect.anchorMin = new Vector2(1f, 0.5f);
            rect.anchorMax = new Vector2(1f, 0.5f);
            rect.sizeDelta = new Vector2(ToggleWidth, HudHeight - 2f * Gap);
            rect.anchoredPosition = new Vector2(-Gap - fromRight * (ToggleWidth + Gap) - ToggleWidth / 2f, 0f);
            return button.GetComponentInChildren<Text>();
        }

        Text HudText(RectTransform hud, float left, float right, float bottom, float top)
        {
            var text = NewText("HudText", hud, TokenColor.TextPrimary, HudTextFontSize);
            text.alignment = TextAnchor.MiddleLeft;
            var rect = text.rectTransform;
            rect.anchorMin = new Vector2(0f, bottom);
            rect.anchorMax = new Vector2(1f, top);
            rect.offsetMin = new Vector2(left, 0f);
            rect.offsetMax = new Vector2(-right, 0f);
            return text;
        }

        void NextLevel()
        {
            level = level == EffectLevel.High ? EffectLevel.Low : level + 1;
            Refresh();
        }

        void ToggleMotion()
        {
            reduceMotion = !reduceMotion;
            Refresh();
        }

        void NextRate()
        {
            rateIndex = (rateIndex + 1) % LoopRates.Length;
            foreach (var sheet in sheets)
            {
                foreach (var loop in sheet.Loops) loop.Art = ArtOf(loop.Source);
                foreach (var table in sheet.Tables)
                foreach (var layer in table.Layers) layer.Art = ArtOf(layer.Source);
            }
            Refresh();
        }

        void NextShape()
        {
            shapeIndex = (shapeIndex + 1) % TableShapes.Length;
            foreach (var sheet in sheets)
            foreach (var table in sheet.Tables) ApplyShape(table);
            Refresh();
        }

        void ApplyShape(TableSlot table)
        {
            var shape = TableShapes[shapeIndex];
            table.Area.offsetMax = new Vector2(0f, shape.belowHud ? -HudHeight : 0f);
            table.Fitter.enabled = shape.aspect > 0f;
            if (shape.aspect > 0f) table.Fitter.aspectRatio = shape.aspect;
            else Stretch(table.Surface);
        }

        void Refresh()
        {
            RefreshLabels();
            Show(sheets[current]);
        }

        void RefreshLabels()
        {
            levelLabel.text = $"effects {level}";
            motionLabel.text = reduceMotion ? "reduce motion on" : "reduce motion off";
            rateLabel.text = $"loops {LoopRates[rateIndex]:F0} fps";
            shapeLabel.text = $"table {TableShapes[shapeIndex].name}";
            if (sheets.Count > 0) title.text = $"{current + 1}/{sheets.Count}  {sheets[current].Title}";
        }

        void Open(int index)
        {
            if (sheets.Count == 0) return;
            sheets[current].Root.gameObject.SetActive(false);
            current = Wrap(index);
            sheets[current].Root.gameObject.SetActive(true);
            Debug.Log($"ArtBench sheet {current + 1}/{sheets.Count} section={sheets[current].Section} title={sheets[current].Title}");
            Refresh();
        }

        int NextSection(int from)
        {
            var at = from;
            do at = Wrap(at + 1);
            while (sheets[at].Section == sheets[from].Section && at != from);
            return at;
        }

        int PreviousSection(int from) => FirstOfSection(Wrap(FirstOfSection(from) - 1));

        int FirstOfSection(int at)
        {
            for (var steps = 0; steps < sheets.Count && sheets[Wrap(at - 1)].Section == sheets[at].Section; steps++) at = Wrap(at - 1);
            return at;
        }

        int Wrap(int at) => (at % sheets.Count + sheets.Count) % sheets.Count;

        void Show(Sheet sheet)
        {
            Application.targetFrameRate = TokenGraphics.TargetFpsOfEffectLevel[(int)level];
            budget = AnimationBudget.For(level);
            var now = Time.unscaledTime;
            foreach (var loop in sheet.Loops) loop.Image.Show(loop.Art, level, reduceMotion, budget, now);
            foreach (var table in sheet.Tables) ShowTable(table, now);
            if (sheet.Effect != null) Debug.Log($"ArtBench effect={sheet.Effect.Kind} level={level} reduceMotion={reduceMotion} {sheet.Effect.Play(level, reduceMotion, now)}");
        }

        void ShowTable(TableSlot table, float now)
        {
            var animate = level != EffectLevel.Low && !reduceMotion && table.Layers.Count > 0 && budget.HasRoomFor(table.Layers.Count);
            table.Still.gameObject.SetActive(!animate);
            foreach (var layer in table.Layers)
            {
                layer.Image.gameObject.SetActive(animate);
                if (animate) layer.Image.Show(layer.Art, level, reduceMotion, budget, now);
            }
        }

        void TrackStats()
        {
            FrameTimingManager.CaptureFrameTimings();
            var delta = Time.unscaledDeltaTime;
            statFrames++;
            statSeconds += delta;
            statWorst = Mathf.Max(statWorst, delta);
            if (FrameTimingManager.GetLatestTimings(1, latestTiming) > 0)
            {
                var work = FrameWindow.WorkMs(latestTiming[0]);
                workSum += work;
                workWorst = Math.Max(workWorst, work);
                mainSum += latestTiming[0].cpuMainThreadFrameTime;
                gpuSum += latestTiming[0].gpuFrameTime;
                timedFrames++;
            }
            if (statSeconds < StatsIntervalSeconds || stats == null) return;
            var line = $"{statFrames / statSeconds:F1} fps  frame {statSeconds * 1000f / statFrames:F1}/{statWorst * 1000f:F1} ms  " +
                       $"work {Average(workSum):F1}/{workWorst:F1} ms  main {Average(mainSum):F1}  gpu {Average(gpuSum):F1}  playing {budget.Active}";
            stats.text = line;
            if (++statReports % StatsLogEvery == 0) Debug.Log($"ArtBench stats sheet={current + 1} level={level} reduceMotion={reduceMotion} {line}");
            statFrames = 0;
            statSeconds = 0f;
            statWorst = 0f;
            workSum = 0;
            workWorst = 0;
            mainSum = 0;
            gpuSum = 0;
            timedFrames = 0;
        }

        double Average(double sum) => timedFrames > 0 ? sum / timedFrames : 0;

        void TrackSwipe()
        {
            if (Input.touchCount != 1) return;
            var touch = Input.GetTouch(0);
            if (touch.phase == TouchPhase.Began) touchStart = touch.position;
            if (touch.phase != TouchPhase.Ended) return;
            var units = ReferenceShortSide / Mathf.Min(Screen.width, Screen.height);
            var travel = (touch.position - touchStart) * units;
            if (Mathf.Abs(travel.x) < SwipeMinUnits || Mathf.Abs(travel.x) < Mathf.Abs(travel.y)) return;
            Open(current + (travel.x < 0f ? 1 : -1));
        }

        void IndexEffects()
        {
            foreach (var sprite in content.effectSprites) effectSprites[sprite.name] = sprite;
            if (content.effectTable == null) return;
            library = EffectLibrary.Parse(content.effectTable.text);
            for (var sprite = 0; sprite < library.SpriteCount; sprite++)
                effectMaterials[library.SpriteName(sprite)] = library.MaterialOf(sprite) == EffectMaterial.Additive ? content.additive : content.alpha;
            Debug.Log($"ArtBench effect sprites={effectSprites.Count} materials={effectMaterials.Count} additive={content.additive} alpha={content.alpha}");
        }

        Sprite EffectSprite(string name) => effectSprites.TryGetValue(name, out var sprite) ? sprite : null;

        Material MaterialOf(string name) => effectMaterials.TryGetValue(name, out var material) ? material : null;

        static ArtBenchLoop FirstStill(ArtBenchLoop[] items)
        {
            foreach (var item in items)
                if (!item.IsAnimated) return item;
            return items.Length > 0 ? items[0] : null;
        }

        static List<List<ArtBenchLoop>> Chunks(ArtBenchLoop[] items, int capacity, int slotsPerItem)
        {
            var animated = new Queue<ArtBenchLoop>();
            var still = new Queue<ArtBenchLoop>();
            foreach (var item in items) (item.IsAnimated ? animated : still).Enqueue(item);
            var animatedPerSheet = Math.Max(1, Math.Min(capacity, AnimatedSlots / slotsPerItem));
            var chunks = new List<List<ArtBenchLoop>>();
            while (animated.Count > 0 || still.Count > 0)
            {
                var chunk = new List<ArtBenchLoop>();
                while (animated.Count > 0 && chunk.Count < animatedPerSheet) chunk.Add(animated.Dequeue());
                while (still.Count > 0 && chunk.Count < capacity) chunk.Add(still.Dequeue());
                chunks.Add(chunk);
            }
            return chunks;
        }

        LoopArt ArtOf(ArtBenchLoop item) => new LoopArt(item.still, item.frames, LoopRates[rateIndex]);

        RectTransform AddLoop(Sheet sheet, RectTransform parent, ArtBenchLoop item, float side) => AddLoop(sheet, parent, item, side, CellCentre);

        RectTransform AddLoop(Sheet sheet, RectTransform parent, ArtBenchLoop item, float side, Vector2 anchor)
        {
            var image = NewImage(item.name, parent, item.still, side, anchor);
            sheet.Loops.Add(new LoopSlot { Image = image.gameObject.AddComponent<AnimatedImage>(), Source = item, Art = ArtOf(item) });
            return image.rectTransform;
        }

        Sheet NewSheet(string section, string caption)
        {
            var root = NewRect(caption, sheetsRoot);
            Stretch(root);
            root.gameObject.SetActive(false);
            var sheet = new Sheet { Section = section, Title = caption, Root = root };
            sheets.Add(sheet);
            return sheet;
        }

        static RectTransform Area(RectTransform root)
        {
            var area = NewRect("Area", root);
            Stretch(area);
            area.offsetMin = new Vector2(Margin, Margin);
            area.offsetMax = new Vector2(-Margin, -HudHeight - Margin);
            return area;
        }

        static RectTransform Cell(RectTransform area, int index, int columns, int rows)
        {
            var cell = NewRect("Cell", area);
            var column = index % columns;
            var row = index / columns;
            cell.anchorMin = new Vector2(column / (float)columns, 1f - (row + 1f) / rows);
            cell.anchorMax = new Vector2((column + 1f) / columns, 1f - row / (float)rows);
            cell.offsetMin = Vector2.zero;
            cell.offsetMax = Vector2.zero;
            return cell;
        }

        static float CellArtSide(int rows) => AreaHeight / rows - LabelHeight - Gap;

        static Vector2 CellCentre => new Vector2(0.5f, 0.5f);

        void Label(RectTransform cell, string text)
        {
            foreach (RectTransform child in cell) child.anchoredPosition += new Vector2(0f, LabelHeight / 2f);
            var label = NewText("Label", cell, TokenColor.TextSecondary, LabelFontSize);
            var rect = label.rectTransform;
            rect.anchorMin = Vector2.zero;
            rect.anchorMax = new Vector2(1f, 0f);
            rect.pivot = new Vector2(0.5f, 0f);
            rect.sizeDelta = new Vector2(0f, LabelHeight);
            rect.anchoredPosition = Vector2.zero;
            label.text = text;
        }

        Image NewImage(string name, Transform parent, Sprite sprite, float side, Vector2 anchor)
        {
            var image = new GameObject(name, typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            image.raycastTarget = false;
            image.sprite = sprite;
            image.preserveAspect = true;
            image.transform.SetParent(parent, false);
            var rect = image.rectTransform;
            rect.anchorMin = anchor;
            rect.anchorMax = anchor;
            rect.sizeDelta = new Vector2(side, side);
            return image;
        }

        Text NewText(string name, Transform parent, Color32 color, int size)
        {
            var text = new GameObject(name, typeof(RectTransform), typeof(Text)).GetComponent<Text>();
            text.font = font;
            text.fontSize = size;
            text.alignment = TextAnchor.MiddleCenter;
            text.horizontalOverflow = HorizontalWrapMode.Overflow;
            text.raycastTarget = false;
            text.color = color;
            text.transform.SetParent(parent, false);
            return text;
        }

        Button NewButton(string name, Transform parent, string text, Action press)
        {
            var image = NewImage(name, parent, white, 0f, CellCentre);
            image.raycastTarget = true;
            image.preserveAspect = false;
            image.color = TokenColor.SurfaceRaised;
            var button = image.gameObject.AddComponent<Button>();
            button.onClick.AddListener(() => press());
            var label = NewText("Label", image.transform, TokenColor.TextPrimary, HudFontSize);
            Stretch(label.rectTransform);
            label.horizontalOverflow = HorizontalWrapMode.Wrap;
            label.text = text;
            return button;
        }

        static TableImage NewTable(string name, Transform parent, Sprite sprite)
        {
            var image = new GameObject(name, typeof(RectTransform), typeof(TableImage)).GetComponent<TableImage>();
            image.raycastTarget = false;
            image.sprite = sprite;
            image.transform.SetParent(parent, false);
            Stretch(image.rectTransform);
            return image;
        }

        static RectTransform NewRect(string name, Transform parent)
        {
            var rect = new GameObject(name, typeof(RectTransform)).GetComponent<RectTransform>();
            rect.SetParent(parent, false);
            return rect;
        }

        static void Stretch(RectTransform rect)
        {
            rect.anchorMin = Vector2.zero;
            rect.anchorMax = Vector2.one;
            rect.offsetMin = Vector2.zero;
            rect.offsetMax = Vector2.zero;
        }
    }
}
