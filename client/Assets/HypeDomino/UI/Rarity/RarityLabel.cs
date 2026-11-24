using HypeDomino.UI.Localization;
using TMPro;
using UnityEngine;

namespace HypeDomino.UI.Rarity
{
    [RequireComponent(typeof(TMP_Text))]
    public sealed class RarityLabel : MonoBehaviour
    {
        TMP_Text label;
        RarityTable table;
        GameStrings strings;
        string rarityId;

        public bool IsShown => Target.enabled;

        TMP_Text Target => label != null ? label : label = GetComponent<TMP_Text>();

        public void Show(RarityTable rarities, GameStrings names, string id)
        {
            table = rarities;
            rarityId = id;
            Follow(names);
            Refresh();
        }

        void Follow(GameStrings names)
        {
            if (strings == names) return;
            if (strings != null) strings.Changed -= Refresh;
            strings = names;
            strings.Changed += Refresh;
        }

        void Refresh()
        {
            if (!table.TryGet(rarityId, out var tier))
            {
                Target.enabled = false;
                return;
            }
            Target.text = strings.Text(tier.NameKey);
            Target.color = tier.Text;
            Target.enabled = true;
        }

        void OnDestroy()
        {
            if (strings != null) strings.Changed -= Refresh;
        }
    }
}
