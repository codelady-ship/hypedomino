using System;
using HypeDomino.UI.Tokens;

namespace HypeDomino.UI.Motion
{
    public sealed class ParticleBudget
    {
        public ParticleBudget(EffectLevel level, bool reduceMotion) => Configure(level, reduceMotion);

        public EffectLevel Level { get; private set; }

        public bool ReduceMotion { get; private set; }

        public int Limit { get; private set; }

        public int Live { get; private set; }

        public EffectLevel ParticleLevel => ReduceMotion ? EffectLevel.Low : Level;

        public void Configure(EffectLevel level, bool reduceMotion)
        {
            Level = level;
            ReduceMotion = reduceMotion;
            Limit = LimitOf(ParticleLevel);
        }

        public int MaxFor(TokenEffectKind kind) => ParticlesOf(ParticleLevel)[(int)kind];

        public int Take(TokenEffectKind kind, int wanted)
        {
            var granted = Math.Min(Math.Min(Math.Max(wanted, 0), MaxFor(kind)), Math.Max(Limit - Live, 0));
            Live += granted;
            return granted;
        }

        public void Return(int count) => Live = Math.Max(Live - Math.Max(count, 0), 0);

        static int[] ParticlesOf(EffectLevel level)
        {
            switch (level)
            {
                case EffectLevel.High: return TokenEffect.HighParticlesOfKind;
                case EffectLevel.Medium: return TokenEffect.MediumParticlesOfKind;
                default: return TokenEffect.LowParticlesOfKind;
            }
        }

        static int LimitOf(EffectLevel level)
        {
            switch (level)
            {
                case EffectLevel.High: return TokenEffectLevel.HighMaxParticles;
                case EffectLevel.Medium: return TokenEffectLevel.MediumMaxParticles;
                default: return TokenEffectLevel.LowMaxParticles;
            }
        }
    }
}
