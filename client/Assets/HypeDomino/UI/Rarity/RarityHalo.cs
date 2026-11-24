using HypeDomino.UI.Motion;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Rarity
{
    [RequireComponent(typeof(Image))]
    public sealed class RarityHalo : MonoBehaviour
    {
        const float FullTurnDegrees = 360f;

        [SerializeField] Sprite halo;
        [SerializeField] Sprite flowHalo;

        Image image;
        float periodSeconds;
        float startedAt;

        public bool IsShown => Target.enabled;

        public bool IsRotating { get; private set; }

        Image Target => image != null ? image : image = GetComponent<Image>();

        public void Configure(Sprite haloSprite, Sprite flowHaloSprite)
        {
            halo = haloSprite;
            flowHalo = flowHaloSprite;
        }

        public void Show(RarityTable table, string rarityId, EffectLevel level, bool reduceMotion, float now)
        {
            IsRotating = false;
            transform.localRotation = Quaternion.identity;
            if (!table.TryGet(rarityId, out var tier) || tier.GlowAlpha <= 0f)
            {
                Target.enabled = false;
                enabled = false;
                return;
            }
            var tint = tier.Fill;
            tint.a = tier.GlowAlpha;
            Target.sprite = tier.Flow ? flowHalo : halo;
            Target.color = tint;
            Target.enabled = true;
            transform.localScale = Vector3.one * tier.GlowScale;
            periodSeconds = table.FlowPeriodSeconds;
            startedAt = now;
            IsRotating = tier.Flow && level != EffectLevel.Low && !reduceMotion && periodSeconds > 0f;
            enabled = IsRotating;
        }

        public void Advance(float now)
        {
            if (!IsRotating) return;
            var turns = Mathf.Repeat(now - startedAt, periodSeconds) / periodSeconds;
            transform.localRotation = Quaternion.Euler(0f, 0f, -FullTurnDegrees * turns);
        }

        void Update() => Advance(Time.unscaledTime);
    }
}
