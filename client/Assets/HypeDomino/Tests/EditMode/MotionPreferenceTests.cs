using HypeDomino.UI.Motion;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MotionPreferenceTests
    {
        int changes;

        MotionPreference Watched(bool? savedChoice, bool systemReduce)
        {
            changes = 0;
            var motion = new MotionPreference(savedChoice, systemReduce);
            motion.Changed += () => changes++;
            return motion;
        }

        [Test]
        public void WithoutAChoiceTheSystemSettingDecides()
        {
            Assert.IsTrue(Watched(null, true).ReduceMotion);
            Assert.IsFalse(Watched(null, false).ReduceMotion);
        }

        [TestCase(true, false)]
        [TestCase(false, true)]
        public void ASavedChoiceWinsOverTheSystem(bool choice, bool system)
        {
            Assert.AreEqual(choice, Watched(choice, system).ReduceMotion);
        }

        [Test]
        public void ASystemChangeAppliesAtOnceOnlyWhileFollowingTheSystem()
        {
            var motion = Watched(null, false);
            motion.SetSystemReduce(true);
            Assert.IsTrue(motion.ReduceMotion);
            Assert.AreEqual(1, changes);

            motion.Choose(false);
            Assert.AreEqual(2, changes);
            motion.SetSystemReduce(false);
            motion.SetSystemReduce(true);
            Assert.IsFalse(motion.ReduceMotion);
            Assert.AreEqual(2, changes);
            Assert.IsTrue(motion.SystemReduce);
        }

        [Test]
        public void ChoosingTheSameValueAgainChangesNothing()
        {
            var motion = Watched(true, false);
            motion.Choose(true);
            Assert.AreEqual(0, changes);
            motion.Choose(null);
            Assert.IsFalse(motion.ReduceMotion);
            Assert.AreEqual(1, changes);
        }
    }
}
