using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ScoreCascadeTests
    {
        [TestCase(0u, 0)]
        [TestCase(5u, 1)]
        [TestCase(10u, 2)]
        [TestCase(15u, 3)]
        [TestCase(20u, 4)]
        [TestCase(25u, 5)]
        [TestCase(30u, 6)]
        [TestCase(80u, 6)]
        public void TheStepComesFromThePoints(uint points, int step) => Assert.AreEqual(step, ScoreCascade.StepOf(points));

        [TestCase(1, 108, 500, 0, 0)]
        [TestCase(2, 124, 550, 0, 0)]
        [TestCase(3, 140, 600, 0, 0)]
        [TestCase(4, 157, 650, 6, 180)]
        [TestCase(5, 173, 700, 6, 180)]
        [TestCase(6, 194, 800, 12, 240)]
        public void EachStepMatchesTheSpecTable(int step, int scoreSize, int holdMs, int shakeAmplitude, int shakeDurationMs)
        {
            var burst = ScoreCascade.Burst(step, EffectLevel.High, false);

            Assert.AreEqual(step, burst.Step);
            Assert.AreEqual(scoreSize, burst.ScoreSize);
            Assert.AreEqual(holdMs, burst.HoldMs);
            Assert.AreEqual(shakeAmplitude, burst.ShakeAmplitude);
            Assert.AreEqual(shakeDurationMs, burst.ShakeDurationMs);
        }

        [TestCase(1, 5, 14, 27)]
        [TestCase(2, 9, 27, 54)]
        [TestCase(3, 14, 41, 81)]
        [TestCase(4, 18, 54, 108)]
        [TestCase(5, 24, 72, 144)]
        [TestCase(6, 30, 90, 180)]
        public void ParticlesFollowTheLevelAndStep(int step, int low, int medium, int high)
        {
            Assert.AreEqual(low, ScoreCascade.Burst(step, EffectLevel.Low, false).Particles);
            Assert.AreEqual(medium, ScoreCascade.Burst(step, EffectLevel.Medium, false).Particles);
            Assert.AreEqual(high, ScoreCascade.Burst(step, EffectLevel.High, false).Particles);
        }

        [Test]
        public void ReduceMotionUsesLowParticlesAndNoShake()
        {
            var burst = ScoreCascade.Burst(6, EffectLevel.High, true);

            Assert.AreEqual(30, burst.Particles);
            Assert.AreEqual(0, burst.ShakeAmplitude);
            Assert.AreEqual(0, burst.ShakeDurationMs);
            Assert.AreEqual(194, burst.ScoreSize);
        }

        [Test]
        public void NoStepHasNoBurst()
        {
            var burst = ScoreCascade.Burst(0, EffectLevel.High, false);

            Assert.AreEqual(0, burst.Step);
            Assert.AreEqual(0, burst.Particles);
            Assert.AreEqual(0f, burst.LifetimeMs);
        }

        [Test]
        public void TheLongestPopupIsTheScorePopupLifetime()
        {
            Assert.AreEqual(TokenEffect.ScorePopupDurationMs, ScoreCascade.Burst(6, EffectLevel.High, false).LifetimeMs);
            Assert.AreEqual(TokenDurationMs.Fast + 500 + TokenDurationMs.Base, ScoreCascade.Burst(1, EffectLevel.High, false).LifetimeMs);
        }

        [Test]
        public void SeveralDoublesInOneTurnBoostTheChain()
        {
            Assert.AreEqual(1, ScoreCascade.ChainStepOf(5, 1));
            Assert.AreEqual(3, ScoreCascade.ChainStepOf(10, 2));
            Assert.AreEqual(3, ScoreCascade.StepOf(15));
            Assert.IsTrue(ScoreCascade.ShowsTurnTotal(2));
            Assert.IsFalse(ScoreCascade.ShowsTurnTotal(1));
        }

        [Test]
        public void TheChainBoostGrowsToTwoStepsAndIsCappedAtTheLastStep()
        {
            Assert.AreEqual(3, ScoreCascade.ChainStepOf(5, 3));
            Assert.AreEqual(3, ScoreCascade.ChainStepOf(5, 7));
            Assert.AreEqual(6, ScoreCascade.ChainStepOf(25, 3));
            Assert.AreEqual(0, ScoreCascade.ChainStepOf(0, 3));
        }

        [Test]
        public void ChainPopupsStartAGapApart()
        {
            Assert.AreEqual(0f, ScoreCascade.ChainStartMs(1));
            Assert.AreEqual(TokenScoreCascade.ChainStepGapMs, ScoreCascade.ChainStartMs(2));
            Assert.AreEqual(2 * TokenScoreCascade.ChainStepGapMs, ScoreCascade.ChainStartMs(3));
        }

        [TestCase(25u, 200f)]
        [TestCase(50u, 400f)]
        [TestCase(80u, 640f)]
        [TestCase(150u, 1200f)]
        [TestCase(300u, 1200f)]
        public void TheMeterRollFollowsThePointsUpToTheCap(uint points, float rollMs) => Assert.AreEqual(rollMs, ScoreCascade.RollMs(points), 1e-3f);

        [Test]
        public void HandPointsWithCarryOverPopTwice()
        {
            Assert.AreEqual(3, ScoreCascade.StepOf(15));
            Assert.AreEqual(6, ScoreCascade.StepOf(35));
            Assert.AreEqual(TokenScoreCascade.CarryOverGapMs, ScoreCascade.CarryOverStartMs);
            Assert.AreEqual(400f, ScoreCascade.RollMs(50), 1e-3f);
        }

        [Test]
        public void ThePopupPopsInHoldsAndPopsOut()
        {
            Assert.IsTrue(ScorePopup.At(0f, 500, false, out var start));
            Assert.AreEqual(0f, start.Scale);

            var overshot = false;
            for (var ms = 1f; ms < TokenDurationMs.Fast; ms += 5f)
            {
                ScorePopup.At(ms, 500, false, out var popping);
                overshot |= popping.Scale > 1f;
            }
            Assert.IsTrue(overshot);

            Assert.IsTrue(ScorePopup.At(TokenDurationMs.Fast + 500, 500, false, out var held));
            Assert.AreEqual(1f, held.Scale);
            Assert.AreEqual(1f, held.Alpha);

            var lifetime = ScorePopup.LifetimeMs(500);
            Assert.IsTrue(ScorePopup.At(lifetime - 1f, 500, false, out var leaving));
            Assert.Less(leaving.Scale, 0.1f);
            Assert.IsFalse(ScorePopup.At(lifetime, 500, false, out _));
        }

        [Test]
        public void WithReduceMotionThePopupFades()
        {
            Assert.IsTrue(ScorePopup.At(0f, 500, true, out var start));
            Assert.AreEqual(1f, start.Scale);
            Assert.AreEqual(0f, start.Alpha);

            Assert.IsTrue(ScorePopup.At(TokenDurationMs.Fast, 500, true, out var shown));
            Assert.AreEqual(1f, shown.Alpha);

            Assert.IsTrue(ScorePopup.At(ScorePopup.LifetimeMs(500) - 1f, 500, true, out var leaving));
            Assert.AreEqual(1f, leaving.Scale);
            Assert.Less(leaving.Alpha, 0.1f);
        }

        [Test]
        public void ComputingABurstAndAPopupAllocatesNothing()
        {
            var total = 0f;
            TestDelegate frame = () =>
            {
                var burst = ScoreCascade.Burst(ScoreCascade.ChainStepOf(10, 2), EffectLevel.Medium, false);
                ScorePopup.At(300f, burst.HoldMs, false, out var popup);
                total += burst.Particles + popup.Scale + ScoreCascade.RollMs(50);
            };
            frame();

            Assert.That(frame, Allocates.Nothing);
        }
    }
}
