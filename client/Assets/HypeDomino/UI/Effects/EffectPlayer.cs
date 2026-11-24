using System;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using Unity.Profiling;
using UnityEngine;
using UnityEngine.Sprites;

namespace HypeDomino.UI.Effects
{
    [DisallowMultipleComponent]
    [RequireComponent(typeof(RectTransform))]
    public sealed class EffectPlayer : MonoBehaviour, IEffectSettingsTarget
    {
        public const string ProfilerMarkerName = "Effect.player";
        public const string TrailSprite = "streak";

        const uint XorShiftSeed = 2463534242u;
        const float UnitRange = 16777216f;
        const uint UnitMask = 0xFFFFFFu;
        const float FullTurnRadians = Mathf.PI * 2f;
        const float MillisecondsPerSecond = 1000f;
        const float Half = 0.5f;

        static readonly ProfilerMarker TickMarker = new(ProfilerMarkerName);

        [SerializeField] TextAsset effects;
        [SerializeField] Sprite[] sprites = Array.Empty<Sprite>();
        [SerializeField] Material additive;
        [SerializeField] Material alpha;

        EffectLibrary library;
        ParticleBudget budget;
        EffectPool[] pools;
        EffectQuads[] quadsOf;
        bool[] drawn;
        Vector2[] halfSizes;
        int slotsPerKind;
        int trailSprite;
        uint random;

        public bool IsBuilt => library != null;

        public EffectLibrary Library => library;

        public ParticleBudget Budget => budget;

        public EffectLevel Level => budget.Level;

        public bool ReduceMotion => budget.ReduceMotion;

        public EffectQuads QuadsOf(EffectMaterial material) => quadsOf[(int)material];

        public EffectPool PoolOf(TokenEffectKind kind) => pools[(int)kind];

        public int LiveQuads => quadsOf[0].Count + quadsOf[1].Count;

        public int ParticleQuads => CountParticles(quadsOf[0]) + CountParticles(quadsOf[1]);

        void Awake()
        {
            if (effects != null && library == null) Build(EffectLibrary.Parse(effects.text), sprites, additive, alpha, EffectLevel.Medium, false);
        }

        public void Build(EffectLibrary effectLibrary, Sprite[] effectSprites, Material additiveMaterial, Material alphaMaterial, EffectLevel level, bool reduceMotion)
        {
            library = effectLibrary;
            budget = new ParticleBudget(level, reduceMotion);
            BuildPools();
            if (!TryGetComponent(out Canvas _)) gameObject.AddComponent<Canvas>();
            var bySprite = SpritesByName(effectSprites);
            var uvs = new Vector4[bySprite.Length];
            halfSizes = new Vector2[bySprite.Length];
            var pixelsPerUnit = GetComponentInParent<Canvas>().referencePixelsPerUnit;
            for (var sprite = 0; sprite < bySprite.Length; sprite++)
            {
                uvs[sprite] = DataUtility.GetOuterUV(bySprite[sprite]);
                halfSizes[sprite] = bySprite[sprite].rect.size * (pixelsPerUnit / bySprite[sprite].pixelsPerUnit) * Half;
            }
            var capacity = TokenEffectLevel.HighMaxParticles + LayerCapacity();
            quadsOf ??= new[] { QuadLayer(nameof(EffectMaterial.Additive)), QuadLayer(nameof(EffectMaterial.Alpha)) };
            quadsOf[(int)EffectMaterial.Additive].Build(capacity, uvs, TextureOf(bySprite, EffectMaterial.Additive), additiveMaterial);
            quadsOf[(int)EffectMaterial.Alpha].Build(capacity, uvs, TextureOf(bySprite, EffectMaterial.Alpha), alphaMaterial);
            quadsOf[(int)EffectMaterial.Alpha].transform.SetAsFirstSibling();
            drawn = new bool[quadsOf.Length];
            trailSprite = IndexOfSprite(TrailSprite);
            random = XorShiftSeed;
        }

        public void Configure(EffectLevel level, bool reduceMotion)
        {
            if (level == budget.Level && reduceMotion == budget.ReduceMotion) return;
            EndAll();
            budget.Configure(level, reduceMotion);
        }

        public void EndAll()
        {
            foreach (var pool in pools) pool.EndAll();
            foreach (var quads in quadsOf) quads.Clear();
        }

        public bool Play(in EffectRequest request)
        {
            if (library == null) return false;
            var kind = request.Kind;
            if (kind == TokenEffectKind.ScoreCascade && request.Step <= 0) return false;
            var particleRecipe = library.Recipe(kind, budget.ParticleLevel);
            var wanted = kind == TokenEffectKind.ScoreCascade
                ? ScoreCascade.Burst(request.Step, budget.Level, budget.ReduceMotion).Particles
                : particleRecipe.Particles;
            var pool = pools[(int)kind];
            var slot = pool.Begin(request.StartMs, wanted);
            var owner = (int)kind * slotsPerKind + slot;
            Purge(owner);
            var space = quadsOf[(int)EffectMaterial.Additive].rectTransform;
            var at = (Vector2)space.InverseTransformPoint(request.At);
            var reach = LocalReach(space, request.Reach);
            var trail = (Vector2)space.InverseTransformPoint(request.Trail);
            SpawnLayers(library.Recipe(kind, budget.Level), owner, at, in request);
            SpawnParticles(particleRecipe, pool.ParticlesOf(slot), owner, at, reach, trail, in request);
            return true;
        }

        public void Tick(float nowMs)
        {
            using (TickMarker.Auto())
            {
                foreach (var pool in pools) pool.Tick(nowMs);
                for (var material = 0; material < quadsOf.Length; material++) Advance(material, nowMs);
            }
        }

        void Update()
        {
            if (library != null) Tick(Time.unscaledTime * MillisecondsPerSecond);
        }

        void BuildPools()
        {
            var kinds = TokenEffect.NameOfKind.Length;
            pools = new EffectPool[kinds];
            slotsPerKind = 0;
            for (var kind = 0; kind < kinds; kind++)
            {
                pools[kind] = new EffectPool((TokenEffectKind)kind, budget);
                slotsPerKind = Math.Max(slotsPerKind, pools[kind].Capacity);
            }
        }

        int LayerCapacity()
        {
            var slots = 0;
            foreach (var pooled in TokenEffect.PooledOfKind) slots += pooled;
            return slots * library.MaxLayers;
        }

        Sprite[] SpritesByName(Sprite[] given)
        {
            var bySprite = new Sprite[library.SpriteCount];
            for (var sprite = 0; sprite < bySprite.Length; sprite++)
            {
                foreach (var candidate in given)
                    if (candidate != null && candidate.name == library.SpriteName(sprite)) bySprite[sprite] = candidate;
                if (bySprite[sprite] == null) throw new ArgumentException($"effect sprite {library.SpriteName(sprite)} is not set", nameof(given));
            }
            return bySprite;
        }

        Texture TextureOf(Sprite[] bySprite, EffectMaterial material)
        {
            for (var sprite = 0; sprite < bySprite.Length; sprite++)
                if (library.MaterialOf(sprite) == material) return bySprite[sprite].texture;
            return null;
        }

        EffectQuads QuadLayer(string layerName)
        {
            var layer = new GameObject(layerName, typeof(RectTransform));
            var rect = (RectTransform)layer.transform;
            rect.SetParent(transform, false);
            rect.anchorMin = Vector2.zero;
            rect.anchorMax = Vector2.one;
            rect.offsetMin = Vector2.zero;
            rect.offsetMax = Vector2.zero;
            return layer.AddComponent<EffectQuads>();
        }

        int IndexOfSprite(string spriteName)
        {
            for (var sprite = 0; sprite < library.SpriteCount; sprite++)
                if (library.SpriteName(sprite) == spriteName) return sprite;
            return -1;
        }

        void SpawnLayers(EffectRecipe recipe, int owner, Vector2 at, in EffectRequest request)
        {
            var durationMs = TokenEffect.DurationMsOfKind[(int)request.Kind];
            var holdMs = Mathf.Max(durationMs - TokenDurationMs.Fast - TokenDurationMs.Base, 0);
            for (var index = 0; index < recipe.Layers; index++)
            {
                var layer = library.Layer(recipe.FirstLayer + index);
                var glow = EffectLibrary.GlowOf(layer.Glow, request.Step);
                Add(new EffectQuad
                {
                    Owner = owner,
                    Sprite = layer.Sprite,
                    Motion = EffectMotion.Pop,
                    From = at,
                    To = at,
                    Center = at,
                    HalfSize = halfSizes[layer.Sprite],
                    StartMs = request.StartMs,
                    DurationMs = durationMs,
                    HoldMs = holdMs,
                    Strength = glow >= 0 ? TokenGlow.AlphaOfLevel[glow] : 1f,
                    Grow = glow >= 0 ? TokenGlow.ScaleOfLevel[glow] : 1f,
                    Tint = layer.Tint.For(request.Team),
                    Cos = 1f,
                });
            }
        }

        void SpawnParticles(EffectRecipe recipe, int granted, int owner, Vector2 at, Vector2 reach, Vector2 trail, in EffectRequest request)
        {
            if (granted <= 0 || recipe.Particles <= 0) return;
            var left = granted;
            var last = recipe.ParticleEntries - 1;
            for (var entry = 0; entry <= last; entry++)
            {
                var particles = library.Particles(recipe.FirstParticles + entry);
                var count = entry == last ? left : Math.Min(left, (granted * particles.Count + recipe.Particles / 2) / recipe.Particles);
                left -= count;
                for (var n = 0; n < count; n++) Add(Particle(particles, owner, at, reach, trail, in request));
            }
        }

        EffectQuad Particle(EffectParticles particles, int owner, Vector2 at, Vector2 reach, Vector2 trail, in EffectRequest request)
        {
            var durationMs = TokenEffect.DurationMsOfKind[(int)request.Kind];
            var quad = new EffectQuad
            {
                Owner = owner,
                Sprite = particles.Sprite,
                Particle = true,
                HalfSize = halfSizes[particles.Sprite],
                Strength = 1f,
                Grow = 1f,
                Tint = particles.Tint.For(request.Team),
            };
            if (particles.Sprite == trailSprite && request.HasTrail)
            {
                quad.Motion = EffectMotion.Travel;
                quad.From = (request.TrailArrives ? trail : at) + Disc(reach);
                quad.To = request.TrailArrives ? at : trail;
                quad.StartMs = request.StartMs;
                quad.DurationMs = Mathf.Min(request.TravelMs, durationMs);
            }
            else
            {
                var delayMs = request.HasTrail && request.TrailArrives ? Mathf.Min(request.TravelMs, durationMs) : 0f;
                quad.Motion = EffectMotion.Burst;
                quad.From = at;
                quad.To = at + Disc(reach);
                quad.StartMs = request.StartMs + delayMs;
                quad.DurationMs = durationMs - delayMs;
            }
            var turn = particles.Sprite == trailSprite ? Mathf.Atan2(quad.To.y - quad.From.y, quad.To.x - quad.From.x) : Next() * FullTurnRadians;
            quad.Cos = Mathf.Cos(turn);
            quad.Sin = Mathf.Sin(turn);
            quad.Center = quad.From;
            return quad;
        }

        void Add(in EffectQuad quad) => quadsOf[(int)library.MaterialOf(quad.Sprite)].TryAdd(in quad);

        void Purge(int owner)
        {
            foreach (var quads in quadsOf)
                for (var at = quads.Count - 1; at >= 0; at--)
                    if (quads[at].Owner == owner) quads.RemoveAt(at);
        }

        void Advance(int material, float nowMs)
        {
            var quads = quadsOf[material];
            for (var at = quads.Count - 1; at >= 0; at--)
            {
                ref var quad = ref quads[at];
                if (!pools[quad.Owner / slotsPerKind].IsLive(quad.Owner % slotsPerKind))
                {
                    quads.RemoveAt(at);
                    continue;
                }
                Animate(ref quad, nowMs - quad.StartMs);
            }
            if (quads.Count == 0 && !drawn[material]) return;
            drawn[material] = quads.Count > 0;
            quads.SetVerticesDirty();
        }

        void Animate(ref EffectQuad quad, float elapsedMs)
        {
            if (elapsedMs < 0f)
            {
                quad.Alpha = 0;
                return;
            }
            if (quad.Motion == EffectMotion.Pop)
            {
                var shown = ScorePopup.At(elapsedMs, (int)quad.HoldMs, budget.ReduceMotion, out var popup);
                quad.Scale = quad.Grow * popup.Scale;
                quad.Alpha = shown ? AlphaOf(quad.Tint, quad.Strength * popup.Alpha) : (byte)0;
                return;
            }
            var t = quad.DurationMs > 0f ? elapsedMs / quad.DurationMs : 1f;
            if (t >= 1f)
            {
                quad.Alpha = 0;
                return;
            }
            var curve = quad.Motion == EffectMotion.Travel ? TokenEasing.Standard : TokenEasing.Enter;
            quad.Center = Vector2.LerpUnclamped(quad.From, quad.To, Easing.Evaluate(curve, t));
            quad.Scale = 1f;
            quad.Alpha = AlphaOf(quad.Tint, 1f - Easing.Evaluate(TokenEasing.Exit, t));
        }

        static byte AlphaOf(Color32 tint, float strength) => (byte)Mathf.Clamp(Mathf.RoundToInt(tint.a * strength), 0, byte.MaxValue);

        static Vector2 LocalReach(Transform space, Vector2 reach)
        {
            var scale = space.lossyScale;
            return new Vector2(scale.x != 0f ? reach.x / scale.x : 0f, scale.y != 0f ? reach.y / scale.y : 0f);
        }

        static int CountParticles(EffectQuads quads)
        {
            var count = 0;
            for (var at = 0; at < quads.Count; at++)
                if (quads[at].Particle) count++;
            return count;
        }

        Vector2 Disc(Vector2 reach)
        {
            var turn = Next() * FullTurnRadians;
            var distance = Mathf.Sqrt(Next());
            return new Vector2(Mathf.Cos(turn) * reach.x * distance, Mathf.Sin(turn) * reach.y * distance);
        }

        float Next()
        {
            random ^= random << 13;
            random ^= random >> 17;
            random ^= random << 5;
            return (random & UnitMask) / UnitRange;
        }
    }
}
