using System;

namespace HypeDomino.Game.Ranking
{
    public enum RankTier
    {
        None,
        Iron,
        Bronze,
        Silver,
        Gold,
        Platinum,
        Emerald,
        Diamond,
        Master,
        Grandmaster,
        Challenger,
    }

    public readonly struct RankBadge
    {
        public const int NoDivision = 0;
        public const int LowestDivision = 4;

        static readonly string[] TierLabels =
        {
            "", "Iron", "Bronze", "Silver", "Gold", "Platinum", "Emerald", "Diamond", "Master", "Grandmaster", "Challenger",
        };
        static readonly string[] DivisionLabels = { "", "I", "II", "III", "IV" };

        public readonly RankTier Tier;
        public readonly int Division;

        RankBadge(RankTier tier, int division)
        {
            Tier = tier;
            Division = division;
        }

        public bool Placed => Tier != RankTier.None;

        public bool Apex => Tier >= RankTier.Master;

        public static bool TryRead(string label, out RankBadge badge)
        {
            badge = default;
            if (string.IsNullOrEmpty(label)) return true;
            var text = label.AsSpan();
            var gap = text.IndexOf(' ');
            var tier = TierOf(gap < 0 ? text : text.Slice(0, gap));
            if (tier == RankTier.None) return false;
            if (tier >= RankTier.Master) return gap < 0 && Accept(tier, NoDivision, out badge);
            if (gap < 0) return false;
            var division = DivisionOf(text.Slice(gap + 1));
            return division != NoDivision && Accept(tier, division, out badge);
        }

        static bool Accept(RankTier tier, int division, out RankBadge badge)
        {
            badge = new RankBadge(tier, division);
            return true;
        }

        static RankTier TierOf(ReadOnlySpan<char> name)
        {
            for (var tier = (int)RankTier.Iron; tier < TierLabels.Length; tier++)
                if (name.SequenceEqual(TierLabels[tier].AsSpan())) return (RankTier)tier;
            return RankTier.None;
        }

        static int DivisionOf(ReadOnlySpan<char> numeral)
        {
            for (var division = 1; division <= LowestDivision; division++)
                if (numeral.SequenceEqual(DivisionLabels[division].AsSpan())) return division;
            return NoDivision;
        }
    }
}
