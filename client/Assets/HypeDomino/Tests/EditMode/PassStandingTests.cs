using System;
using HypeDomino.Game.Pass;
using HypeDomino.Net;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PassStandingTests
    {
        const ulong SeasonEndsUnixMs = 1_801_440_000_000;

        const string MidLevelJson = @"{""season"":""season_1"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z"",""config_version"":4,""levels"":100,""xp"":1700,""level"":5,""level_xp"":1500,""next_level_xp"":2000,""premium"":false,
""tracks"":{""free"":{""1"":{""id"":""pass.season_1.free.1"",""revision"":1,""coins"":30,""cosmetics"":null,""medals"":null,""passXp"":0}},""premium"":{""1"":{""id"":""pass.season_1.premium.1"",""revision"":1,""coins"":0,""crystals"":6,""cosmetics"":[""avatar.s1_navigator""],""medals"":null,""passXp"":0}}}}";

        const string LastLevelJson = @"{""season"":""season_1"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00.5Z"",""config_version"":4,""levels"":100,""xp"":94200,""level"":100,""level_xp"":93750,""next_level_xp"":null,""premium"":true,""tracks"":{}}";

        static PassStanding Read(string json)
        {
            Assert.IsTrue(PassStanding.TryRead(PassReply.FromJson(json), out var standing));
            return standing;
        }

        static PassReply MidLevel() => PassReply.FromJson(MidLevelJson);

        [Test]
        public void TheStandingIsWhatTheServerSent()
        {
            var standing = Read(MidLevelJson);

            Assert.AreEqual("season_1", standing.Season);
            Assert.AreEqual(100, standing.Levels);
            Assert.AreEqual(5, standing.Level);
            Assert.AreEqual(1700, standing.Xp);
            Assert.IsFalse(standing.Premium);
            Assert.AreEqual(SeasonEndsUnixMs, standing.EndsUnixMs);
        }

        [Test]
        public void TheLevelPricesAreWhatTheServerSent()
        {
            var reply = MidLevel();
            reply.level_coin_price = 200;
            reply.level_crystal_price = 40;
            Assert.IsTrue(PassStanding.TryRead(reply, out var standing));

            Assert.AreEqual(200, standing.LevelCoinPrice);
            Assert.AreEqual(40, standing.LevelCrystalPrice);
            Assert.AreEqual(0, Read(MidLevelJson).LevelCoinPrice);
            Assert.AreEqual(0, Read(MidLevelJson).LevelCrystalPrice);
        }

        [Test]
        public void TheLevelBarFillsFromTheServerLevelBounds()
        {
            var standing = Read(MidLevelJson);

            Assert.IsFalse(standing.AtLastLevel);
            Assert.AreEqual(200, standing.XpIntoLevel);
            Assert.AreEqual(500, standing.XpForLevel);
            Assert.AreEqual(0.4f, standing.LevelFill, 0.0001f);
        }

        [Test]
        public void TheLastLevelIsFullAndHasNoNextLevel()
        {
            var standing = Read(LastLevelJson);

            Assert.IsTrue(standing.AtLastLevel);
            Assert.AreEqual(450, standing.XpIntoLevel);
            Assert.AreEqual(0, standing.XpForLevel);
            Assert.AreEqual(1f, standing.LevelFill);
            Assert.AreEqual(SeasonEndsUnixMs + 500, standing.EndsUnixMs);
        }

        [TestCase("2027-02-01T00:00:00Z", 0UL)]
        [TestCase("2027-02-01T00:00:00.123456789Z", 123UL)]
        [TestCase("2027-02-01T04:00:00+04:00", 0UL)]
        public void TheSeasonEndIsReadAsTheServerWritesIt(string endsAt, ulong extraMs)
        {
            var reply = MidLevel();
            reply.ends_at = endsAt;
            Assert.IsTrue(PassStanding.TryRead(reply, out var standing));

            Assert.AreEqual(SeasonEndsUnixMs + extraMs, standing.EndsUnixMs);
        }

        [TestCase(PassTrack.Free, 1, PassRewardState.Earned)]
        [TestCase(PassTrack.Free, 5, PassRewardState.Earned)]
        [TestCase(PassTrack.Free, 6, PassRewardState.Upcoming)]
        [TestCase(PassTrack.Free, 100, PassRewardState.Upcoming)]
        [TestCase(PassTrack.Premium, 1, PassRewardState.Locked)]
        [TestCase(PassTrack.Premium, 5, PassRewardState.Locked)]
        [TestCase(PassTrack.Premium, 6, PassRewardState.Locked)]
        public void WithoutPremiumThePremiumTrackIsLocked(PassTrack track, int level, PassRewardState state)
        {
            Assert.AreEqual(state, Read(MidLevelJson).StateAt(track, level));
        }

        [TestCase(PassTrack.Free, 5, PassRewardState.Earned)]
        [TestCase(PassTrack.Free, 6, PassRewardState.Upcoming)]
        [TestCase(PassTrack.Premium, 5, PassRewardState.Earned)]
        [TestCase(PassTrack.Premium, 6, PassRewardState.Upcoming)]
        public void WithPremiumBothTracksAreEarnedUpToTheLevel(PassTrack track, int level, PassRewardState state)
        {
            var reply = MidLevel();
            reply.premium = true;
            Assert.IsTrue(PassStanding.TryRead(reply, out var standing));

            Assert.AreEqual(state, standing.StateAt(track, level));
        }

        [Test]
        public void ALevelOutsideTheSeasonHasNoState()
        {
            var standing = Read(MidLevelJson);

            Assert.Throws<ArgumentOutOfRangeException>(() => standing.StateAt(PassTrack.Free, 0));
            Assert.Throws<ArgumentOutOfRangeException>(() => standing.StateAt(PassTrack.Premium, 101));
        }

        [Test]
        public void AReplyThatDoesNotHoldTogetherIsRefused()
        {
            Assert.IsFalse(PassStanding.TryRead(null, out _));
            AssertRefused(reply => reply.season = "");
            AssertRefused(reply => reply.levels = 0);
            AssertRefused(reply => reply.level = 0);
            AssertRefused(reply => reply.level = 101);
            AssertRefused(reply => reply.xp = 1499);
            AssertRefused(reply => reply.next_level_xp = 1700);
            AssertRefused(reply => reply.ends_at = null);
            AssertRefused(reply => reply.ends_at = "soon");
            AssertRefused(reply => reply.ends_at = "1970-01-01T00:00:00Z");
            AssertRefused(reply => reply.level_coin_price = -1);
            AssertRefused(reply => reply.level_crystal_price = -1);
        }

        static void AssertRefused(Action<PassReply> spoil)
        {
            var reply = MidLevel();
            spoil(reply);
            Assert.IsFalse(PassStanding.TryRead(reply, out var standing));
            Assert.IsNull(standing);
        }

        [Test]
        public void TheSeasonCountsDownOnServerTime()
        {
            var standing = Read(MidLevelJson);
            var clock = new ServerClock();
            clock.Sync(SeasonEndsUnixMs - 90_000, 0, 0);

            Assert.IsTrue(standing.TrySecondsLeft(clock, 0, out var atSync));
            Assert.AreEqual(90, atSync, 0.001);
            Assert.IsTrue(standing.TrySecondsLeft(clock, 30.5, out var later));
            Assert.AreEqual(59.5, later, 0.001);
            Assert.IsTrue(standing.TrySecondsLeft(clock, 120, out var past));
            Assert.AreEqual(0, past);
        }

        [Test]
        public void NoCountdownRunsBeforeTheClockIsSynced()
        {
            var standing = Read(MidLevelJson);

            Assert.IsFalse(standing.TrySecondsLeft(new ServerClock(), 10, out var seconds));
            Assert.AreEqual(0, seconds);
            Assert.IsFalse(standing.TrySecondsLeft(null, 10, out _));
        }

        [Test]
        public void TickingTheCountdownAndReadingTheTracksAllocatesNothing()
        {
            var standing = Read(MidLevelJson);
            var clock = new ServerClock();
            clock.Sync(SeasonEndsUnixMs - 90_000, 0, 0);
            var now = 0.0;
            TestDelegate step = () =>
            {
                standing.TrySecondsLeft(clock, now, out _);
                for (var level = 1; level <= standing.Levels; level++)
                {
                    standing.StateAt(PassTrack.Free, level);
                    standing.StateAt(PassTrack.Premium, level);
                }
                _ = standing.LevelFill;
                now += 0.016;
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
