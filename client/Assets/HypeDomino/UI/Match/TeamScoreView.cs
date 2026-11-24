using HypeDomino.UI.Tokens;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Match
{
    public sealed class TeamScoreView : MonoBehaviour
    {
        [SerializeField] TMP_Text score;
        [SerializeField] Graphic gem;
        [SerializeField] Image meter;
        [SerializeField] GameObject lockBadge;

        bool hasScore;
        uint shownScore;

        public void Configure(TMP_Text scoreText, Graphic teamGem, Image meterFill, GameObject badge)
        {
            score = scoreText;
            gem = teamGem;
            meter = meterFill;
            lockBadge = badge;
            hasScore = false;
        }

        public void Show(Color32 team, uint points, float fill, bool locked)
        {
            gem.color = team;
            meter.color = team;
            meter.fillAmount = fill;
            score.color = TokenColor.RoleScoreText;
            if (lockBadge.activeSelf != locked) lockBadge.SetActive(locked);
            if (hasScore && shownScore == points) return;
            hasScore = true;
            shownScore = points;
            score.SetText("{0}", points);
        }
    }
}
