using System;
using System.Collections.Generic;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.UI.Effects
{
    public enum EffectMaterial
    {
        Additive,
        Alpha
    }

    public readonly struct EffectTint
    {
        public const string TeamName = "team";

        public readonly Color32 Color;
        public readonly bool Team;

        public EffectTint(Color32 color, bool team)
        {
            Color = color;
            Team = team;
        }

        public Color32 For(Color32 team) => Team ? team : Color;
    }

    public readonly struct EffectLayer
    {
        public const int NoGlow = -1;
        public const int StepGlow = -2;

        public readonly int Sprite;
        public readonly EffectTint Tint;
        public readonly int Glow;

        public EffectLayer(int sprite, EffectTint tint, int glow)
        {
            Sprite = sprite;
            Tint = tint;
            Glow = glow;
        }
    }

    public readonly struct EffectParticles
    {
        public readonly int Sprite;
        public readonly int Count;
        public readonly EffectTint Tint;

        public EffectParticles(int sprite, int count, EffectTint tint)
        {
            Sprite = sprite;
            Count = count;
            Tint = tint;
        }
    }

    public readonly struct EffectRecipe
    {
        public readonly int FirstLayer;
        public readonly int Layers;
        public readonly int FirstParticles;
        public readonly int ParticleEntries;
        public readonly int Particles;
        public readonly int OverdrawLayers;

        public EffectRecipe(int firstLayer, int layers, int firstParticles, int particleEntries, int particles, int overdrawLayers)
        {
            FirstLayer = firstLayer;
            Layers = layers;
            FirstParticles = firstParticles;
            ParticleEntries = particleEntries;
            Particles = particles;
            OverdrawLayers = overdrawLayers;
        }
    }

    public sealed class EffectLibrary
    {
        public const int Levels = 3;

        const string AdditiveName = "additive";
        const string AlphaName = "alpha";
        const string StepGlowName = "step";

        [Serializable]
        sealed class SpriteEntry
        {
            public string name;
            public string material;
        }

        [Serializable]
        sealed class LayerEntry
        {
            public string sprite;
            public string tint;
            public string glow;
        }

        [Serializable]
        sealed class ParticleEntry
        {
            public string sprite;
            public int count;
            public string tint;
        }

        [Serializable]
        sealed class LevelEntry
        {
            public LayerEntry[] sprites;
            public ParticleEntry[] particles;
            public int overdrawLayers;
        }

        [Serializable]
        sealed class LevelsEntry
        {
            public LevelEntry low;
            public LevelEntry medium;
            public LevelEntry high;
        }

        [Serializable]
        sealed class EffectEntry
        {
            public string name;
            public LevelsEntry levels;
        }

        [Serializable]
        sealed class LibraryFile
        {
            public SpriteEntry[] sprites;
            public EffectEntry[] effects;
        }

        static readonly int[] MaxOverdrawLayersOfLevel = { TokenEffectLevel.LowMaxOverdrawLayers, TokenEffectLevel.MediumMaxOverdrawLayers, TokenEffectLevel.HighMaxOverdrawLayers };
        static readonly int[][] MaxParticlesOfLevel = { TokenEffect.LowParticlesOfKind, TokenEffect.MediumParticlesOfKind, TokenEffect.HighParticlesOfKind };

        readonly string[] spriteNames;
        readonly EffectMaterial[] materials;
        readonly List<EffectLayer> layers = new();
        readonly List<EffectParticles> particles = new();
        readonly EffectRecipe[] recipes = new EffectRecipe[TokenEffect.NameOfKind.Length * Levels];

        EffectLibrary(LibraryFile file)
        {
            var sprites = file.sprites ?? Array.Empty<SpriteEntry>();
            spriteNames = new string[sprites.Length];
            materials = new EffectMaterial[sprites.Length];
            for (var at = 0; at < sprites.Length; at++) AddSprite(at, sprites[at]);
            var effects = file.effects ?? Array.Empty<EffectEntry>();
            for (var kind = 0; kind < TokenEffect.NameOfKind.Length; kind++) AddEffect(kind, Find(effects, TokenEffect.NameOfKind[kind]));
            foreach (var effect in effects) KindOf(effect.name);
        }

        public int SpriteCount => spriteNames.Length;

        public int MaxLayers { get; private set; }

        public static EffectLibrary Parse(string json)
        {
            if (string.IsNullOrEmpty(json)) throw new FormatException("effects.json is empty");
            return new EffectLibrary(JsonUtility.FromJson<LibraryFile>(json));
        }

        public string SpriteName(int sprite) => spriteNames[sprite];

        public EffectMaterial MaterialOf(int sprite) => materials[sprite];

        public EffectRecipe Recipe(TokenEffectKind kind, EffectLevel level) => recipes[(int)kind * Levels + (int)level];

        public EffectLayer Layer(int index) => layers[index];

        public EffectParticles Particles(int index) => particles[index];

        public static int GlowOf(int glow, int step)
        {
            if (glow != EffectLayer.StepGlow) return glow;
            var steps = TokenScoreCascade.StepGlowLevel;
            return steps[Math.Min(Math.Max(step, 1), steps.Length) - 1];
        }

        void AddSprite(int at, SpriteEntry entry)
        {
            if (string.IsNullOrEmpty(entry.name)) throw new FormatException($"sprites[{at}] has no name");
            if (Array.IndexOf(spriteNames, entry.name, 0, at) >= 0) throw new FormatException($"sprite {entry.name} is listed twice");
            spriteNames[at] = entry.name;
            materials[at] = entry.material switch
            {
                AdditiveName => EffectMaterial.Additive,
                AlphaName => EffectMaterial.Alpha,
                _ => throw new FormatException($"sprite {entry.name}: unknown material {entry.material}")
            };
        }

        void AddEffect(int kind, EffectEntry effect)
        {
            var name = TokenEffect.NameOfKind[kind];
            if (effect?.levels == null) throw new FormatException($"effect {name} is missing");
            AddLevel(kind, EffectLevel.Low, effect.levels.low);
            AddLevel(kind, EffectLevel.Medium, effect.levels.medium);
            AddLevel(kind, EffectLevel.High, effect.levels.high);
        }

        void AddLevel(int kind, EffectLevel level, LevelEntry entry)
        {
            var where = $"{TokenEffect.NameOfKind[kind]}.{level}";
            if (entry == null) throw new FormatException($"{where} is missing");
            if (entry.overdrawLayers > MaxOverdrawLayersOfLevel[(int)level]) throw new FormatException($"{where}: {entry.overdrawLayers} overdraw layers pass the level's {MaxOverdrawLayersOfLevel[(int)level]}");
            var firstLayer = layers.Count;
            foreach (var layer in entry.sprites ?? Array.Empty<LayerEntry>()) layers.Add(new EffectLayer(SpriteOf(layer.sprite, where), TintOf(layer.tint, where), GlowNamed(layer.glow, where)));
            var firstParticles = particles.Count;
            var total = 0;
            foreach (var particle in entry.particles ?? Array.Empty<ParticleEntry>())
            {
                if (particle.count < 0) throw new FormatException($"{where}: negative count of {particle.sprite}");
                particles.Add(new EffectParticles(SpriteOf(particle.sprite, where), particle.count, TintOf(particle.tint, where)));
                total += particle.count;
            }
            var budget = MaxParticlesOfLevel[(int)level][kind];
            if (total > budget) throw new FormatException($"{where}: {total} particles pass the budget of {budget}");
            MaxLayers = Math.Max(MaxLayers, layers.Count - firstLayer);
            recipes[kind * Levels + (int)level] = new EffectRecipe(firstLayer, layers.Count - firstLayer, firstParticles, particles.Count - firstParticles, total, entry.overdrawLayers);
        }

        int SpriteOf(string name, string where)
        {
            var sprite = Array.IndexOf(spriteNames, name);
            if (sprite < 0) throw new FormatException($"{where}: unknown sprite {name}");
            return sprite;
        }

        static EffectTint TintOf(string path, string where)
        {
            if (path == EffectTint.TeamName) return new EffectTint(default, true);
            var at = Array.IndexOf(TokenColor.Paths, path);
            if (at < 0) throw new FormatException($"{where}: tint {path} is not a colour token");
            return new EffectTint(TokenColor.OfPath[at], false);
        }

        static int GlowNamed(string name, string where)
        {
            if (string.IsNullOrEmpty(name)) return EffectLayer.NoGlow;
            if (name == StepGlowName) return EffectLayer.StepGlow;
            var glow = Array.IndexOf(TokenGlow.NameOfLevel, name);
            if (glow < 0) throw new FormatException($"{where}: glow {name} is not a glow token");
            return glow;
        }

        static EffectEntry Find(EffectEntry[] effects, string name)
        {
            foreach (var effect in effects)
                if (effect.name == name) return effect;
            return null;
        }

        static void KindOf(string name)
        {
            if (Array.IndexOf(TokenEffect.NameOfKind, name) < 0) throw new FormatException($"effect {name} has no budget in effects.budgets");
        }
    }
}
