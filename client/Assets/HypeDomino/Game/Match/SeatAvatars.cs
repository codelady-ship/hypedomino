using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Match;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public sealed class SeatAvatars
    {
        readonly CosmeticStage stage;
        readonly AvatarView[] bySide;
        readonly AvatarLook[] shown = new AvatarLook[TableSides.Count];
        readonly bool[] showing = new bool[TableSides.Count];

        public SeatAvatars(CosmeticStage stage, AvatarView[] bySide)
        {
            this.stage = stage;
            this.bySide = bySide;
        }

        public void Show(MatchSnapshot view, CosmeticManifest manifest)
        {
            if (view.Mode != Mode._1V1 && view.Mode != Mode._2V2) return;
            var seatCount = SeatOrder.Size(view.Mode);
            var viewer = SeatOrder.Viewer(view.Seat, view.Mode);
            var roster = view.Roster;
            for (var at = 0; at < roster.Count; at++)
            {
                var entry = roster[at];
                if (!SeatOrder.IsIn(entry.Seat, seatCount)) continue;
                var side = (int)TableSides.Of(viewer, SeatOrder.Clockwise(entry.Seat), seatCount);
                Place(side, AvatarLook.Of(manifest, entry.Avatar, entry.Frame));
            }
        }

        public void RefreshEffects()
        {
            for (var side = 0; side < bySide.Length; side++)
                if (bySide[side] != null) bySide[side].RefreshEffects();
        }

        void Place(int side, AvatarLook look)
        {
            if (side >= bySide.Length || bySide[side] == null) return;
            if (showing[side] && shown[side].IsSameAs(look)) return;
            shown[side] = look;
            showing[side] = true;
            bySide[side].Show(stage, look);
        }
    }
}
