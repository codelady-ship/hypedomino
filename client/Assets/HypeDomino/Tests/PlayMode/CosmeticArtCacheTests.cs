using System.Collections.Generic;
using HypeDomino.UI.Cosmetics;
using NUnit.Framework;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class CosmeticArtCacheTests
    {
        const string Address = "cosmetic/avatar/a/static";
        const string Art = "art";

        FakeArtSource<string> source;
        CosmeticArtCache<string> cache;
        List<string> received;

        [SetUp]
        public void Build()
        {
            source = new FakeArtSource<string>();
            cache = new CosmeticArtCache<string>(source);
            received = new List<string>();
        }

        [Test]
        public void NothingLoadsUntilItIsAskedFor()
        {
            Assert.IsEmpty(source.Loads);
        }

        [Test]
        public void TwoViewsOfOneItemShareOneLoad()
        {
            cache.Acquire(Address, received.Add);
            cache.Acquire(Address, received.Add);
            Assert.AreEqual(1, source.Loads.Count);
            Assert.IsEmpty(received);

            source.Finish(Address, Art);
            Assert.AreEqual(new[] { Art, Art }, received);
        }

        [Test]
        public void ALoadedItemIsGivenAtOnceWithoutLoadingAgain()
        {
            cache.Acquire(Address, received.Add);
            source.Finish(Address, Art);
            cache.Acquire(Address, received.Add);
            Assert.AreEqual(1, source.Loads.Count);
            Assert.AreEqual(2, received.Count);
        }

        [Test]
        public void TheLastReleaseFreesTheArt()
        {
            cache.Acquire(Address, received.Add);
            cache.Acquire(Address, received.Add);
            source.Finish(Address, Art);

            cache.Release(Address);
            Assert.IsEmpty(source.Releases);
            cache.Release(Address);
            Assert.AreEqual(new[] { Address }, source.Releases);
        }

        [Test]
        public void ArtReleasedWhileLoadingIsFreedWhenTheLoadEnds()
        {
            cache.Acquire(Address, received.Add);
            cache.Release(Address);
            Assert.IsEmpty(source.Releases);

            source.Finish(Address, Art);
            Assert.AreEqual(new[] { Address }, source.Releases);
        }

        [Test]
        public void AFailedLoadGivesNothingAndIsTriedAgainAfterItsLastRelease()
        {
            cache.Acquire(Address, received.Add);
            source.Finish(Address, null);
            Assert.AreEqual(new string[] { null }, received);

            cache.Release(Address);
            Assert.IsEmpty(source.Releases, "nothing was loaded, so nothing is freed");

            cache.Acquire(Address, received.Add);
            Assert.AreEqual(2, source.Loads.Count);
        }

        [Test]
        public void ReleasingAnUnknownAddressDoesNothing()
        {
            cache.Release(Address);
            Assert.IsEmpty(source.Releases);
        }
    }
}
