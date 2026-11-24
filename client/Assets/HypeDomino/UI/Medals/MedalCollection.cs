using System;
using System.Collections.Generic;
using HypeDomino.UI.Missions;
using HypeDomino.UI.Rarity;

namespace HypeDomino.UI.Medals
{
    public readonly struct MedalEntry
    {
        public MedalEntry(MedalReply medal, bool earned, string earnedAt)
        {
            Medal = medal;
            Earned = earned;
            EarnedAt = earnedAt;
        }

        public MedalReply Medal { get; }

        public bool Earned { get; }

        public string EarnedAt { get; }
    }

    public sealed class MedalCollection
    {
        public const string CollectionPath = "/medals";
        public const string ShowcasePath = "/medals/showcase";

        const string MissionSource = "mission";
        static readonly string[] GroupOrder = { "achievement", "event", "rank_reward" };

        readonly Dictionary<string, MedalReply> shown = new();
        readonly Dictionary<string, string> earnedAt = new();
        readonly List<MedalReply> showcase = new();
        readonly List<MedalReply> grid = new();
        readonly RarityTable rarities;
        readonly Comparison<MedalReply> gridOrder;

        public MedalCollection(RarityTable rarities = null)
        {
            this.rarities = rarities;
            gridOrder = CompareInGrid;
        }

        public int ShownCount => shown.Count;

        public MedalReply ShownAt(int index) => grid[index];

        public int EarnedCount => earnedAt.Count;

        public int Slots { get; private set; }

        public int ShowcaseCount => showcase.Count;

        public MedalReply ShowcaseAt(int index) => showcase[index];

        public void Show(MedalCollectionReply reply)
        {
            shown.Clear();
            earnedAt.Clear();
            Slots = Math.Max(0, reply?.slots ?? 0);
            if (reply?.medals != null)
                foreach (var medal in reply.medals)
                    if (medal != null && !string.IsNullOrEmpty(medal.id)) shown[medal.id] = medal;
            grid.Clear();
            grid.AddRange(shown.Values);
            grid.Sort(gridOrder);
            if (reply?.owned != null)
                foreach (var owned in reply.owned)
                    if (owned != null && !string.IsNullOrEmpty(owned.medal) && shown.ContainsKey(owned.medal)) earnedAt[owned.medal] = owned.earned_at;
            FillShowcase(reply?.showcase);
        }

        public void Apply(MedalShowcaseReply reply)
        {
            if (reply?.showcase == null) return;
            FillShowcase(reply.showcase);
        }

        public bool TryGet(string id, out MedalEntry entry)
        {
            entry = default;
            if (string.IsNullOrEmpty(id) || !shown.TryGetValue(id, out var medal)) return false;
            entry = new MedalEntry(medal, earnedAt.TryGetValue(id, out var at), at);
            return true;
        }

        public bool TryMissionProgress(string id, MissionBoard board, out MissionCard mission)
        {
            mission = default;
            if (board == null || string.IsNullOrEmpty(id) || earnedAt.ContainsKey(id) || !shown.TryGetValue(id, out var medal) || medal.sources == null) return false;
            foreach (var source in medal.sources)
                if (source?.type == MissionSource && TryAchievement(board, source.@ref, out mission)) return true;
            return false;
        }

        public bool IsShowcased(string id) => ShowcaseIndexOf(id) >= 0;

        public bool TryAddToShowcase(string id, out MedalShowcaseRequest request)
        {
            request = null;
            if (string.IsNullOrEmpty(id) || !earnedAt.ContainsKey(id) || IsShowcased(id) || showcase.Count >= Slots) return false;
            var draft = Draft();
            draft.Add(id);
            request = Request(draft);
            return true;
        }

        public bool TryRemoveFromShowcase(string id, out MedalShowcaseRequest request)
        {
            request = null;
            var at = ShowcaseIndexOf(id);
            if (at < 0) return false;
            var draft = Draft();
            draft.RemoveAt(at);
            request = Request(draft);
            return true;
        }

        public bool TryMoveInShowcase(int from, int to, out MedalShowcaseRequest request)
        {
            request = null;
            if (from == to || from < 0 || to < 0 || from >= showcase.Count || to >= showcase.Count) return false;
            var draft = Draft();
            var moved = draft[from];
            draft.RemoveAt(from);
            draft.Insert(to, moved);
            request = Request(draft);
            return true;
        }

        void FillShowcase(string[] ids)
        {
            showcase.Clear();
            if (ids == null) return;
            foreach (var id in ids)
                if (!string.IsNullOrEmpty(id) && earnedAt.ContainsKey(id) && !IsShowcased(id)) showcase.Add(shown[id]);
        }

        int ShowcaseIndexOf(string id)
        {
            if (string.IsNullOrEmpty(id)) return -1;
            for (var at = 0; at < showcase.Count; at++)
                if (showcase[at].id == id) return at;
            return -1;
        }

        List<string> Draft()
        {
            var draft = new List<string>(showcase.Count + 1);
            foreach (var medal in showcase) draft.Add(medal.id);
            return draft;
        }

        static bool TryAchievement(MissionBoard board, string missionId, out MissionCard mission)
        {
            for (var at = 0; at < board.Count(MissionTab.Achievement); at++)
            {
                mission = board.Card(MissionTab.Achievement, at);
                if (mission.Id == missionId) return true;
            }
            mission = default;
            return false;
        }

        int CompareInGrid(MedalReply a, MedalReply b)
        {
            var byGroup = GroupRank(a.group).CompareTo(GroupRank(b.group));
            if (byGroup != 0) return byGroup;
            var byRarity = RarityOrdinal(b.rarityId).CompareTo(RarityOrdinal(a.rarityId));
            return byRarity != 0 ? byRarity : string.CompareOrdinal(a.id, b.id);
        }

        int RarityOrdinal(string rarityId) => rarities != null && rarities.TryGet(rarityId, out var tier) ? tier.Ordinal : int.MinValue;

        static int GroupRank(string group)
        {
            var rank = Array.IndexOf(GroupOrder, group);
            return rank < 0 ? GroupOrder.Length : rank;
        }

        static MedalShowcaseRequest Request(List<string> draft) => new() { medals = draft.ToArray() };
    }
}
