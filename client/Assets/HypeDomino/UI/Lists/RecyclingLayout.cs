using UnityEngine;

namespace HypeDomino.UI.Lists
{
    public readonly struct RecyclingLayout
    {
        const int BufferRows = 3;

        public RecyclingLayout(float rowHeight, float gap, float viewportHeight)
        {
            RowHeight = rowHeight;
            Stride = rowHeight + gap;
            RowCount = Mathf.CeilToInt(viewportHeight / Stride) + BufferRows;
        }

        public float RowHeight { get; }

        public float Stride { get; }

        public int RowCount { get; }

        public float ContentHeight(int itemCount) => itemCount > 0 ? itemCount * Stride - (Stride - RowHeight) : 0f;

        public float RowTop(int index) => index * Stride;

        public int FirstRow(float scrollOffset, int itemCount)
        {
            var first = Mathf.FloorToInt(Mathf.Max(0f, scrollOffset) / Stride) - 1;
            return Mathf.Clamp(first, 0, Mathf.Max(0, itemCount - RowCount));
        }
    }
}
