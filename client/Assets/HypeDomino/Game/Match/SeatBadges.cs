using System;
using HypeDomino.Net;
using HypeDomino.UI.Match;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public sealed class SeatBadges
    {
        readonly SeatNotice[] bySide = new SeatNotice[TableSides.Count];

        public SeatNotice At(TableSide side) => bySide[(int)side];

        public bool Read(MatchSnapshot view)
        {
            Span<SeatNotice> next = stackalloc SeatNotice[TableSides.Count];
            Place(view, next);
            var changed = false;
            for (var side = 0; side < bySide.Length; side++)
            {
                if (bySide[side] == next[side]) continue;
                bySide[side] = next[side];
                changed = true;
            }
            return changed;
        }

        static void Place(MatchSnapshot view, Span<SeatNotice> next)
        {
            if (view.Mode != Mode._1V1 && view.Mode != Mode._2V2) return;
            var seatCount = SeatOrder.Size(view.Mode);
            var viewer = SeatOrder.Viewer(view.Seat, view.Mode);
            foreach (var seat in view.Seats)
            {
                if (!SeatOrder.IsIn(seat.Seat, seatCount)) continue;
                var side = TableSides.Of(viewer, SeatOrder.Clockwise(seat.Seat), seatCount);
                next[(int)side] = ConnectionNotices.Seat(view, seat);
            }
        }
    }
}
