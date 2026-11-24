using System;
using System.Collections.Generic;
using System.Globalization;
using HypeDomino.Net;

namespace HypeDomino.Game.Pass
{
    public sealed class PassReward
    {
        public PassReward(string id, long coins, long crystals, IReadOnlyList<string> cosmetics)
        {
            Id = id;
            Coins = coins;
            Crystals = crystals;
            Cosmetics = cosmetics;
        }

        public string Id { get; }

        public long Coins { get; }

        public long Crystals { get; }

        public IReadOnlyList<string> Cosmetics { get; }
    }

    public sealed class PassTracks
    {
        const string FreeKey = "free";
        const string PremiumKey = "premium";

        readonly Dictionary<int, PassReward> free;
        readonly Dictionary<int, PassReward> premium;

        PassTracks(Dictionary<int, PassReward> free, Dictionary<int, PassReward> premium)
        {
            this.free = free;
            this.premium = premium;
        }

        public static bool TryRead(string json, int levels, out PassTracks tracks)
        {
            tracks = null;
            if (!JsonMap.TryParse(json, out var reply) || !reply.TryMap("tracks", out var byTrack)) return false;
            if (!TryReadTrack(byTrack, FreeKey, levels, out var free)) return false;
            if (!TryReadTrack(byTrack, PremiumKey, levels, out var premium)) return false;
            tracks = new PassTracks(free, premium);
            return true;
        }

        public bool TryRewardAt(PassTrack track, int level, out PassReward reward) =>
            (track == PassTrack.Premium ? premium : free).TryGetValue(level, out reward);

        static bool TryReadTrack(JsonMap byTrack, string name, int levels, out Dictionary<int, PassReward> rewards)
        {
            rewards = new Dictionary<int, PassReward>();
            if (byTrack.IsAbsent(name)) return true;
            if (!byTrack.TryMap(name, out var byLevel)) return false;
            foreach (var key in byLevel.Keys)
            {
                if (!TryReadLevel(key, levels, out var level)) return false;
                if (!byLevel.TryMap(key, out var bundle) || !TryReadReward(bundle, out var reward)) return false;
                rewards.Add(level, reward);
            }
            return true;
        }

        static bool TryReadLevel(string key, int levels, out int level)
        {
            if (!int.TryParse(key, NumberStyles.None, CultureInfo.InvariantCulture, out level)) return false;
            return level >= 1 && level <= levels && key == level.ToString(CultureInfo.InvariantCulture);
        }

        static bool TryReadReward(JsonMap bundle, out PassReward reward)
        {
            reward = null;
            if (!bundle.TryString("id", out var id) || id.Length == 0) return false;
            if (!bundle.TryInteger("coins", out var coins) || coins < 0) return false;
            long crystals = 0;
            if (!bundle.IsAbsent("crystals") && (!bundle.TryInteger("crystals", out crystals) || crystals < 0)) return false;
            var cosmetics = Array.Empty<string>();
            if (!bundle.IsAbsent("cosmetics") && !bundle.TryStrings("cosmetics", out cosmetics)) return false;
            reward = new PassReward(id, coins, crystals, cosmetics);
            return true;
        }
    }
}
