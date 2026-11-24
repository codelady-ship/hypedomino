using System;
using System.Collections.Generic;
using HypeDomino.Content;
using HypeDomino.Game;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class CacheClearKeysTests
    {
        const string GameStringKeyFormat = "^[a-z0-9_]+(\\.[a-z0-9_]+)+$";

        sealed class BusyStore : ICacheStore
        {
            public long OccupiedBytes => 0;

            public bool ClearAtStart { get; set; }

            public void ApplyLimits(long limitBytes, int unusedForSeconds)
            {
            }

            public bool TryClear() => false;
        }

        [Test]
        public void EachResultNamesItsOwnKey()
        {
            Assert.AreEqual(CacheClearKeys.Cleared, CacheClearKeys.Of(CacheClearResult.Cleared));
            Assert.AreEqual(CacheClearKeys.AtNextStart, CacheClearKeys.Of(CacheClearResult.AtNextStart));
            Assert.AreEqual(CacheClearKeys.NotInMatch, CacheClearKeys.Of(CacheClearResult.NotInMatch));
        }

        [Test]
        public void EveryResultHasADistinctKeyInTheGameStringFormat()
        {
            var keys = new HashSet<string>();
            foreach (CacheClearResult result in Enum.GetValues(typeof(CacheClearResult)))
            {
                var key = CacheClearKeys.Of(result);
                StringAssert.IsMatch(GameStringKeyFormat, key, result.ToString());
                Assert.IsTrue(keys.Add(key), result.ToString());
            }
        }

        [Test]
        public void TheShippedBundleHasEveryCacheClearKeyInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            foreach (CacheClearResult result in Enum.GetValues(typeof(CacheClearResult)))
            {
                var key = CacheClearKeys.Of(result);
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }

        [Test]
        public void AClearBlockedByABundleInUseTellsThePlayerItRunsAtTheNextStart()
        {
            var store = new BusyStore();
            var cache = new ContentCache(store, () => false);
            Assert.AreEqual(CacheClearKeys.AtNextStart, CacheClearKeys.Of(cache.Clear()));
            Assert.IsTrue(store.ClearAtStart);
        }

        [Test]
        public void AClearDuringAMatchTellsThePlayerToFinishTheMatchFirst()
        {
            var store = new BusyStore();
            var cache = new ContentCache(store, () => true);
            Assert.AreEqual(CacheClearKeys.NotInMatch, CacheClearKeys.Of(cache.Clear()));
            Assert.IsFalse(store.ClearAtStart);
        }
    }
}
