using System.Collections;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Rarity;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.TestTools;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class TileBackViewTests
    {
        const float FramesPerSecond = 10f;
        const int SeatTiles = 14;
        const string SkinStatic = "cosmetic/tile/storm/static";
        const string SkinAnim = "cosmetic/tile/storm/anim";
        const string Rarities = @"{""diamondMinSize"":72,""flowPeriodMs"":2000,""tiers"":[
{""id"":""top"",""ordinal"":1,""nameKey"":""rarity.top.name"",""fill"":""#0000FF"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":1,""glowAlpha"":0.75,""glowScale"":1.5,""flow"":true}]}";

        static readonly CosmeticArt AnimatedSkin = new CosmeticArt(SkinStatic, SkinAnim);

        GameObject root;
        FakeArtSource<Sprite> stills;
        FakeArtSource<LoopArt> loops;
        Sprite builtInBack;
        Sprite still;
        Sprite frameOne;
        Sprite frameTwo;
        LoopArt loop;
        Image[] followers;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Screen", typeof(RectTransform), typeof(Canvas));
            stills = new FakeArtSource<Sprite>();
            loops = new FakeArtSource<LoopArt>();
            builtInBack = NewSprite();
            still = NewSprite();
            frameOne = NewSprite();
            frameTwo = NewSprite();
            var frames = new Sprite[20];
            for (var i = 0; i < frames.Length; i++) frames[i] = i == 0 ? frameOne : frameTwo;
            loop = new LoopArt(still, frames, FramesPerSecond);
        }

        [TearDown]
        public void DestroyScreen() => Object.Destroy(root);

        static Sprite NewSprite() => Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);

        CosmeticStage NewStage(EffectLevel level) =>
            new CosmeticStage(stills, loops, RarityTable.Parse(Rarities), NewSprite(), NewSprite(), level, false);

        TileBackView NewView()
        {
            var lead = new GameObject("Lead", typeof(RectTransform), typeof(Image)).AddComponent<AnimatedImage>().gameObject.AddComponent<CosmeticImage>();
            lead.transform.SetParent(root.transform, false);
            followers = new Image[SeatTiles - 1];
            for (var i = 0; i < followers.Length; i++)
            {
                followers[i] = new GameObject("Back", typeof(RectTransform)).AddComponent<Image>();
                followers[i].transform.SetParent(root.transform, false);
            }
            var view = new GameObject("Seat backs", typeof(RectTransform)).AddComponent<TileBackView>();
            view.transform.SetParent(root.transform, false);
            view.Configure(lead, followers, builtInBack);
            return view;
        }

        void AssertBacks(int count, Sprite expected)
        {
            for (var i = 0; i < followers.Length; i++)
            {
                Assert.AreEqual(i + 1 < count, followers[i].enabled, $"back {i + 1}");
                if (followers[i].enabled) Assert.AreSame(expected, followers[i].sprite, $"back {i + 1}");
            }
        }

        [Test]
        public void ASeatWithoutASkinShowsTheBuiltInBackOnEveryTile()
        {
            var view = NewView();
            view.Show(NewStage(EffectLevel.High), new CosmeticArt(null, null), 7);

            Assert.AreSame(builtInBack, view.Lead.Shown);
            AssertBacks(7, builtInBack);
            Assert.IsEmpty(stills.Loads);
        }

        [Test]
        public void EveryTileShowsTheSkinOnceItsStaticHasLoaded()
        {
            var view = NewView();
            view.Show(NewStage(EffectLevel.Medium), new CosmeticArt(SkinStatic, null), 5);
            AssertBacks(5, builtInBack);

            stills.Finish(SkinStatic, still);
            view.RefreshEffects();

            AssertBacks(5, still);
            Assert.AreEqual(new[] { SkinStatic }, stills.Loads);
        }

        [UnityTest]
        public IEnumerator AllTilesOfASeatPlayOneLoopOnOneSlot()
        {
            var stage = NewStage(EffectLevel.High);
            var view = NewView();
            view.Show(stage, AnimatedSkin, SeatTiles);
            stills.Finish(SkinStatic, still);
            loops.Finish(SkinAnim, loop);
            view.RefreshEffects();

            Assert.AreEqual(1, stage.Budget.Active);
            Assert.AreEqual(new[] { SkinAnim }, loops.Loads);
            AssertBacks(SeatTiles, frameOne);

            yield return new WaitForSecondsRealtime(1.5f / FramesPerSecond);
            yield return null;

            Assert.AreSame(frameTwo, view.Lead.Shown);
            AssertBacks(SeatTiles, frameTwo);
        }

        [Test]
        public void OnTheLowLevelTheSkinStaysStaticAndNoLoopLoads()
        {
            var view = NewView();
            view.Show(NewStage(EffectLevel.Low), AnimatedSkin, 7);
            stills.Finish(SkinStatic, still);
            view.RefreshEffects();

            Assert.IsFalse(view.Lead.IsPlaying);
            Assert.IsEmpty(loops.Loads);
            AssertBacks(7, still);
        }

        [Test]
        public void PlayingTilesChangesOnlyHowManyBacksShow()
        {
            var view = NewView();
            view.Show(NewStage(EffectLevel.Medium), new CosmeticArt(SkinStatic, null), 7);
            stills.Finish(SkinStatic, still);

            view.SetCount(3);
            AssertBacks(3, still);
            view.SetCount(0);
            Assert.AreEqual(0f, view.Lead.GetComponent<CanvasRenderer>().GetAlpha());
            AssertBacks(0, still);
            view.SetCount(40);
            Assert.AreEqual(SeatTiles, view.Count);
            Assert.AreEqual(1f, view.Lead.GetComponent<CanvasRenderer>().GetAlpha());
            AssertBacks(SeatTiles, still);

            Assert.AreEqual(new[] { SkinStatic }, stills.Loads);
        }

        [Test]
        public void ChangingTheCountAndPlayingTheLoopAllocateNothing()
        {
            var view = NewView();
            view.Show(NewStage(EffectLevel.High), AnimatedSkin, SeatTiles);
            stills.Finish(SkinStatic, still);
            loops.Finish(SkinAnim, loop);
            view.RefreshEffects();
            var count = SeatTiles;

            void Step()
            {
                count = count == SeatTiles ? 3 : SeatTiles;
                view.SetCount(count);
                view.RefreshEffects();
            }

            Step();
            Assert.That(Step, Allocates.Nothing);
        }
    }
}
