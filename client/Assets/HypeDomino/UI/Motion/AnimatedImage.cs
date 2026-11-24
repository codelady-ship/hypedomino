using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Motion
{
    [RequireComponent(typeof(Image))]
    public sealed class AnimatedImage : MonoBehaviour
    {
        Image image;
        LoopArt art;
        AnimationBudget budget;
        EffectLevel level;
        float startedAt;

        public bool IsPlaying { get; private set; }

        public Sprite Shown => Target.sprite;

        Image Target => image != null ? image : image = GetComponent<Image>();

        public void Show(LoopArt loop, EffectLevel effectLevel, bool reduceMotion, AnimationBudget shared, float now)
        {
            GiveBackSlot();
            art = loop;
            budget = shared;
            level = effectLevel;
            startedAt = now;
            IsPlaying = effectLevel != EffectLevel.Low && !reduceMotion && loop.IsAnimated && shared.TryAcquire();
            Target.sprite = IsPlaying ? loop.Frame(effectLevel, 0f) : loop.Still;
            enabled = IsPlaying;
        }

        public void ShowStill(Sprite sprite)
        {
            GiveBackSlot();
            Target.sprite = sprite;
            enabled = false;
        }

        public void Advance(float now)
        {
            if (!IsPlaying) return;
            var frame = art.Frame(level, now - startedAt);
            if (!ReferenceEquals(frame, Target.sprite)) Target.sprite = frame;
        }

        void Update() => Advance(Time.unscaledTime);

        void OnDisable()
        {
            if (!IsPlaying) return;
            GiveBackSlot();
            if (image != null) image.sprite = art.Still;
        }

        void GiveBackSlot()
        {
            if (!IsPlaying) return;
            IsPlaying = false;
            budget.Release();
        }
    }
}
