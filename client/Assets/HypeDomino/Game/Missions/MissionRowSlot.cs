using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game.Missions
{
    public sealed class MissionRowSlot : MonoBehaviour
    {
        [SerializeField] TMP_Text missionName;
        [SerializeField] TMP_Text progress;
        [SerializeField] TMP_Text target;
        [SerializeField] Image progressFill;
        [SerializeField] GameObject completedMark;

        public void Configure(TMP_Text nameText, TMP_Text progressText, TMP_Text targetText, Image progressFillImage, GameObject completedRoot)
        {
            missionName = nameText;
            progress = progressText;
            target = targetText;
            progressFill = progressFillImage;
            completedMark = completedRoot;
        }

        public void Show(string nameLabel, string progressLabel, string targetLabel, float fill, bool completed)
        {
            missionName.text = nameLabel;
            progress.text = progressLabel;
            target.text = targetLabel;
            progressFill.fillAmount = fill;
            if (completedMark.activeSelf != completed) completedMark.SetActive(completed);
        }
    }
}
