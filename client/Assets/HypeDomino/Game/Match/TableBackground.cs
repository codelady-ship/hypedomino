using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;

namespace HypeDomino.Game.Match
{
    public sealed class TableBackground
    {
        readonly CosmeticStage stage;
        readonly TableView view;
        TableLook shown;
        bool showing;

        public TableBackground(CosmeticStage stage, TableView view)
        {
            this.stage = stage;
            this.view = view;
        }

        public void Show(MatchSnapshot snapshot, CosmeticManifest manifest)
        {
            var look = LookOf(manifest, snapshot.Table);
            if (showing && shown.IsSameAs(look)) return;
            shown = look;
            showing = true;
            view.Show(stage, look);
        }

        public void RefreshEffects()
        {
            if (showing) view.RefreshEffects();
        }

        static TableLook LookOf(CosmeticManifest manifest, string tableId)
        {
            var item = manifest?.ShownOf(CosmeticAddresses.Table, tableId);
            return item?.Table ?? default(TableLook);
        }
    }
}
