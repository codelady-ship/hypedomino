using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Rarity;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class TableBackgroundTests
    {
        const string StarterStatic = "cosmetic/table/starter/static";
        const string PlainStatic = "cosmetic/table/plain/static";
        const string NeonStatic = "cosmetic/table/neon/static";
        const string NeonBase = "cosmetic/table/neon/layer/base";
        const string NeonMotion = "cosmetic/table/neon/layer/motion";
        const string Rarities = @"{""diamondMinSize"":72,""flowPeriodMs"":2000,""tiers"":[
{""id"":""common"",""ordinal"":1,""nameKey"":""rarity.common.name"",""fill"":""#0000FF"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":1,""glowAlpha"":0.5,""glowScale"":1.2,""flow"":false}]}";
        const string Catalog = @"{""schemaVersion"":1,""manifestVersion"":1,""items"":[
{""id"":""table.starter"",""type"":""table"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/table/starter/static"",""sources"":[{""type"":""default""}]},
{""id"":""table.plain"",""type"":""table"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/table/plain/static""},
{""id"":""table.gone"",""type"":""table"",""rarityId"":""common"",""availability"":""disabled"",""staticAddress"":""cosmetic/table/gone/static""},
{""id"":""avatar.plain"",""type"":""avatar"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/plain/static""},
{""id"":""table.neon"",""type"":""table"",""rarityId"":""common"",""availability"":""live"",""animated"":true,""staticAddress"":""cosmetic/table/neon/static"",
""layers"":[{""name"":""base"",""address"":""cosmetic/table/neon/layer/base"",""animated"":false},{""name"":""motion"",""address"":""cosmetic/table/neon/layer/motion"",""animated"":true}]}]}";

        GameObject root;
        FakeArtSource<Sprite> stills;
        FakeArtSource<LoopArt> loops;
        CosmeticManifest manifest;
        CosmeticStage stage;
        TableView view;
        TableBackground background;
        Sprite builtIn;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Match", typeof(RectTransform), typeof(Canvas));
            stills = new FakeArtSource<Sprite>();
            loops = new FakeArtSource<LoopArt>();
            Assert.IsTrue(CosmeticManifest.TryParse(Catalog, out manifest));
            stage = new CosmeticStage(stills, loops, RarityTable.Parse(Rarities), null, null, EffectLevel.Medium, false);
            builtIn = NewSprite();
            view = NewView();
            background = new TableBackground(stage, view);
        }

        [TearDown]
        public void DestroyMatch() => Object.Destroy(root);

        static Sprite NewSprite() => Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);

        TableView NewView()
        {
            var table = new GameObject("Table", typeof(RectTransform)).AddComponent<TableView>();
            table.transform.SetParent(root.transform, false);
            var layers = new AnimatedImage[2];
            for (var at = 0; at < layers.Length; at++)
            {
                layers[at] = new GameObject("Layer", typeof(RectTransform), typeof(Image)).AddComponent<AnimatedImage>();
                layers[at].transform.SetParent(table.transform, false);
            }
            var flat = new GameObject("Flat", typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            flat.transform.SetParent(table.transform, false);
            table.Configure(flat, layers, builtIn);
            return table;
        }

        static MatchSnapshot Snapshot(string table)
        {
            var match = new MatchView { State = MatchState.Playing, Mode = Mode._1V1, Seat = Seat.X, Table = table };
            match.Roster.Add(new RosterEntry { Seat = Seat.X });
            var snapshot = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(match.ToByteArray()));
            return snapshot;
        }

        void ComposeNeon(MatchSnapshot snapshot)
        {
            background.Show(snapshot, manifest);
            stills.Finish(NeonStatic, NewSprite());
            loops.Finish(NeonBase, new LoopArt(NewSprite(), null, 0f));
            loops.Finish(NeonMotion, new LoopArt(NewSprite(), new[] { NewSprite(), NewSprite() }, 8f));
        }

        [Test]
        public void TheMatchTableShowsItsStaticAfterTheBuiltIn()
        {
            background.Show(Snapshot("table.plain"), manifest);
            Assert.AreSame(builtIn, view.Flat);

            var plain = NewSprite();
            stills.Finish(PlainStatic, plain);

            Assert.AreSame(plain, view.Flat);
            Assert.IsEmpty(loops.Loads);
        }

        [TestCase("")]
        [TestCase("table.unknown")]
        [TestCase("table.gone")]
        [TestCase("avatar.plain")]
        public void AnEmptyUnknownDisabledOrOtherTypeIdShowsTheDefaultTable(string table)
        {
            background.Show(Snapshot(table), manifest);

            Assert.AreEqual(new[] { StarterStatic }, stills.Loads);
        }

        [Test]
        public void WithoutAManifestTheBuiltInTableStays()
        {
            background.Show(Snapshot("table.plain"), null);

            Assert.AreSame(builtIn, view.Flat);
            Assert.IsEmpty(stills.Loads);
        }

        [Test]
        public void AnAnimatedTableComposesItsLayers()
        {
            ComposeNeon(Snapshot("table.neon"));

            Assert.IsTrue(view.IsComposed);
            Assert.IsTrue(view.Layer(1).IsPlaying);
            Assert.AreEqual(1, stage.Budget.Active);
        }

        [Test]
        public void TheSameTableAgainNeitherReloadsNorRestartsAndAllocatesNothing()
        {
            var snapshot = Snapshot("table.neon");
            ComposeNeon(snapshot);

            background.Show(snapshot, manifest);
            Assert.That(() => background.Show(snapshot, manifest), Allocates.Nothing);

            Assert.IsTrue(view.IsComposed);
            Assert.AreEqual(1, stills.Loads.Count);
            Assert.AreEqual(2, loops.Loads.Count);
            Assert.IsEmpty(loops.Releases);
            Assert.AreEqual(1, stage.Budget.Active);
        }

        [Test]
        public void DroppingToTheLowLevelReturnsTheTableToItsStatic()
        {
            ComposeNeon(Snapshot("table.neon"));

            stage.SetEffects(EffectLevel.Low, false);
            background.RefreshEffects();

            Assert.IsFalse(view.IsComposed);
            Assert.IsNotNull(view.Flat);
            CollectionAssert.AreEquivalent(new[] { NeonBase, NeonMotion }, loops.Releases);
        }

        [Test]
        public void AnotherTableFreesThePreviousOne()
        {
            background.Show(Snapshot("table.plain"), manifest);
            stills.Finish(PlainStatic, NewSprite());

            background.Show(Snapshot("table.starter"), manifest);

            Assert.AreEqual(new[] { PlainStatic }, stills.Releases);
            Assert.AreEqual(new[] { PlainStatic, StarterStatic }, stills.Loads);
        }

        [Test]
        public void TheTableTakesOverThePrefetchedStaticWithoutLoadingItAgain()
        {
            var snapshot = Snapshot("table.plain");
            var prefetch = new MatchPrefetch(stage.Stills);
            prefetch.Start(snapshot, manifest);
            var plain = NewSprite();
            stills.Finish(PlainStatic, plain);
            Assert.IsTrue(prefetch.Done);

            background.Show(snapshot, manifest);
            prefetch.Release();

            Assert.AreEqual(new[] { PlainStatic }, stills.Loads);
            Assert.IsEmpty(stills.Releases);
            Assert.AreSame(plain, view.Flat);
        }
    }
}
