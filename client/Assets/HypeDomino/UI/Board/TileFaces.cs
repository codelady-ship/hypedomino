using System;
using UnityEngine;

namespace HypeDomino.UI.Board
{
    public static class TileFaces
    {
        public const int SetSize = 28;
        public const int HighestValue = 6;
        const int Values = HighestValue + 1;

        public static int Index(Domino tile)
        {
            var low = Math.Min(tile.First, tile.Second);
            var high = Math.Max(tile.First, tile.Second);
            if (high >= Values) throw new ArgumentOutOfRangeException(nameof(tile), high, "a half holds 0 to 6 pips");
            return low * Values - low * (low - 1) / 2 + high - low;
        }

        public static float Angle(PlacedDomino tile)
        {
            var top = TopDirection(tile);
            return Mathf.Atan2(top.y, top.x) * Mathf.Rad2Deg - 90f;
        }

        static Vector2 TopDirection(PlacedDomino tile)
        {
            if (tile.Across) return new Vector2(-tile.Facing.y, tile.Facing.x);
            return tile.Near <= tile.Far ? -tile.Facing : tile.Facing;
        }
    }
}
