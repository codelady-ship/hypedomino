using System;
using HypeDomino.Net;
using UnityEngine;

namespace HypeDomino.Game.Ranking
{
    [Serializable]
    public sealed class RankSeasonReply
    {
        public string id;
        public string name_key;
        public string starts_at;
        public string ends_at;
    }

    [Serializable]
    public sealed class RankModeReply
    {
        public string mode;
        public bool placed;
        public int placement_played;
        public int placement_matches;
        public string rank;
        public string tier;
        public int lp;
        public int division_lp;
        public int wins;
    }

    [Serializable]
    public sealed class RankStandingReply
    {
        public RankSeasonReply season;
        public RankModeReply[] modes;

        public static RankStandingReply FromJson(string json) => JsonUtility.FromJson<RankStandingReply>(json);
    }

    public readonly struct RankModeStanding
    {
        public RankModeStanding(string mode, RankBadge badge, int placementPlayed, int placementMatches, int lp, int divisionLp, int wins)
        {
            Mode = mode;
            Badge = badge;
            PlacementPlayed = placementPlayed;
            PlacementMatches = placementMatches;
            LP = lp;
            DivisionLP = divisionLp;
            Wins = wins;
        }

        public string Mode { get; }

        public RankBadge Badge { get; }

        public bool Placed => Badge.Placed;

        public int PlacementPlayed { get; }

        public int PlacementMatches { get; }

        public int LP { get; }

        public int DivisionLP { get; }

        public int Wins { get; }

        public bool HasLpBar => Placed && !Badge.Apex;

        public float LpFill => HasLpBar ? Mathf.Clamp01((float)LP / DivisionLP) : 0f;
    }

    public sealed class RankStanding
    {
        public const string Path = "/ranked/me";
        public const string NoSeason = "rating.season_unknown";

        readonly RankModeStanding[] modes;

        RankStanding(RankSeasonReply season, ulong startsUnixMs, ulong endsUnixMs, RankModeStanding[] modes)
        {
            SeasonId = season.id;
            SeasonNameKey = season.name_key ?? string.Empty;
            StartsUnixMs = startsUnixMs;
            EndsUnixMs = endsUnixMs;
            this.modes = modes;
        }

        public string SeasonId { get; }

        public string SeasonNameKey { get; }

        public ulong StartsUnixMs { get; }

        public ulong EndsUnixMs { get; }

        public int ModeCount => modes.Length;

        public RankModeStanding ModeAt(int index) => modes[index];

        public bool TryFind(string mode, out RankModeStanding standing)
        {
            for (var i = 0; i < modes.Length; i++)
            {
                if (string.Equals(modes[i].Mode, mode, StringComparison.Ordinal))
                {
                    standing = modes[i];
                    return true;
                }
            }
            standing = default;
            return false;
        }

        public bool TrySecondsLeft(ServerClock clock, double nowSeconds, out double seconds)
        {
            seconds = 0;
            return clock != null && clock.TrySecondsLeft(EndsUnixMs, nowSeconds, out seconds);
        }

        public static bool TryRead(string json, out RankStanding standing)
        {
            standing = null;
            if (!TryParse(json, out var reply) || reply.season == null || string.IsNullOrEmpty(reply.season.id))
            {
                return false;
            }
            if (!ServerClock.TryReadUnixMs(reply.season.starts_at, out var startsUnixMs) || !ServerClock.TryReadUnixMs(reply.season.ends_at, out var endsUnixMs) || endsUnixMs <= startsUnixMs)
            {
                return false;
            }
            if (!TryModes(reply.modes, out var modes))
            {
                return false;
            }
            standing = new RankStanding(reply.season, startsUnixMs, endsUnixMs, modes);
            return true;
        }

        static bool TryParse(string json, out RankStandingReply reply)
        {
            reply = null;
            if (string.IsNullOrEmpty(json))
            {
                return false;
            }
            try
            {
                reply = RankStandingReply.FromJson(json);
            }
            catch (ArgumentException)
            {
                return false;
            }
            return reply != null;
        }

        static bool TryModes(RankModeReply[] replies, out RankModeStanding[] modes)
        {
            modes = Array.Empty<RankModeStanding>();
            if (replies == null || replies.Length == 0)
            {
                return true;
            }
            var read = new RankModeStanding[replies.Length];
            for (var i = 0; i < replies.Length; i++)
            {
                if (!TryMode(replies[i], out read[i]))
                {
                    return false;
                }
                for (var j = 0; j < i; j++)
                {
                    if (string.Equals(read[j].Mode, read[i].Mode, StringComparison.Ordinal))
                    {
                        return false;
                    }
                }
            }
            modes = read;
            return true;
        }

        static bool TryMode(RankModeReply reply, out RankModeStanding mode)
        {
            mode = default;
            if (reply == null || string.IsNullOrEmpty(reply.mode) || reply.placement_matches < 1 || reply.placement_played < 0 || reply.placement_played > reply.placement_matches)
            {
                return false;
            }
            if (reply.lp < 0 || reply.division_lp < 1 || reply.wins < 0 || !TryBadge(reply, out var badge))
            {
                return false;
            }
            mode = new RankModeStanding(reply.mode, badge, reply.placement_played, reply.placement_matches, reply.lp, reply.division_lp, reply.wins);
            return true;
        }

        static bool TryBadge(RankModeReply reply, out RankBadge badge)
        {
            if (!RankBadge.TryRead(reply.rank, out badge) || badge.Placed != reply.placed)
            {
                return false;
            }
            if (!reply.placed)
            {
                return reply.lp == 0;
            }
            if (!badge.Apex)
            {
                return true;
            }
            return RankBadge.TryRead(reply.tier, out badge) && badge.Apex;
        }
    }
}
