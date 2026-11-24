using HypeDomino.Game.Ranking;
using HypeDomino.Net;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class RankStandingTests
    {
        const ulong SeasonEndsUnixMs = 1_801_440_000_000;

        const string StandingJson = @"{""season"":{""id"":""season.2026_11"",""name_key"":""season.2026_11.name"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":true,""placement_played"":5,""placement_matches"":5,""rank"":""Gold II"",""tier"":""Gold"",""lp"":42,""division_lp"":100,""wins"":9},{""mode"":""2v2"",""placed"":false,""placement_played"":2,""placement_matches"":5,""rank"":"""",""tier"":"""",""lp"":0,""division_lp"":100,""wins"":1}]}";
        const string ApexJson = @"{""season"":{""id"":""season.2026_11"",""name_key"":""season.2026_11.name"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":true,""placement_played"":5,""placement_matches"":5,""rank"":""Grandmaster"",""tier"":""Grandmaster"",""lp"":340,""division_lp"":100,""wins"":80}]}";

        [Test]
        public void ReadsTheSeasonAndEachModeTheServerSends()
        {
            Assert.IsTrue(RankStanding.TryRead(StandingJson, out var standing));
            Assert.AreEqual("season.2026_11", standing.SeasonId);
            Assert.AreEqual("season.2026_11.name", standing.SeasonNameKey);
            Assert.AreEqual(SeasonEndsUnixMs, standing.EndsUnixMs);
            Assert.AreEqual(2, standing.ModeCount);

            Assert.IsTrue(standing.TryFind("1v1", out var solo));
            Assert.IsTrue(solo.Placed);
            Assert.AreEqual(RankTier.Gold, solo.Badge.Tier);
            Assert.AreEqual(2, solo.Badge.Division);
            Assert.AreEqual(42, solo.LP);
            Assert.AreEqual(100, solo.DivisionLP);
            Assert.AreEqual(9, solo.Wins);
            Assert.IsTrue(solo.HasLpBar);
            Assert.AreEqual(0.42f, solo.LpFill, 0.0001f);

            Assert.IsTrue(standing.TryFind("2v2", out var duo));
            Assert.IsFalse(duo.Placed);
            Assert.AreEqual(RankTier.None, duo.Badge.Tier);
            Assert.AreEqual(2, duo.PlacementPlayed);
            Assert.AreEqual(5, duo.PlacementMatches);
            Assert.IsFalse(duo.HasLpBar);
            Assert.AreEqual(0f, duo.LpFill);

            Assert.IsFalse(standing.TryFind("3v3", out _));
        }

        [Test]
        public void AnApexRankReadsItsTierAndHasNoLpBar()
        {
            Assert.IsTrue(RankStanding.TryRead(ApexJson, out var standing));
            var solo = standing.ModeAt(0);
            Assert.AreEqual(RankTier.Grandmaster, solo.Badge.Tier);
            Assert.AreEqual(RankBadge.NoDivision, solo.Badge.Division);
            Assert.AreEqual(340, solo.LP);
            Assert.IsFalse(solo.HasLpBar);
            Assert.AreEqual(0f, solo.LpFill);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("not json")]
        [TestCase(@"{""modes"":[]}")]
        [TestCase(@"{""season"":{""id"":"""",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""""},""modes"":[]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2027-02-01T00:00:00Z"",""ends_at"":""2026-11-01T00:00:00Z""},""modes"":[]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":"""",""placement_matches"":5,""division_lp"":100}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placement_matches"":0,""division_lp"":100}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placement_played"":6,""placement_matches"":5,""division_lp"":100}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placement_matches"":5,""division_lp"":0}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":true,""placement_played"":5,""placement_matches"":5,""rank"":""Gold II"",""tier"":""Gold"",""lp"":-1,""division_lp"":100}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":true,""placement_played"":5,""placement_matches"":5,""rank"":"""",""lp"":10,""division_lp"":100}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":false,""placement_matches"":5,""rank"":""Gold II"",""division_lp"":100}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":false,""placement_matches"":5,""lp"":20,""division_lp"":100}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":true,""placement_matches"":5,""placement_played"":5,""rank"":""Master"",""tier"":""Gold"",""division_lp"":100}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":true,""placement_matches"":5,""placement_played"":5,""rank"":""Gold 2"",""tier"":""Gold"",""division_lp"":100}]}")]
        [TestCase(@"{""season"":{""id"":""s"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placement_matches"":5,""division_lp"":100},{""mode"":""1v1"",""placement_matches"":5,""division_lp"":100}]}")]
        public void AnUnreadableReplyIsRefused(string json)
        {
            Assert.IsFalse(RankStanding.TryRead(json, out var standing));
            Assert.IsNull(standing);
        }

        [Test]
        public void TheSeasonCountdownRunsOnServerTime()
        {
            RankStanding.TryRead(StandingJson, out var standing);
            var clock = new ServerClock();
            Assert.IsFalse(standing.TrySecondsLeft(clock, 0, out _));
            Assert.IsFalse(standing.TrySecondsLeft(null, 0, out _));

            clock.Sync(SeasonEndsUnixMs - 90_000, 0, 0);
            Assert.IsTrue(standing.TrySecondsLeft(clock, 0, out var atSync));
            Assert.AreEqual(90, atSync, 0.001);
            Assert.IsTrue(standing.TrySecondsLeft(clock, 120, out var afterEnd));
            Assert.AreEqual(0, afterEnd, 0.001);
        }

        [Test]
        public void ReadingTheModesAllocatesNothing()
        {
            RankStanding.TryRead(StandingJson, out var standing);
            TestDelegate step = () =>
            {
                for (var i = 0; i < standing.ModeCount; i++) _ = standing.ModeAt(i).LpFill + standing.ModeAt(i).Wins;
                standing.TryFind("2v2", out var duo);
                _ = duo.PlacementPlayed;
            };
            step();
            Assert.That(step, Allocates.Nothing);
        }
    }
}
