using System;
using System.Collections.Generic;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Medals;

namespace HypeDomino.UI.Profile
{
    public enum ProfileStat
    {
        Games,
        Wins,
        Losses,
        WinRatePercent,
        Streak,
        BestStreak,
    }

    public readonly struct ProfileStatValue
    {
        public ProfileStatValue(ProfileStat stat, int value)
        {
            Stat = stat;
            Value = value;
        }

        public ProfileStat Stat { get; }
        public int Value { get; }
    }

    public readonly struct ProfileRank
    {
        public ProfileRank(string mode, string label)
        {
            Mode = mode;
            Label = label;
        }

        public string Mode { get; }
        public string Label { get; }
    }

    public readonly struct ProfileModeStats
    {
        public ProfileModeStats(string mode, int games, int wins, int losses)
        {
            Mode = mode;
            Games = games;
            Wins = wins;
            Losses = losses;
        }

        public string Mode { get; }
        public int Games { get; }
        public int Wins { get; }
        public int Losses { get; }
    }

    public readonly struct ShowcaseMedal
    {
        public ShowcaseMedal(MedalReply reply)
        {
            Id = reply.id;
            NameKey = reply.nameKey;
            RarityId = reply.rarityId;
            StaticAddress = reply.staticAddress;
        }

        public string Id { get; }
        public string NameKey { get; }
        public string RarityId { get; }
        public string StaticAddress { get; }
    }

    public readonly struct ShowcaseEmblem
    {
        public ShowcaseEmblem(ShowcaseEmblemReply reply)
        {
            Id = reply.id;
            Season = reply.season;
            Mode = reply.mode;
            Board = reply.board;
            Bracket = reply.bracket;
            Position = reply.position;
        }

        public string Id { get; }
        public string Season { get; }
        public string Mode { get; }
        public string Board { get; }
        public int Bracket { get; }
        public int Position { get; }
    }

    public sealed class PlayerProfile
    {
        public const string OwnPath = "/profile/me";

        const string PublicPathPrefix = "/profile/";

        readonly ProfileStatValue[] stats;
        readonly ProfileModeStats[] modeStats;
        readonly ProfileRank[] ranks;
        readonly ShowcaseMedal[] medals;
        readonly ShowcaseEmblem[] emblems;

        PlayerProfile(string accountId, string nickname, int level, long xp, bool isOwn, LoadoutReply loadout, ProfileStatValue[] stats, ProfileModeStats[] modeStats,
            ProfileRank[] ranks, ShowcaseMedal[] medals, ShowcaseEmblem[] emblems)
        {
            AccountId = accountId;
            Nickname = nickname;
            Level = level;
            Xp = xp;
            IsOwn = isOwn;
            AvatarId = loadout?.avatar;
            FrameId = loadout?.frame;
            this.stats = stats;
            this.modeStats = modeStats;
            this.ranks = ranks;
            this.medals = medals;
            this.emblems = emblems;
        }

        public string AccountId { get; }
        public string Nickname { get; }
        public int Level { get; }
        public long Xp { get; }
        public bool IsOwn { get; }
        public string AvatarId { get; }
        public string FrameId { get; }
        public bool RenameRequired { get; private set; }
        public ulong RenameAvailableUnixMs { get; private set; }
        public SanctionAppeal RenameAppeal { get; private set; }
        public int StatCount => stats.Length;
        public int ModeStatsCount => modeStats.Length;
        public int RankCount => ranks.Length;
        public int MedalCount => medals.Length;
        public int EmblemCount => emblems.Length;

        public ProfileStatValue Stat(int index) => stats[index];

        public ProfileModeStats ModeStats(int index) => modeStats[index];

        public ProfileRank Rank(int index) => ranks[index];

        public ShowcaseMedal Medal(int index) => medals[index];

        public ShowcaseEmblem Emblem(int index) => emblems[index];

        public AvatarLook Look(CosmeticManifest manifest) => AvatarLook.Of(manifest, AvatarId, FrameId);

        public static string PublicPath(string accountId) => PublicPathPrefix + Uri.EscapeDataString(accountId);

        public static bool TryFromOwn(string json, out PlayerProfile profile)
        {
            profile = null;
            if (!JsonMap.TryParse(json, out var map) || !TryModeStats(map, out var modeStats))
            {
                return false;
            }
            var reply = OwnProfileReply.FromJson(json);
            if (reply.setup_required || !Named(reply.account_id, reply.nickname) || !TryRenameAvailable(reply.rename_available_at, out var renameAvailableUnixMs))
            {
                return false;
            }
            profile = new PlayerProfile(reply.account_id, reply.nickname, reply.level, reply.xp, true, reply.loadout, OwnStats(reply.stats ?? new OwnStatsReply()), modeStats,
                Array.Empty<ProfileRank>(), Array.Empty<ShowcaseMedal>(), Array.Empty<ShowcaseEmblem>())
            {
                RenameRequired = reply.rename_required,
                RenameAvailableUnixMs = renameAvailableUnixMs,
                RenameAppeal = reply.rename_required ? SanctionAppeal.Of(reply.rename_appeal) : default,
            };
            return true;
        }

        public bool TryRenameSecondsLeft(ServerClock clock, double nowSeconds, out double seconds)
        {
            seconds = 0;
            return clock != null && clock.TrySecondsLeft(RenameAvailableUnixMs, nowSeconds, out seconds);
        }

        static bool TryRenameAvailable(string text, out ulong unixMs)
        {
            unixMs = 0;
            return string.IsNullOrEmpty(text) || ServerClock.TryReadUnixMs(text, out unixMs);
        }

        public static bool TryFromPublic(string json, out PlayerProfile profile)
        {
            profile = null;
            if (!JsonMap.TryParse(json, out var map) || !TryRanks(map, out var ranks))
            {
                return false;
            }
            var reply = PublicProfileReply.FromJson(json);
            if (!Named(reply.account_id, reply.nickname))
            {
                return false;
            }
            profile = new PlayerProfile(reply.account_id, reply.nickname, reply.level, 0, false, reply.loadout, PublicStats(reply.stats ?? new PublicStatsReply()),
                Array.Empty<ProfileModeStats>(), ranks, Medals(reply.medals), Emblems(reply.ranked?.emblems));
            return true;
        }

        static bool Named(string accountId, string nickname) => !string.IsNullOrEmpty(accountId) && !string.IsNullOrEmpty(nickname);

        static ProfileStatValue[] OwnStats(OwnStatsReply reply) => new[]
        {
            new ProfileStatValue(ProfileStat.Games, reply.games),
            new ProfileStatValue(ProfileStat.Wins, reply.wins),
            new ProfileStatValue(ProfileStat.Losses, reply.losses),
            new ProfileStatValue(ProfileStat.WinRatePercent, reply.win_rate_percent),
            new ProfileStatValue(ProfileStat.Streak, reply.streak),
            new ProfileStatValue(ProfileStat.BestStreak, reply.best_streak),
        };

        static ProfileStatValue[] PublicStats(PublicStatsReply reply) => new[]
        {
            new ProfileStatValue(ProfileStat.Games, reply.games),
            new ProfileStatValue(ProfileStat.WinRatePercent, reply.win_rate_percent),
            new ProfileStatValue(ProfileStat.BestStreak, reply.best_streak),
        };

        static bool TryModeStats(JsonMap reply, out ProfileModeStats[] modeStats)
        {
            modeStats = Array.Empty<ProfileModeStats>();
            if (!TryOptionalMap(reply, "stats", out var stats)) return false;
            if (stats == null) return true;
            if (!TryOptionalMap(stats, "modes", out var byMode)) return false;
            if (byMode == null) return true;
            var read = new List<ProfileModeStats>();
            foreach (var mode in byMode.Keys)
            {
                if (mode.Length == 0 || !byMode.TryMap(mode, out var counters)) return false;
                if (!TryCounter(counters, "games", out var games) || !TryCounter(counters, "wins", out var wins) || !TryCounter(counters, "losses", out var losses)) return false;
                read.Add(new ProfileModeStats(mode, games, wins, losses));
            }
            modeStats = read.ToArray();
            return true;
        }

        static bool TryRanks(JsonMap reply, out ProfileRank[] ranks)
        {
            ranks = Array.Empty<ProfileRank>();
            if (!TryOptionalMap(reply, "ranked", out var ranked)) return false;
            if (ranked == null) return true;
            if (!TryOptionalMap(ranked, "ranks", out var byMode)) return false;
            if (byMode == null) return true;
            var read = new List<ProfileRank>();
            foreach (var mode in byMode.Keys)
            {
                if (mode.Length == 0 || !byMode.TryString(mode, out var label) || label.Length == 0) return false;
                read.Add(new ProfileRank(mode, label));
            }
            ranks = read.ToArray();
            return true;
        }

        static bool TryOptionalMap(JsonMap parent, string key, out JsonMap map)
        {
            map = null;
            return parent.IsAbsent(key) || parent.TryMap(key, out map);
        }

        static bool TryCounter(JsonMap counters, string key, out int value)
        {
            value = 0;
            if (!counters.TryInteger(key, out var number) || number < 0 || number > int.MaxValue) return false;
            value = (int)number;
            return true;
        }

        static ShowcaseMedal[] Medals(MedalReply[] replies)
        {
            var shown = new List<ShowcaseMedal>();
            foreach (var reply in replies ?? Array.Empty<MedalReply>())
            {
                if (!string.IsNullOrEmpty(reply?.id))
                {
                    shown.Add(new ShowcaseMedal(reply));
                }
            }
            return shown.ToArray();
        }

        static ShowcaseEmblem[] Emblems(ShowcaseEmblemReply[] replies)
        {
            var shown = new List<ShowcaseEmblem>();
            foreach (var reply in replies ?? Array.Empty<ShowcaseEmblemReply>())
            {
                if (!string.IsNullOrEmpty(reply?.id))
                {
                    shown.Add(new ShowcaseEmblem(reply));
                }
            }
            return shown.ToArray();
        }
    }
}
