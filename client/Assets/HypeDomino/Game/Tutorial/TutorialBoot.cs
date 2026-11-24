using UnityEngine;

namespace HypeDomino.Game.Tutorial
{
    public static class TutorialBoot
    {
        public static TutorialOffer Offer { get; private set; }

        [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterSceneLoad)]
        static void Start()
        {
            Offer = new TutorialOffer();
            Offer.Follow(AppBoot.Flow);
        }
    }
}
