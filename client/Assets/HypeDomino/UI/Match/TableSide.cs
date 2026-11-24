using System;

namespace HypeDomino.UI.Match
{
    public enum TableSide
    {
        Bottom,
        Left,
        Top,
        Right,
    }

    public static class TableSides
    {
        public const int Count = 4;

        public static TableSide Of(int viewer, int seat, int seatCount)
        {
            CheckSeatCount(seatCount);
            if ((uint)viewer >= (uint)seatCount) throw new ArgumentOutOfRangeException(nameof(viewer), viewer, null);
            if ((uint)seat >= (uint)seatCount) throw new ArgumentOutOfRangeException(nameof(seat), seat, null);
            var turnsAway = (seat - viewer + seatCount) % seatCount;
            if (seatCount == 2) return turnsAway == 0 ? TableSide.Bottom : TableSide.Top;
            return (TableSide)turnsAway;
        }

        public static bool IsUsed(TableSide side, int seatCount)
        {
            CheckSeatCount(seatCount);
            return seatCount == 4 || side == TableSide.Bottom || side == TableSide.Top;
        }

        internal static void CheckSeatCount(int seatCount)
        {
            if (seatCount != 2 && seatCount != 4) throw new ArgumentOutOfRangeException(nameof(seatCount), seatCount, null);
        }
    }
}
