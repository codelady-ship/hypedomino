using HypeDomino.Content;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ContentCacheTests
    {
        sealed class FakeStore : ICacheStore
        {
            public bool InUse;
            public int Clears;
            public long LimitBytes;
            public int UnusedForSeconds;

            public long OccupiedBytes { get; set; }

            public bool ClearAtStart { get; set; }

            public void ApplyLimits(long limitBytes, int unusedForSeconds)
            {
                LimitBytes = limitBytes;
                UnusedForSeconds = unusedForSeconds;
            }

            public bool TryClear()
            {
                Clears++;
                return !InUse;
            }
        }

        FakeStore store;
        bool inMatch;
        ContentCache cache;

        [SetUp]
        public void Create()
        {
            store = new FakeStore();
            inMatch = false;
            cache = new ContentCache(store, () => inMatch);
        }

        [Test]
        public void StartCapsTheCacheAtOneGigabyteAndThirtyDaysUnused()
        {
            cache.Start();
            Assert.AreEqual(1_000_000_000L, store.LimitBytes);
            Assert.AreEqual(2_592_000, store.UnusedForSeconds);
        }

        [Test]
        public void AClearOutsideAMatchClearsAtOnce()
        {
            Assert.AreEqual(CacheClearResult.Cleared, cache.Clear());
            Assert.AreEqual(1, store.Clears);
            Assert.IsFalse(store.ClearAtStart);
        }

        [Test]
        public void AClearDuringAMatchDoesNothing()
        {
            inMatch = true;
            Assert.AreEqual(CacheClearResult.NotInMatch, cache.Clear());
            Assert.AreEqual(0, store.Clears);
            Assert.IsFalse(store.ClearAtStart);
        }

        [Test]
        public void ABundleInUseMovesTheClearToTheNextStart()
        {
            store.InUse = true;
            Assert.AreEqual(CacheClearResult.AtNextStart, cache.Clear());
            Assert.IsTrue(store.ClearAtStart);
            store.InUse = false;
            cache.Start();
            Assert.AreEqual(2, store.Clears);
            Assert.IsFalse(store.ClearAtStart);
        }

        [Test]
        public void AClearThatSucceedsLaterCancelsThePendingClear()
        {
            store.InUse = true;
            Assert.AreEqual(CacheClearResult.AtNextStart, cache.Clear());
            store.InUse = false;
            Assert.AreEqual(CacheClearResult.Cleared, cache.Clear());
            Assert.IsFalse(store.ClearAtStart);
            cache.Start();
            Assert.AreEqual(2, store.Clears);
        }

        [Test]
        public void AClearThatFailsAtStartIsTriedAgainNextStart()
        {
            store.ClearAtStart = true;
            store.InUse = true;
            cache.Start();
            Assert.IsTrue(store.ClearAtStart);
        }

        [Test]
        public void StartWithoutAPendingClearKeepsTheCache()
        {
            cache.Start();
            Assert.AreEqual(0, store.Clears);
        }

        [Test]
        public void TheShownSizeIsWhatTheStoreHolds()
        {
            store.OccupiedBytes = 123_456;
            Assert.AreEqual(123_456L, cache.OccupiedBytes);
        }
    }
}
