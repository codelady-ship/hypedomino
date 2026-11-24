using System;
using Google.Protobuf;
using HypeDomino.Game;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Match;
using HypeDomino.UI.Tokens;
using Hypedomino.V1;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class MatchHudViewTests
    {
        const ulong ServerNowMs = 1_000_000;
        const int Slots = 4;
        const string Catalog = @"{""schemaVersion"":1,""manifestVersion"":1,""items"":[{""id"":""avatar.fox"",""type"":""avatar"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/fox/static""}]}";

        public sealed class HudParts
        {
            public MatchHudView View;
            public TileFaceSet Faces;
            public Button[] Tiles;
            public GameObject[] Marks;
            public Button Draw;
            public Button Pass;
            public Button EndTurn;
            public Image Ring;
            public TMP_Text Network;
            public TMP_Text Afk;
            public Button TakeBack;
            public TMP_Text TakeBackText;
            public TMP_Text[] Badges;
            public CanvasGroup[] PassLabels;
        }

        GameObject root;
        HudParts hud;
        Connection connection;
        MatchHost host;
        GameStrings strings;
        readonly MatchSnapshot snapshot = new();

        [SetUp]
        public void Build()
        {
            root = new GameObject("Match", typeof(RectTransform), typeof(Canvas));
            hud = NewHud(root.transform, Slots);
            connection = new Connection(() => throw new InvalidOperationException(), () => new ClientHello(), () => 0d, new System.Random(1));
            Assert.IsTrue(CosmeticManifest.TryParse(Catalog, out var manifest));
            host = new MatchHost(new AppFlow(), connection, new ActiveCosmeticManifest(manifest));
            strings = new GameStrings(BuiltinStrings.Load(), LanguageChoice.Load(GameLanguage.English, () => Array.Empty<string>()), false);
            hud.View.Bind(host, connection, strings);
        }

        [TearDown]
        public void Destroy()
        {
            Object.DestroyImmediate(root);
            Object.DestroyImmediate(hud.Faces);
        }

        public static HudParts NewHud(Transform parent, int slots)
        {
            var parts = new HudParts
            {
                View = new GameObject("Hud", typeof(RectTransform)).AddComponent<MatchHudView>(),
                Faces = ScriptableObject.CreateInstance<TileFaceSet>(),
                Tiles = new Button[slots],
                Marks = new GameObject[slots],
                Badges = new TMP_Text[TableSides.Count],
                PassLabels = new CanvasGroup[TableSides.Count],
            };
            var hudTransform = parts.View.transform;
            hudTransform.SetParent(parent, false);
            var faces = new Sprite[TileFaces.SetSize];
            for (var at = 0; at < faces.Length; at++) faces[at] = NewSprite();
            parts.Faces.SetFaces(faces);
            for (var at = 0; at < slots; at++)
            {
                parts.Tiles[at] = NewButton("Tile " + at, hudTransform);
                parts.Marks[at] = new GameObject("Legal " + at, typeof(RectTransform));
                parts.Marks[at].transform.SetParent(parts.Tiles[at].transform, false);
            }
            parts.Draw = NewButton("Draw", hudTransform);
            parts.Pass = NewButton("Pass", hudTransform);
            parts.EndTurn = NewButton("End turn", hudTransform);
            parts.Ring = new GameObject("Ring", typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            parts.Ring.transform.SetParent(hudTransform, false);
            parts.Ring.type = Image.Type.Filled;
            parts.Network = NewText("Network", hudTransform);
            parts.Afk = NewText("Afk", hudTransform);
            parts.TakeBack = NewButton("Take back", hudTransform);
            parts.TakeBackText = NewText("Take back text", parts.TakeBack.transform);
            for (var side = 0; side < TableSides.Count; side++)
            {
                parts.Badges[side] = NewText("Badge " + side, hudTransform);
                parts.PassLabels[side] = new GameObject("Pass " + side, typeof(RectTransform)).AddComponent<CanvasGroup>();
                parts.PassLabels[side].transform.SetParent(hudTransform, false);
            }
            parts.View.Configure(parts.Faces, parts.Tiles, parts.Marks, parts.Draw, parts.Pass, parts.EndTurn, parts.Ring,
                parts.Network, parts.Afk, parts.TakeBack, parts.TakeBackText, parts.Badges, parts.PassLabels);
            return parts;
        }

        static Sprite NewSprite() => Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);

        static Button NewButton(string name, Transform parent)
        {
            var button = new GameObject(name, typeof(RectTransform), typeof(Image)).AddComponent<Button>();
            button.transform.SetParent(parent, false);
            button.targetGraphic = button.GetComponent<Image>();
            return button;
        }

        static TMP_Text NewText(string name, Transform parent)
        {
            var text = new GameObject(name, typeof(RectTransform)).AddComponent<TextMeshProUGUI>();
            text.transform.SetParent(parent, false);
            return text;
        }

        static Hypedomino.V1.Tile T(uint low, uint high) => new() { Low = low, High = high };

        static MatchView Playing(params LegalMove[] legal)
        {
            var view = new MatchView
            {
                State = MatchState.Playing,
                Seat = Seat.South,
                Mode = Mode._2V2,
                Current = Seat.South,
                TurnDeadlineUnixMs = ServerNowMs + 10_000,
                TurnSeconds = 15,
                TurnWarningSeconds = 5,
            };
            view.Hand.Add(new[] { T(1, 2), T(3, 3), T(4, 6) });
            view.LegalMoves.Add(legal);
            return view;
        }

        MatchSnapshot Read(MatchView view)
        {
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        HandChoice ShowHand(MatchView view)
        {
            var hand = new HandChoice();
            hand.Read(Read(view));
            host.Hud.ShowHand(hand);
            hud.View.Refresh();
            return hand;
        }

        TurnTimer RunTimer(double nowSeconds)
        {
            var timer = new TurnTimer();
            timer.Read(Read(Playing()));
            connection.Clock.Sync(ServerNowMs, 0d, 0d);
            timer.Tick(connection.Clock, nowSeconds);
            host.Hud.ShowTurn(timer);
            hud.View.Refresh();
            return timer;
        }

        void ShowNotices(MatchView view)
        {
            var notices = new MatchNotices();
            notices.Read(Read(view));
            host.Hud.ShowNotices(notices);
            hud.View.Refresh();
        }

        Sprite FaceOf(byte first, byte second) => hud.Faces.Face(new Domino(first, second));

        [Test]
        public void TheHandShowsEachTileFaceAndMarksOnlyTheLegalOnes()
        {
            ShowHand(Playing(new LegalMove { Tile = T(3, 3), End = BoardEnd.Right }));

            for (var at = 0; at < 3; at++) Assert.IsTrue(hud.Tiles[at].gameObject.activeSelf);
            Assert.IsFalse(hud.Tiles[3].gameObject.activeSelf);
            Assert.AreSame(FaceOf(1, 2), hud.Tiles[0].image.sprite);
            Assert.AreSame(FaceOf(3, 3), hud.Tiles[1].image.sprite);
            Assert.AreSame(FaceOf(4, 6), hud.Tiles[2].image.sprite);
            CollectionAssert.AreEqual(new[] { false, true, false, false }, Array.ConvertAll(hud.Marks, mark => mark.activeSelf));
            CollectionAssert.AreEqual(new[] { false, true, false, false }, Array.ConvertAll(hud.Tiles, tile => tile.interactable));
            Assert.IsFalse(hud.Pass.gameObject.activeSelf);
            Assert.IsFalse(hud.Draw.gameObject.activeSelf);
            Assert.IsFalse(hud.EndTurn.gameObject.activeSelf);
        }

        [Test]
        public void WithoutALegalMoveOnlyThePassButtonShows()
        {
            ShowHand(Playing());

            Assert.IsTrue(hud.Pass.gameObject.activeSelf);
            Assert.IsFalse(hud.Draw.gameObject.activeSelf);
            Assert.IsFalse(hud.EndTurn.gameObject.activeSelf);
            foreach (var mark in hud.Marks) Assert.IsFalse(mark.activeSelf);
        }

        [Test]
        public void AClearedHudHidesTheHandAndItsButtons()
        {
            ShowHand(Playing());

            host.Hud.Clear();
            hud.View.Refresh();

            foreach (var tile in hud.Tiles) Assert.IsFalse(tile.gameObject.activeSelf);
            Assert.IsFalse(hud.Pass.gameObject.activeSelf);
            Assert.IsFalse(hud.Ring.gameObject.activeSelf);
        }

        [Test]
        public void TapsOutsideAMatchSendNothing()
        {
            ShowHand(Playing(new LegalMove { Tile = T(3, 3), End = BoardEnd.Right }));

            hud.Tiles[1].onClick.Invoke();
            hud.Pass.onClick.Invoke();
            hud.TakeBack.onClick.Invoke();

            Assert.IsFalse(host.Live);
            Assert.IsFalse(connection.RoundTrip.HasPending);
        }

        [Test]
        public void TheRingFillsWithTheTimeLeftInGold()
        {
            var timer = RunTimer(0d);

            Assert.IsTrue(hud.Ring.gameObject.activeSelf);
            Assert.AreEqual(timer.Fill, hud.Ring.fillAmount, 1e-5f);
            Assert.AreEqual((Color)TokenColor.GoldDefault, hud.Ring.color);
        }

        [Test]
        public void TheRingTurnsToTheWarningRoleNearTheDeadline()
        {
            RunTimer(6d);

            Assert.AreEqual((Color)TokenColor.RoleWarningFill, hud.Ring.color);
        }

        [Test]
        public void AStoppedTimerHidesTheRing()
        {
            var timer = new TurnTimer();
            host.Hud.ShowTurn(timer);

            hud.View.Refresh();

            Assert.IsFalse(hud.Ring.gameObject.activeSelf);
        }

        [Test]
        public void SeatBadgesShowOnTheirSideAndTheTakeBackBannerShowsForTheOwnSeat()
        {
            var view = Playing();
            view.Seats.Add(new SeatView { Seat = Seat.North, Controller = Controller.Player, Disconnected = true });
            view.Seats.Add(new SeatView { Seat = Seat.South, Controller = Controller.Bot });

            ShowNotices(view);

            Assert.IsTrue(hud.Badges[(int)TableSide.Top].enabled);
            Assert.AreEqual(strings.Text(ConnectionNoticeKeys.Dropped), hud.Badges[(int)TableSide.Top].text);
            Assert.IsTrue(hud.Badges[(int)TableSide.Bottom].enabled);
            Assert.AreEqual(strings.Text(ConnectionNoticeKeys.BotPlaying), hud.Badges[(int)TableSide.Bottom].text);
            Assert.IsFalse(hud.Badges[(int)TableSide.Left].enabled);
            Assert.IsFalse(hud.Badges[(int)TableSide.Right].enabled);
            Assert.IsTrue(hud.TakeBack.gameObject.activeSelf);
            Assert.AreEqual(strings.Text(ConnectionNoticeKeys.TakeBack), hud.TakeBackText.text);
            Assert.IsFalse(hud.Network.enabled);
            Assert.IsFalse(hud.Afk.enabled);
        }

        [Test]
        public void TheAfkWarningShowsWhenTheNextTimeoutHandsTheSeatToTheBot()
        {
            var view = Playing();
            view.AfkConsecutiveTimeouts = 2;
            view.Seats.Add(new SeatView { Seat = Seat.South, Controller = Controller.Player, ConsecutiveTimeouts = 1 });

            ShowNotices(view);

            Assert.IsTrue(hud.Afk.enabled);
            Assert.AreEqual(strings.Text(ConnectionNoticeKeys.OneTimeoutFromBot), hud.Afk.text);
            Assert.IsFalse(hud.TakeBack.gameObject.activeSelf);
        }

        [Test]
        public void APassLabelPopsAtItsSideUntilHidden()
        {
            Assert.IsTrue(PassNotice.At(0f, false, out var notice));

            host.Hud.ShowPass(TableSide.Left, notice);
            hud.View.Refresh();

            var label = hud.PassLabels[(int)TableSide.Left];
            Assert.IsTrue(label.gameObject.activeSelf);
            Assert.AreEqual(notice.Alpha, label.alpha, 1e-5f);
            Assert.AreEqual(notice.Scale, label.transform.localScale.x, 1e-5f);
            Assert.IsFalse(hud.PassLabels[(int)TableSide.Right].gameObject.activeSelf);

            host.Hud.HidePass(TableSide.Left);
            hud.View.Refresh();

            Assert.IsFalse(label.gameObject.activeSelf);
        }

        [Test]
        public void ARefreshWithTheTimerRunningAndNothingElseChangedAllocatesNothing()
        {
            ShowHand(Playing(new LegalMove { Tile = T(3, 3), End = BoardEnd.Right }));
            var timer = RunTimer(0d);
            hud.View.Refresh();

            Assert.That(() =>
            {
                timer.Tick(connection.Clock, 1d);
                hud.View.Refresh();
            }, Allocates.Nothing);
        }

        [Test]
        public void AMovingPassLabelAllocatesNothing()
        {
            Assert.IsTrue(PassNotice.At(0f, false, out var first));
            host.Hud.ShowPass(TableSide.Top, first);
            hud.View.Refresh();
            Assert.IsTrue(PassNotice.At(40f, false, out var next));

            Assert.That(() =>
            {
                host.Hud.ShowPass(TableSide.Top, next);
                hud.View.Refresh();
            }, Allocates.Nothing);
        }
    }
}
