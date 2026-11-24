using HypeDomino.UI.Localization;
using TMPro;
using UnityEngine;

namespace HypeDomino.Game.Pass
{
    public sealed class PassLevelRow : MonoBehaviour
    {
        [SerializeField] TMP_Text level;
        [SerializeField] PassCellView free;
        [SerializeField] PassCellView premium;

        public void Configure(TMP_Text levelText, PassCellView freeCell, PassCellView premiumCell)
        {
            level = levelText;
            free = freeCell;
            premium = premiumCell;
        }

        public void Show(PassLane lane, int row, string language)
        {
            var freeCell = lane.CellAt(PassTrack.Free, row);
            level.text = MessageFormat.Number(freeCell.Level, language);
            free.Show(freeCell, language);
            premium.Show(lane.CellAt(PassTrack.Premium, row), language);
        }
    }
}
