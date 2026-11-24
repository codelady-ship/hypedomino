using HypeDomino.UI.Graphics;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class FrameRatePolicyTests
    {
        static readonly int[] Panel120 = { 60, 120 };
        static readonly int[] Panel60 = { 60 };
        static readonly int[] Panel90 = { 60, 90 };

        static FrameRatePolicy Policy(GraphicsLevel level, int[] refreshRatesHz, int? choice = null, bool batterySaver = false) =>
            new FrameRatePolicy(new GraphicsQuality(DeviceTier.High, level, 0, false), refreshRatesHz, choice, batterySaver);

        [TestCase(GraphicsLevel.Low, 30)]
        [TestCase(GraphicsLevel.Medium, 60)]
        [TestCase(GraphicsLevel.High, 120)]
        [TestCase(GraphicsLevel.Ultra, 120)]
        public void TheLevelInUseCapsTheRate(GraphicsLevel level, int highest)
        {
            Assert.AreEqual(highest, Policy(level, Panel120).Highest);
        }

        [Test]
        public void OnlyRatesThatDivideARefreshRateAreOffered()
        {
            var on120 = Policy(GraphicsLevel.Ultra, Panel120);
            Assert.IsTrue(on120.IsOffered(30));
            Assert.IsTrue(on120.IsOffered(120));
            Assert.IsFalse(on120.IsOffered(90));
            Assert.IsFalse(on120.IsOffered(45));

            Assert.IsTrue(Policy(GraphicsLevel.Ultra, Panel90).IsOffered(90));
            Assert.AreEqual(60, Policy(GraphicsLevel.Ultra, Panel60).Highest);
        }

        [Test]
        public void AutoStopsAt60AndOneHundredTwentyIsOptIn()
        {
            var policy = Policy(GraphicsLevel.High, Panel120);
            policy.SetInMatch(true);
            Assert.AreEqual(60, policy.Target);

            Assert.IsTrue(policy.Choose(120));
            Assert.AreEqual(120, policy.Target);
        }

        [Test]
        public void AutoOnLowRunsAt30()
        {
            var policy = Policy(GraphicsLevel.Low, Panel120);
            policy.SetInMatch(true);
            Assert.AreEqual(30, policy.Target);
        }

        [Test]
        public void ARateAboveTheCapCannotBeChosen()
        {
            var policy = Policy(GraphicsLevel.Medium, Panel120);
            var changes = 0;
            policy.Changed += () => changes++;

            Assert.IsFalse(policy.Choose(120));
            Assert.IsFalse(policy.Choose(90));
            Assert.IsNull(policy.Choice);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void MenusRunAtMost60AndTheChosenRateOnlyInAMatch()
        {
            var policy = Policy(GraphicsLevel.Ultra, Panel120, 120);
            Assert.AreEqual(60, policy.Target);
            policy.SetInMatch(true);
            Assert.AreEqual(120, policy.Target);
            policy.SetInMatch(false);
            Assert.AreEqual(60, policy.Target);
        }

        [Test]
        public void ThirtyChosenHoldsInMenusToo()
        {
            var policy = Policy(GraphicsLevel.High, Panel120, 30);
            Assert.AreEqual(30, policy.Target);
            policy.SetInMatch(true);
            Assert.AreEqual(30, policy.Target);
        }

        [Test]
        public void EitherSaverRunsEverythingAt30WithoutTouchingTheLevel()
        {
            var policy = Policy(GraphicsLevel.Ultra, Panel120, 120);
            policy.SetInMatch(true);

            policy.SetBatterySaver(true);
            Assert.AreEqual(30, policy.Target);
            policy.SetBatterySaver(false);
            policy.SetSystemSaver(true);
            Assert.AreEqual(30, policy.Target);
            policy.SetSystemSaver(false);
            Assert.AreEqual(120, policy.Target);
        }

        [Test]
        public void ASavedRateAboveTheCapCountsAsTheCap()
        {
            var policy = Policy(GraphicsLevel.Medium, Panel120, 120);
            policy.SetInMatch(true);
            Assert.AreEqual(60, policy.Target);
        }

        [Test]
        public void HeatLowersTheRateWithTheLevelAndItComesBack()
        {
            var quality = new GraphicsQuality(DeviceTier.High, GraphicsLevel.High, 0, false);
            var policy = new FrameRatePolicy(quality, Panel120, 120, false);
            policy.SetInMatch(true);
            var changes = 0;
            policy.Changed += () => changes++;

            quality.UpdateHeat(ThermalStatus.Moderate, 0f);
            Assert.AreEqual(60, policy.Target);
            quality.UpdateHeat(ThermalStatus.Severe, 1f);
            Assert.AreEqual(30, policy.Target);
            quality.UpdateHeat(ThermalStatus.Normal, 2f);
            quality.UpdateHeat(ThermalStatus.Normal, 62f);
            Assert.AreEqual(120, policy.Target);
            Assert.AreEqual(3, changes);
        }

        [Test]
        public void ADisplayThatReportsNoRateOffersEveryRateUpToTheCap()
        {
            Assert.AreEqual(120, Policy(GraphicsLevel.Ultra, new int[0]).Highest);
        }
    }
}
