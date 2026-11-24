using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Tokens;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Match
{
    public sealed class LoadingCardView : MonoBehaviour
    {
        [SerializeField] AvatarView avatar;
        [SerializeField] TMP_Text nickname;
        [SerializeField] Image progress;
        [SerializeField] TMP_Text status;

        AvatarLook shownLook;
        bool showingLook;
        bool showingName;
        string shownName;
        bool showingStatus;
        string shownStatus;

        public void Configure(AvatarView avatarView, TMP_Text nicknameText, Image progressFill, TMP_Text statusText)
        {
            avatar = avatarView;
            nickname = nicknameText;
            progress = progressFill;
            status = statusText;
            showingLook = false;
            showingName = false;
            showingStatus = false;
        }

        public void SetVisible(bool visible)
        {
            if (gameObject.activeSelf != visible) gameObject.SetActive(visible);
        }

        public void ShowPlayer(CosmeticStage stage, AvatarLook look, string name)
        {
            if (!showingLook || !shownLook.IsSameAs(look))
            {
                shownLook = look;
                showingLook = true;
                avatar.Show(stage, look);
            }
            var text = name ?? string.Empty;
            if (showingName && text == shownName) return;
            showingName = true;
            shownName = text;
            nickname.color = TokenColor.TextPrimary;
            nickname.SetText(text);
        }

        public void ShowProgress(float fill, string statusText, Color32 statusColor)
        {
            progress.color = TokenColor.RolePrimaryFill;
            progress.fillAmount = fill;
            status.color = statusColor;
            var text = statusText ?? string.Empty;
            if (showingStatus && text == shownStatus) return;
            showingStatus = true;
            shownStatus = text;
            status.SetText(text);
        }

        public void RefreshEffects()
        {
            if (showingLook) avatar.RefreshEffects();
        }
    }
}
