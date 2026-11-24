using HypeDomino.UI.Localization;
using TMPro;
using UnityEngine;

namespace HypeDomino.Game.Pass
{
    public sealed class PassCellView : MonoBehaviour
    {
        [SerializeField] TMP_Text coins;
        [SerializeField] TMP_Text crystals;
        [SerializeField] GameObject cosmetics;
        [SerializeField] GameObject earned;
        [SerializeField] GameObject locked;

        public void Configure(TMP_Text coinsText, TMP_Text crystalsText, GameObject cosmeticsMark, GameObject earnedMark, GameObject lockedMark)
        {
            coins = coinsText;
            crystals = crystalsText;
            cosmetics = cosmeticsMark;
            earned = earnedMark;
            locked = lockedMark;
        }

        public void Show(PassCell cell, string language)
        {
            var reward = cell.Reward;
            ShowAmount(coins, reward?.Coins ?? 0, language);
            ShowAmount(crystals, reward?.Crystals ?? 0, language);
            cosmetics.SetActive(cell.CosmeticCount > 0);
            earned.SetActive(cell.State == PassRewardState.Earned);
            locked.SetActive(cell.IsLocked);
        }

        static void ShowAmount(TMP_Text text, long amount, string language)
        {
            text.enabled = amount > 0;
            if (amount > 0) text.text = MessageFormat.Number(amount, language);
        }
    }
}
