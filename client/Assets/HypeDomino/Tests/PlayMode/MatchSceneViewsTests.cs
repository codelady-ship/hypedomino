using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Match;
using HypeDomino.UI.Motion;
using Hypedomino.V1;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class MatchSceneViewsTests
    {
        const ulong ServerNowMs = 1_000_000;
        const string FoxStatic = "cosmetic/avatar/fox/static";
        const string NeonStatic = "cosmetic/table/neon/static";
        const string Rarities = @"{""diamondMinSize"":72,""flowPeriodMs"":2000,""tiers"":[
{""id"":""common"",""ordinal"":1,""nameKey"":""rarity.common.name"",""fill"":""#0000FF"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":1,""glowAlpha"":0.5,""glowScale"":1.2,""flow"":false}]}";
        const string Catalog = @"{""schemaVersion"":1,""manifestVersion"":1,""items"":[
{""id"":""avatar.fox"",""type"":""avatar"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/fox/static""},
{""id"":""table.neon"",""type"":""table"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/table/neon/static""}]}";

        sealed class ServerFrames : IFrameTransport
        {
            readonly Queue<byte[]> pending = new();

            public void Send(ReadOnlySpan<byte> frame)
            {
            }

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                var has = pending.Count > 0;
                frame = has ? pending.Dequeue() : null;
                return has;
            }

            public void Deliver(Envelope.PayloadOneofCase payload, IMessage message) =>
                pending.Enqueue(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, 1), payload, message).ToArray());
        }

        GameObject root;
        MatchLayoutView layout;
        MatchHudViewTests.HudParts hud;
        MatchSceneViews views;
        TextAsset rarities;
        FakeArtSource<Sprite> stills;
        ServerFrames frames;
        Connection connection;
        AppFlow flow;
        MatchHost host;
        ActiveCosmeticManifest manifest;
        GameStrings strings;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Match", typeof(RectTransform), typeof(Canvas));
            ((RectTransform)root.transform).sizeDelta = new Vector2(1920f, 1080f);
            layout = NewChild<MatchLayoutView>("Layout");
            ((RectTransform)layout.transform).sizeDelta = new Vector2(1920f, 1080f);
            var tiles = NewChild<BoardTilesView>("Board");
            hud = MatchHudViewTests.NewHud(root.transform, 1);
            tiles.Configure(hud.Faces, Array.Empty<Image>(), null, Array.Empty<Image>());
            var seats = new AvatarView[TableSides.Count];
            for (var side = 0; side < seats.Length; side++)
            {
                seats[side] = NewChild<AvatarView>("Seat " + side);
                seats[side].Configure(NewArt("Avatar", seats[side].transform), NewArt("Frame", seats[side].transform), null, null);
            }
            var loading = NewChild<LoadingScreenView>("Loading");
            loading.Configure(Array.Empty<LoadingCardView>(), NewImage("Total", loading.transform), NewTable(loading.transform));
            rarities = new TextAsset(Rarities);
            views = NewChild<MatchSceneViews>("Views");
            views.Configure(layout, NewScorePanel(), tiles, seats, NewTable(root.transform), loading, hud.View, rarities, NewSprite(), NewSprite());
            stills = new FakeArtSource<Sprite>();
            frames = new ServerFrames();
            connection = new Connection(() => throw new InvalidOperationException(), () => new ClientHello(), () => 0d, new System.Random(1));
            flow = new AppFlow();
            connection.Session.HelloReceived += flow.Hello;
            connection.Session.ViewReceived += FollowView;
            Assert.IsTrue(CosmeticManifest.TryParse(Catalog, out var parsed));
            manifest = new ActiveCosmeticManifest(parsed);
            host = new MatchHost(flow, connection, manifest);
            strings = new GameStrings(BuiltinStrings.Load(), LanguageChoice.Load(GameLanguage.English, () => Array.Empty<string>()), false);
            connection.Session.Hello(frames, new ClientHello());
            frames.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = 1, ServerTimeUnixMs = ServerNowMs });
            connection.Session.Poll();
        }

        [TearDown]
        public void Destroy()
        {
            views.Detach();
            Object.DestroyImmediate(root);
            Object.DestroyImmediate(hud.Faces);
            Object.DestroyImmediate(rarities);
        }

        void FollowView(MatchSnapshot view) => flow.View(view.State);

        void Attach() => views.Attach(host, connection, manifest, strings, stills, new FakeArtSource<LoopArt>(), new GraphicsQuality(DeviceTier.Mid, null, 0, false), new MotionPreference(null, false));

        T NewChild<T>(string name) where T : Component
        {
            var child = new GameObject(name, typeof(RectTransform)).AddComponent<T>();
            child.transform.SetParent(root.transform, false);
            return child;
        }

        static Sprite NewSprite() => Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);

        static Image NewImage(string name, Transform parent)
        {
            var image = new GameObject(name, typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            image.transform.SetParent(parent, false);
            image.type = Image.Type.Filled;
            return image;
        }

        static TMP_Text NewText(string name, Transform parent)
        {
            var text = new GameObject(name, typeof(RectTransform)).AddComponent<TextMeshProUGUI>();
            text.transform.SetParent(parent, false);
            return text;
        }

        static CosmeticImage NewArt(string name, Transform parent)
        {
            var image = new GameObject(name, typeof(RectTransform), typeof(Image)).AddComponent<AnimatedImage>().gameObject.AddComponent<CosmeticImage>();
            image.transform.SetParent(parent, false);
            return image;
        }

        static TableView NewTable(Transform parent)
        {
            var built = new GameObject("Table", typeof(RectTransform)).AddComponent<TableView>();
            built.transform.SetParent(parent, false);
            var layer = new GameObject("Layer", typeof(RectTransform), typeof(Image)).AddComponent<AnimatedImage>();
            layer.transform.SetParent(built.transform, false);
            built.Configure(NewImage("Flat", built.transform), new[] { layer }, NewSprite());
            return built;
        }

        ScorePanelView NewScorePanel()
        {
            var panel = NewChild<ScorePanelView>("Score");
            panel.Configure(NewTeam("Mine", panel.transform), NewTeam("Theirs", panel.transform), NewText("Target", panel.transform));
            return panel;
        }

        static TeamScoreView NewTeam(string name, Transform parent)
        {
            var team = new GameObject(name, typeof(RectTransform)).AddComponent<TeamScoreView>();
            team.transform.SetParent(parent, false);
            var badge = new GameObject("Lock", typeof(RectTransform));
            badge.transform.SetParent(team.transform, false);
            team.Configure(NewText("Score", team.transform), NewImage("Gem", team.transform), NewImage("Meter", team.transform), badge);
            return team;
        }

        static MatchView View(MatchState state, Mode mode)
        {
            var two = mode == Mode._1V1;
            var view = new MatchView
            {
                State = state,
                Mode = mode,
                Seat = two ? Seat.X : Seat.South,
                Current = two ? Seat.X : Seat.South,
                Table = "table.neon",
                TurnDeadlineUnixMs = ServerNowMs + 10_000,
                TurnSeconds = 15,
                TurnWarningSeconds = 5,
            };
            view.Roster.Add(new RosterEntry { Seat = two ? Seat.Y : Seat.North, Avatar = "avatar.fox" });
            view.Hand.Add(new Hypedomino.V1.Tile { Low = 2, High = 5 });
            return view;
        }

        void Receive(MatchView view)
        {
            frames.Deliver(Envelope.PayloadOneofCase.MatchView, view);
            connection.Session.Poll();
        }

        [Test]
        public void TheViewsAreNotReadyUntilAttached()
        {
            Assert.IsFalse(views.Ready);

            Attach();

            Assert.IsTrue(views.Ready);
            Assert.IsNotNull(views.Table);
            Assert.IsNotNull(views.LoadingScreen);
            Assert.IsNotNull(views.Stills);
            Assert.IsNotNull(views.Emoji);
        }

        [Test]
        public void AttachedViewsLetTheHostBuildTheMatchAndShowTheTableAndSeats()
        {
            Attach();

            Receive(View(MatchState.Playing, Mode._1V1));

            Assert.IsTrue(host.Live);
            CollectionAssert.Contains(stills.Loads, NeonStatic);
            CollectionAssert.Contains(stills.Loads, FoxStatic);
        }

        [Test]
        public void TheLayoutTakesTheSeatCountOfTheMode()
        {
            Attach();

            Receive(View(MatchState.Playing, Mode._1V1));
            Assert.AreEqual(2, layout.SeatCount);

            Receive(View(MatchState.Playing, Mode._2V2));
            Assert.AreEqual(4, layout.SeatCount);
        }

        [Test]
        public void TheHudShowsTheHandOfTheHostedMatch()
        {
            Attach();

            Receive(View(MatchState.Playing, Mode._2V2));
            hud.View.Refresh();

            Assert.IsTrue(hud.Tiles[0].gameObject.activeSelf);
            Assert.AreSame(hud.Faces.Face(new Domino(2, 5)), hud.Tiles[0].image.sprite);
        }

        [Test]
        public void DetachingTearsTheMatchDown()
        {
            Attach();
            Receive(View(MatchState.Playing, Mode._1V1));

            views.Detach();
            Receive(View(MatchState.Playing, Mode._1V1));

            Assert.IsFalse(host.Live);
            Assert.IsFalse(views.Ready);
        }
    }
}
