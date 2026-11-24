using System;
using System.Collections.Generic;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Emoji;
using HypeDomino.UI.Match;
using HypeDomino.UI.Motion;
using NUnit.Framework;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class EmojiArtShelfTests
    {
        const string Laugh = "emoji.laugh";
        const string Cry = "emoji.cry";
        const string Party = "emoji.party";
        const string LaughStill = "cosmetic/emoji/laugh/static";
        const string CryStill = "cosmetic/emoji/cry/static";
        const string PartyStill = "cosmetic/emoji/party/static";
        const string PartyLoop = "cosmetic/emoji/party/anim";

        const string Manifest = "{\"schemaVersion\":1,\"manifestVersion\":3,\"items\":["
            + "{\"id\":\"emoji.laugh\",\"type\":\"emoji\",\"availability\":\"live\",\"staticAddress\":\"cosmetic/emoji/laugh/static\"},"
            + "{\"id\":\"emoji.cry\",\"type\":\"emoji\",\"availability\":\"retired\",\"staticAddress\":\"cosmetic/emoji/cry/static\"},"
            + "{\"id\":\"emoji.party\",\"type\":\"emoji\",\"availability\":\"live\",\"animated\":true,\"staticAddress\":\"cosmetic/emoji/party/static\",\"animationAddress\":\"cosmetic/emoji/party/anim\"},"
            + "{\"id\":\"emoji.gone\",\"type\":\"emoji\",\"availability\":\"disabled\",\"staticAddress\":\"cosmetic/emoji/gone/static\"},"
            + "{\"id\":\"avatar.fox\",\"type\":\"avatar\",\"availability\":\"live\",\"staticAddress\":\"cosmetic/avatar/fox/static\"}"
            + "]}";

        sealed class Source<T> : ICosmeticArtSource<T> where T : class
        {
            readonly Dictionary<string, Action<T>> pending = new();

            public readonly List<string> Loads = new();
            public readonly List<string> Releases = new();

            public void Load(string address, Action<T> done)
            {
                Loads.Add(address);
                pending[address] = done;
            }

            public void Release(string address) => Releases.Add(address);

            public void Finish(string address, T art)
            {
                var done = pending[address];
                pending.Remove(address);
                done(art);
            }
        }

        Source<Sprite> stills;
        Source<LoopArt> loops;
        Texture2D texture;
        Sprite sprite;
        LoopArt loop;
        int shownCount;
        TableSide shownSide;
        LoopArt shownArt;

        [SetUp]
        public void CreateArt()
        {
            stills = new Source<Sprite>();
            loops = new Source<LoopArt>();
            texture = new Texture2D(4, 4);
            sprite = Sprite.Create(texture, new Rect(0, 0, 4, 4), Vector2.zero);
            loop = new LoopArt(sprite, new[] { sprite, sprite }, 12f);
            shownCount = 0;
            shownArt = null;
        }

        [TearDown]
        public void DestroyArt()
        {
            Object.DestroyImmediate(sprite);
            Object.DestroyImmediate(texture);
        }

        EmojiArtShelf Shelf(EffectLevel level)
        {
            Assert.IsTrue(CosmeticManifest.TryParse(Manifest, out var manifest));
            var stage = new CosmeticStage(stills, loops, null, null, null, level, false);
            return new EmojiArtShelf(manifest, stage, Record);
        }

        void Record(TableSide side, LoopArt art)
        {
            shownCount++;
            shownSide = side;
            shownArt = art;
        }

        [Test]
        public void APrefetchedEmojiPopsItsStaticArtAtOnceOnItsSide()
        {
            var shelf = Shelf(EffectLevel.Medium);
            shelf.Prefetch(new[] { Laugh, Cry });
            CollectionAssert.AreEqual(new[] { LaughStill, CryStill }, stills.Loads);
            stills.Finish(LaughStill, sprite);

            Assert.IsTrue(shelf.Show(TableSide.Left, Laugh));

            Assert.AreEqual(1, shownCount);
            Assert.AreEqual(TableSide.Left, shownSide);
            Assert.AreSame(sprite, shownArt.Still);
            Assert.IsFalse(shownArt.IsAnimated);
        }

        [Test]
        public void AnEmojiStillLoadingPopsWhenItsStaticArtArrives()
        {
            var shelf = Shelf(EffectLevel.Medium);

            Assert.IsTrue(shelf.Show(TableSide.Top, Laugh));
            Assert.AreEqual(0, shownCount);
            Assert.IsTrue(shelf.IsWaiting(TableSide.Top));

            stills.Finish(LaughStill, sprite);

            Assert.AreEqual(1, shownCount);
            Assert.AreEqual(TableSide.Top, shownSide);
            Assert.IsFalse(shelf.IsWaiting(TableSide.Top));
        }

        [Test]
        public void ANewerEmojiReplacesOneStillLoadingOnTheSameSide()
        {
            var shelf = Shelf(EffectLevel.Medium);
            shelf.Show(TableSide.Right, Laugh);
            shelf.Show(TableSide.Right, Cry);

            stills.Finish(LaughStill, sprite);
            Assert.AreEqual(0, shownCount);

            stills.Finish(CryStill, sprite);
            Assert.AreEqual(1, shownCount);
            Assert.AreEqual(TableSide.Right, shownSide);
        }

        [Test]
        public void OneLoadServesEverySideWaitingForTheSameEmoji()
        {
            var shelf = Shelf(EffectLevel.Medium);
            shelf.Show(TableSide.Bottom, Laugh);
            shelf.Show(TableSide.Top, Laugh);

            stills.Finish(LaughStill, sprite);

            CollectionAssert.AreEqual(new[] { LaughStill }, stills.Loads);
            Assert.AreEqual(2, shownCount);
        }

        [Test]
        public void AnAnimatedEmojiPlaysItsLoopOnceTheLoopIsLoaded()
        {
            var shelf = Shelf(EffectLevel.Medium);
            shelf.Show(TableSide.Top, Party);
            CollectionAssert.AreEqual(new[] { PartyLoop }, loops.Loads);

            stills.Finish(PartyStill, sprite);
            Assert.IsFalse(shownArt.IsAnimated);

            loops.Finish(PartyLoop, loop);
            shelf.Show(TableSide.Top, Party);

            Assert.AreSame(loop, shownArt);
            Assert.AreEqual(1, loops.Loads.Count);
        }

        [Test]
        public void TheLowLevelShowsTheStaticArtAndNeverLoadsTheLoop()
        {
            var shelf = Shelf(EffectLevel.Low);
            shelf.Show(TableSide.Top, Party);
            stills.Finish(PartyStill, sprite);

            Assert.IsEmpty(loops.Loads);
            Assert.IsFalse(shownArt.IsAnimated);
            Assert.AreSame(sprite, shownArt.Still);
        }

        [TestCase("emoji.missing")]
        [TestCase("emoji.gone")]
        [TestCase("avatar.fox")]
        [TestCase("")]
        [TestCase(null)]
        public void AnIdTheManifestDoesNotShowAsAnEmojiIsDropped(string itemId)
        {
            var shelf = Shelf(EffectLevel.Medium);

            Assert.IsFalse(shelf.Show(TableSide.Top, itemId));

            Assert.IsEmpty(stills.Loads);
            Assert.AreEqual(0, shelf.Count);
            Assert.IsFalse(shelf.IsWaiting(TableSide.Top));
        }

        [Test]
        public void AnEmojiWhoseStaticArtFailsIsDropped()
        {
            var shelf = Shelf(EffectLevel.Medium);
            shelf.Show(TableSide.Top, Laugh);

            stills.Finish(LaughStill, null);

            Assert.AreEqual(0, shownCount);
            Assert.IsFalse(shelf.IsWaiting(TableSide.Top));
            Assert.IsFalse(shelf.Show(TableSide.Top, Laugh));
        }

        [Test]
        public void ClearReleasesTheHeldArtAndIgnoresArtArrivingLater()
        {
            var shelf = Shelf(EffectLevel.Medium);
            shelf.Prefetch(new[] { Laugh });
            stills.Finish(LaughStill, sprite);
            shelf.Show(TableSide.Top, Party);

            shelf.Clear();
            stills.Finish(PartyStill, sprite);
            loops.Finish(PartyLoop, loop);

            Assert.AreEqual(0, shownCount);
            Assert.AreEqual(0, shelf.Count);
            CollectionAssert.AreEquivalent(new[] { LaughStill, PartyStill }, stills.Releases);
            CollectionAssert.AreEqual(new[] { PartyLoop }, loops.Releases);
        }

        [Test]
        public void ShowingALoadedEmojiDoesNotAllocate()
        {
            var shelf = Shelf(EffectLevel.Medium);
            shelf.Prefetch(new[] { Laugh, Party });
            stills.Finish(LaughStill, sprite);
            stills.Finish(PartyStill, sprite);
            loops.Finish(PartyLoop, loop);
            TestDelegate showBoth = () =>
            {
                for (var at = 0; at < 100; at++)
                {
                    shelf.Show(TableSide.Top, Laugh);
                    shelf.Show(TableSide.Left, Party);
                }
            };
            showBoth();

            Assert.That(showBoth, Allocates.Nothing);
        }
    }
}
