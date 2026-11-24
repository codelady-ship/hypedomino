using System.IO;
using System.Linq;
using System.Text.RegularExpressions;
using NUnit.Framework;
using UnityEditor;
using UnityEditor.Build;
using UnityEngine;
using UnityEngine.Rendering;
using UnityEngine.Rendering.Universal;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ProjectSettingsTests
    {
        static readonly NamedBuildTarget[] MobileTargets = { NamedBuildTarget.Android, NamedBuildTarget.iOS };

        [Test]
        public void MobileTargetsUseIl2CppWithHighStripping()
        {
            foreach (var target in MobileTargets)
            {
                Assert.AreEqual(ScriptingImplementation.IL2CPP, PlayerSettings.GetScriptingBackend(target), target.TargetName);
                Assert.AreEqual(ManagedStrippingLevel.High, PlayerSettings.GetManagedStrippingLevel(target), target.TargetName);
            }
            Assert.IsTrue(PlayerSettings.stripEngineCode);
        }

        [Test]
        public void AndroidBuildsArm64Only()
        {
            Assert.AreEqual(AndroidArchitecture.ARM64, PlayerSettings.Android.targetArchitectures);
        }

        [Test]
        public void AndroidTexturesCompressToAstcOnly()
        {
            CollectionAssert.AreEqual(new[] { TextureCompressionFormat.ASTC }, PlayerSettings.Android.textureCompressionFormats);
        }

        [Test]
        public void FrameTimingStatsAreEnabled()
        {
            Assert.IsTrue(PlayerSettings.enableFrameTimingStats);
        }

        [Test]
        public void UniversalRendererUsesForwardWithoutShadowsOrHdr()
        {
            var pipeline = GraphicsSettings.defaultRenderPipeline as UniversalRenderPipelineAsset;
            Assert.IsNotNull(pipeline);
            Assert.IsFalse(pipeline.supportsHDR);
            Assert.IsFalse(pipeline.supportsMainLightShadows);
            Assert.IsFalse(pipeline.supportsAdditionalLightShadows);

            Assert.AreEqual(1, pipeline.rendererDataList.Length);
            var renderer = pipeline.rendererDataList[0] as UniversalRendererData;
            Assert.IsNotNull(renderer);
            Assert.AreEqual(RenderingMode.Forward, renderer.renderingMode);
            Assert.IsNull(renderer.postProcessData);
        }

        [Test]
        public void QualityLevelsUseTheDefaultPipeline()
        {
            var current = QualitySettings.GetQualityLevel();
            for (var level = 0; level < QualitySettings.names.Length; level++)
                Assert.IsNull(QualitySettings.GetRenderPipelineAssetAt(level), QualitySettings.names[level]);
            Assert.AreEqual(current, QualitySettings.GetQualityLevel());
        }

        [Test]
        public void NoQualityLevelWaitsForVsyncSoTheTargetFrameRateDecides()
        {
            var settings = File.ReadAllText("ProjectSettings/QualitySettings.asset");
            var counts = Regex.Matches(settings, @"vSyncCount: (\d+)");
            Assert.AreEqual(QualitySettings.names.Length, counts.Count);
            foreach (Match count in counts) Assert.AreEqual("0", count.Groups[1].Value);
        }

        [Test]
        public void BootSceneIsTheFirstBuildScene()
        {
            Assert.AreEqual("Assets/HypeDomino/Scenes/Boot.unity", EditorBuildSettings.scenes.First().path);
        }

        [Test]
        public void SpriteAtlasV2IsEnabled()
        {
            Assert.AreEqual(SpritePackerMode.SpriteAtlasV2, EditorSettings.spritePackerMode);
        }
    }
}
