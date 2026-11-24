using System;
using Hypedomino.V1;

namespace HypeDomino.Net.Integrity
{
    public enum IntegrityPurpose
    {
        SessionStart,
        Purchase,
        RankedMatch,
        CasualMatch,
    }

    public static class IntegrityPurposes
    {
        public static string WireName(IntegrityPurpose purpose) => purpose switch
        {
            IntegrityPurpose.SessionStart => "session_start",
            IntegrityPurpose.Purchase => "purchase",
            IntegrityPurpose.RankedMatch => "ranked_match",
            IntegrityPurpose.CasualMatch => "casual_match",
            _ => throw new ArgumentOutOfRangeException(nameof(purpose), purpose, null),
        };

        public static IntegrityPurpose ForQueue(QueueKind kind) => kind == QueueKind.Ranked ? IntegrityPurpose.RankedMatch : IntegrityPurpose.CasualMatch;
    }
}
