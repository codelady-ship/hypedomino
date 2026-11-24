using System;
using HypeDomino.UI.Effects;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using UnityEditor;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class EffectPlayerTests
    {
        static readonly Vector3 Hand = new Vector3(0f, -400f, 0f);
        static readonly Vector3 Meter = new Vector3(300f, 500f, 0f);

        GameObject root;
        EffectPlayer player;
        EffectLibrary library;

        [SetUp]
        public void BuildPlayer()
        {
            root = new GameObject("Screen", typeof(RectTransform), typeof(Canvas));
            root.GetComponent<Canvas>().renderMode = RenderMode.WorldSpace;
            var layer = new GameObject("Effects", typeof(RectTransform));
            layer.transform.SetParent(root.transform, false);
            player = layer.AddComponent<EffectPlayer>();
            library = EffectLibraryTests.Committed();
            player.Build(library, CommittedSprites(library), Material(EffectLibraryTests.EffectsFolder + "EffectAdditive.mat"), Material(EffectLibraryTests.EffectsFolder + "EffectAlpha.mat"), EffectLevel.Medium, false);
        }

        [TearDown]
        public void DestroyPlayer() => Object.DestroyImmediate(root);

        public static Sprite[] CommittedSprites(EffectLibrary effects)
        {
            var sprites = new Sprite[effects.SpriteCount];
            for (var sprite = 0; sprite < sprites.Length; sprite++)
                sprites[sprite] = AssetDatabase.LoadAssetAtPath<Sprite>(EffectLibraryTests.EffectsFolder + "Sprites/" + effects.SpriteName(sprite) + ".png");
            return sprites;
        }

        Vector2 Local(Vector3 world) => player.QuadsOf(EffectMaterial.Additive).rectTransform.InverseTransformPoint(world);

        static Material Material(string path) => AssetDatabase.LoadAssetAtPath<Material>(path);

        static EffectRequest On(TokenEffectKind kind, float startMs = 0f) => EffectRequest.On(kind, Vector3.zero, new Vector2(100f, 100f), startMs).ForTeam(TokenColor.RoleTeamAFill);

        int CountOf(string spriteName)
        {
            var count = 0;
            foreach (EffectMaterial material in Enum.GetValues(typeof(EffectMaterial)))
            {
                var quads = player.QuadsOf(material);
                for (var at = 0; at < quads.Count; at++)
                    if (library.SpriteName(quads[at].Sprite) == spriteName) count++;
            }
            return count;
        }

        ref EffectQuad FirstOf(string spriteName)
        {
            foreach (EffectMaterial material in Enum.GetValues(typeof(EffectMaterial)))
            {
                var quads = player.QuadsOf(material);
                for (var at = 0; at < quads.Count; at++)
                    if (library.SpriteName(quads[at].Sprite) == spriteName) return ref quads[at];
            }
            throw new AssertionException($"no {spriteName} quad");
        }

        [TestCase(EffectLevel.Low)]
        [TestCase(EffectLevel.Medium)]
        [TestCase(EffectLevel.High)]
        public void EveryEffectSpawnsItsLevelsSpritesAndParticles(EffectLevel level)
        {
            player.Configure(level, false);

            foreach (TokenEffectKind kind in Enum.GetValues(typeof(TokenEffectKind)))
            {
                var recipe = library.Recipe(kind, level);
                Assert.IsTrue(player.Play(On(kind).AtStep(ScoreCascade.MaxStep)), kind.ToString());
                Assert.AreEqual(recipe.Particles, player.ParticleQuads, $"{kind} {level}");
                Assert.AreEqual(recipe.Particles + recipe.Layers, player.LiveQuads, $"{kind} {level}");
                Assert.AreEqual(player.Budget.Live, player.ParticleQuads);
                player.EndAll();
            }
        }

        [TestCase(EffectLevel.Low, 3, 3, 3, 8)]
        [TestCase(EffectLevel.Medium, 3, 8, 8, 25)]
        [TestCase(EffectLevel.High, 5, 29, 29, 86)]
        public void AScoreStepSplitsItsBurstLikeTheVfxTable(EffectLevel level, int step, int stars, int streaks, int sparks)
        {
            player.Configure(level, false);

            player.Play(On(TokenEffectKind.ScoreCascade).AtStep(step));

            Assert.AreEqual(stars, CountOf("star"));
            Assert.AreEqual(streaks, CountOf("streak"));
            Assert.AreEqual(sparks, CountOf("spark"));
            Assert.AreEqual(ScoreCascade.Burst(step, level, false).Particles, player.ParticleQuads);
        }

        [Test]
        public void AScoreWithoutAStepPlaysNothing()
        {
            Assert.IsFalse(player.Play(On(TokenEffectKind.ScoreCascade)));
            Assert.AreEqual(0, player.LiveQuads);
        }

        [Test]
        public void TheLevelsLiveParticleCapHoldsAcrossEffects()
        {
            player.Configure(EffectLevel.Low, false);

            player.Play(On(TokenEffectKind.MatchWin));
            player.Play(On(TokenEffectKind.RankUp));
            player.Play(On(TokenEffectKind.ScoreCascade).AtStep(ScoreCascade.MaxStep));
            player.Play(On(TokenEffectKind.Domino));
            player.Play(On(TokenEffectKind.TilePlace));

            Assert.AreEqual(TokenEffectLevel.LowMaxParticles, player.Budget.Live);
            Assert.AreEqual(TokenEffectLevel.LowMaxParticles, player.ParticleQuads);
            Assert.AreEqual(10, player.PoolOf(TokenEffectKind.Domino).ParticlesOf(0));
            Assert.AreEqual(2, CountOf("glow"), "the score and domino glows still tell their events when particles are cut");
        }

        [Test]
        public void AFullPoolReplacesItsOldestInstance()
        {
            player.Configure(EffectLevel.Low, false);

            for (var placement = 0; placement <= TokenEffect.TilePlacePooled; placement++) player.Play(On(TokenEffectKind.TilePlace, placement));

            Assert.AreEqual(TokenEffect.TilePlacePooled, player.PoolOf(TokenEffectKind.TilePlace).LiveCount);
            Assert.AreEqual(TokenEffect.TilePlacePooled * library.Recipe(TokenEffectKind.TilePlace, EffectLevel.Low).Particles, player.ParticleQuads);
            Assert.AreEqual(player.Budget.Live, player.ParticleQuads);
        }

        [Test]
        public void AnEffectEndsAtItsMaxMilliseconds()
        {
            player.Play(On(TokenEffectKind.Block));
            player.Tick(TokenEffect.BlockDurationMs - 1f);
            Assert.Greater(player.LiveQuads, 0);

            player.Tick(TokenEffect.BlockDurationMs);

            Assert.AreEqual(0, player.LiveQuads);
            Assert.AreEqual(0, player.Budget.Live);
        }

        [Test]
        public void ReduceMotionUsesTheLowCountsButKeepsTheLevelsSprites()
        {
            player.Configure(EffectLevel.High, true);

            player.Play(On(TokenEffectKind.RankUp));

            Assert.AreEqual(library.Recipe(TokenEffectKind.RankUp, EffectLevel.Low).Particles, player.ParticleQuads);
            Assert.AreEqual(1, CountOf("rays"));
            Assert.AreEqual(1, CountOf("glow"));
        }

        [Test]
        public void ReduceMotionFadesASpriteInWithoutTheScalePop()
        {
            player.Configure(EffectLevel.Medium, true);
            player.Play(On(TokenEffectKind.Block));

            player.Tick(TokenDurationMs.Fast * 0.5f);

            ref var stamp = ref FirstOf("stamp");
            Assert.AreEqual(1f, stamp.Scale, 1e-4f);
            Assert.Less(stamp.Alpha, byte.MaxValue);
            Assert.Greater(stamp.Alpha, 0);
        }

        [Test]
        public void WithoutReduceMotionASpritePopsInFromNothing()
        {
            player.Play(On(TokenEffectKind.Block));

            player.Tick(0f);

            ref var stamp = ref FirstOf("stamp");
            Assert.AreEqual(0f, stamp.Scale, 1e-4f);
            Assert.AreEqual(byte.MaxValue, stamp.Alpha);
        }

        [TestCase(1, "soft")]
        [TestCase(6, "intense")]
        public void TheScoreGlowTakesTheStepsGlowToken(int step, string glow)
        {
            player.Play(On(TokenEffectKind.ScoreCascade).AtStep(step));
            player.Tick(TokenEffect.ScoreCascadeDurationMs * 0.5f);

            ref var quad = ref FirstOf("glow");
            var level = Array.IndexOf(TokenGlow.NameOfLevel, glow);
            Assert.AreEqual(TokenGlow.ScaleOfLevel[level], quad.Scale, 1e-4f);
            Assert.AreEqual(Mathf.RoundToInt(byte.MaxValue * TokenGlow.AlphaOfLevel[level]), quad.Alpha);
            Assert.AreEqual(TokenColor.RoleScoreFill, quad.Tint);
        }

        [Test]
        public void TeamEffectsTakeTheTeamsFill()
        {
            player.Play(EffectRequest.On(TokenEffectKind.Domino, Vector3.zero, Vector2.zero, 0f).ForTeam(TokenColor.RoleTeamBFill));

            Assert.AreEqual(TokenColor.RoleTeamBFill, FirstOf("glow").Tint);
            Assert.AreEqual(TokenColor.RoleTeamBFill, FirstOf("spark").Tint);
            Assert.AreEqual(TokenColor.GoldDefault, FirstOf("star").Tint);
        }

        [Test]
        public void AnArrivingTrailTravelsFirstAndTheBurstFollowsItsEnd()
        {
            player.Configure(EffectLevel.High, false);
            var request = EffectRequest.On(TokenEffectKind.TilePlace, Vector3.zero, Vector2.zero, 0f).TrailFrom(Hand, TokenDurationMs.Base);
            player.Play(request);

            player.Tick(0f);
            Assert.AreEqual(Local(Hand), FirstOf("streak").Center);
            Assert.AreEqual(0, FirstOf("dust").Alpha);

            player.Tick(TokenDurationMs.Base + 1f);
            Assert.Greater(FirstOf("dust").Alpha, 0);
        }

        [Test]
        public void ALeavingTrailFliesToItsTarget()
        {
            player.Play(EffectRequest.On(TokenEffectKind.ScoreCascade, Vector3.zero, Vector2.zero, 0f).AtStep(1).TrailTo(Meter));

            player.Tick(TokenDurationMs.Base - 1f);

            Assert.AreEqual(Local(Meter).x, FirstOf("streak").Center.x, 1f);
            Assert.AreEqual(Local(Meter).y, FirstOf("streak").Center.y, 1f);
        }

        [Test]
        public void ChangingTheLevelEndsWhatIsPlaying()
        {
            player.Play(On(TokenEffectKind.MatchWin));

            player.Configure(EffectLevel.Low, false);

            Assert.AreEqual(0, player.LiveQuads);
            Assert.AreEqual(0, player.Budget.Live);
            Assert.AreEqual(TokenEffectLevel.LowMaxParticles, player.Budget.Limit);
        }

        [Test]
        public void TheQuadsUseOnlyTheTwoSharedMaterials()
        {
            Assert.AreSame(Material(EffectLibraryTests.EffectsFolder + "EffectAdditive.mat"), player.QuadsOf(EffectMaterial.Additive).material);
            Assert.AreSame(Material(EffectLibraryTests.EffectsFolder + "EffectAlpha.mat"), player.QuadsOf(EffectMaterial.Alpha).material);
        }

        [Test]
        public void AMissingSpriteStopsTheBuild()
        {
            var sprites = CommittedSprites(library);
            sprites[0] = null;

            Assert.Throws<ArgumentException>(() => player.Build(library, sprites, null, null, EffectLevel.Low, false));
        }
    }
}
