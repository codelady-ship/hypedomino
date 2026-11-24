using HypeDomino.UI.Localization;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game.Promo
{
    public sealed class PromoCodeView : MonoBehaviour
    {
        [SerializeField] TMP_InputField field;
        [SerializeField] Button paste;
        [SerializeField] Button redeem;
        [SerializeField] TMP_Text message;

        PromoCodeEntry entry;
        GameStrings strings;

        public void Configure(TMP_InputField codeField, Button pasteButton, Button redeemButton, TMP_Text messageLine)
        {
            Unbind();
            field = codeField;
            paste = pasteButton;
            redeem = redeemButton;
            message = messageLine;
        }

        public void Bind(PromoCodeEntry codeEntry, GameStrings gameStrings)
        {
            Unbind();
            entry = codeEntry;
            strings = gameStrings;
            entry.Changed += Show;
            strings.Changed += Show;
            field.onValueChanged.AddListener(OnEdited);
            paste.onClick.AddListener(OnPaste);
            redeem.onClick.AddListener(OnRedeem);
            Show();
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (entry == null) return;
            entry.Changed -= Show;
            strings.Changed -= Show;
            field.onValueChanged.RemoveListener(OnEdited);
            paste.onClick.RemoveListener(OnPaste);
            redeem.onClick.RemoveListener(OnRedeem);
            entry = null;
            strings = null;
        }

        void OnEdited(string typed) => entry.Edit(typed);

        void OnPaste() => entry.Paste(GUIUtility.systemCopyBuffer);

        void OnRedeem() => entry.Submit();

        void Show()
        {
            if (field.text != entry.Text) field.SetTextWithoutNotify(entry.Text);
            field.readOnly = entry.Busy;
            paste.interactable = !entry.Busy;
            redeem.interactable = entry.CanSubmit;
            var key = entry.MessageKey;
            message.enabled = key != null;
            message.text = key == null ? string.Empty : strings.Text(key);
        }
    }
}
