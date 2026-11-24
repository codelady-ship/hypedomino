using System.Threading;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AllocatesTests
    {
        const int Repeats = 500;
        static object kept;
        static volatile bool othersAllocating;

        static void Nothing() { }

        static void AllocateThree()
        {
            kept = new object();
            kept = new object();
            kept = new object();
        }

        [Test]
        public void EveryAllocationOnTheCallingThreadIsCounted()
        {
            AllocateThree();

            Assert.AreEqual(3, AllocatesNothingConstraint.CountOnCallingThread(AllocateThree));
            Assert.IsFalse(Allocates.Nothing.ApplyTo((TestDelegate)AllocateThree).IsSuccess);
        }

        [Test]
        public void AStepThatAllocatesNothingPasses()
        {
            Nothing();

            for (var i = 0; i < Repeats; i++) Assert.That(Nothing, Allocates.Nothing);
        }

        [Test]
        public void AllocationsOnOtherThreadsAreNotCounted()
        {
            Nothing();
            othersAllocating = true;
            var other = new Thread(() =>
            {
                while (othersAllocating) kept = new byte[48];
            }) { IsBackground = true };
            other.Start();
            try
            {
                for (var i = 0; i < Repeats; i++) Assert.That(Nothing, Allocates.Nothing);
            }
            finally
            {
                othersAllocating = false;
                other.Join();
            }
        }
    }
}
