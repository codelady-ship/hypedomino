using System;
using System.Collections.Generic;
using HypeDomino.Game.Match;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchResultKeysTests
    {
        static readonly string[] AllKeys =
        {
            MatchResultKeys.Won,
            MatchResultKeys.Lost,
            MatchResultKeys.Over,
            MatchResultKeys.PlayAgain,
            MatchResultKeys.BackToLobby,
        };

        [Test]
        public void TheTitleFollowsTheOutcome()
        {
            Assert.AreEqual(MatchResultKeys.Won, MatchResultKeys.Title(MatchOutcome.Won));
            Assert.AreEqual(MatchResultKeys.Lost, MatchResultKeys.Title(MatchOutcome.Lost));
            Assert.AreEqual(MatchResultKeys.Over, MatchResultKeys.Title(MatchOutcome.None));
        }

        [Test]
        public void EveryOutcomeHasATitle()
        {
            foreach (MatchOutcome outcome in Enum.GetValues(typeof(MatchOutcome)))
            {
                Assert.IsNotNull(MatchResultKeys.Title(outcome), outcome.ToString());
            }
        }

        [Test]
        public void TheKeysAreDistinct() =>
            Assert.AreEqual(AllKeys.Length, new HashSet<string>(AllKeys).Count);

        [Test]
        public void TheShippedBundleHasEveryKeyInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            foreach (var key in AllKeys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
            Assert.AreEqual("Qələbə", strings.Value(MatchResultKeys.Won, "az"));
            Assert.AreEqual("Məğlubiyyət", strings.Value(MatchResultKeys.Lost, "az"));
            Assert.AreEqual("Victory", strings.Value(MatchResultKeys.Won, GameLanguage.English));
        }
    }
}
