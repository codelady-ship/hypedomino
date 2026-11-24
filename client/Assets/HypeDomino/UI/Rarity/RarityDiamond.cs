using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Rarity
{
    public sealed class RarityDiamond : MonoBehaviour
    {
        [SerializeField] Image diamond;
        [SerializeField] Image pips;
        [SerializeField] Sprite[] pipOverlays;

        RarityTable table;
        string rarityId;

        public bool IsShown => diamond.enabled;

        public void Configure(Image diamondImage, Image pipImage, Sprite[] overlaysByPipCount)
        {
            diamond = diamondImage;
            pips = pipImage;
            pipOverlays = overlaysByPipCount;
        }

        public void Show(RarityTable rarities, string id)
        {
            table = rarities;
            rarityId = id;
            Refresh();
        }

        void OnRectTransformDimensionsChange()
        {
            if (table != null) Refresh();
        }

        void Refresh()
        {
            if (!Fits() || !table.TryGet(rarityId, out var tier) || tier.Pips > pipOverlays.Length)
            {
                SetShown(false);
                return;
            }
            diamond.color = tier.Fill;
            pips.color = tier.On;
            pips.sprite = pipOverlays[tier.Pips - 1];
            SetShown(true);
        }

        bool Fits()
        {
            var box = ((RectTransform)transform).rect;
            return Mathf.Min(box.width, box.height) >= table.DiamondMinSize;
        }

        void SetShown(bool shown)
        {
            diamond.enabled = shown;
            pips.enabled = shown;
        }
    }
}
