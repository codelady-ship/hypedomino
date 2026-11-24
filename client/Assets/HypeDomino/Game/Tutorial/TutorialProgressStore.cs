using UnityEngine;

namespace HypeDomino.Game.Tutorial
{
    public static class TutorialProgressStore
    {
        public const string FinishedKey = "tutorial.finished";

        public static bool ShouldOffer => PlayerPrefs.GetInt(FinishedKey, 0) == 0;

        public static bool Keep(TutorialRun run)
        {
            if (!run.IsFinished || !ShouldOffer) return false;
            PlayerPrefs.SetInt(FinishedKey, 1);
            PlayerPrefs.Save();
            return true;
        }
    }
}
