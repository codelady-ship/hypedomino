using System.IO;
using HypeDomino.Core.Diagnostics;
using UnityEditor;
using UnityEditor.Build;
using UnityEditor.Presets;
using UnityEditor.SceneManagement;
using UnityEngine;
using UnityEngine.Rendering;
using UnityEngine.Rendering.Universal;

namespace HypeDomino.Editor
{
    public static class ProjectSetup
    {
        public const TextureImporterFormat SpriteFormat = TextureImporterFormat.ASTC_6x6;
        const string SettingsFolder = "Assets/HypeDomino/Settings";
        const string ScenesFolder = "Assets/HypeDomino/Scenes";
        const string RendererPath = SettingsFolder + "/UniversalRenderer.asset";
        const string PipelinePath = SettingsFolder + "/UniversalRenderPipeline.asset";
        const string SpritePresetPath = SettingsFolder + "/SpriteTextureASTC.preset";
        const string PresetSourcePath = SettingsFolder + "/PresetSource.png";
        const string BootScenePath = ScenesFolder + "/Boot.unity";

        static readonly NamedBuildTarget[] MobileTargets = { NamedBuildTarget.Android, NamedBuildTarget.iOS };
        static readonly string[] TexturePlatforms = { "Android", "iPhone" };

        [MenuItem("HypeDomino/Apply Project Setup")]
        public static void Apply()
        {
            Directory.CreateDirectory(SettingsFolder);
            Directory.CreateDirectory(ScenesFolder);
            ApplyPlayerSettings();
            ApplyRenderPipeline();
            ApplySpriteTexturePreset();
            EditorSettings.spritePackerMode = SpritePackerMode.SpriteAtlasV2;
            CreateBootScene();
            ContentSetup.Apply();
            AssetDatabase.SaveAssets();
            Debug.Log("HypeDomino project setup applied");
        }

        static void ApplyPlayerSettings()
        {
            foreach (var target in MobileTargets)
            {
                PlayerSettings.SetScriptingBackend(target, ScriptingImplementation.IL2CPP);
                PlayerSettings.SetManagedStrippingLevel(target, ManagedStrippingLevel.High);
                PlayerSettings.SetApiCompatibilityLevel(target, ApiCompatibilityLevel.NET_Standard);
            }

            PlayerSettings.stripEngineCode = true;
            PlayerSettings.enableFrameTimingStats = true;
            PlayerSettings.colorSpace = ColorSpace.Linear;
            PlayerSettings.Android.targetArchitectures = AndroidArchitecture.ARM64;
            PlayerSettings.Android.textureCompressionFormats = new[] { TextureCompressionFormat.ASTC };
            EditorUserBuildSettings.androidBuildSubtarget = MobileTextureSubtarget.ASTC;
            PlayerSettings.SetUseDefaultGraphicsAPIs(BuildTarget.Android, false);
            PlayerSettings.SetGraphicsAPIs(BuildTarget.Android, new[] { GraphicsDeviceType.Vulkan, GraphicsDeviceType.OpenGLES3 });
            PlayerSettings.SetUseDefaultGraphicsAPIs(BuildTarget.iOS, false);
            PlayerSettings.SetGraphicsAPIs(BuildTarget.iOS, new[] { GraphicsDeviceType.Metal });
        }

        static void ApplyRenderPipeline()
        {
            var pipeline = AssetDatabase.LoadAssetAtPath<UniversalRenderPipelineAsset>(PipelinePath);
            if (pipeline == null) pipeline = CreateRenderPipeline();
            GraphicsSettings.defaultRenderPipeline = pipeline;
            for (var level = 0; level < QualitySettings.names.Length; level++)
            {
                QualitySettings.SetQualityLevel(level, false);
                QualitySettings.renderPipeline = null;
                QualitySettings.vSyncCount = 0;
            }
        }

        static UniversalRenderPipelineAsset CreateRenderPipeline()
        {
            var renderer = ScriptableObject.CreateInstance<UniversalRendererData>();
            renderer.renderingMode = RenderingMode.Forward;
            renderer.postProcessData = null;
            AssetDatabase.CreateAsset(renderer, RendererPath);

            var pipeline = UniversalRenderPipelineAsset.Create(renderer);
            pipeline.supportsHDR = false;
            pipeline.msaaSampleCount = 1;
            pipeline.supportsCameraDepthTexture = false;
            pipeline.supportsCameraOpaqueTexture = false;
            AssetDatabase.CreateAsset(pipeline, PipelinePath);

            var serialized = new SerializedObject(pipeline);
            serialized.FindProperty("m_MainLightShadowsSupported").boolValue = false;
            serialized.FindProperty("m_AdditionalLightShadowsSupported").boolValue = false;
            serialized.FindProperty("m_AnyShadowsSupported").boolValue = false;
            serialized.FindProperty("m_AdditionalLightsRenderingMode").intValue = (int)LightRenderingMode.Disabled;
            serialized.ApplyModifiedPropertiesWithoutUndo();
            return pipeline;
        }

        static void ApplySpriteTexturePreset()
        {
            if (AssetDatabase.LoadAssetAtPath<Preset>(SpritePresetPath) != null) return;

            var source = new Texture2D(4, 4);
            File.WriteAllBytes(PresetSourcePath, source.EncodeToPNG());
            Object.DestroyImmediate(source);
            AssetDatabase.ImportAsset(PresetSourcePath);

            var importer = (TextureImporter)AssetImporter.GetAtPath(PresetSourcePath);
            importer.textureType = TextureImporterType.Sprite;
            importer.spriteImportMode = SpriteImportMode.Single;
            importer.mipmapEnabled = false;
            importer.alphaIsTransparency = true;
            foreach (var platform in TexturePlatforms)
            {
                var settings = importer.GetPlatformTextureSettings(platform);
                settings.overridden = true;
                settings.format = SpriteFormat;
                importer.SetPlatformTextureSettings(settings);
            }

            var preset = new Preset(importer);
            AssetDatabase.CreateAsset(preset, SpritePresetPath);
            Preset.SetDefaultPresetsForType(new PresetType(importer), new[] { new DefaultPreset(string.Empty, preset) });
            AssetDatabase.DeleteAsset(PresetSourcePath);
        }

        static void CreateBootScene()
        {
            if (File.Exists(BootScenePath)) return;

            var scene = EditorSceneManager.NewScene(NewSceneSetup.EmptyScene, NewSceneMode.Single);
            var cameraObject = new GameObject("Main Camera", typeof(Camera), typeof(UniversalAdditionalCameraData));
            cameraObject.tag = "MainCamera";
            var camera = cameraObject.GetComponent<Camera>();
            camera.orthographic = true;
            camera.clearFlags = CameraClearFlags.SolidColor;
            camera.backgroundColor = Color.black;
            new GameObject(nameof(FrameTimingSampler), typeof(FrameTimingSampler));

            EditorSceneManager.SaveScene(scene, BootScenePath);
            EditorBuildSettings.scenes = new[] { new EditorBuildSettingsScene(BootScenePath, true) };
        }
    }
}
