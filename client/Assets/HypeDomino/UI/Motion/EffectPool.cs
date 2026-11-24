using HypeDomino.UI.Tokens;
using Unity.Profiling;

namespace HypeDomino.UI.Motion
{
    public sealed class EffectPool
    {
        readonly ParticleBudget particles;
        readonly float[] startedMs;
        readonly int[] granted;
        readonly bool[] live;
        readonly float lifetimeMs;

        public EffectPool(TokenEffectKind kind, ParticleBudget particles)
        {
            Kind = kind;
            this.particles = particles;
            var pooled = TokenEffect.PooledOfKind[(int)kind];
            startedMs = new float[pooled];
            granted = new int[pooled];
            live = new bool[pooled];
            lifetimeMs = TokenEffect.DurationMsOfKind[(int)kind];
            Marker = new ProfilerMarker(MarkerNameOf(kind));
        }

        public TokenEffectKind Kind { get; }

        public ProfilerMarker Marker { get; }

        public int Capacity => live.Length;

        public int LiveCount { get; private set; }

        public static string MarkerNameOf(TokenEffectKind kind) => "Effect." + TokenEffect.NameOfKind[(int)kind];

        public bool IsLive(int slot) => live[slot];

        public int ParticlesOf(int slot) => granted[slot];

        public int Begin(float nowMs, int wantedParticles)
        {
            var slot = FreeOrOldest();
            End(slot);
            live[slot] = true;
            startedMs[slot] = nowMs;
            granted[slot] = particles.Take(Kind, wantedParticles);
            LiveCount++;
            return slot;
        }

        public void End(int slot)
        {
            if (!live[slot]) return;
            live[slot] = false;
            particles.Return(granted[slot]);
            granted[slot] = 0;
            LiveCount--;
        }

        public void Tick(float nowMs)
        {
            using (Marker.Auto())
            {
                for (var slot = 0; slot < live.Length; slot++)
                    if (live[slot] && nowMs - startedMs[slot] >= lifetimeMs) End(slot);
            }
        }

        public void EndAll()
        {
            for (var slot = 0; slot < live.Length; slot++) End(slot);
        }

        int FreeOrOldest()
        {
            var oldest = 0;
            for (var slot = 0; slot < live.Length; slot++)
            {
                if (!live[slot]) return slot;
                if (startedMs[slot] < startedMs[oldest]) oldest = slot;
            }
            return oldest;
        }
    }
}
