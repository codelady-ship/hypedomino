using System.Collections.Generic;
using UnityEngine;

namespace HypeDomino.UI.Board
{
    public enum ChainEnd
    {
        West,
        East,
        North,
        South,
    }

    public sealed class ChainLayout
    {
        const float Tolerance = 0.01f;
        const int NoTile = -1;

        static readonly Vector2[] Outward = { Vector2.left, Vector2.right, Vector2.up, Vector2.down };
        static readonly float[] FoldBoxAspectsOfBoard = { 1f, 0.5f, 2f };

        readonly int[] endTiles = new int[4];
        readonly Rect[] areas;
        PlacedDomino[] placed;
        PlacedDomino[] candidate;
        int anchor;

        public ChainLayout(int capacity)
        {
            placed = new PlacedDomino[capacity];
            candidate = new PlacedDomino[capacity];
            areas = new Rect[capacity];
        }

        public int Count { get; private set; }

        public Rect Bounds { get; private set; }

        public float Scale { get; private set; }

        public Vector2 Focus => Bounds.center;

        public PlacedDomino this[int index] => placed[index];

        public Vector2 OnBoard(Vector2 point) => (point - Focus) * Scale;

        public bool TryGetEnd(ChainEnd end, out Rect area)
        {
            area = default;
            if (Count == 0) return false;
            var last = endTiles[(int)end];
            var tile = placed[last == NoTile ? anchor : last];
            area = HalfToward(tile.Area, last == NoTile ? Outward[(int)end] : tile.Facing);
            return true;
        }

        public void Arrange(IReadOnlyList<Domino> line, int anchorIndex, IReadOnlyList<Domino> top, IReadOnlyList<Domino> bottom, Vector2 board, float maxScale)
        {
            Count = line.Count + top.Count + bottom.Count;
            Scale = 0f;
            Bounds = default;
            if (line.Count == 0) return;
            MarkEnds(line.Count, anchorIndex, top.Count, bottom.Count);

            var straight = Length(line) + Length(top) + Length(bottom) + 2f;
            for (var i = 0; i < FoldBoxAspectsOfBoard.Length; i++)
                TryFoldBoxes(line, anchorIndex, top, bottom, board, maxScale, straight, board.y / board.x * FoldBoxAspectsOfBoard[i]);
        }

        void TryFoldBoxes(IReadOnlyList<Domino> line, int anchorIndex, IReadOnlyList<Domino> top, IReadOnlyList<Domino> bottom, Vector2 board, float maxScale, float straight, float aspect)
        {
            var tried = new Vector2(-1f, -1f);
            for (var halfWidth = straight * Mathf.Max(1f, 1f / aspect); halfWidth >= 1f; halfWidth -= 1f)
            {
                var box = new Vector2(FoldLimit(halfWidth, straight), FoldLimit(halfWidth * aspect, straight));
                if (box == tried) continue;
                tried = box;
                Lay(line, anchorIndex, top, bottom, box);
                MeasureAreas();
                if (Overlaps()) continue;
                var bounds = Enclose();
                var scale = Mathf.Min(maxScale, Mathf.Min(board.x / bounds.width, board.y / bounds.height));
                if (scale <= Scale) continue;
                Scale = scale;
                Bounds = bounds;
                (placed, candidate) = (candidate, placed);
            }
        }

        static float FoldLimit(float half, float straight) => Mathf.Floor((Mathf.Min(half, straight) + Tolerance) * 2f) * 0.5f;

        void MarkEnds(int lineCount, int anchorIndex, int topCount, int bottomCount)
        {
            anchor = anchorIndex;
            endTiles[(int)ChainEnd.West] = anchorIndex > 0 ? 0 : NoTile;
            endTiles[(int)ChainEnd.East] = anchorIndex < lineCount - 1 ? lineCount - 1 : NoTile;
            endTiles[(int)ChainEnd.North] = topCount > 0 ? lineCount + topCount - 1 : NoTile;
            endTiles[(int)ChainEnd.South] = bottomCount > 0 ? lineCount + topCount + bottomCount - 1 : NoTile;
        }

        static Rect HalfToward(Rect area, Vector2 heading)
        {
            var center = area.center;
            if (heading.x > 0f) area.xMin = center.x;
            else if (heading.x < 0f) area.xMax = center.x;
            else if (heading.y > 0f) area.yMin = center.y;
            else area.yMax = center.y;
            return area;
        }

        static float Length(IReadOnlyList<Domino> tiles)
        {
            var length = 0f;
            for (var i = 0; i < tiles.Count; i++) length += tiles[i].IsDouble ? 1f : 2f;
            return length;
        }

        void Lay(IReadOnlyList<Domino> line, int anchorIndex, IReadOnlyList<Domino> top, IReadOnlyList<Domino> bottom, Vector2 box)
        {
            var anchor = line[anchorIndex];
            candidate[anchorIndex] = new PlacedDomino(Vector2.zero, Vector2.right, anchor.First, anchor.Second, anchor.IsDouble);
            var reach = anchor.IsDouble ? 0.5f : 1f;

            var arm = new Arm(new Vector2(reach, 0f), Vector2.right, box);
            for (var i = anchorIndex + 1; i < line.Count; i++) candidate[i] = arm.Place(line[i].First, line[i].Second, line[i].IsDouble);

            arm = new Arm(new Vector2(-reach, 0f), Vector2.left, box);
            for (var i = anchorIndex - 1; i >= 0; i--) candidate[i] = arm.Place(line[i].Second, line[i].First, line[i].IsDouble);

            LayArm(top, line.Count, new Arm(new Vector2(0f, 1f), Vector2.up, box));
            LayArm(bottom, line.Count + top.Count, new Arm(new Vector2(0f, -1f), Vector2.down, box));
        }

        void LayArm(IReadOnlyList<Domino> tiles, int offset, Arm arm)
        {
            for (var i = 0; i < tiles.Count; i++) candidate[offset + i] = arm.Place(tiles[i].First, tiles[i].Second, tiles[i].IsDouble);
        }

        void MeasureAreas()
        {
            for (var i = 0; i < Count; i++) areas[i] = candidate[i].Area;
        }

        bool Overlaps()
        {
            for (var i = 0; i < Count; i++)
            {
                var area = areas[i];
                for (var j = i + 1; j < Count; j++)
                {
                    var other = areas[j];
                    var width = Mathf.Min(area.xMax, other.xMax) - Mathf.Max(area.xMin, other.xMin);
                    var height = Mathf.Min(area.yMax, other.yMax) - Mathf.Max(area.yMin, other.yMin);
                    if (width > Tolerance && height > Tolerance) return true;
                }
            }
            return false;
        }

        Rect Enclose()
        {
            var min = areas[0].min;
            var max = areas[0].max;
            for (var i = 1; i < Count; i++)
            {
                min = Vector2.Min(min, areas[i].min);
                max = Vector2.Max(max, areas[i].max);
            }
            return Rect.MinMaxRect(min.x, min.y, max.x, max.y);
        }

        struct Arm
        {
            Vector2 end;
            Vector2 heading;
            readonly Vector2 box;

            public Arm(Vector2 end, Vector2 heading, Vector2 box)
            {
                this.end = end;
                this.heading = heading;
                this.box = box;
            }

            public PlacedDomino Place(byte near, byte far, bool isDouble)
            {
                if (isDouble) return Advance(near, far, 1f, true);
                if (Crosses()) return Turn(near, far);
                return Advance(near, far, 2f, false);
            }

            bool Crosses()
            {
                var limit = heading.x != 0f ? box.x : box.y;
                return Vector2.Dot(end + heading * 2f, heading) > limit + Tolerance;
            }

            PlacedDomino Advance(byte near, byte far, float length, bool across)
            {
                var center = end + heading * (length * 0.5f);
                end += heading * length;
                return new PlacedDomino(center, heading, near, far, across);
            }

            PlacedDomino Turn(byte near, byte far)
            {
                var turned = new Vector2(heading.y, -heading.x);
                var corner = end + heading * 0.5f;
                end = corner + turned * 1.5f;
                heading = turned;
                return new PlacedDomino(corner + turned * 0.5f, turned, near, far, false);
            }
        }
    }
}
