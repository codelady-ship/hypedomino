using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Rarity;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class TableViewTests
    {
        const float FramesPerSecond = 10f;
        const string NeonStatic = "cosmetic/table/neon/static";
        const string NeonSky = "cosmetic/table/neon/layer/sky";
        const string NeonCloth = "cosmetic/table/neon/layer/cloth";
        const string PlainStatic = "cosmetic/table/plain/static";
        const string Rarities = @"{""diamondMinSize"":72,""flowPeriodMs"":2000,""tiers"":[
{""id"":""top"",""ordinal"":1,""nameKey"":""rarity.top.name"",""fill"":""#0000FF"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":1,""glowAlpha"":0.75,""glowScale"":1.5,""flow"":true}]}";

        static readonly TableLook Neon = new(NeonStatic, new[] { new TableLayer("sky", NeonSky, true), new TableLayer("cloth", NeonCloth, false) });
        static readonly TableLook Plain = new(PlainStatic, null);

        GameObject root;
        FakeArtSource<Sprite> stills;
        FakeArtSource<LoopArt> loops;
        RarityTable rarities;
        Sprite builtIn;
        Sprite still;
        Sprite skyFrame;
        LoopArt sky;
        LoopArt cloth;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Screen", typeof(RectTransform), typeof(Canvas));
            stills = new FakeArtSource<Sprite>();
            loops = new FakeArtSource<LoopArt>();
            rarities = RarityTable.Parse(Rarities);
            builtIn = NewSprite();
            still = NewSprite();
            skyFrame = NewSprite();
            sky = new LoopArt(NewSprite(), new[] { skyFrame, NewSprite() }, FramesPerSecond);
            cloth = new LoopArt(NewSprite(), null, 0f);
        }

        [TearDown]
        public void DestroyScreen() => Object.Destroy(root);

        static Sprite NewSprite() => Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);

        CosmeticStage NewStage(EffectLevel level, bool reduceMotion = false) =>
            new(stills, loops, rarities, null, null, level, reduceMotion);

        TableView NewView(int capacity = 2)
        {
            var view = new GameObject("Table", typeof(RectTransform)).AddComponent<TableView>();
            view.transform.SetParent(root.transform, false);
            var layers = new AnimatedImage[capacity];
            for (var at = 0; at < capacity; at++)
            {
                layers[at] = new GameObject("Layer", typeof(RectTransform), typeof(Image)).AddComponent<AnimatedImage>();
                layers[at].transform.SetParent(view.transform, false);
            }
            var flat = new GameObject("Flat", typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            flat.transform.SetParent(view.transform, false);
            view.Configure(flat, layers, builtIn);
            return view;
        }

        void FinishLayers()
        {
            loops.Finish(NeonSky, sky);
            loops.Finish(NeonCloth, cloth);
        }

        [Test]
        public void TheBuiltInTableShowsUntilTheStaticHasLoaded()
        {
            var view = NewView();
            view.Show(NewStage(EffectLevel.Medium), Plain);
            Assert.AreSame(builtIn, view.Flat);

            stills.Finish(PlainStatic, still);

            Assert.AreSame(still, view.Flat);
            Assert.IsFalse(view.IsComposed);
            Assert.IsEmpty(loops.Loads);
        }

        [Test]
        public void TheLayersLoadOnlyAfterTheStatic()
        {
            var view = NewView();
            view.Show(NewStage(EffectLevel.Medium), Neon);
            Assert.IsEmpty(loops.Loads);

            stills.Finish(NeonStatic, still);

            CollectionAssert.AreEqual(new[] { NeonSky, NeonCloth }, loops.Loads);
        }

        [Test]
        public void TheStaticStaysUntilEveryLayerHasLoaded()
        {
            var view = NewView();
            view.Show(NewStage(EffectLevel.Medium), Neon);
            stills.Finish(NeonStatic, still);

            loops.Finish(NeonSky, sky);
            Assert.AreSame(still, view.Flat);
            Assert.IsFalse(view.IsComposed);

            loops.Finish(NeonCloth, cloth);
            Assert.IsTrue(view.IsComposed);
            Assert.IsNull(view.Flat);
        }

        [Test]
        public void TheComposedTablePlaysItsAnimatedLayersBackToFront()
        {
            var view = NewView();
            var stage = NewStage(EffectLevel.Medium);
            view.Show(stage, Neon);
            stills.Finish(NeonStatic, still);
            FinishLayers();

            Assert.IsTrue(view.Layer(0).IsPlaying);
            Assert.AreSame(skyFrame, view.Layer(0).Shown);
            Assert.IsFalse(view.Layer(1).IsPlaying);
            Assert.AreSame(cloth.Still, view.Layer(1).Shown);
            Assert.AreEqual(1, stage.Budget.Active);
        }

        [TestCase(EffectLevel.Low, false)]
        [TestCase(EffectLevel.High, true)]
        public void TheLowLevelAndReduceMotionKeepTheStatic(EffectLevel level, bool reduceMotion)
        {
            var view = NewView();
            view.Show(NewStage(level, reduceMotion), Neon);
            stills.Finish(NeonStatic, still);

            Assert.IsEmpty(loops.Loads);
            Assert.AreSame(still, view.Flat);
        }

        [Test]
        public void AFailedLayerKeepsTheStaticAndFreesTheOthers()
        {
            var view = NewView();
            view.Show(NewStage(EffectLevel.Medium), Neon);
            stills.Finish(NeonStatic, still);

            loops.Finish(NeonSky, sky);
            loops.Finish(NeonCloth, null);

            Assert.IsFalse(view.IsComposed);
            Assert.AreSame(still, view.Flat);
            CollectionAssert.AreEqual(new[] { NeonSky }, loops.Releases);
        }

        [Test]
        public void ATableWithMoreLayersThanTheViewHoldsStaysStatic()
        {
            var view = NewView(1);
            view.Show(NewStage(EffectLevel.Medium), Neon);
            stills.Finish(NeonStatic, still);

            Assert.IsEmpty(loops.Loads);
            Assert.AreSame(still, view.Flat);
        }

        [Test]
        public void ASpentBudgetKeepsTheStatic()
        {
            var view = NewView();
            var stage = NewStage(EffectLevel.Medium);
            while (stage.Budget.TryAcquire()) { }
            view.Show(stage, Neon);
            stills.Finish(NeonStatic, still);

            Assert.IsEmpty(loops.Loads);
            Assert.AreSame(still, view.Flat);
        }

        [Test]
        public void DroppingToTheLowLevelReturnsToTheStaticAndFreesTheLayers()
        {
            var view = NewView();
            var stage = NewStage(EffectLevel.Medium);
            view.Show(stage, Neon);
            stills.Finish(NeonStatic, still);
            FinishLayers();

            stage.SetEffects(EffectLevel.Low, false);
            view.RefreshEffects();

            Assert.IsFalse(view.IsComposed);
            Assert.AreSame(still, view.Flat);
            Assert.IsFalse(view.Layer(0).IsPlaying);
            CollectionAssert.AreEquivalent(new[] { NeonSky, NeonCloth }, loops.Releases);
        }

        [Test]
        public void ShowingAnotherTableFreesThePreviousOne()
        {
            var view = NewView();
            var stage = NewStage(EffectLevel.Medium);
            view.Show(stage, Neon);
            stills.Finish(NeonStatic, still);
            FinishLayers();

            view.Show(stage, Plain);

            Assert.IsFalse(view.IsComposed);
            Assert.AreSame(builtIn, view.Flat);
            CollectionAssert.AreEquivalent(new[] { NeonSky, NeonCloth }, loops.Releases);
            CollectionAssert.AreEqual(new[] { NeonStatic }, stills.Releases);
        }

        [Test]
        public void ShowingTheSameTableAgainLoadsNothingNew()
        {
            var view = NewView();
            var stage = NewStage(EffectLevel.Medium);
            view.Show(stage, Neon);
            stills.Finish(NeonStatic, still);
            FinishLayers();

            view.Show(stage, new TableLook(NeonStatic, new[] { new TableLayer("sky", NeonSky, true), new TableLayer("cloth", NeonCloth, false) }));

            Assert.IsTrue(view.IsComposed);
            Assert.AreEqual(1, stills.Loads.Count);
            Assert.AreEqual(2, loops.Loads.Count);
            Assert.IsEmpty(loops.Releases);
        }

        [Test]
        public void ALateStaticOfAReplacedTableIsIgnored()
        {
            var view = NewView();
            var stage = NewStage(EffectLevel.Medium);
            view.Show(stage, Neon);
            view.Show(stage, Plain);

            stills.Finish(NeonStatic, still);

            Assert.AreSame(builtIn, view.Flat);
            Assert.IsEmpty(loops.Loads);
        }
    }
}
