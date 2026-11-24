using System;
using System.Collections.Generic;
using System.Globalization;
using HypeDomino.Net;

namespace HypeDomino.UI.Missions
{
    public enum MissionTab
    {
        Daily,
        Weekly,
        Achievement,
    }

    public readonly struct MissionCard
    {
        public MissionCard(AssignmentReply reply, MissionTab tab)
        {
            Id = reply.mission.id;
            NameKey = reply.mission.nameKey;
            Tab = tab;
            Target = reply.mission.target;
            Progress = reply.progress;
            Completed = !string.IsNullOrEmpty(reply.completed_at);
        }

        public string Id { get; }
        public string NameKey { get; }
        public MissionTab Tab { get; }
        public long Target { get; }
        public long Progress { get; }
        public bool Completed { get; }
        public long ShownProgress => Completed ? Target : Math.Clamp(Progress, 0, Target);
        public float Fill => Completed ? 1f : Target > 0 ? (float)ShownProgress / Target : 0f;
    }

    public sealed class MissionBoard
    {
        public const string BoardPath = "/missions";
        public const string RerollPath = "/missions/reroll";

        const int TabCount = 3;
        const int FirstYear = 1970;
        const int LastYear = 9998;
        const int LastIsoWeek = 53;
        const string WeekSeparator = "-W";

        readonly MissionCard[][] tabs;
        readonly int dailyRerolls;
        readonly int weeklyRerolls;

        MissionBoard(MissionBoardReply reply, ulong dailyResetUnixMs, ulong weeklyResetUnixMs)
        {
            Version = reply.version;
            DailyResetUnixMs = dailyResetUnixMs;
            WeeklyResetUnixMs = weeklyResetUnixMs;
            dailyRerolls = reply.daily_rerolls;
            weeklyRerolls = reply.weekly_rerolls;
            tabs = ToTabs(reply.missions);
        }

        public long Version { get; }
        public ulong DailyResetUnixMs { get; }
        public ulong WeeklyResetUnixMs { get; }

        public int Count(MissionTab tab) => tabs[(int)tab].Length;

        public MissionCard Card(MissionTab tab, int index) => tabs[(int)tab][index];

        public int RerollsLeft(MissionTab tab) => tab switch
        {
            MissionTab.Daily => dailyRerolls,
            MissionTab.Weekly => weeklyRerolls,
            _ => 0,
        };

        public bool CanReroll(in MissionCard card) => !card.Completed && RerollsLeft(card.Tab) > 0;

        public ulong ResetUnixMs(MissionTab tab) => tab switch
        {
            MissionTab.Daily => DailyResetUnixMs,
            MissionTab.Weekly => WeeklyResetUnixMs,
            _ => 0,
        };

        public bool TrySecondsUntilReset(MissionTab tab, ServerClock clock, double nowSeconds, out double seconds)
        {
            seconds = 0;
            return clock != null && clock.TrySecondsLeft(ResetUnixMs(tab), nowSeconds, out seconds);
        }

        public bool IsPastReset(ServerClock clock, double nowSeconds) =>
            TrySecondsUntilReset(MissionTab.Daily, clock, nowSeconds, out var seconds) && seconds <= 0;

        public static bool TryFrom(MissionBoardReply reply, out MissionBoard board)
        {
            board = null;
            if (reply == null || !TryDayEnd(reply.day, out var dayEnd) || !TryWeekEnd(reply.week, out var weekEnd))
            {
                return false;
            }
            board = new MissionBoard(reply, dayEnd, weekEnd);
            return true;
        }

        static MissionCard[][] ToTabs(AssignmentReply[] replies)
        {
            var lists = new List<MissionCard>[TabCount];
            for (var i = 0; i < TabCount; i++)
            {
                lists[i] = new List<MissionCard>();
            }
            foreach (var reply in replies ?? Array.Empty<AssignmentReply>())
            {
                if (reply?.mission != null && TryTab(reply.mission.type, out var tab))
                {
                    lists[(int)tab].Add(new MissionCard(reply, tab));
                }
            }
            var result = new MissionCard[TabCount][];
            for (var i = 0; i < TabCount; i++)
            {
                result[i] = lists[i].ToArray();
            }
            return result;
        }

        static bool TryTab(string type, out MissionTab tab)
        {
            tab = type switch
            {
                "weekly" => MissionTab.Weekly,
                "achievement" => MissionTab.Achievement,
                _ => MissionTab.Daily,
            };
            return type == "daily" || type == "weekly" || type == "achievement";
        }

        static bool TryDayEnd(string day, out ulong unixMs)
        {
            unixMs = 0;
            if (!DateTime.TryParseExact(day, "yyyy-MM-dd", CultureInfo.InvariantCulture, DateTimeStyles.AssumeUniversal | DateTimeStyles.AdjustToUniversal, out var start)
                || start.Year < FirstYear || start.Year > LastYear)
            {
                return false;
            }
            unixMs = UnixMs(start.AddDays(1));
            return true;
        }

        static bool TryWeekEnd(string week, out ulong unixMs)
        {
            unixMs = 0;
            var separator = week?.IndexOf(WeekSeparator, StringComparison.Ordinal) ?? -1;
            if (separator < 0
                || !int.TryParse(week.Substring(0, separator), NumberStyles.None, CultureInfo.InvariantCulture, out var year)
                || !int.TryParse(week.Substring(separator + WeekSeparator.Length), NumberStyles.None, CultureInfo.InvariantCulture, out var number)
                || year < FirstYear || year > LastYear || number < 1 || number > LastIsoWeek)
            {
                return false;
            }
            var fourthOfJanuary = new DateTime(year, 1, 4, 0, 0, 0, DateTimeKind.Utc);
            var firstMonday = fourthOfJanuary.AddDays(-(((int)fourthOfJanuary.DayOfWeek + 6) % 7));
            unixMs = UnixMs(firstMonday.AddDays(7 * number));
            return true;
        }

        static ulong UnixMs(DateTime utc) => (ulong)new DateTimeOffset(utc).ToUnixTimeMilliseconds();
    }
}
