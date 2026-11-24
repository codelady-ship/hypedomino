using System;
using HypeDomino.UI.Effects;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using UnityEditor;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class EffectLibraryTests
    {
        public const string EffectsFolder = "Assets/HypeDomino/UI/Effects/";

        const string Minimal = @"{
  ""sprites"": [{ ""name"": ""spark"", ""material"": ""additive"" }, { ""name"": ""glow"", ""material"": ""additive"" }],
  ""effects"": [EFFECTS]
}";

        static readonly EffectLevel[] AllLevels = { EffectLevel.Low, EffectLevel.Medium, EffectLevel.High };

        public static EffectLibrary Committed() => EffectLibrary.Parse(AssetDatabase.LoadAssetAtPath<TextAsset>(EffectsFolder + "effects.json").text);

        static string EmptyEffect(string name) =>
            $@"{{ ""name"": ""{name}"", ""levels"": {{ ""low"": {{ ""overdrawLayers"": 1 }}, ""medium"": {{ ""overdrawLayers"": 1 }}, ""high"": {{ ""overdrawLayers"": 1 }} }} }}";

        static string With(string tilePlaceLow)
        {
            var effects = new string[TokenEffect.NameOfKind.Length];
            for (var kind = 0; kind < effects.Length; kind++) effects[kind] = EmptyEffect(TokenEffect.NameOfKind[kind]);
            effects[(int)TokenEffectKind.TilePlace] = $@"{{ ""name"": ""tilePlace"", ""levels"": {{ ""low"": {tilePlaceLow}, ""medium"": {{ ""overdrawLayers"": 1 }}, ""high"": {{ ""overdrawLayers"": 1 }} }} }}";
            return Minimal.Replace("EFFECTS", string.Join(",", effects));
        }

        [Test]
        public void TheCommittedLibraryDefinesEveryBudgetedEffectAtEveryLevel()
        {
            var library = Committed();

            foreach (TokenEffectKind kind in Enum.GetValues(typeof(TokenEffectKind)))
            foreach (var level in AllLevels)
            {
                var recipe = library.Recipe(kind, level);
                Assert.LessOrEqual(recipe.Particles, new ParticleBudget(level, false).MaxFor(kind), $"{kind} {level}");
                Assert.LessOrEqual(recipe.OverdrawLayers, MaxOverdrawLayers(level), $"{kind} {level}");
            }
            Assert.AreEqual(2, library.MaxLayers);
        }

        [TestCase(EffectLevel.Low, 4)]
        [TestCase(EffectLevel.Medium, 12)]
        [TestCase(EffectLevel.High, 24)]
        public void EachLevelReadsItsOwnCounts(EffectLevel level, int tilePlaceParticles)
        {
            var library = Committed();

            Assert.AreEqual(tilePlaceParticles, library.Recipe(TokenEffectKind.TilePlace, level).Particles);
            Assert.AreEqual(TokenScoreCascade.ParticlesOfEffectLevel[(int)level], library.Recipe(TokenEffectKind.ScoreCascade, level).Particles);
        }

        [Test]
        public void HigherLevelsOnlyAddSpritesAndParticles()
        {
            var library = Committed();

            foreach (TokenEffectKind kind in Enum.GetValues(typeof(TokenEffectKind)))
            {
                var low = library.Recipe(kind, EffectLevel.Low);
                var medium = library.Recipe(kind, EffectLevel.Medium);
                var high = library.Recipe(kind, EffectLevel.High);
                Assert.LessOrEqual(low.Layers, medium.Layers, kind.ToString());
                Assert.LessOrEqual(medium.Layers, high.Layers, kind.ToString());
                Assert.LessOrEqual(low.Particles, medium.Particles, kind.ToString());
                Assert.LessOrEqual(medium.Particles, high.Particles, kind.ToString());
            }
        }

        [Test]
        public void TintsResolveToColourTokensAndTheTeam()
        {
            var library = Committed();
            var tilePlace = library.Recipe(TokenEffectKind.TilePlace, EffectLevel.Low);
            var domino = library.Recipe(TokenEffectKind.Domino, EffectLevel.Low);

            Assert.AreEqual(TokenColor.GoldHi, library.Particles(tilePlace.FirstParticles).Tint.Color);
            Assert.IsFalse(library.Particles(tilePlace.FirstParticles).Tint.Team);
            Assert.IsTrue(library.Layer(domino.FirstLayer).Tint.Team);
            Assert.AreEqual(TokenColor.RoleTeamBFill, library.Layer(domino.FirstLayer).Tint.For(TokenColor.RoleTeamBFill));
        }

        [Test]
        public void GlowsComeFromTheGlowTokensAndTheScoreStep()
        {
            var library = Committed();
            var domino = library.Layer(library.Recipe(TokenEffectKind.Domino, EffectLevel.Low).FirstLayer);
            var cascade = library.Layer(library.Recipe(TokenEffectKind.ScoreCascade, EffectLevel.Low).FirstLayer);

            Assert.AreEqual(Array.IndexOf(TokenGlow.NameOfLevel, "intense"), domino.Glow);
            Assert.AreEqual(EffectLayer.StepGlow, cascade.Glow);
            for (var step = 1; step <= ScoreCascade.MaxStep; step++)
                Assert.AreEqual(TokenScoreCascade.StepGlowLevel[step - 1], EffectLibrary.GlowOf(cascade.Glow, step));
            Assert.AreEqual(Array.IndexOf(TokenGlow.NameOfLevel, "soft"), EffectLibrary.GlowOf(cascade.Glow, 1));
            Assert.AreEqual(Array.IndexOf(TokenGlow.NameOfLevel, "intense"), EffectLibrary.GlowOf(cascade.Glow, ScoreCascade.MaxStep));
        }

        [Test]
        public void EveryListedSpriteIsInTheEffectsFolderWithItsMaterial()
        {
            var library = Committed();

            for (var sprite = 0; sprite < library.SpriteCount; sprite++)
                Assert.IsNotNull(AssetDatabase.LoadAssetAtPath<Sprite>(EffectsFolder + "Sprites/" + library.SpriteName(sprite) + ".png"), library.SpriteName(sprite));
            Assert.AreEqual(EffectMaterial.Alpha, library.MaterialOf(IndexOf(library, "stamp")));
            Assert.AreEqual(EffectMaterial.Additive, library.MaterialOf(IndexOf(library, "glow")));
        }

        [Test]
        public void AValidMinimalLibraryParses()
        {
            var library = EffectLibrary.Parse(With(@"{ ""sprites"": [{ ""sprite"": ""glow"", ""tint"": ""color.gold.hi"", ""glow"": ""soft"" }], ""particles"": [{ ""sprite"": ""spark"", ""count"": 4, ""tint"": ""team"" }], ""overdrawLayers"": 2 }"));

            Assert.AreEqual(4, library.Recipe(TokenEffectKind.TilePlace, EffectLevel.Low).Particles);
            Assert.AreEqual(1, library.Recipe(TokenEffectKind.TilePlace, EffectLevel.Low).Layers);
        }

        [TestCase(@"{ ""particles"": [{ ""sprite"": ""spark"", ""count"": 5, ""tint"": ""color.gold.hi"" }], ""overdrawLayers"": 1 }", TestName = "MoreParticlesThanTheBudgetAreRefused")]
        [TestCase(@"{ ""particles"": [{ ""sprite"": ""spark"", ""count"": 4, ""tint"": ""color.gold.hi"" }], ""overdrawLayers"": 3 }", TestName = "MoreOverdrawLayersThanTheLevelAreRefused")]
        [TestCase(@"{ ""particles"": [{ ""sprite"": ""spark"", ""count"": 4, ""tint"": ""#FFFFFF"" }], ""overdrawLayers"": 1 }", TestName = "ATintThatIsNotAColourTokenIsRefused")]
        [TestCase(@"{ ""particles"": [{ ""sprite"": ""coin"", ""count"": 4, ""tint"": ""color.gold.hi"" }], ""overdrawLayers"": 1 }", TestName = "AnUnknownSpriteIsRefused")]
        [TestCase(@"{ ""sprites"": [{ ""sprite"": ""glow"", ""tint"": ""color.gold.hi"", ""glow"": ""blinding"" }], ""overdrawLayers"": 1 }", TestName = "AGlowThatIsNotAGlowTokenIsRefused")]
        [TestCase(@"{ ""particles"": [{ ""sprite"": ""spark"", ""count"": -1, ""tint"": ""color.gold.hi"" }], ""overdrawLayers"": 1 }", TestName = "ANegativeCountIsRefused")]
        public void ABrokenDefinitionIsRefused(string tilePlaceLow)
        {
            Assert.Throws<FormatException>(() => EffectLibrary.Parse(With(tilePlaceLow)));
        }

        [Test]
        public void AMissingEffectIsRefused()
        {
            var json = With(@"{ ""overdrawLayers"": 1 }").Replace(EmptyEffect("emoji"), EmptyEffect("fireworks"));

            Assert.Throws<FormatException>(() => EffectLibrary.Parse(json));
        }

        [Test]
        public void AnEffectWithoutABudgetIsRefused()
        {
            var json = With(@"{ ""overdrawLayers"": 1 }").Replace("\"effects\": [", "\"effects\": [" + EmptyEffect("fireworks") + ",");

            Assert.Throws<FormatException>(() => EffectLibrary.Parse(json));
        }

        static int MaxOverdrawLayers(EffectLevel level)
        {
            switch (level)
            {
                case EffectLevel.High: return TokenEffectLevel.HighMaxOverdrawLayers;
                case EffectLevel.Medium: return TokenEffectLevel.MediumMaxOverdrawLayers;
                default: return TokenEffectLevel.LowMaxOverdrawLayers;
            }
        }

        static int IndexOf(EffectLibrary library, string name)
        {
            for (var sprite = 0; sprite < library.SpriteCount; sprite++)
                if (library.SpriteName(sprite) == name) return sprite;
            return -1;
        }
    }
}
