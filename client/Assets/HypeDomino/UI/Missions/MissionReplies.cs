using System;
using UnityEngine;

namespace HypeDomino.UI.Missions
{
    [Serializable]
    public sealed class MissionReply
    {
        public string id;
        public string type;
        public string nameKey;
        public long target;
    }

    [Serializable]
    public sealed class AssignmentReply
    {
        public MissionReply mission;
        public string period;
        public long progress;
        public string completed_at;
    }

    [Serializable]
    public sealed class MissionBoardReply
    {
        public long version;
        public string day;
        public string week;
        public AssignmentReply[] missions;
        public int daily_rerolls;
        public int weekly_rerolls;

        public static MissionBoardReply FromJson(string json) => JsonUtility.FromJson<MissionBoardReply>(json);
    }

    [Serializable]
    public sealed class RerollRequest
    {
        public string mission;

        public static string Json(string missionId) => JsonUtility.ToJson(new RerollRequest { mission = missionId });
    }
}
