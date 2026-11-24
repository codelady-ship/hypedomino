using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Match;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Rarity;
using HypeDomino.UI.Tokens;
using Hypedomino.V1;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class LoadingScreenTests
    {
        const string FoxStatic = "cosmetic/avatar/fox/static";
        const string NeonStatic = "cosmetic/table/neon/static";
        const string Rarities = @"{""diamondMinSize"":72,""flowPeriodMs"":2000,""tiers"":[
{""id"":""common"",""ordinal"":1,""nameKey"":""rarity.common.name"",""fill"":""#0000FF"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":1,""glowAlpha"":0.5,""glowScale"":1.2,""flow"":false}]}";
        const string Catalog = @"{""schemaVersion"":1,""manifestVersion"":1,""items"":[
{""id"":""avatar.fox"",""type"":""avatar"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/fox/static""},
{""id"":""table.neon"",""type"":""table"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/table/neon/static""}]}";
        const string Strings = @"{""entries"":[
{""key"":""match.loading.ready"",""en"":""Ready""},
{""key"":""match.loading.bot_playing"",""en"":""Couldn't connect, bot playing""}]}";

        sealed class CardParts
        {
            public LoadingCardView View;
            public AvatarView Avatar;
            public TMP_Text Nickname;
            public Image Progress;
            public TMP_Text Status;
        }

        GameObject root;
        FakeArtSource<Sprite> stills;
        CardParts[] parts;
        LoadingScreenView view;
        Image total;
        TableView table;
        LoadingScreen screen;
        readonly LoadingCards cards = new();
        readonly MatchSnapshot snapshot = new();

        [SetUp]
        public void Build()
        {
            root = new GameObject("Screen", typeof(RectTransform), typeof(Canvas));
            stills = new FakeArtSource<Sprite>();
            Assert.IsTrue(CosmeticManifest.TryParse(Catalog, out var manifest));
            var stage = new CosmeticStage(stills, new FakeArtSource<LoopArt>(), RarityTable.Parse(Rarities), NewSprite(), NewSprite(), EffectLevel.Medium, false);
            view = new GameObject("Loading", typeof(RectTransform)).AddComponent<LoadingScreenView>();
            view.transform.SetParent(root.transform, false);
            parts = new CardParts[LoadingScreenView.Rows * LoadingScreenView.CardsPerRow];
            var views = new LoadingCardView[parts.Length];
            for (var at = 0; at < parts.Length; at++)
            {
                parts[at] = NewCard("Card " + at, view.transform);
                views[at] = parts[at].View;
            }
            total = NewImage("Total", view.transform);
            table = NewTable(view.transform);
            view.Configure(views, total, table);
            var strings = new GameStrings(BuiltinStrings.Parse(Strings), LanguageChoice.Load("en", () => new string[0]), false);
            screen = new LoadingScreen(view, stage, new ActiveCosmeticManifest(manifest), strings);
        }

        [TearDown]
        public void DestroyScreen() => Object.Destroy(root);

        static Sprite NewSprite() => Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);

        static TMP_Text NewText(string name, Transform parent)
        {
            var text = new GameObject(name, typeof(RectTransform)).AddComponent<TextMeshProUGUI>();
            text.transform.SetParent(parent, false);
            return text;
        }

        static Image NewImage(string name, Transform parent)
        {
            var image = new GameObject(name, typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            image.transform.SetParent(parent, false);
            image.type = Image.Type.Filled;
            return image;
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
            var layers = new AnimatedImage[2];
            for (var at = 0; at < layers.Length; at++)
            {
                layers[at] = new GameObject("Layer", typeof(RectTransform), typeof(Image)).AddComponent<AnimatedImage>();
                layers[at].transform.SetParent(built.transform, false);
            }
            var flat = new GameObject("Flat", typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            flat.transform.SetParent(built.transform, false);
            built.Configure(flat, layers, NewSprite());
            return built;
        }

        static CardParts NewCard(string name, Transform parent)
        {
            var card = new GameObject(name, typeof(RectTransform)).AddComponent<LoadingCardView>();
            card.transform.SetParent(parent, false);
            var avatar = new GameObject("Avatar", typeof(RectTransform)).AddComponent<AvatarView>();
            avatar.transform.SetParent(card.transform, false);
            avatar.Configure(NewArt("Art", avatar.transform), NewArt("Frame", avatar.transform), null, null);
            var built = new CardParts
            {
                View = card,
                Avatar = avatar,
                Nickname = NewText("Nickname", card.transform),
                Progress = NewImage("Progress", card.transform),
                Status = NewText("Status", card.transform),
            };
            card.Configure(avatar, built.Nickname, built.Progress, built.Status);
            return built;
        }

        void Show(MatchView match)
        {
            Assert.IsTrue(snapshot.TryRead(match.ToByteArray()));
            cards.Read(snapshot);
            screen.ShowCards(snapshot, cards);
        }

        static MatchView View(MatchState state, Mode mode, Seat seat, params SeatView[] seats)
        {
            var match = new MatchView { State = state, Mode = mode, Seat = seat };
            match.Seats.Add(seats);
            return match;
        }

        static SeatView At(Seat seat, uint progress, bool ready = false) => new() { Seat = seat, LoadingProgress = progress, Ready = ready };

        [Test]
        public void TwoVersusTwoFillsEveryCardWithItsSeatsPlayer()
        {
            var match = View(MatchState.Loading, Mode._2V2, Seat.South, At(Seat.South, 40), At(Seat.North, 0, true), At(Seat.West, 20), At(Seat.East, 0));
            match.Roster.Add(new RosterEntry { Seat = Seat.South, Nickname = "Ayla", Avatar = "avatar.fox" });
            match.Roster.Add(new RosterEntry { Seat = Seat.North, Nickname = "Bora" });
            match.Roster.Add(new RosterEntry { Seat = Seat.West, Nickname = "Cem" });
            match.Roster.Add(new RosterEntry { Seat = Seat.East, Nickname = "Dina" });

            Show(match);

            Assert.IsTrue(view.gameObject.activeSelf);
            Assert.AreEqual(new[] { "Ayla", "Bora", "Cem", "Dina" }, new[] { parts[0].Nickname.text, parts[1].Nickname.text, parts[2].Nickname.text, parts[3].Nickname.text });
            Assert.AreEqual(0.4f, parts[0].Progress.fillAmount, 1e-6f);
            Assert.AreEqual(1f, parts[1].Progress.fillAmount, 1e-6f);
            Assert.AreEqual(string.Empty, parts[0].Status.text);
            Assert.AreEqual("Ready", parts[1].Status.text);
            Assert.AreEqual((Color)TokenColor.RoleSuccessText, parts[1].Status.color);
            Assert.AreEqual(0.4f, total.fillAmount, 1e-6f);
            CollectionAssert.Contains(stills.Loads, FoxStatic);
        }

        [Test]
        public void OneVersusOneShowsTheFirstCardOfEachRowAndHidesTheRest()
        {
            var match = View(MatchState.Loading, Mode._1V1, Seat.Y, At(Seat.X, 0), At(Seat.Y, 0));
            match.Roster.Add(new RosterEntry { Seat = Seat.X, Nickname = "Xan" });
            match.Roster.Add(new RosterEntry { Seat = Seat.Y, Nickname = "Yaz" });

            Show(match);

            Assert.IsTrue(parts[0].View.gameObject.activeSelf);
            Assert.IsFalse(parts[1].View.gameObject.activeSelf);
            Assert.IsTrue(parts[2].View.gameObject.activeSelf);
            Assert.IsFalse(parts[3].View.gameObject.activeSelf);
            Assert.AreEqual("Yaz", parts[0].Nickname.text);
            Assert.AreEqual("Xan", parts[2].Nickname.text);
        }

        [Test]
        public void ASeatThatNeverLoadedShowsTheBotInTheWarningRole()
        {
            Show(View(MatchState.Opening, Mode._1V1, Seat.X, At(Seat.X, 100, true), At(Seat.Y, 30)));

            Assert.AreEqual("Couldn't connect, bot playing", parts[2].Status.text);
            Assert.AreEqual((Color)TokenColor.RoleWarningText, parts[2].Status.color);
            Assert.AreEqual(0.3f, parts[2].Progress.fillAmount, 1e-6f);
        }

        [Test]
        public void ARepeatedViewReloadsNoArt()
        {
            var match = View(MatchState.Loading, Mode._1V1, Seat.X, At(Seat.X, 0), At(Seat.Y, 0));
            match.Roster.Add(new RosterEntry { Seat = Seat.X, Avatar = "avatar.fox" });

            Show(match);
            var loads = stills.Loads.Count;
            Show(match);

            Assert.AreEqual(loads, stills.Loads.Count);
        }

        [Test]
        public void TheTableTheServerChoseLoadsOnceBehindTheCards()
        {
            var match = View(MatchState.Loading, Mode._1V1, Seat.X, At(Seat.X, 0), At(Seat.Y, 0));
            match.Table = "table.neon";

            Show(match);
            Show(match);

            Assert.AreEqual(1, stills.Loads.FindAll(address => address == NeonStatic).Count);
        }

        [Test]
        public void ATableTheManifestDoesNotShowLoadsNothing()
        {
            var match = View(MatchState.Loading, Mode._1V1, Seat.X, At(Seat.X, 0), At(Seat.Y, 0));
            match.Table = "table.gone";

            Show(match);

            CollectionAssert.DoesNotContain(stills.Loads, NeonStatic);
            Assert.IsTrue(table.Flat != null);
        }

        [Test]
        public void CloseHidesTheScreen()
        {
            Show(View(MatchState.Loading, Mode._1V1, Seat.X, At(Seat.X, 0), At(Seat.Y, 0)));

            screen.Close();

            Assert.IsFalse(view.gameObject.activeSelf);
        }
    }
}
