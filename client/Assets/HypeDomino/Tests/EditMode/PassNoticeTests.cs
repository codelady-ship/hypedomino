using HypeDomino.UI.Match;
using HypeDomino.UI.Tokens;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PassNoticeTests
    {
        [Test]
        public void TheNoticeLastsPopInHoldAndExit() =>
            Assert.AreEqual(TokenDurationMs.Fast + TokenDurationMs.Slow + TokenDurationMs.Base, PassNotice.LifetimeMs);

        [Test]
        public void TheNoticePopsInHoldsAndFadesOut()
        {
            Assert.IsTrue(PassNotice.At(0f, false, out var start));
            Assert.AreEqual(0f, start.Scale);
            Assert.AreEqual(1f, start.Alpha);

            var overshot = false;
            for (var ms = 1f; ms < TokenDurationMs.Fast; ms += 5f)
            {
                PassNotice.At(ms, false, out var popping);
                overshot |= popping.Scale > 1f;
            }
            Assert.IsTrue(overshot);

            Assert.IsTrue(PassNotice.At(TokenDurationMs.Fast + TokenDurationMs.Slow, false, out var held));
            Assert.AreEqual(1f, held.Scale);
            Assert.AreEqual(1f, held.Alpha);

            Assert.IsTrue(PassNotice.At(PassNotice.LifetimeMs - 1f, false, out var leaving));
            Assert.Less(leaving.Alpha, 0.1f);

            Assert.IsFalse(PassNotice.At(PassNotice.LifetimeMs, false, out var gone));
            Assert.AreEqual(0f, gone.Alpha);
        }

        [Test]
        public void ReduceMotionFadesInstead()
        {
            Assert.IsTrue(PassNotice.At(0f, true, out var start));
            Assert.AreEqual(1f, start.Scale);
            Assert.AreEqual(0f, start.Alpha);

            for (var ms = 0f; ms < PassNotice.LifetimeMs; ms += 10f)
            {
                PassNotice.At(ms, true, out var notice);
                Assert.AreEqual(1f, notice.Scale, ms.ToString());
            }

            Assert.IsTrue(PassNotice.At(TokenDurationMs.Fast, true, out var shown));
            Assert.AreEqual(1f, shown.Alpha);
        }

        [Test]
        public void EvaluatingTheNoticeAllocatesNothing()
        {
            var alpha = 0f;
            TestDelegate frame = () =>
            {
                PassNotice.At(300f, false, out var notice);
                alpha += notice.Alpha;
            };
            frame();

            Assert.That(frame, Allocates.Nothing);
        }
    }
}
