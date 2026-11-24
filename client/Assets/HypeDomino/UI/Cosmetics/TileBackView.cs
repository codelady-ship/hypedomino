using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Cosmetics
{
    public sealed class TileBackView : MonoBehaviour
    {
        [SerializeField] CosmeticImage lead;
        [SerializeField] Image[] followers;
        [SerializeField] Sprite builtIn;

        Sprite mirrored;

        public int Count { get; private set; }

        public int Capacity => followers.Length + 1;

        public CosmeticImage Lead => lead;

        public void Configure(CosmeticImage leadBack, Image[] followerBacks, Sprite builtInBack)
        {
            lead = leadBack;
            followers = followerBacks;
            builtIn = builtInBack;
        }

        public void Show(CosmeticStage stage, CosmeticArt skin, int count)
        {
            lead.Show(stage, skin, builtIn);
            SetCount(count);
        }

        public void SetCount(int count)
        {
            Count = Mathf.Clamp(count, 0, Capacity);
            lead.GetComponent<CanvasRenderer>().SetAlpha(Count > 0 ? 1f : 0f);
            for (var i = 0; i < followers.Length; i++) followers[i].enabled = i + 1 < Count;
            Mirror();
        }

        public void RefreshEffects()
        {
            lead.Refresh();
            Mirror();
        }

        void LateUpdate() => Mirror();

        void Mirror()
        {
            var shown = lead.Shown;
            if (ReferenceEquals(shown, mirrored)) return;
            mirrored = shown;
            for (var i = 0; i < followers.Length; i++) followers[i].sprite = shown;
        }
    }
}
