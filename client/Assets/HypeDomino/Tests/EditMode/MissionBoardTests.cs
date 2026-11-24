using System;
using HypeDomino.Net;
using HypeDomino.UI.Missions;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MissionBoardTests
    {
        const string BoardJson = @"{""version"":7,""day"":""2026-11-14"",""week"":""2026-W46"",""missions"":[
{""mission"":{""id"":""mission.achievement_wins_10"",""revision"":1,""type"":""achievement"",""nameKey"":""mission.achievement_wins_10.name"",""event"":""match_won"",""target"":10,""filters"":{},""bundle"":{""id"":""bundle.wins_10"",""revision"":1,""coins"":100,""cosmetics"":[],""medals"":[],""passXp"":0},""availability"":""live""},""period"":""once"",""progress"":12,""completed_at"":""2026-11-02T18:04:11.5Z""},
{""mission"":{""id"":""mission.daily_play_1"",""revision"":2,""type"":""daily"",""nameKey"":""mission.daily_play_1.name"",""event"":""match_completed"",""target"":1,""filters"":{},""slot"":1,""weight"":1,""availability"":""live""},""period"":""2026-11-14"",""progress"":0,""completed_at"":null},
{""mission"":{""id"":""mission.daily_points_150"",""revision"":1,""type"":""daily"",""nameKey"":""mission.daily_points_150.name"",""event"":""board_points"",""target"":150,""filters"":{""queue"":""ranked""},""slot"":3,""weight"":2,""availability"":""live""},""period"":""2026-11-14"",""progress"":45,""completed_at"":null},
{""mission"":{""id"":""mission.event_spin"",""revision"":1,""type"":""event"",""nameKey"":""mission.event_spin.name"",""event"":""match_won"",""target"":3,""filters"":{},""eventId"":""event.spin"",""availability"":""live""},""period"":""event.spin"",""progress"":1,""completed_at"":null},
{""mission"":{""id"":""mission.weekly_wins_15"",""revision"":1,""type"":""weekly"",""nameKey"":""mission.weekly_wins_15.name"",""event"":""match_won"",""target"":15,""filters"":{},""slot"":2,""weight"":1,""availability"":""live""},""period"":""2026-W46"",""progress"":15,""completed_at"":""2026-11-13T09:00:00Z""}],
""daily_rerolls"":1,""weekly_rerolls"":0}";

        static MissionBoard Board(string json = BoardJson)
        {
            Assert.IsTrue(MissionBoard.TryFrom(MissionBoardReply.FromJson(json), out var board));
            return board;
        }

        static ulong UnixMs(int year, int month, int day) => (ulong)new DateTimeOffset(year, month, day, 0, 0, 0, TimeSpan.Zero).ToUnixTimeMilliseconds();

        [Test]
        public void CardsGoToTheirTabInServerOrderAndUnknownTypesAreLeftOut()
        {
            var board = Board();
            Assert.AreEqual(7, board.Version);
            Assert.AreEqual(2, board.Count(MissionTab.Daily));
            Assert.AreEqual(1, board.Count(MissionTab.Weekly));
            Assert.AreEqual(1, board.Count(MissionTab.Achievement));
            Assert.AreEqual("mission.daily_play_1", board.Card(MissionTab.Daily, 0).Id);
            Assert.AreEqual("mission.daily_points_150", board.Card(MissionTab.Daily, 1).Id);
            Assert.AreEqual("mission.weekly_wins_15", board.Card(MissionTab.Weekly, 0).Id);
            Assert.AreEqual("mission.achievement_wins_10", board.Card(MissionTab.Achievement, 0).Id);
        }

        [Test]
        public void ACardShowsTheServerProgressAndCompletion()
        {
            var board = Board();
            var open = board.Card(MissionTab.Daily, 1);
            Assert.AreEqual("mission.daily_points_150.name", open.NameKey);
            Assert.AreEqual(MissionTab.Daily, open.Tab);
            Assert.AreEqual(150, open.Target);
            Assert.AreEqual(45, open.Progress);
            Assert.AreEqual(45, open.ShownProgress);
            Assert.AreEqual(0.3f, open.Fill, 1e-6f);
            Assert.IsFalse(open.Completed);
            var fresh = board.Card(MissionTab.Daily, 0);
            Assert.IsFalse(fresh.Completed);
            Assert.AreEqual(0f, fresh.Fill);
            var achieved = board.Card(MissionTab.Achievement, 0);
            Assert.IsTrue(achieved.Completed);
            Assert.AreEqual(12, achieved.Progress);
            Assert.AreEqual(10, achieved.ShownProgress);
            Assert.AreEqual(1f, achieved.Fill);
        }

        [Test]
        public void OnlyAnOpenDailyOrWeeklyMissionWithARerollLeftOffersTheReroll()
        {
            var board = Board();
            Assert.AreEqual(1, board.RerollsLeft(MissionTab.Daily));
            Assert.AreEqual(0, board.RerollsLeft(MissionTab.Weekly));
            Assert.AreEqual(0, board.RerollsLeft(MissionTab.Achievement));
            Assert.IsTrue(board.CanReroll(board.Card(MissionTab.Daily, 0)));
            Assert.IsFalse(board.CanReroll(board.Card(MissionTab.Weekly, 0)));
            Assert.IsFalse(board.CanReroll(board.Card(MissionTab.Achievement, 0)));
            var spent = Board(BoardJson.Replace(@"""daily_rerolls"":1,""weekly_rerolls"":0", @"""daily_rerolls"":0,""weekly_rerolls"":1"));
            Assert.IsFalse(spent.CanReroll(spent.Card(MissionTab.Daily, 0)));
            Assert.IsFalse(spent.CanReroll(spent.Card(MissionTab.Weekly, 0)));
        }

        [Test]
        public void TheRerollIntentNamesOnlyTheMission()
        {
            Assert.AreEqual("/missions", MissionBoard.BoardPath);
            Assert.AreEqual("/missions/reroll", MissionBoard.RerollPath);
            Assert.AreEqual(@"{""mission"":""mission.daily_play_1""}", RerollRequest.Json("mission.daily_play_1"));
        }

        [Test]
        public void ResetsAreTheEndsOfTheServerDayAndIsoWeekAtMidnightUtc()
        {
            var board = Board();
            Assert.AreEqual(UnixMs(2026, 11, 15), board.DailyResetUnixMs);
            Assert.AreEqual(UnixMs(2026, 11, 16), board.WeeklyResetUnixMs);
        }

        static ServerClock ClockAt(ulong serverUnixMs)
        {
            var clock = new ServerClock();
            clock.Sync(serverUnixMs, 0, 0);
            return clock;
        }

        [Test]
        public void DailyAndWeeklyTabsCountDownToTheirResetOnServerTime()
        {
            var board = Board();
            var clock = ClockAt(UnixMs(2026, 11, 15) - 90_000);
            Assert.AreEqual(board.DailyResetUnixMs, board.ResetUnixMs(MissionTab.Daily));
            Assert.AreEqual(board.WeeklyResetUnixMs, board.ResetUnixMs(MissionTab.Weekly));
            Assert.IsTrue(board.TrySecondsUntilReset(MissionTab.Daily, clock, 0, out var daily));
            Assert.AreEqual(90, daily, 0.001);
            Assert.IsTrue(board.TrySecondsUntilReset(MissionTab.Weekly, clock, 30.5, out var weekly));
            Assert.AreEqual(86_459.5, weekly, 0.001);
            Assert.IsTrue(board.TrySecondsUntilReset(MissionTab.Daily, clock, 120, out var past));
            Assert.AreEqual(0, past);
        }

        [Test]
        public void TheAchievementTabHasNoResetCountdown()
        {
            var board = Board();
            Assert.AreEqual(0UL, board.ResetUnixMs(MissionTab.Achievement));
            Assert.IsFalse(board.TrySecondsUntilReset(MissionTab.Achievement, ClockAt(UnixMs(2026, 11, 14)), 0, out var seconds));
            Assert.AreEqual(0, seconds);
        }

        [Test]
        public void NoResetCountdownRunsBeforeTheClockIsSynced()
        {
            var board = Board();
            Assert.IsFalse(board.TrySecondsUntilReset(MissionTab.Daily, new ServerClock(), 10, out var seconds));
            Assert.AreEqual(0, seconds);
            Assert.IsFalse(board.TrySecondsUntilReset(MissionTab.Weekly, null, 10, out _));
            Assert.IsFalse(board.IsPastReset(new ServerClock(), 10));
            Assert.IsFalse(board.IsPastReset(null, 10));
        }

        [Test]
        public void TheBoardIsPastItsResetOnceTheServerDayEnds()
        {
            var board = Board();
            var clock = ClockAt(UnixMs(2026, 11, 15) - 1_000);
            Assert.IsFalse(board.IsPastReset(clock, 0));
            Assert.IsFalse(board.IsPastReset(clock, 0.5));
            Assert.IsTrue(board.IsPastReset(clock, 1));
            Assert.IsTrue(board.IsPastReset(clock, 3_600));
        }

        [Test]
        public void TickingTheResetCountdownsAllocatesNothing()
        {
            var board = Board();
            var clock = ClockAt(UnixMs(2026, 11, 15) - 90_000);
            var now = 0.0;
            var sink = 0.0;
            TestDelegate tick = () =>
            {
                board.TrySecondsUntilReset(MissionTab.Daily, clock, now, out var daily);
                board.TrySecondsUntilReset(MissionTab.Weekly, clock, now, out var weekly);
                board.TrySecondsUntilReset(MissionTab.Achievement, clock, now, out var none);
                sink += daily + weekly + none + (board.IsPastReset(clock, now) ? 1 : 0);
                now += 0.016;
            };
            tick();

            Assert.That(tick, Allocates.Nothing);
            Assert.Greater(sink, 0d);
        }

        [TestCase("2026-12-31", "2026-W53", 2027, 1, 1, 2027, 1, 4)]
        [TestCase("2024-12-31", "2025-W01", 2025, 1, 1, 2025, 1, 6)]
        [TestCase("2027-01-04", "2027-W01", 2027, 1, 5, 2027, 1, 11)]
        [TestCase("2028-02-29", "2028-W09", 2028, 3, 1, 2028, 3, 6)]
        public void ResetsCrossMonthAndYearBoundaries(string day, string week, int dayYear, int dayMonth, int dayDate, int weekYear, int weekMonth, int weekDate)
        {
            var board = Board(BoardJson.Replace("2026-11-14", day).Replace("2026-W46", week));
            Assert.AreEqual(UnixMs(dayYear, dayMonth, dayDate), board.DailyResetUnixMs);
            Assert.AreEqual(UnixMs(weekYear, weekMonth, weekDate), board.WeeklyResetUnixMs);
        }

        [TestCase(@"""day"":""2026-11-14""", @"""day"":""14.11.2026""")]
        [TestCase(@"""day"":""2026-11-14""", @"""day"":""""")]
        [TestCase(@"""week"":""2026-W46""", @"""week"":""2026-46""")]
        [TestCase(@"""week"":""2026-W46""", @"""week"":""2026-W54""")]
        [TestCase(@"""week"":""2026-W46""", @"""week"":""2026-W-1""")]
        [TestCase(@"""week"":""2026-W46""", @"""week"":null")]
        public void ABoardWithoutAReadablePeriodIsRefused(string sent, string broken)
        {
            Assert.IsFalse(MissionBoard.TryFrom(MissionBoardReply.FromJson(BoardJson.Replace(sent, broken)), out var board));
            Assert.IsNull(board);
            Assert.IsFalse(MissionBoard.TryFrom(null, out _));
        }

        [Test]
        public void AnEmptyBoardHasEmptyTabs()
        {
            var board = Board(@"{""version"":1,""day"":""2026-11-14"",""week"":""2026-W46"",""missions"":[],""daily_rerolls"":1,""weekly_rerolls"":1}");
            Assert.AreEqual(0, board.Count(MissionTab.Daily));
            Assert.AreEqual(0, board.Count(MissionTab.Weekly));
            Assert.AreEqual(0, board.Count(MissionTab.Achievement));
        }

        [Test]
        public void ReadingTheBoardAllocatesNothing()
        {
            var board = Board();
            var sink = 0f;
            TestDelegate read = () =>
            {
                for (var tab = MissionTab.Daily; tab <= MissionTab.Achievement; tab++)
                {
                    for (var i = 0; i < board.Count(tab); i++)
                    {
                        var card = board.Card(tab, i);
                        sink += card.Fill + card.ShownProgress + (board.CanReroll(card) ? 1 : 0);
                    }
                }
                sink += board.DailyResetUnixMs + board.WeeklyResetUnixMs;
            };
            read();

            Assert.That(read, Allocates.Nothing);
            Assert.Greater(sink, 0f);
        }
    }
}
