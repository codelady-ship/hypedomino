using System;
using HypeDomino.Content;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Emoji;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Match;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Rarity;
using UnityEngine;

namespace HypeDomino.Game.Match
{
    public sealed class MatchSceneViews : MonoBehaviour, IMatchViews
    {
        public const float NoTileSizeCap = float.MaxValue;
        const int NoEmojiSlots = 0;

        [SerializeField] MatchLayoutView layout;
        [SerializeField] ScorePanelView score;
        [SerializeField] BoardTilesView tiles;
        [SerializeField] AvatarView[] seats = new AvatarView[TableSides.Count];
        [SerializeField] TableView table;
        [SerializeField] LoadingScreenView loadingScreen;
        [SerializeField] MatchHudView hud;
        [SerializeField] TextAsset rarities;
        [SerializeField] Sprite builtInAvatar;
        [SerializeField] Sprite builtInFrame;

        MatchHost host;
        CosmeticStage stage;
        CosmeticEffectsLink effects;
        MatchTable matchTable;
        LoadingScreen loading;

        public bool Ready => host != null;
        public IMatchTable Table => matchTable;
        public ILoadingScreen LoadingScreen => loading;
        public CosmeticArtCache<Sprite> Stills => stage?.Stills;
        public MatchEmoji Emoji { get; private set; }

        public void Configure(MatchLayoutView layoutView, ScorePanelView scorePanel, BoardTilesView boardTiles, AvatarView[] seatAvatars, TableView tableView,
            LoadingScreenView loadingView, MatchHudView hudView, TextAsset rarityTable, Sprite avatar, Sprite frame)
        {
            Detach();
            layout = layoutView;
            score = scorePanel;
            tiles = boardTiles;
            seats = seatAvatars;
            table = tableView;
            loadingScreen = loadingView;
            hud = hudView;
            rarities = rarityTable;
            builtInAvatar = avatar;
            builtInFrame = frame;
        }

        public void Attach(MatchHost matchHost, Connection connection, ActiveCosmeticManifest manifest, GameStrings strings,
            ICosmeticArtSource<Sprite> stills, ICosmeticArtSource<LoopArt> loops, GraphicsQuality graphics, MotionPreference motion)
        {
            if (matchHost == null) throw new ArgumentNullException(nameof(matchHost));
            Detach();
            stage = new CosmeticStage(stills, loops, RarityTable.Parse(rarities.text), builtInAvatar, builtInFrame, graphics.Effects, motion.ReduceMotion);
            matchTable = new MatchTable(new SeatAvatars(stage, seats), new TableBackground(stage, table), manifest, score, tiles, layout, NoTileSizeCap);
            loading = new LoadingScreen(loadingScreen, stage, manifest, strings);
            Emoji = new MatchEmoji(new EmojiWheel(NoEmojiSlots), new SeatMessages(), strings, new EmojiArtShelf(manifest.Current, stage, ShowEmoji), ClockMs);
            effects = new CosmeticEffectsLink(stage, graphics, motion, RefreshEffects);
            host = matchHost;
            hud.Bind(host, connection, strings);
            host.Attach(this);
        }

        public void Detach()
        {
            if (host == null) return;
            host.Detach(this);
            hud.Unbind();
            effects.Dispose();
            host = null;
        }

        void Start()
        {
            if (host != null || MatchBoot.Host == null) return;
            Attach(MatchBoot.Host, NetBoot.Connection, ContentBoot.Cosmetics, LanguageBoot.Strings,
                new AddressablesArtSource<Sprite>(), new StillsOnly(), GraphicsBoot.Quality, MotionBoot.Motion);
        }

        void OnDestroy() => Detach();

        void RefreshEffects()
        {
            matchTable.RefreshEffects();
            loading.RefreshEffects();
        }

        static void ShowEmoji(TableSide side, LoopArt art)
        {
        }

        static long ClockMs() => (long)(Time.realtimeSinceStartupAsDouble * 1000d);

        sealed class StillsOnly : ICosmeticArtSource<LoopArt>
        {
            public void Load(string address, Action<LoopArt> done) => done(null);

            public void Release(string address)
            {
            }
        }
    }
}
