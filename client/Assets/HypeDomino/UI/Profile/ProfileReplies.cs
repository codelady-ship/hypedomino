using System;
using HypeDomino.Net;
using HypeDomino.UI.Medals;
using UnityEngine;

namespace HypeDomino.UI.Profile
{
    [Serializable]
    public sealed class LoadoutReply
    {
        public string avatar;
        public string frame;
    }

    [Serializable]
    public sealed class OwnStatsReply
    {
        public int games;
        public int wins;
        public int losses;
        public int win_rate_percent;
        public int streak;
        public int best_streak;
    }

    [Serializable]
    public sealed class OwnProfileReply
    {
        public string account_id;
        public string nickname;
        public bool setup_required;
        public bool rename_required;
        public string rename_available_at;
        public AppealGrantReply rename_appeal;
        public int level;
        public long xp;
        public LoadoutReply loadout;
        public OwnStatsReply stats;

        public static OwnProfileReply FromJson(string json) => JsonUtility.FromJson<OwnProfileReply>(json);
    }

    [Serializable]
    public sealed class PublicStatsReply
    {
        public int games;
        public int win_rate_percent;
        public int best_streak;
    }

    [Serializable]
    public sealed class ShowcaseEmblemReply
    {
        public string id;
        public string season;
        public string mode;
        public string board;
        public int bracket;
        public int position;
    }

    [Serializable]
    public sealed class PublicRankedReply
    {
        public ShowcaseEmblemReply[] emblems;
    }

    [Serializable]
    public sealed class PublicProfileReply
    {
        public string account_id;
        public string nickname;
        public int level;
        public LoadoutReply loadout;
        public PublicStatsReply stats;
        public PublicRankedReply ranked;
        public MedalReply[] medals;

        public static PublicProfileReply FromJson(string json) => JsonUtility.FromJson<PublicProfileReply>(json);
    }
}
