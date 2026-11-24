using HypeDomino.UI.Tokens;
using TMPro;
using UnityEngine;

namespace HypeDomino.UI.Match
{
    public sealed class ScorePanelView : MonoBehaviour
    {
        [SerializeField] TeamScoreView mine;
        [SerializeField] TeamScoreView theirs;
        [SerializeField] TMP_Text target;

        uint shownTarget;

        public TeamScoreView Mine => mine;
        public TeamScoreView Theirs => theirs;

        public void Configure(TeamScoreView mineView, TeamScoreView theirsView, TMP_Text targetText)
        {
            mine = mineView;
            theirs = theirsView;
            target = targetText;
            shownTarget = 0;
        }

        public void ShowTarget(uint points)
        {
            var visible = points > 0;
            if (target.enabled != visible) target.enabled = visible;
            if (!visible || points == shownTarget) return;
            shownTarget = points;
            target.color = TokenColor.RoleScoreText;
            target.SetText("{0}", points);
        }
    }
}
