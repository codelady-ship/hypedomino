using System.Collections.Generic;
using HypeDomino.Net;

namespace HypeDomino.Game
{
    public static class GrantDuplicates
    {
        const string GrantKey = "grant";
        const string DuplicatesKey = "duplicates";

        public static readonly IReadOnlyDictionary<string, long> None = new Dictionary<string, long>();

        public static bool TryRead(string body, out IReadOnlyDictionary<string, long> duplicates)
        {
            duplicates = None;
            if (!JsonMap.TryParse(body, out var root)) return false;
            if (root.IsAbsent(GrantKey)) return true;
            if (!root.TryMap(GrantKey, out var grant)) return false;
            if (grant.IsAbsent(DuplicatesKey)) return true;
            if (!grant.TryMap(DuplicatesKey, out var byItem)) return false;
            var read = new Dictionary<string, long>();
            foreach (var item in byItem.Keys)
            {
                if (!byItem.TryInteger(item, out var coins) || coins < 0) return false;
                read.Add(item, coins);
            }
            duplicates = read;
            return true;
        }
    }
}
