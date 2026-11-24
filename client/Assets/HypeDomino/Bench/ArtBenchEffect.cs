using HypeDomino.UI.Effects;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.Bench
{
    public sealed class ArtBenchEffect : MonoBehaviour
    {
        const float RepeatGapSeconds = 0.6f;
        const float MillisecondsPerSecond = 1000f;
        const int CascadeStep = 1;

        EffectPlayer player;
        EffectLibrary library;
        ArtBenchContent content;
        TokenEffectKind kind;
        float durationSeconds;
        float nextAt;

        public TokenEffectKind Kind => kind;

        public void Configure(TokenEffectKind effectKind, EffectLibrary effects, ArtBenchContent art)
        {
            kind = effectKind;
            library = effects;
            content = art;
            durationSeconds = TokenEffect.DurationMsOfKind[(int)effectKind] / MillisecondsPerSecond;
            player = gameObject.AddComponent<EffectPlayer>();
        }

        public string Play(EffectLevel level, bool reduceMotion, float now)
        {
            if (!player.IsBuilt) player.Build(library, content.effectSprites, content.additive, content.alpha, level, reduceMotion);
            player.Configure(level, reduceMotion);
            Trigger(now);
            return $"particleLevel={player.Budget.ParticleLevel} quads={player.LiveQuads} particles={player.ParticleQuads}";
        }

        void Update()
        {
            if (player.IsBuilt && Time.unscaledTime >= nextAt) Trigger(Time.unscaledTime);
        }

        void Trigger(float now)
        {
            var request = EffectRequest.On(kind, (RectTransform)transform, now * MillisecondsPerSecond).ForTeam(TokenColor.RoleTeamAFill).AtStep(CascadeStep);
            player.Play(in request);
            nextAt = now + durationSeconds + RepeatGapSeconds;
        }
    }
}
