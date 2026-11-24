using HypeDomino.UI.Emoji;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class EmojiTests
    {
        static readonly string[] Equipped = { "emoji.laugh", "emoji.cry", "emoji.wave" };

        [Test]
        public void TheWheelHoldsTheEquippedEmojiUpToItsSlots()
        {
            var wheel = new EmojiWheel(2);
            Assert.IsFalse(wheel.Visible);
            wheel.Toggle();
            Assert.IsFalse(wheel.IsOpen);

            wheel.Equip(Equipped);
            Assert.AreEqual(2, wheel.Count);
            Assert.AreEqual("emoji.cry", wheel.Slot(1));
            wheel.Toggle();
            Assert.IsTrue(wheel.IsOpen);

            wheel.Equip(new string[0]);
            Assert.IsFalse(wheel.Visible);
            Assert.IsFalse(wheel.IsOpen);
            Assert.IsNull(wheel.Slot(0));
        }

        [TestCase(6, 3)]
        [TestCase(2, 2)]
        [TestCase(0, 0)]
        [TestCase(-1, 0)]
        public void TheWheelHoldsNoMoreThanTheSlotCount(int slotCount, int expected)
        {
            var wheel = new EmojiWheel(4);
            wheel.Equip(Equipped, slotCount);
            Assert.AreEqual(expected, wheel.Count);
            for (var slot = expected; slot < 4; slot++) Assert.IsNull(wheel.Slot(slot));
        }

        [Test]
        public void ASendClosesTheWheelAndCalmsItForTheEmojiDuration()
        {
            var wheel = new EmojiWheel(6);
            wheel.Equip(Equipped);
            Assert.IsFalse(wheel.TrySend(0, 0, out _), "a closed wheel sends nothing");

            wheel.Toggle();
            Assert.IsFalse(wheel.TrySend(3, 0, out _), "an empty slot sends nothing");
            Assert.IsTrue(wheel.TrySend(1, 1000, out var item));
            Assert.AreEqual("emoji.cry", item);
            Assert.IsFalse(wheel.IsOpen);

            wheel.Toggle();
            Assert.IsTrue(wheel.IsOpen, "the wheel opens while calm");
            Assert.IsTrue(wheel.Calm(1000 + TokenEffect.EmojiDurationMs - 1));
            Assert.IsFalse(wheel.TrySend(0, 1000 + TokenEffect.EmojiDurationMs - 1, out _));
            Assert.IsTrue(wheel.TrySend(0, 1000 + TokenEffect.EmojiDurationMs, out item));
            Assert.AreEqual("emoji.laugh", item);
        }

        [Test]
        public void TheOwnEmojiArrivingLateKeepsTheWheelCalmUntilItEnds()
        {
            var wheel = new EmojiWheel(6);
            wheel.Equip(Equipped);
            wheel.Toggle();
            wheel.TrySend(0, 0, out _);
            wheel.OwnEmojiShown(300);
            Assert.IsTrue(wheel.Calm(TokenEffect.EmojiDurationMs + 299));
            Assert.IsFalse(wheel.Calm(TokenEffect.EmojiDurationMs + 300));
        }

        [Test]
        public void EasingFollowsTheTokenCurves()
        {
            Assert.AreEqual(0f, Easing.Evaluate(TokenEasing.Standard, 0f));
            Assert.AreEqual(1f, Easing.Evaluate(TokenEasing.Standard, 1f));
            Assert.AreEqual(0.5f, Easing.Evaluate(TokenEasing.Linear, 0.5f), 1e-4f);
            Assert.Greater(Easing.Evaluate(TokenEasing.Overshoot, 0.6f), 1f, "overshoot passes its end value");
            var previous = 0f;
            for (var step = 1; step <= 10; step++)
            {
                var value = Easing.Evaluate(TokenEasing.Standard, step / 10f);
                Assert.GreaterOrEqual(value, previous);
                previous = value;
            }
        }

        [Test]
        public void ThePopScalesInHoldsAndFadesOut()
        {
            Assert.IsTrue(EmojiPop.At(0f, false, out var pop));
            Assert.AreEqual(0f, pop.Scale);
            Assert.AreEqual(1f, pop.Alpha);

            EmojiPop.At(TokenDurationMs.Base, false, out pop);
            Assert.AreEqual(1f, pop.Scale);

            EmojiPop.At(TokenEffect.EmojiDurationMs / 2f, false, out pop);
            Assert.AreEqual(1f, pop.Scale);
            Assert.AreEqual(1f, pop.Alpha);

            EmojiPop.At(TokenEffect.EmojiDurationMs - TokenDurationMs.Base / 2f, false, out pop);
            Assert.That(pop.Alpha, Is.InRange(0.01f, 0.99f));

            Assert.IsFalse(EmojiPop.At(TokenEffect.EmojiDurationMs, false, out pop));
            Assert.AreEqual(0f, pop.Alpha);
        }

        [Test]
        public void ReduceMotionFadesInWithoutScaling()
        {
            EmojiPop.At(0f, true, out var pop);
            Assert.AreEqual(1f, pop.Scale);
            Assert.AreEqual(0f, pop.Alpha);
            EmojiPop.At(TokenDurationMs.Fast, true, out pop);
            Assert.AreEqual(1f, pop.Alpha);
            EmojiPop.At(TokenDurationMs.Base / 2f, true, out pop);
            Assert.AreEqual(1f, pop.Scale);
        }

        [Test]
        public void TheLowLevelShowsTheStaticArtAndOthersPlayTheLoop()
        {
            var still = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);
            var first = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 2, 2), Vector2.zero);
            var second = Sprite.Create(Texture2D.whiteTexture, new Rect(2, 2, 2, 2), Vector2.zero);
            var art = new LoopArt(still, new[] { first, second }, 10f);

            Assert.AreSame(still, art.Frame(EffectLevel.Low, 0.15f));
            Assert.AreSame(first, art.Frame(EffectLevel.Medium, 0.05f));
            Assert.AreSame(second, art.Frame(EffectLevel.High, 0.15f));
            Assert.AreSame(first, art.Frame(EffectLevel.High, 0.25f));
            Assert.AreSame(still, new LoopArt(still, null, 10f).Frame(EffectLevel.High, 0.15f), "a loop not loaded yet shows the static art");
        }
    }
}
