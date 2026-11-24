using HypeDomino.UI.Motion;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class AnimatedImageTests
    {
        const float FramesPerSecond = 10f;

        GameObject root;
        Sprite still;
        Sprite first;
        Sprite second;
        LoopArt loop;

        [SetUp]
        public void BuildArt()
        {
            root = new GameObject("Screen", typeof(RectTransform));
            still = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);
            first = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 2, 2), Vector2.zero);
            second = Sprite.Create(Texture2D.whiteTexture, new Rect(2, 2, 2, 2), Vector2.zero);
            loop = new LoopArt(still, new[] { first, second }, FramesPerSecond);
        }

        [TearDown]
        public void DestroyScreen() => Object.Destroy(root);

        AnimatedImage NewImage()
        {
            var image = new GameObject("Avatar", typeof(RectTransform), typeof(Image)).AddComponent<AnimatedImage>();
            image.transform.SetParent(root.transform);
            return image;
        }

        [Test]
        public void ItPlaysTheLoopWhenAllowed()
        {
            var budget = new AnimationBudget(1);
            var image = NewImage();

            image.Show(loop, EffectLevel.Medium, false, budget, 10f);
            Assert.IsTrue(image.IsPlaying);
            Assert.AreSame(first, image.Shown);
            Assert.AreEqual(1, budget.Active);

            image.Advance(10.15f);
            Assert.AreSame(second, image.Shown);
            image.Advance(10.25f);
            Assert.AreSame(first, image.Shown);
        }

        [Test]
        public void TheLowLevelReduceMotionAndAStaticItemShowTheStillWithoutASlot()
        {
            var budget = new AnimationBudget(4);
            var image = NewImage();

            image.Show(loop, EffectLevel.Low, false, budget, 10f);
            Assert.AreSame(still, image.Shown);
            image.Show(loop, EffectLevel.High, true, budget, 10f);
            Assert.AreSame(still, image.Shown);
            image.Show(new LoopArt(still, null, FramesPerSecond), EffectLevel.High, false, budget, 10f);
            Assert.AreSame(still, image.Shown);

            Assert.IsFalse(image.IsPlaying);
            Assert.AreEqual(0, budget.Active);
            image.Advance(10.15f);
            Assert.AreSame(still, image.Shown);
        }

        [Test]
        public void ASpentBudgetShowsTheStillAndDisablingFreesTheSlot()
        {
            var budget = new AnimationBudget(1);
            var playing = NewImage();
            var waiting = NewImage();

            playing.Show(loop, EffectLevel.Medium, false, budget, 10f);
            waiting.Show(loop, EffectLevel.Medium, false, budget, 10f);
            Assert.IsFalse(waiting.IsPlaying);
            Assert.AreSame(still, waiting.Shown);

            playing.gameObject.SetActive(false);
            Assert.AreEqual(0, budget.Active);
            Assert.AreSame(still, playing.Shown, "a row hidden and shown again must not hold a mid-loop frame");

            waiting.Show(loop, EffectLevel.Medium, false, budget, 11f);
            Assert.IsTrue(waiting.IsPlaying);
        }

        [Test]
        public void ShowingAStillGivesBackTheSlot()
        {
            var budget = new AnimationBudget(1);
            var image = NewImage();

            image.Show(loop, EffectLevel.Medium, false, budget, 10f);
            image.ShowStill(still);

            Assert.IsFalse(image.IsPlaying);
            Assert.AreSame(still, image.Shown);
            Assert.AreEqual(0, budget.Active);
        }

        [Test]
        public void ShowingNewArtKeepsOneSlot()
        {
            var budget = new AnimationBudget(1);
            var image = NewImage();

            image.Show(loop, EffectLevel.Medium, false, budget, 10f);
            image.Show(loop, EffectLevel.High, false, budget, 12f);

            Assert.IsTrue(image.IsPlaying);
            Assert.AreEqual(1, budget.Active);
        }

        [Test]
        public void AdvancingAllocatesNothing()
        {
            var image = NewImage();
            image.Show(loop, EffectLevel.Medium, false, new AnimationBudget(1), 10f);
            image.Advance(10.15f);

            Assert.That(() => image.Advance(10.25f), Allocates.Nothing);
        }
    }
}
