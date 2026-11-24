using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using HypeDomino.Content;
using UnityEditor;
using UnityEditor.AddressableAssets;
using UnityEditor.AddressableAssets.Settings;
using UnityEditor.AddressableAssets.Settings.GroupSchemas;
using UnityEditor.U2D;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Editor
{
    public static class CosmeticContent
    {
        public const string Root = "Assets/Cosmetics";
        public const string BuiltinPlacement = "builtin";
        public const string RemotePlacement = "remote";
        public const string StaticFile = "static.png";
        public const string AnimSlot = "anim";
        public const string LayerSlot = "layer";
        public const string LoopExtension = ".asset";
        public const string AtlasExtension = ".spriteatlasv2";
        public const string CosmeticLabel = "cosmetic";
        public const string StaticLabel = "cosmetic-static";
        public const string SliceFile = "slice.json";
        public const int LoopAtlasPadding = 0;

        [Serializable]
        public sealed class TableSlice
        {
            public int depth;
            public int step;
        }

        static readonly string[] TexturePlatforms = { "Android", "iPhone" };

        [MenuItem("HypeDomino/Import Cosmetics")]
        public static void ImportAll()
        {
            foreach (var item in ItemFolders(Root).ToList()) Import(item);
            AssetDatabase.SaveAssets();
            Debug.Log("HypeDomino cosmetics imported");
        }

        public static void ImportAllAndExit() => EditorMethod.Run(ImportAll);

        public static IEnumerable<string> ItemFolders(string root)
        {
            foreach (var placement in new[] { BuiltinPlacement, RemotePlacement })
            {
                var folder = $"{root}/{placement}";
                if (!AssetDatabase.IsValidFolder(folder)) continue;
                foreach (var type in AssetDatabase.GetSubFolders(folder))
                foreach (var item in AssetDatabase.GetSubFolders(type))
                    yield return item;
            }
        }

        public static string GroupName(string type, string slug) => $"cosmetic-{type}-{slug}";

        public static string[] Labels(string type, bool isStatic) =>
            isStatic ? new[] { CosmeticLabel, $"cosmetic-{type}", StaticLabel } : new[] { CosmeticLabel, $"cosmetic-{type}" };

        public static AddressableAssetGroup Import(string itemFolder)
        {
            var parts = itemFolder.Split('/');
            if (parts.Length < 4) throw new ArgumentException($"{itemFolder} is not <root>/<placement>/<type>/<slug>");
            var slug = parts[^1];
            var type = parts[^2];
            var placement = parts[^3];
            var stillPath = $"{itemFolder}/{StaticFile}";
            if (!File.Exists(stillPath)) throw new ArgumentException($"{itemFolder} has no {StaticFile}");

            var settings = AddressableAssetSettingsDefaultObject.GetSettings(true);
            var group = TargetGroup(settings, placement, type, slug);
            var addressBase = $"cosmetic/{type}/{slug}";
            RemoveEntries(settings, addressBase);

            var border = SliceBorder(itemFolder, stillPath);
            ImportSprite(stillPath, border);
            AddEntry(settings, group, stillPath, $"{addressBase}/static", Labels(type, true));
            foreach (var (slot, folder) in Loops(itemFolder))
                AddEntry(settings, group, BuildLoop(folder, border), $"{addressBase}/{slot}", Labels(type, false));
            EditorUtility.SetDirty(group);
            return group;
        }

        static AddressableAssetGroup TargetGroup(AddressableAssetSettings settings, string placement, string type, string slug)
        {
            var itemGroup = settings.FindGroup(GroupName(type, slug));
            if (placement == RemotePlacement) return itemGroup ?? CreateItemGroup(settings, GroupName(type, slug));
            if (placement != BuiltinPlacement) throw new ArgumentException($"placement {placement} is neither {BuiltinPlacement} nor {RemotePlacement}");
            if (itemGroup != null) settings.RemoveGroup(itemGroup);
            return settings.FindGroup(ContentSetup.BuiltinGroupName) ?? throw new InvalidOperationException("cosmetic-builtin is missing; run HypeDomino/Apply Content Setup");
        }

        static AddressableAssetGroup CreateItemGroup(AddressableAssetSettings settings, string name)
        {
            var group = settings.CreateGroup(name, false, false, true, null, typeof(BundledAssetGroupSchema), typeof(ContentUpdateGroupSchema));
            ContentSetup.ApplyRemoteItem(settings, group.GetSchema<BundledAssetGroupSchema>());
            group.GetSchema<ContentUpdateGroupSchema>().StaticContent = false;
            return group;
        }

        static void RemoveEntries(AddressableAssetSettings settings, string addressBase)
        {
            var prefix = addressBase + "/";
            var stale = new List<AddressableAssetEntry>();
            settings.GetAllAssets(stale, false, null, entry => entry.address.StartsWith(prefix, StringComparison.Ordinal));
            foreach (var entry in stale) settings.RemoveAssetEntry(entry.guid);
        }

        public static Vector4 Border(TableSlice slice, int width)
        {
            var side = (width - slice.step) / 2f;
            return new Vector4(side, slice.depth, side, slice.depth);
        }

        public static Vector4 SliceBorder(string itemFolder, string stillPath)
        {
            var path = $"{itemFolder}/{SliceFile}";
            if (!File.Exists(path)) return Vector4.zero;
            var importer = (TextureImporter)AssetImporter.GetAtPath(stillPath);
            importer.GetSourceTextureWidthAndHeight(out var width, out _);
            return Border(JsonUtility.FromJson<TableSlice>(File.ReadAllText(path)), width);
        }

        static IEnumerable<(string slot, string folder)> Loops(string itemFolder)
        {
            var anim = $"{itemFolder}/{AnimSlot}";
            if (HasFrames(anim)) yield return (AnimSlot, anim);
            else DeleteLoop(anim);

            var layers = $"{itemFolder}/{LayerSlot}";
            DeleteOrphanLoops(layers);
            if (!AssetDatabase.IsValidFolder(layers)) yield break;
            foreach (var layer in AssetDatabase.GetSubFolders(layers).OrderBy(path => path, StringComparer.Ordinal))
                yield return ($"{LayerSlot}/{Path.GetFileName(layer)}", layer);
        }

        internal static bool HasFrames(string folder) => AssetDatabase.IsValidFolder(folder) && Frames(folder).Length > 0;

        static string[] Frames(string folder) =>
            Directory.GetFiles(folder, "*.png").Select(path => path.Replace('\\', '/')).OrderBy(path => path, StringComparer.Ordinal).ToArray();

        static void DeleteLoop(string folder)
        {
            AssetDatabase.DeleteAsset(folder + LoopExtension);
            AssetDatabase.DeleteAsset(folder + AtlasExtension);
            if (AssetDatabase.IsValidFolder(folder)) AssetDatabase.DeleteAsset(folder);
        }

        static void DeleteOrphanLoops(string layers)
        {
            if (!AssetDatabase.IsValidFolder(layers)) return;
            foreach (var layer in AssetDatabase.GetSubFolders(layers))
                if (!HasFrames(layer)) DeleteLoop(layer);
            foreach (var generated in Directory.GetFiles(layers).Where(IsGenerated))
            {
                var folder = generated.Replace('\\', '/');
                folder = folder.Substring(0, folder.LastIndexOf('.'));
                if (!AssetDatabase.IsValidFolder(folder)) DeleteLoop(folder);
            }
        }

        static bool IsGenerated(string path) => path.EndsWith(LoopExtension, StringComparison.Ordinal) || path.EndsWith(AtlasExtension, StringComparison.Ordinal);

        internal static string BuildLoop(string folder, Vector4 border = default)
        {
            var frames = Frames(folder).Select(path => ImportSprite(path, border)).ToArray();
            var loopPath = folder + LoopExtension;
            var loop = AssetDatabase.LoadAssetAtPath<SpriteLoop>(loopPath);
            if (loop == null)
            {
                loop = ScriptableObject.CreateInstance<SpriteLoop>();
                AssetDatabase.CreateAsset(loop, loopPath);
            }
            loop.SetFrames(frames);
            EditorUtility.SetDirty(loop);
            BuildAtlas(folder);
            return loopPath;
        }

        internal static Sprite ImportSprite(string path, Vector4 border = default)
        {
            var importer = (TextureImporter)AssetImporter.GetAtPath(path);
            var texture = new TextureImporterSettings();
            importer.ReadTextureSettings(texture);
            texture.textureType = TextureImporterType.Sprite;
            texture.spriteMode = (int)SpriteImportMode.Single;
            texture.spriteMeshType = SpriteMeshType.FullRect;
            texture.spriteBorder = border;
            texture.mipmapEnabled = false;
            texture.alphaIsTransparency = true;
            texture.sRGBTexture = true;
            importer.SetTextureSettings(texture);
            foreach (var platform in TexturePlatforms)
            {
                var settings = importer.GetPlatformTextureSettings(platform);
                settings.overridden = true;
                settings.format = ProjectSetup.SpriteFormat;
                importer.SetPlatformTextureSettings(settings);
            }
            importer.SaveAndReimport();
            return AssetDatabase.LoadAssetAtPath<Sprite>(path);
        }

        static void BuildAtlas(string folder)
        {
            var path = folder + AtlasExtension;
            if (!File.Exists(path))
            {
                var atlas = new SpriteAtlasAsset();
                atlas.Add(new Object[] { AssetDatabase.LoadAssetAtPath<DefaultAsset>(folder) });
                SpriteAtlasAsset.Save(atlas, path);
                AssetDatabase.ImportAsset(path);
            }
            var importer = (SpriteAtlasImporter)AssetImporter.GetAtPath(path);
            var packing = importer.packingSettings;
            packing.enableRotation = false;
            packing.enableTightPacking = false;
            packing.padding = LoopAtlasPadding;
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
                importer.SetPlatformSettings(settings);
            }
            importer.SaveAndReimport();
        }

        static void AddEntry(AddressableAssetSettings settings, AddressableAssetGroup group, string path, string address, IEnumerable<string> labels)
        {
            var entry = settings.CreateOrMoveEntry(AssetDatabase.AssetPathToGUID(path), group);
            entry.address = address;
            foreach (var label in labels)
            {
                settings.AddLabel(label);
                entry.SetLabel(label, true, true);
            }
        }
    }
}
