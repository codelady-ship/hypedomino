using UnityEngine;

namespace HypeDomino.UI.Match
{
    [RequireComponent(typeof(RectTransform))]
    public sealed class MatchLayoutView : MonoBehaviour
    {
        [SerializeField] RectTransform board;
        [SerializeField] RectTransform hand;
        [SerializeField] RectTransform[] seats = new RectTransform[TableSides.Count];
        [SerializeField] int seatCount = 4;

        readonly MatchLayout layout = new();

        public MatchLayout Layout => layout;
        public int SeatCount => seatCount;

        public void Bind(RectTransform boardRect, RectTransform handRect, RectTransform[] seatRects)
        {
            board = boardRect;
            hand = handRect;
            seats = seatRects;
            Apply();
        }

        public void SetSeatCount(int count)
        {
            TableSides.CheckSeatCount(count);
            seatCount = count;
            Apply();
        }

        void OnEnable() => Apply();

        void OnRectTransformDimensionsChange()
        {
            if (isActiveAndEnabled) Apply();
        }

        public void Apply()
        {
            var root = (RectTransform)transform;
            var area = root.rect;
            layout.Arrange(area, seatCount);
            Place(board, layout.Board, area);
            Place(hand, layout.Hand, area);
            if (seats == null) return;
            for (var i = 0; i < seats.Length && i < TableSides.Count; i++)
            {
                var seat = seats[i];
                if (seat == null) continue;
                var side = (TableSide)i;
                var used = TableSides.IsUsed(side, seatCount);
                if (seat.gameObject.activeSelf != used) seat.gameObject.SetActive(used);
                if (used) Place(seat, layout.Seat(side), area);
            }
        }

        static void Place(RectTransform target, Rect rect, Rect area)
        {
            if (target == null) return;
            target.anchorMin = Vector2.zero;
            target.anchorMax = Vector2.zero;
            target.sizeDelta = rect.size;
            target.anchoredPosition = rect.position - area.min + Vector2.Scale(target.pivot, rect.size);
        }
    }
}
