using System;
using HypeDomino.UI.Board;
using HypeDomino.UI.Effects;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using UnityEditor;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class EffectSettingsLinkTests
    {
        sealed class RecordingTarget : IEffectSettingsTarget
        {
            public EffectLevel Level = EffectLevel.High;
            public bool ReduceMotion = true;
            public int Configured;

            public void Configure(EffectLevel level, bool reduceMotion)
            {
                Level = level;
                ReduceMotion = reduceMotion;
                Configured++;
            }
        }

        static GraphicsQuality MidDevice() => new GraphicsQuality(DeviceTier.Mid, null, 0, false);

        static MotionPreference FullMotion() => new MotionPreference(null, false);

        [Test]
        public void LinkingConfiguresEveryTargetWithTheCurrentSettings()
        {
            var first = new RecordingTarget();
            var second = new RecordingTarget();

            new EffectSettingsLink(MidDevice(), FullMotion(), first, second);

            foreach (var target in new[] { first, second })
            {
                Assert.AreEqual(EffectLevel.Medium, target.Level);
                Assert.IsFalse(target.ReduceMotion);
                Assert.AreEqual(1, target.Configured);
            }
        }

        [Test]
        public void ChoosingLowGraphicsConfiguresTheLowEffectLevel()
        {
            var target = new RecordingTarget();
            var graphics = MidDevice();
            new EffectSettingsLink(graphics, FullMotion(), target);

            graphics.Choose(GraphicsLevel.Low);

            Assert.AreEqual(EffectLevel.Low, target.Level);
            Assert.AreEqual(2, target.Configured);
        }

        [Test]
        public void UltraGraphicsUsesTheHighEffectLevel()
        {
            var target = new RecordingTarget();
            var graphics = new GraphicsQuality(DeviceTier.High, GraphicsLevel.Medium, 0, false);
            new EffectSettingsLink(graphics, FullMotion(), target);

            graphics.Choose(GraphicsLevel.Ultra);

            Assert.AreEqual(EffectLevel.High, target.Level);
        }

        [Test]
        public void SevereHeatConfiguresTheLowEffectLevel()
        {
            var target = new RecordingTarget();
            var graphics = MidDevice();
            new EffectSettingsLink(graphics, FullMotion(), target);

            graphics.UpdateHeat(ThermalStatus.Severe, 0f);

            Assert.AreEqual(EffectLevel.Low, target.Level);
        }

        [Test]
        public void ReduceMotionReachesEveryTargetAndKeepsTheLevel()
        {
            var target = new RecordingTarget();
            var motion = FullMotion();
            new EffectSettingsLink(MidDevice(), motion, target);

            motion.Choose(true);

            Assert.IsTrue(target.ReduceMotion);
            Assert.AreEqual(EffectLevel.Medium, target.Level);
        }

        [Test]
        public void ADisposedLinkNoLongerFollowsTheSettings()
        {
            var target = new RecordingTarget();
            var graphics = MidDevice();
            var motion = FullMotion();
            new EffectSettingsLink(graphics, motion, target).Dispose();

            graphics.Choose(GraphicsLevel.Low);
            motion.Choose(true);

            Assert.AreEqual(EffectLevel.Medium, target.Level);
            Assert.IsFalse(target.ReduceMotion);
            Assert.AreEqual(1, target.Configured);
        }

        [Test]
        public void AMissingTargetIsRefused()
        {
            Assert.Throws<ArgumentException>(() => new EffectSettingsLink(MidDevice(), FullMotion(), new RecordingTarget(), null));
        }

        [Test]
        public void LowGraphicsLandsLiveEffectsAndShrinksTheTilePlacePool()
        {
            var root = new GameObject("Screen", typeof(RectTransform), typeof(Canvas));
            try
            {
                root.GetComponent<Canvas>().renderMode = RenderMode.WorldSpace;
                var layer = new GameObject("Effects", typeof(RectTransform));
                layer.transform.SetParent(root.transform, false);
                var player = layer.AddComponent<EffectPlayer>();
                var library = EffectLibraryTests.Committed();
                player.Build(library, EffectPlayerTests.CommittedSprites(library), Material("EffectAdditive.mat"), Material("EffectAlpha.mat"), EffectLevel.Medium, false);
                var tiles = root.AddComponent<TilePlaceLayer>();
                tiles.Build(Array.Empty<Transform>());
                var graphics = MidDevice();
                new EffectSettingsLink(graphics, FullMotion(), player, tiles);
                player.Play(EffectRequest.On(TokenEffectKind.Domino, Vector3.zero, new Vector2(100f, 100f), 0f));
                Assert.AreEqual(1, player.PoolOf(TokenEffectKind.Domino).LiveCount);

                graphics.Choose(GraphicsLevel.Low);

                Assert.AreEqual(EffectLevel.Low, player.Level);
                Assert.AreEqual(0, player.PoolOf(TokenEffectKind.Domino).LiveCount);
                Assert.AreEqual(0, player.LiveQuads);
                Assert.AreEqual(TokenEffectLevel.LowMaxTweens, tiles.Capacity);
            }
            finally
            {
                Object.DestroyImmediate(root);
            }
        }

        static Material Material(string file) => AssetDatabase.LoadAssetAtPath<Material>(EffectLibraryTests.EffectsFolder + file);
    }
}
