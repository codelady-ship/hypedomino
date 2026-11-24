using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.UI.Match
{
    public sealed class MatchLayout
    {
        public const float Margin = TokenSpacing.Md;
        public static readonly Vector2 SeatPanelSize = new(TokenSize.SeatPanelWidth, TokenSize.SeatPanelHeight);
        public static readonly float BottomBandHeight = Mathf.Max(TokenSize.HandHeight, TokenSize.SeatPanelHeight);

        readonly Rect[] seats = new Rect[TableSides.Count];

        public Rect Safe { get; private set; }
        public int SeatCount { get; private set; }
        public Rect Board { get; private set; }
        public Rect Hand { get; private set; }

        public Rect Seat(TableSide side) => seats[(int)side];

        public void Arrange(Rect safe, int seatCount)
        {
            TableSides.CheckSeatCount(seatCount);
            Safe = safe;
            SeatCount = seatCount;

            var panel = SeatPanelSize;
            var bottom = safe.yMin + Margin;
            var bandTop = bottom + BottomBandHeight;

            var mine = new Rect(safe.xMin + Margin, bottom, panel.x, panel.y);
            seats[(int)TableSide.Bottom] = mine;
            var handLeft = mine.xMax + Margin;
            Hand = new Rect(handLeft, bottom, Mathf.Max(0f, safe.xMax - Margin - handLeft), TokenSize.HandHeight);

            var top = new Rect(safe.center.x - panel.x / 2f, safe.yMax - Margin - panel.y, panel.x, panel.y);
            seats[(int)TableSide.Top] = top;

            var boardLeft = safe.xMin + Margin;
            var boardRight = safe.xMax - Margin;
            if (seatCount == 4)
            {
                var sideBottom = (bandTop + safe.yMax) / 2f - panel.y / 2f;
                seats[(int)TableSide.Left] = new Rect(safe.xMin + Margin, sideBottom, panel.x, panel.y);
                seats[(int)TableSide.Right] = new Rect(safe.xMax - Margin - panel.x, sideBottom, panel.x, panel.y);
                boardLeft += panel.x + Margin;
                boardRight -= panel.x + Margin;
            }
            else
            {
                seats[(int)TableSide.Left] = Rect.zero;
                seats[(int)TableSide.Right] = Rect.zero;
            }

            var boardBottom = bandTop + Margin;
            var boardTop = top.yMin - Margin;
            Board = Rect.MinMaxRect(boardLeft, boardBottom, Mathf.Max(boardLeft, boardRight), Mathf.Max(boardBottom, boardTop));
        }
    }
}
