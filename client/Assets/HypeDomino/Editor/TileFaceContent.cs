using System;
using System.IO;
using HypeDomino.UI.Board;
using UnityEditor;
using UnityEditor.AddressableAssets;
using UnityEditor.U2D;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Editor
{
    public static class TileFaceContent
    {
        public const string DefaultSet = "ivory";
        public const string Root = "Assets/HypeDomino/UI/Board/Tiles";
        public const float PixelsPerUnit = 100f;
        public const int FaceWidth = 128;
        public const int FaceHeight = 256;
        public const int MaxAtlasSize = 2048;
        public const int AtlasPadding = 4;

        static readonly string[] TexturePlatforms = { "Android", "iPhone" };

        [MenuItem("HypeDomino/Import Tile Faces")]
        public static void ImportAll()
        {
            ImportSet(DefaultSet);
            AddToBuiltinGroup(DefaultSet);
            AssetDatabase.SaveAssets();
            Debug.Log("HypeDomino tile faces imported");
        }

        public static void ImportAllAndExit() => EditorMethod.Run(ImportAll);

        public static string SetFolder(string slug) => $"{Root}/{slug}";

        public static string FacesFolder(string slug) => SetFolder(slug) + "/face";

        public static string AtlasPath(string slug) => SetFolder(slug) + "/faces.spriteatlasv2";

        public static string SetPath(string slug) => SetFolder(slug) + "/faces.asset";

        public static string Address(string slug) => $"tile/{slug}/faces";

        public static string[] FacePaths(string slug)
        {
            var paths = new string[TileFaces.SetSize];
            var index = 0;
            for (var low = 0; low <= TileFaces.HighestValue; low++)
            for (var high = low; high <= TileFaces.HighestValue; high++)
                paths[index++] = $"{FacesFolder(slug)}/{low}_{high}.png";
            return paths;
        }

        static void ImportSet(string slug)
        {
            var paths = FacePaths(slug);
            var faces = new Sprite[paths.Length];
            for (var i = 0; i < paths.Length; i++)
            {
                if (!File.Exists(paths[i])) throw new FileNotFoundException("tile face missing", paths[i]);
                ImportSprite(paths[i]);
                faces[i] = AssetDatabase.LoadAssetAtPath<Sprite>(paths[i]);
            }
            BuildAtlas(slug);
            SaveSet(slug, faces);
        }

        static void ImportSprite(string path)
        {
            var importer = (TextureImporter)AssetImporter.GetAtPath(path);
            var texture = new TextureImporterSettings();
            importer.ReadTextureSettings(texture);
            texture.textureType = TextureImporterType.Sprite;
            texture.spriteMode = (int)SpriteImportMode.Single;
            texture.spriteMeshType = SpriteMeshType.FullRect;
            texture.spritePixelsPerUnit = PixelsPerUnit;
            texture.spriteAlignment = (int)SpriteAlignment.Center;
            texture.spriteBorder = Vector4.zero;
            texture.mipmapEnabled = false;
            texture.alphaIsTransparency = true;
            texture.sRGBTexture = true;
            texture.wrapMode = TextureWrapMode.Clamp;
            texture.filterMode = FilterMode.Bilinear;
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

        static void BuildAtlas(string slug)
        {
            var path = AtlasPath(slug);
            if (!File.Exists(path))
            {
                var atlas = new SpriteAtlasAsset();
                atlas.Add(new Object[] { AssetDatabase.LoadAssetAtPath<DefaultAsset>(FacesFolder(slug)) });
                SpriteAtlasAsset.Save(atlas, path);
                AssetDatabase.ImportAsset(path);
            }
            var importer = (SpriteAtlasImporter)AssetImporter.GetAtPath(path);
            var packing = importer.packingSettings;
            packing.enableRotation = false;
            packing.enableTightPacking = false;
            packing.padding = AtlasPadding;
            importer.packingSettings = packing;
            var texture = importer.textureSettings;
            texture.generateMipMaps = false;
            texture.sRGB = true;
            texture.filterMode = FilterMode.Bilinear;
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

        static void SaveSet(string slug, Sprite[] faces)
        {
            var path = SetPath(slug);
            var set = AssetDatabase.LoadAssetAtPath<TileFaceSet>(path);
            if (set == null)
            {
                set = ScriptableObject.CreateInstance<TileFaceSet>();
                set.SetFaces(faces);
                AssetDatabase.CreateAsset(set, path);
                return;
            }
            set.SetFaces(faces);
            EditorUtility.SetDirty(set);
        }

        static void AddToBuiltinGroup(string slug)
        {
            var settings = AddressableAssetSettingsDefaultObject.Settings ?? throw new InvalidOperationException("Addressables settings are missing; run HypeDomino/Apply Content Setup");
            var group = settings.FindGroup(ContentSetup.BuiltinGroupName) ?? throw new InvalidOperationException($"{ContentSetup.BuiltinGroupName} is missing; run HypeDomino/Apply Content Setup");
            var entry = settings.CreateOrMoveEntry(AssetDatabase.AssetPathToGUID(SetPath(slug)), group);
            entry.address = Address(slug);
            EditorUtility.SetDirty(group);
            EditorUtility.SetDirty(settings);
        }
    }
}
