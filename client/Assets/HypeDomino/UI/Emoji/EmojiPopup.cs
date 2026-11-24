using HypeDomino.UI.Motion;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Emoji
{
    [RequireComponent(typeof(Image))]
    public sealed class EmojiPopup : MonoBehaviour
    {
        Image image;
        LoopArt art;
        EffectLevel level;
        bool reduceMotion;
        float startedAt;

        public Sprite Shown => image.sprite;
        public float Alpha => image.color.a;

        void Awake() => image = GetComponent<Image>();

        public void Begin(LoopArt emoji, EffectLevel effectLevel, bool reduce, float now)
        {
            if (image == null) image = GetComponent<Image>();
            art = emoji;
            level = effectLevel;
            reduceMotion = reduce;
            startedAt = now;
            Advance(now);
        }

        public bool Advance(float now)
        {
            var elapsed = now - startedAt;
            var live = EmojiPop.At(elapsed * 1000f, reduceMotion, out var pop);
            image.sprite = art.Frame(level, elapsed);
            transform.localScale = new Vector3(pop.Scale, pop.Scale, 1f);
            var tint = image.color;
            tint.a = pop.Alpha;
            image.color = tint;
            return live;
        }
    }
}
