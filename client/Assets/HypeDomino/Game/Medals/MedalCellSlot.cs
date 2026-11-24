using HypeDomino.UI.Cosmetics;
using TMPro;
using UnityEngine;

namespace HypeDomino.Game.Medals
{
    public sealed class MedalCellSlot : MonoBehaviour
    {
        [SerializeField] CosmeticImage icon;
        [SerializeField] TMP_Text medalName;
        [SerializeField] GameObject lockedLook;

        public void Configure(CosmeticImage iconImage, TMP_Text nameText, GameObject lockedRoot)
        {
            icon = iconImage;
            medalName = nameText;
            lockedLook = lockedRoot;
        }

        public void Show(CosmeticStage stage, string staticAddress, string nameLabel, bool earned)
        {
            if (!gameObject.activeSelf) gameObject.SetActive(true);
            icon.Show(stage, new CosmeticArt(staticAddress, null), null);
            medalName.text = nameLabel;
            if (lockedLook.activeSelf == earned) lockedLook.SetActive(!earned);
        }

        public void Hide()
        {
            if (gameObject.activeSelf) gameObject.SetActive(false);
        }
    }
}
