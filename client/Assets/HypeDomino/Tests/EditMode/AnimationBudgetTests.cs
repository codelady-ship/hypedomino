using HypeDomino.UI.Motion;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AnimationBudgetTests
    {
        [Test]
        public void ItGrantsSlotsUpToItsLimitAndTakesThemBack()
        {
            var budget = new AnimationBudget(2);
            Assert.IsTrue(budget.TryAcquire());
            Assert.IsTrue(budget.TryAcquire());
            Assert.IsFalse(budget.TryAcquire());
            Assert.AreEqual(2, budget.Active);

            budget.Release();
            Assert.IsTrue(budget.TryAcquire());
        }

        [Test]
        public void ItSaysWhetherASlotIsFreeWithoutTakingOne()
        {
            var budget = new AnimationBudget(1);
            Assert.IsTrue(budget.HasRoom);
            Assert.AreEqual(0, budget.Active);
            budget.TryAcquire();
            Assert.IsFalse(budget.HasRoom);
        }

        [Test]
        public void ItSaysWhetherSeveralSlotsAreFree()
        {
            var budget = new AnimationBudget(3);
            budget.TryAcquire();
            Assert.IsTrue(budget.HasRoomFor(2));
            Assert.IsFalse(budget.HasRoomFor(3));
            Assert.AreEqual(1, budget.Active);
        }

        [Test]
        public void ReleasingAnEmptyBudgetAddsNoSlot()
        {
            var budget = new AnimationBudget(1);
            budget.Release();
            Assert.AreEqual(0, budget.Active);
            Assert.IsTrue(budget.TryAcquire());
            Assert.IsFalse(budget.TryAcquire());
        }

        [Test]
        public void TheLowLevelGrantsNoSlotAndTheOthersGrantTen()
        {
            Assert.IsFalse(AnimationBudget.For(EffectLevel.Low).TryAcquire());
            foreach (var level in new[] { EffectLevel.Medium, EffectLevel.High })
            {
                var budget = AnimationBudget.For(level);
                for (var slot = 0; slot < 10; slot++) Assert.IsTrue(budget.TryAcquire(), level.ToString());
                Assert.IsFalse(budget.TryAcquire(), level.ToString());
            }
        }
    }
}
