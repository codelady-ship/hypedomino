using HypeDomino.UI.Emoji;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class EffectPoolTests
    {
        [TestCase(EffectLevel.Low, 150)]
        [TestCase(EffectLevel.Medium, 400)]
        [TestCase(EffectLevel.High, 800)]
        public void TheLiveParticleCapIsTheLevelsBudget(EffectLevel level, int limit)
        {
            Assert.AreEqual(limit, new ParticleBudget(level, false).Limit);
            Assert.AreEqual(150, new ParticleBudget(level, true).Limit);
        }

        [TestCase(TokenEffectKind.TilePlace, 4, 12, 24)]
        [TestCase(TokenEffectKind.ScorePopup, 0, 0, 8)]
        [TestCase(TokenEffectKind.ScoreCascade, 30, 90, 180)]
        [TestCase(TokenEffectKind.SpinnerFormation, 10, 30, 60)]
        [TestCase(TokenEffectKind.Domino, 40, 120, 240)]
        [TestCase(TokenEffectKind.Block, 12, 40, 80)]
        [TestCase(TokenEffectKind.MatchWin, 60, 200, 400)]
        [TestCase(TokenEffectKind.RankUp, 50, 160, 320)]
        [TestCase(TokenEffectKind.Emoji, 0, 32, 64)]
        public void EachEffectGetsAtMostItsBudgetedParticles(TokenEffectKind kind, int low, int medium, int high)
        {
            Assert.AreEqual(low, new ParticleBudget(EffectLevel.Low, false).MaxFor(kind));
            Assert.AreEqual(medium, new ParticleBudget(EffectLevel.Medium, false).MaxFor(kind));
            Assert.AreEqual(high, new ParticleBudget(EffectLevel.High, false).MaxFor(kind));
            Assert.AreEqual(low, new ParticleBudget(EffectLevel.High, true).MaxFor(kind));
        }

        [Test]
        public void ATakeIsCappedByTheEffectAndByWhatTheLevelHasLeft()
        {
            var budget = new ParticleBudget(EffectLevel.Low, false);
            Assert.AreEqual(4, budget.Take(TokenEffectKind.TilePlace, 50));
            Assert.AreEqual(0, budget.Take(TokenEffectKind.TilePlace, -3));
            Assert.AreEqual(60, budget.Take(TokenEffectKind.MatchWin, 60));
            Assert.AreEqual(50, budget.Take(TokenEffectKind.RankUp, 50));
            Assert.AreEqual(30, budget.Take(TokenEffectKind.ScoreCascade, 30));
            Assert.AreEqual(6, budget.Take(TokenEffectKind.Domino, 40));
            Assert.AreEqual(150, budget.Live);
            Assert.AreEqual(0, budget.Take(TokenEffectKind.Block, 12));

            budget.Return(10);
            Assert.AreEqual(10, budget.Take(TokenEffectKind.Block, 12));
            budget.Return(500);
            Assert.AreEqual(0, budget.Live);
        }

        [Test]
        public void TurningOnReduceMotionUsesTheLowCountsAtOnce()
        {
            var budget = new ParticleBudget(EffectLevel.High, false);
            Assert.AreEqual(180, budget.Take(TokenEffectKind.ScoreCascade, 180));

            budget.Configure(EffectLevel.High, true);

            Assert.AreEqual(EffectLevel.Low, budget.ParticleLevel);
            Assert.AreEqual(0, budget.Take(TokenEffectKind.ScoreCascade, 30));
            budget.Return(180);
            Assert.AreEqual(30, budget.Take(TokenEffectKind.ScoreCascade, 180));
        }

        [TestCase(TokenEffectKind.TilePlace, 4)]
        [TestCase(TokenEffectKind.ScorePopup, 6)]
        [TestCase(TokenEffectKind.ScoreCascade, 2)]
        [TestCase(TokenEffectKind.SpinnerFormation, 1)]
        [TestCase(TokenEffectKind.Domino, 1)]
        [TestCase(TokenEffectKind.Block, 1)]
        [TestCase(TokenEffectKind.MatchWin, 1)]
        [TestCase(TokenEffectKind.RankUp, 1)]
        [TestCase(TokenEffectKind.Emoji, 4)]
        public void APoolHoldsTheEffectsPooledCount(TokenEffectKind kind, int pooled)
        {
            Assert.AreEqual(pooled, new EffectPool(kind, new ParticleBudget(EffectLevel.High, false)).Capacity);
        }

        [Test]
        public void AFullPoolEndsItsOldestInstanceAndReusesIt()
        {
            var budget = new ParticleBudget(EffectLevel.High, false);
            var pool = new EffectPool(TokenEffectKind.ScoreCascade, budget);
            var first = pool.Begin(0f, 180);
            var second = pool.Begin(120f, 54);
            Assert.AreNotEqual(first, second);
            Assert.AreEqual(234, budget.Live);

            var third = pool.Begin(240f, 27);

            Assert.AreEqual(first, third);
            Assert.AreEqual(2, pool.LiveCount);
            Assert.AreEqual(27, pool.ParticlesOf(third));
            Assert.AreEqual(81, budget.Live);
        }

        [Test]
        public void AnInstanceEndsAtTheEffectsMaxTimeAndGivesBackItsParticles()
        {
            var budget = new ParticleBudget(EffectLevel.Medium, false);
            var pool = new EffectPool(TokenEffectKind.TilePlace, budget);
            var slot = pool.Begin(100f, 12);

            pool.Tick(100f + TokenEffect.TilePlaceDurationMs - 1f);
            Assert.IsTrue(pool.IsLive(slot));
            Assert.AreEqual(12, budget.Live);

            pool.Tick(100f + TokenEffect.TilePlaceDurationMs);
            Assert.IsFalse(pool.IsLive(slot));
            Assert.AreEqual(0, pool.ParticlesOf(slot));
            Assert.AreEqual(0, budget.Live);
            Assert.AreEqual(0, pool.LiveCount);
        }

        [Test]
        public void EndingTwiceGivesBackTheParticlesOnce()
        {
            var budget = new ParticleBudget(EffectLevel.Medium, false);
            var other = new EffectPool(TokenEffectKind.Block, budget);
            var pool = new EffectPool(TokenEffectKind.Domino, budget);
            other.Begin(0f, 40);
            var slot = pool.Begin(0f, 120);

            pool.End(slot);
            pool.End(slot);

            Assert.AreEqual(40, budget.Live);
            Assert.AreEqual(0, pool.LiveCount);
        }

        [Test]
        public void EndAllIsTheJumpToTheFinalState()
        {
            var budget = new ParticleBudget(EffectLevel.High, false);
            var pool = new EffectPool(TokenEffectKind.TilePlace, budget);
            for (var at = 0; at < pool.Capacity; at++) pool.Begin(at, 24);

            pool.EndAll();

            Assert.AreEqual(0, pool.LiveCount);
            Assert.AreEqual(0, budget.Live);
        }

        [Test]
        public void TheTurnSetFitsTheLevelCapAtEveryLevel()
        {
            var kinds = new[] { TokenEffectKind.TilePlace, TokenEffectKind.ScorePopup, TokenEffectKind.ScoreCascade, TokenEffectKind.SpinnerFormation, TokenEffectKind.Emoji };
            foreach (var level in new[] { EffectLevel.Low, EffectLevel.Medium, EffectLevel.High })
            {
                var budget = new ParticleBudget(level, false);
                foreach (var kind in kinds) Assert.AreEqual(budget.MaxFor(kind), budget.Take(kind, int.MaxValue), $"{level} {kind}");
            }
        }

        [Test]
        public void EachPoolProfilesUnderItsEffectName()
        {
            var budget = new ParticleBudget(EffectLevel.Low, false);
            Assert.AreEqual("Effect.scoreCascade", EffectPool.MarkerNameOf(TokenEffectKind.ScoreCascade));
            Assert.AreEqual("Effect.tilePlace", EffectPool.MarkerNameOf(new EffectPool(TokenEffectKind.TilePlace, budget).Kind));
            Assert.AreEqual(SeatEmojiLayer.ProfilerMarkerName, EffectPool.MarkerNameOf(TokenEffectKind.Emoji));
        }

        [Test]
        public void AFrameAllocatesNothing()
        {
            var budget = new ParticleBudget(EffectLevel.High, false);
            var pool = new EffectPool(TokenEffectKind.ScoreCascade, budget);
            var now = 0f;
            TestDelegate frame = () =>
            {
                now += 400f;
                pool.End(pool.Begin(now, 90));
                pool.Begin(now, 90);
                pool.Tick(now + 16f);
            };
            frame();

            Assert.That(frame, Allocates.Nothing);
        }
    }
}
