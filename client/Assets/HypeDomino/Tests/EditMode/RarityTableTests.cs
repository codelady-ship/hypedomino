using System.Collections.Generic;
using System.IO;
using HypeDomino.UI.Rarity;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class RarityTableTests
    {
        const string DefaultsPath = "Assets/HypeDomino/UI/Rarity/rarity-defaults.json";
        static readonly string[] TierIds = { "common", "rare", "epic", "legendary", "mythic" };
        const string ValidA = @"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.a.name"",""fill"":""#FF0000"",""on"":""#00FF00"",""text"":""#0000FF"",""pips"":1}";
        const string ValidB = @"{""id"":""b"",""ordinal"":2,""nameKey"":""rarity.b.name"",""fill"":""#FF0000"",""on"":""#00FF00"",""text"":""#0000FF"",""pips"":2}";

        static RarityTable Defaults() => RarityTable.Parse(File.ReadAllText(DefaultsPath));

        static RarityTable TableOf(params string[] entries) => RarityTable.Parse(@"{""tiers"":[" + string.Join(",", entries) + "]}");

        [Test]
        public void TheBuiltInDefaultsHoldEveryTierInOrder()
        {
            var table = Defaults();
            Assert.AreEqual(TierIds.Length, table.Count);
            for (var at = 0; at < TierIds.Length; at++)
            {
                Assert.IsTrue(table.TryGet(TierIds[at], out var tier));
                Assert.AreEqual(at + 1, tier.Ordinal);
                Assert.AreEqual($"rarity.{TierIds[at]}.name", tier.NameKey);
            }
        }

        [Test]
        public void EveryTierHasItsOwnPipPatternSoColourIsNeverTheOnlyCue()
        {
            var table = Defaults();
            var patterns = new HashSet<int>();
            foreach (var id in TierIds)
            {
                table.TryGet(id, out var tier);
                Assert.IsTrue(patterns.Add(tier.Pips), id);
            }
        }

        [Test]
        public void OnlyTheTopTierFlows()
        {
            var table = Defaults();
            foreach (var id in TierIds)
            {
                table.TryGet(id, out var tier);
                Assert.AreEqual(id == "mythic", tier.Flow, id);
            }
        }

        [Test]
        public void HexValuesBecomeOpaqueColours()
        {
            TableOf(ValidA).TryGet("a", out var tier);
            Assert.AreEqual(Color.red, tier.Fill);
            Assert.AreEqual(Color.green, tier.On);
            Assert.AreEqual(Color.blue, tier.Text);
        }

        [Test]
        public void AnUnknownOrEmptyIdHasNoTier()
        {
            var table = Defaults();
            Assert.IsFalse(table.TryGet("unknown", out _));
            Assert.IsFalse(table.TryGet("", out _));
            Assert.IsFalse(table.TryGet(null, out _));
        }

        [TestCase(@"{""id"":"""",""ordinal"":1,""nameKey"":""rarity..name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""Gold"",""ordinal"":1,""nameKey"":""rarity.Gold.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""1st"",""ordinal"":1,""nameKey"":""rarity.1st.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a-b"",""ordinal"":1,""nameKey"":""rarity.a-b.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":0,""nameKey"":""rarity.a.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":-1,""nameKey"":""rarity.a.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.b.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.a.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":0}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.a.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":7}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.a.name"",""fill"":""nope"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.a.name"",""fill"":""red"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.a.name"",""fill"":""#F00"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.a.name"",""fill"":""#FF000080"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.a.name"",""fill"":""#000000"",""on"":""#GG0000"",""text"":""#000000"",""pips"":1}")]
        [TestCase(@"{""id"":""a"",""ordinal"":1,""nameKey"":""rarity.a.name"",""fill"":""#000000"",""on"":""#000000"",""pips"":1}")]
        public void AnInvalidTierIsLeftOutSoItDrawsNoMark(string entry)
        {
            Assert.AreEqual(0, TableOf(entry).Count);
        }

        [Test]
        public void AnIdWithDigitsAndUnderscoresAndTheSixthPipPatternAreValid()
        {
            var table = TableOf(@"{""id"":""season_2"",""ordinal"":1,""nameKey"":""rarity.season_2.name"",""fill"":""#a3a6bf"",""on"":""#000000"",""text"":""#000000"",""pips"":6}");
            Assert.IsTrue(table.TryGet("season_2", out var tier));
            Assert.AreEqual(6, tier.Pips);
        }

        [Test]
        public void AnInvalidTierLeavesTheOthersDrawn()
        {
            var table = TableOf(ValidA, @"{""id"":""b"",""ordinal"":2,""nameKey"":""rarity.b.name"",""fill"":""red"",""on"":""#000000"",""text"":""#000000"",""pips"":2}");
            Assert.AreEqual(1, table.Count);
            Assert.IsTrue(table.TryGet("a", out _));
        }

        [TestCase(@"{""id"":""a"",""ordinal"":3,""nameKey"":""rarity.a.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":3}")]
        [TestCase(@"{""id"":""c"",""ordinal"":1,""nameKey"":""rarity.c.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":3}")]
        [TestCase(@"{""id"":""c"",""ordinal"":3,""nameKey"":""rarity.c.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}")]
        public void TiersThatShareAnIdOrdinalOrPipPatternAreAllLeftOut(string clash)
        {
            var table = TableOf(ValidA, ValidB, clash);
            Assert.AreEqual(1, table.Count);
            Assert.IsFalse(table.TryGet("a", out _));
            Assert.IsTrue(table.TryGet("b", out _));
        }

        [Test]
        public void TheClashRuleDoesNotDependOnTheOrderOfTiers()
        {
            const string clash = @"{""id"":""c"",""ordinal"":3,""nameKey"":""rarity.c.name"",""fill"":""#000000"",""on"":""#000000"",""text"":""#000000"",""pips"":1}";
            Assert.AreEqual(1, TableOf(clash, ValidB, ValidA).Count);
            Assert.AreEqual(1, TableOf(ValidA, clash, ValidB).Count);
        }

        [Test]
        public void AnInvalidTierDoesNotClashWithAValidOne()
        {
            var table = TableOf(ValidA, @"{""id"":""c"",""ordinal"":1,""nameKey"":""rarity.c.name"",""fill"":""red"",""on"":""#000000"",""text"":""#000000"",""pips"":1}");
            Assert.IsTrue(table.TryGet("a", out _));
        }

        [Test]
        public void TheFlowPeriodIsInSeconds()
        {
            Assert.AreEqual(2.4f, RarityTable.Parse(@"{""flowPeriodMs"":2400}").FlowPeriodSeconds, 0.0001f);
        }
    }
}
