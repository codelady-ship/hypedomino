using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Motion;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class CosmeticEffectsLinkTests
    {
        int refreshes;

        void CountRefresh() => refreshes++;

        static CosmeticStage Stage(EffectLevel level, bool reduceMotion) => new CosmeticStage(null, null, null, null, null, level, reduceMotion);

        CosmeticEffectsLink Link(CosmeticStage stage, GraphicsQuality graphics, MotionPreference motion)
        {
            var link = new CosmeticEffectsLink(stage, graphics, motion, CountRefresh);
            refreshes = 0;
            return link;
        }

        [SetUp]
        public void ResetCount() => refreshes = 0;

        [Test]
        public void LinkingSetsTheStageToTheCurrentSettingsAndRefreshesOnce()
        {
            var stage = Stage(EffectLevel.High, true);
            var graphics = new GraphicsQuality(DeviceTier.Mid, null, 0, false);

            new CosmeticEffectsLink(stage, graphics, new MotionPreference(null, false), CountRefresh);

            Assert.AreEqual(EffectLevel.Medium, stage.Level);
            Assert.IsFalse(stage.ReduceMotion);
            Assert.AreEqual(1, refreshes);
        }

        [Test]
        public void LinkingAStageThatAlreadyMatchesDoesNotRefresh()
        {
            var stage = Stage(EffectLevel.Medium, false);

            new CosmeticEffectsLink(stage, new GraphicsQuality(DeviceTier.Mid, null, 0, false), new MotionPreference(null, false), CountRefresh);

            Assert.AreEqual(0, refreshes);
        }

        [Test]
        public void ChoosingLowGraphicsMakesEveryCosmeticStaticAndRefreshes()
        {
            var stage = Stage(EffectLevel.Medium, false);
            var graphics = new GraphicsQuality(DeviceTier.Mid, null, 0, false);
            Link(stage, graphics, new MotionPreference(null, false));

            graphics.Choose(GraphicsLevel.Low);

            Assert.AreEqual(EffectLevel.Low, stage.Level);
            Assert.IsFalse(stage.AllowsAnimation);
            Assert.AreEqual(1, refreshes);
        }

        [Test]
        public void SevereHeatMakesEveryCosmeticStatic()
        {
            var stage = Stage(EffectLevel.Medium, false);
            var graphics = new GraphicsQuality(DeviceTier.Mid, null, 0, false);
            Link(stage, graphics, new MotionPreference(null, false));

            graphics.UpdateHeat(ThermalStatus.Severe, 0f);

            Assert.IsFalse(stage.AllowsAnimation);
            Assert.AreEqual(1, refreshes);
        }

        [Test]
        public void ReduceMotionMakesEveryCosmeticStaticAndRefreshes()
        {
            var stage = Stage(EffectLevel.Medium, false);
            var motion = new MotionPreference(null, false);
            Link(stage, new GraphicsQuality(DeviceTier.Mid, null, 0, false), motion);

            motion.Choose(true);

            Assert.IsTrue(stage.ReduceMotion);
            Assert.IsFalse(stage.AllowsAnimation);
            Assert.AreEqual(1, refreshes);
        }

        [Test]
        public void ALevelChangeWithTheSameEffectsKeepsTheBudgetAndDoesNotRefresh()
        {
            var stage = Stage(EffectLevel.High, false);
            var graphics = new GraphicsQuality(DeviceTier.High, GraphicsLevel.High, 0, false);
            Link(stage, graphics, new MotionPreference(null, false));
            var budget = stage.Budget;

            graphics.Choose(GraphicsLevel.Ultra);

            Assert.AreSame(budget, stage.Budget);
            Assert.AreEqual(0, refreshes);
        }

        [Test]
        public void ADisposedLinkNoLongerFollowsTheSettings()
        {
            var stage = Stage(EffectLevel.Medium, false);
            var graphics = new GraphicsQuality(DeviceTier.Mid, null, 0, false);
            var motion = new MotionPreference(null, false);
            Link(stage, graphics, motion).Dispose();

            graphics.Choose(GraphicsLevel.Low);
            motion.Choose(true);

            Assert.AreEqual(EffectLevel.Medium, stage.Level);
            Assert.IsFalse(stage.ReduceMotion);
            Assert.AreEqual(0, refreshes);
        }
    }
}
