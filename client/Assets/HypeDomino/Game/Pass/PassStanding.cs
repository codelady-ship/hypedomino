using System;
using HypeDomino.Net;
using UnityEngine;

namespace HypeDomino.Game.Pass
{
    [Serializable]
    public sealed class PassReply
    {
        public string season;
        public string ends_at;
        public int levels;
        public long xp;
        public int level;
        public long level_xp;
        public long next_level_xp;
        public bool premium;
        public long level_coin_price;
        public long level_crystal_price;

        public static PassReply FromJson(string json) => JsonUtility.FromJson<PassReply>(json);
    }

    public enum PassTrack
    {
        Free,
        Premium,
    }

    public enum PassRewardState
    {
        Upcoming,
        Earned,
        Locked,
    }

    public sealed class PassStanding
    {
        public const string Path = "/pass";
        public const string NoSeason = "pass.no_season";

        readonly long levelXp;
        readonly long nextLevelXp;

        PassStanding(PassReply reply, ulong endsUnixMs)
        {
            Season = reply.season;
            Levels = reply.levels;
            Level = reply.level;
            Xp = reply.xp;
            Premium = reply.premium;
            LevelCoinPrice = reply.level_coin_price;
            LevelCrystalPrice = reply.level_crystal_price;
            EndsUnixMs = endsUnixMs;
            levelXp = reply.level_xp;
            nextLevelXp = reply.next_level_xp;
        }

        public string Season { get; }

        public int Levels { get; }

        public int Level { get; }

        public long Xp { get; }

        public bool Premium { get; }

        public long LevelCoinPrice { get; }

        public long LevelCrystalPrice { get; }

        public ulong EndsUnixMs { get; }

        public bool AtLastLevel => Level == Levels;

        public long XpIntoLevel => Xp - levelXp;

        public long XpForLevel => AtLastLevel ? 0 : nextLevelXp - levelXp;

        public float LevelFill => AtLastLevel ? 1f : Mathf.Clamp01((float)XpIntoLevel / XpForLevel);

        public static bool TryRead(PassReply reply, out PassStanding standing)
        {
            standing = null;
            if (reply == null || string.IsNullOrEmpty(reply.season) || reply.levels < 1) return false;
            if (reply.level < 1 || reply.level > reply.levels || reply.xp < reply.level_xp) return false;
            if (reply.level < reply.levels && reply.next_level_xp <= reply.xp) return false;
            if (reply.level_coin_price < 0 || reply.level_crystal_price < 0) return false;
            if (!ServerClock.TryReadUnixMs(reply.ends_at, out var endsUnixMs)) return false;
            standing = new PassStanding(reply, endsUnixMs);
            return true;
        }

        public PassRewardState StateAt(PassTrack track, int atLevel)
        {
            if (atLevel < 1 || atLevel > Levels) throw new ArgumentOutOfRangeException(nameof(atLevel));
            if (track == PassTrack.Premium && !Premium) return PassRewardState.Locked;
            return atLevel <= Level ? PassRewardState.Earned : PassRewardState.Upcoming;
        }

        public bool TrySecondsLeft(ServerClock clock, double nowSeconds, out double seconds)
        {
            seconds = 0;
            return clock != null && clock.TrySecondsLeft(EndsUnixMs, nowSeconds, out seconds);
        }
    }
}
