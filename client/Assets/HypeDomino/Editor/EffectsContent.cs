using System;
using System.IO;
using System.Linq;
using UnityEditor;
using UnityEditor.U2D;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Editor
{
    public static class EffectsContent
    {
        public const string Root = "Assets/HypeDomino/UI/Effects";
        public const string SpritesFolder = Root + "/Sprites";
        public const string TablePath = Root + "/effects.json";
        public const string AtlasPath = Root + "/Effects.spriteatlasv2";
        public const string AdditiveMaterialPath = Root + "/EffectAdditive.mat";
        public const string AlphaMaterialPath = Root + "/EffectAlpha.mat";
        public const string AdditiveShaderName = "HypeDomino/UI/Effect Additive";
        public const string AlphaShaderName = "UI/Default";
        public const string AdditiveMaterial = "additive";
        public const string AlphaMaterial = "alpha";
        public const float PixelsPerUnit = 100f;
        public const int MaxAtlasSize = 1024;
        public const int AtlasPadding = 4;

        static readonly string[] TexturePlatforms = { "Android", "iPhone" };

        [Serializable]
        public sealed class EffectSprite
        {
            public string name;
            public string material;
        }

        [Serializable]
        public sealed class EffectTable
        {
            public EffectSprite[] sprites;
        }

        [MenuItem("HypeDomino/Import Effects")]
        public static void ImportAll()
        {
            foreach (var path in SpritePaths()) ImportSprite(path);
            BuildAtlas();
            SaveMaterial(AdditiveMaterialPath, AdditiveShaderName);
            SaveMaterial(AlphaMaterialPath, AlphaShaderName);
            AssetDatabase.SaveAssets();
            Debug.Log("HypeDomino effects imported");
        }

        public static void ImportAllAndExit() => EditorMethod.Run(ImportAll);

        public static EffectTable ReadTable() => JsonUtility.FromJson<EffectTable>(File.ReadAllText(TablePath));

        public static string SpritePath(string name) => $"{SpritesFolder}/{name}.png";

        public static string[] SpritePaths() =>
            Directory.GetFiles(SpritesFolder, "*.png").Select(path => path.Replace('\\', '/')).OrderBy(path => path, StringComparer.Ordinal).ToArray();

        static void ImportSprite(string path)
        {
            var importer = (TextureImporter)AssetImporter.GetAtPath(path);
            var texture = new TextureImporterSettings();
            importer.ReadTextureSettings(texture);
            texture.textureType = TextureImporterType.Sprite;
            texture.spriteMode = (int)SpriteImportMode.Single;
            texture.spriteMeshType = SpriteMeshType.Tight;
            texture.spritePixelsPerUnit = PixelsPerUnit;
            texture.spriteBorder = Vector4.zero;
            texture.mipmapEnabled = false;
            texture.alphaIsTransparency = true;
            texture.sRGBTexture = true;
            texture.wrapMode = TextureWrapMode.Clamp;
            importer.SetTextureSettings(texture);
            foreach (var platform in TexturePlatforms)
            {
                var settings = importer.GetPlatformTextureSettings(platform);
                settings.overridden = true;
                settings.format = ProjectSetup.SpriteFormat;
                importer.SetPlatformTextureSettings(settings);
            }
            importer.SaveAndReimport();
        }

        static void BuildAtlas()
        {
            if (!File.Exists(AtlasPath))
            {
                var atlas = new SpriteAtlasAsset();
                atlas.Add(new Object[] { AssetDatabase.LoadAssetAtPath<DefaultAsset>(SpritesFolder) });
                SpriteAtlasAsset.Save(atlas, AtlasPath);
                AssetDatabase.ImportAsset(AtlasPath);
            }
            var importer = (SpriteAtlasImporter)AssetImporter.GetAtPath(AtlasPath);
            var packing = importer.packingSettings;
            packing.enableRotation = false;
            packing.enableTightPacking = false;
            packing.padding = AtlasPadding;
            importer.packingSettings = packing;
            var texture = importer.textureSettings;
            texture.generateMipMaps = false;
            texture.sRGB = true;
            importer.textureSettings = texture;
            foreach (var platform in TexturePlatforms)
            {
                var settings = importer.GetPlatformSettings(platform);
                settings.overridden = true;
                settings.format = ProjectSetup.SpriteFormat;
                settings.maxTextureSize = MaxAtlasSize;
                importer.SetPlatformSettings(settings);
            }
            importer.SaveAndReimport();
        }

        static void SaveMaterial(string path, string shaderName)
        {
            var shader = Shader.Find(shaderName);
            if (shader == null) throw new InvalidOperationException($"shader {shaderName} not found");
            var material = AssetDatabase.LoadAssetAtPath<Material>(path);
            if (material == null)
            {
                AssetDatabase.CreateAsset(new Material(shader), path);
                return;
            }
            material.shader = shader;
            EditorUtility.SetDirty(material);
        }
    }
}
