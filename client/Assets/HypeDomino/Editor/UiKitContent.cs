using System;
using System.IO;
using System.Linq;
using UnityEditor;
using UnityEditor.U2D;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Editor
{
    public static class UiKitContent
    {
        public const string Root = "Assets/HypeDomino/UI/Kit";
        public const string PanelsFolder = Root + "/Panels";
        public const string RarityFolder = Root + "/Rarity";
        public const string BordersPath = PanelsFolder + "/kit.json";
        public const string AtlasPath = Root + "/UiKit.spriteatlasv2";
        public const float DefaultPixelsPerUnit = 100f;
        public const int MaxAtlasSize = 2048;
        public const int AtlasPadding = 4;

        static readonly string[] TexturePlatforms = { "Android", "iPhone" };

        [Serializable]
        public sealed class KitSprite
        {
            public string name;
            public int width;
            public int height;
            public int[] border;
            public float pixelsPerUnit;
        }

        [Serializable]
        public sealed class KitSprites
        {
            public KitSprite[] sprites;
        }

        [MenuItem("HypeDomino/Import UI Kit")]
        public static void ImportAll()
        {
            var kit = ReadKit();
            foreach (var path in SpritePaths()) ImportSprite(path, KitEntry(kit, path));
            BuildAtlas();
            AssetDatabase.SaveAssets();
            Debug.Log("HypeDomino UI kit imported");
        }

        public static void ImportAllAndExit() => EditorMethod.Run(ImportAll);

        public static KitSprites ReadKit() => JsonUtility.FromJson<KitSprites>(File.ReadAllText(BordersPath));

        public static KitSprite KitEntry(KitSprites kit, string path) =>
            path.StartsWith(PanelsFolder + "/", StringComparison.Ordinal)
                ? kit.sprites.FirstOrDefault(sprite => sprite.name == Path.GetFileNameWithoutExtension(path))
                : null;

        public static string[] SpritePaths() =>
            Directory.GetFiles(Root, "*.png", SearchOption.AllDirectories).Select(path => path.Replace('\\', '/')).OrderBy(path => path, StringComparer.Ordinal).ToArray();

        static void ImportSprite(string path, KitSprite kit)
        {
            var importer = (TextureImporter)AssetImporter.GetAtPath(path);
            var texture = new TextureImporterSettings();
            importer.ReadTextureSettings(texture);
            texture.textureType = TextureImporterType.Sprite;
            texture.spriteMode = (int)SpriteImportMode.Single;
            texture.spriteMeshType = SpriteMeshType.FullRect;
            texture.spritePixelsPerUnit = kit?.pixelsPerUnit ?? DefaultPixelsPerUnit;
            texture.spriteBorder = kit == null ? Vector4.zero : new Vector4(kit.border[0], kit.border[1], kit.border[2], kit.border[3]);
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
                atlas.Add(new Object[] { AssetDatabase.LoadAssetAtPath<DefaultAsset>(Root) });
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
    }
}
