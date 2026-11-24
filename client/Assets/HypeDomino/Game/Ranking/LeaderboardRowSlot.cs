using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game.Ranking
{
    public sealed class LeaderboardRowSlot : MonoBehaviour
    {
        [SerializeField] TMP_Text position;
        [SerializeField] TMP_Text nickname;
        [SerializeField] Image badge;
        [SerializeField] TMP_Text lp;
        [SerializeField] GameObject ownHighlight;

        public void Configure(TMP_Text positionText, TMP_Text nicknameText, Image badgeImage, TMP_Text lpText, GameObject ownHighlightRoot)
        {
            position = positionText;
            nickname = nicknameText;
            badge = badgeImage;
            lp = lpText;
            ownHighlight = ownHighlightRoot;
        }

        public void Show(string positionLabel, string nicknameLabel, RankBadge rankBadge, Sprite[] tierBadges, string lpLabel, bool own)
        {
            position.text = positionLabel;
            nickname.text = nicknameLabel;
            RankBadgeArt.Show(badge, tierBadges, rankBadge);
            lp.text = lpLabel;
            if (ownHighlight.activeSelf != own) ownHighlight.SetActive(own);
        }
    }
}
