using System;
using UnityEngine;

namespace HypeDomino.UI.Medals
{
    [Serializable]
    public sealed class MedalSourceReply
    {
        public string type;
        public string @ref;
    }

    [Serializable]
    public sealed class MedalReply
    {
        public string id;
        public string nameKey;
        public string descriptionKey;
        public string rarityId;
        public string staticAddress;
        public MedalSourceReply[] sources;
        public string group;
    }

    [Serializable]
    public sealed class OwnedMedalReply
    {
        public string medal;
        public string earned_at;
    }

    [Serializable]
    public sealed class MedalCollectionReply
    {
        public MedalReply[] medals;
        public OwnedMedalReply[] owned;
        public string[] showcase;
        public int slots;

        public static MedalCollectionReply FromJson(string json) => JsonUtility.FromJson<MedalCollectionReply>(json);
    }

    [Serializable]
    public sealed class MedalShowcaseReply
    {
        public string[] showcase;

        public static MedalShowcaseReply FromJson(string json) => JsonUtility.FromJson<MedalShowcaseReply>(json);
    }

    [Serializable]
    public sealed class MedalShowcaseRequest
    {
        public string[] medals;

        public string ToJson() => JsonUtility.ToJson(this);
    }
}
