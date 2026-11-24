using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using HypeDomino.UI.Typography;
using HypeDomino.UI.Tokens;
using TMPro;
using UnityEditor;
using UnityEngine;
using UnityEngine.TextCore.LowLevel;

namespace HypeDomino.Editor
{
    public static class FontSetup
    {
        public const string FontsFolder = "Assets/HypeDomino/UI/Fonts";
        public const string ManifestPath = FontsFolder + "/fonts.json";
        public const string BuiltinStringsPath = "Assets/HypeDomino/UI/Localization/Resources/builtin-strings.json";
        const string SourceFolder = FontsFolder + "/Source";
        const string ResourcesFolder = FontsFolder + "/Resources";
        const string TypeSetPath = ResourcesFolder + "/" + TypeSet.ResourceName + ".asset";
        const string EssentialsPackage = "TMP Essential Resources.unitypackage";
        const string MobileDistanceField = "TextMeshPro/Mobile/Distance Field";
        static readonly string[] LeftOutEssentials = { "LiberationSans", "EmojiOne", "HDRP" };
        const string TextMeshProFolder = "Assets/TextMesh Pro";

        [Serializable]
        public sealed class AtlasSize
        {
            public int width;
            public int height;
        }

        [Serializable]
        public sealed class Atlas
        {
            public int samplingPointSize;
            public int padding;
            public AtlasSize[] sizes;
        }

        [Serializable]
        public sealed class Family
        {
            public string token;
            public string name;
            public string folder;
        }

        [Serializable]
        public sealed class Face
        {
            public string id;
            public string family;
            public int weight;
            public string[] fallbacks;
        }

        [Serializable]
        public sealed class CharacterSet
        {
            public string name;
            public string characters;
        }

        [Serializable]
        public sealed class UiOnly
        {
            public string characters;
            public string[] families;
        }

        [Serializable]
        public sealed class Manifest
        {
            public Atlas atlas;
            public Family[] families;
            public Face[] faces;
            public string defaultFace;
            public CharacterSet[] characterSets;
            public UiOnly uiOnly;
        }

        [Serializable]
        sealed class BuiltinEntry
        {
            public string en;
            public string az;
            public string tr;
            public string ru;
        }

        [Serializable]
        sealed class BuiltinStrings
        {
            public BuiltinEntry[] entries;
        }

        [MenuItem("HypeDomino/Generate Font Assets")]
        public static void Generate()
        {
            ImportTextMeshPro();
            var manifest = LoadManifest();
            var required = RequiredCharacters(manifest);
            Directory.CreateDirectory(ResourcesFolder);
            var fonts = new Dictionary<string, TMP_FontAsset>();
            foreach (var face in manifest.faces) fonts[face.id] = CreateFace(manifest, face, required);
            foreach (var face in manifest.faces) LinkFallbacks(fonts[face.id], face, fonts);
            WriteTypeSet(manifest, fonts);
            PointTextMeshProAt(fonts[manifest.defaultFace]);
            AssetDatabase.SaveAssets();
            Debug.Log($"Font assets generated: {manifest.faces.Length} faces, {required.Length} required characters");
        }

        public static void GenerateAndExit() => EditorMethod.Run(Generate);

        public static Manifest LoadManifest() => JsonUtility.FromJson<Manifest>(File.ReadAllText(ManifestPath));

        public static string RequiredCharacters(Manifest manifest)
        {
            var characters = new SortedSet<char>();
            foreach (var set in manifest.characterSets) characters.UnionWith(set.characters);
            var builtin = JsonUtility.FromJson<BuiltinStrings>(File.ReadAllText(BuiltinStringsPath));
            foreach (var entry in builtin.entries)
                foreach (var value in new[] { entry.en, entry.az, entry.tr, entry.ru })
                    if (value != null) characters.UnionWith(value);
            var text = new StringBuilder();
            foreach (var character in characters) text.Append(character);
            return text.ToString();
        }

        public static TokenFontFamily FamilyToken(string token) => (TokenFontFamily)Enum.Parse(typeof(TokenFontFamily), char.ToUpperInvariant(token[0]) + token.Substring(1));

        static void ImportTextMeshPro()
        {
            if (Shader.Find(MobileDistanceField) != null) return;
            var package = UnityEditor.PackageManager.PackageInfo.FindForAssembly(typeof(TMP_Text).Assembly);
            if (package == null) throw new InvalidOperationException("TextMeshPro is not part of a package");
            var found = Directory.GetFiles(package.resolvedPath, EssentialsPackage, SearchOption.AllDirectories);
            if (found.Length == 0) throw new FileNotFoundException($"{EssentialsPackage} not found under {package.resolvedPath}");
            var written = UnityPackageFiles.Extract(found[0], IsKeptEssential);
            RemoveEmptyFolders(TextMeshProFolder);
            AssetDatabase.Refresh(ImportAssetOptions.ForceSynchronousImport);
            if (Shader.Find(MobileDistanceField) == null) throw new InvalidOperationException($"{MobileDistanceField} is missing after extracting {found[0]}");
            Debug.Log($"Extracted {written} entries of {found[0]}");
        }

        static bool IsKeptEssential(string path)
        {
            if (path != TextMeshProFolder && !path.StartsWith(TextMeshProFolder + "/", StringComparison.Ordinal)) return false;
            foreach (var name in LeftOutEssentials)
                if (path.Contains(name)) return false;
            return true;
        }

        static void RemoveEmptyFolders(string folder)
        {
            foreach (var child in Directory.GetDirectories(folder)) RemoveEmptyFolders(child);
            if (Directory.GetFileSystemEntries(folder).Length > 0) return;
            Directory.Delete(folder);
            File.Delete(folder + ".meta");
        }

        static Family FamilyOf(Manifest manifest, string token)
        {
            foreach (var family in manifest.families)
                if (family.token == token) return family;
            throw new InvalidOperationException($"fonts.json has no family {token}");
        }

        static TMP_FontAsset CreateFace(Manifest manifest, Face face, string required)
        {
            var family = FamilyOf(manifest, face.family);
            var sourcePath = $"{SourceFolder}/{family.folder}/{face.id}.ttf";
            var source = AssetDatabase.LoadAssetAtPath<Font>(sourcePath);
            if (source == null) throw new FileNotFoundException(sourcePath);
            var excluded = Array.IndexOf(manifest.uiOnly.families, face.family) < 0 ? manifest.uiOnly.characters : string.Empty;
            var characters = CharactersInFont(source, manifest.atlas.samplingPointSize, required, excluded);
            foreach (var size in manifest.atlas.sizes)
            {
                var font = TMP_FontAsset.CreateFontAsset(source, manifest.atlas.samplingPointSize, manifest.atlas.padding, GlyphRenderMode.SDFAA, size.width, size.height, AtlasPopulationMode.Dynamic, false);
                if (font.TryAddCharacters(characters, out var missing, true) && string.IsNullOrEmpty(missing))
                {
                    KeepFeaturesOfDrawnGlyphs(font);
                    return Save(font, face.id);
                }
                UnityEngine.Object.DestroyImmediate(font.atlasTexture);
                UnityEngine.Object.DestroyImmediate(font.material);
                UnityEngine.Object.DestroyImmediate(font);
            }
            throw new InvalidOperationException($"{face.id}: {characters.Length} characters do not fit the largest atlas in fonts.json");
        }

        static string CharactersInFont(Font source, int pointSize, string required, string excluded)
        {
            FontEngine.InitializeFontEngine();
            if (FontEngine.LoadFontFace(source, pointSize) != FontEngineError.Success) throw new InvalidOperationException($"cannot load {source.name}");
            var characters = new StringBuilder();
            foreach (var character in required)
                if (excluded.IndexOf(character) < 0 && FontEngine.TryGetGlyphWithUnicodeValue(character, GlyphLoadFlags.LOAD_NO_BITMAP, out var glyph) && glyph.index != 0)
                    characters.Append(character);
            return characters.ToString();
        }

        static void KeepFeaturesOfDrawnGlyphs(TMP_FontAsset font)
        {
            var drawn = new HashSet<uint>();
            foreach (var character in font.characterTable) drawn.Add(character.glyphIndex);
            var features = font.fontFeatureTable;
            features.glyphPairAdjustmentRecords.RemoveAll(pair => !drawn.Contains(pair.firstAdjustmentRecord.glyphIndex) || !drawn.Contains(pair.secondAdjustmentRecord.glyphIndex));
            features.MarkToBaseAdjustmentRecords.RemoveAll(mark => !drawn.Contains(mark.baseGlyphID) || !drawn.Contains(mark.markGlyphID));
            features.MarkToMarkAdjustmentRecords.RemoveAll(mark => !drawn.Contains(mark.baseMarkGlyphID) || !drawn.Contains(mark.combiningMarkGlyphID));
            font.ReadFontAssetDefinition();
        }

        static TMP_FontAsset Save(TMP_FontAsset font, string id)
        {
            var path = $"{FontsFolder}/{id}.asset";
            AssetDatabase.DeleteAsset(path);
            font.name = id;
            font.atlasPopulationMode = AtlasPopulationMode.Static;
            AssetDatabase.CreateAsset(font, path);
            var atlas = font.atlasTexture;
            atlas.name = id + " Atlas";
            AssetDatabase.AddObjectToAsset(atlas, font);
            font.material.name = id + " Material";
            AssetDatabase.AddObjectToAsset(font.material, font);
            var serializedAtlas = new SerializedObject(atlas);
            serializedAtlas.FindProperty("m_IsReadable").boolValue = false;
            serializedAtlas.ApplyModifiedPropertiesWithoutUndo();
            EditorUtility.SetDirty(font);
            return font;
        }

        static void LinkFallbacks(TMP_FontAsset font, Face face, Dictionary<string, TMP_FontAsset> fonts)
        {
            var fallbacks = new List<TMP_FontAsset>();
            foreach (var id in face.fallbacks) fallbacks.Add(fonts[id]);
            font.fallbackFontAssetTable = fallbacks;
            EditorUtility.SetDirty(font);
        }

        static void WriteTypeSet(Manifest manifest, Dictionary<string, TMP_FontAsset> fonts)
        {
            var typeSet = AssetDatabase.LoadAssetAtPath<TypeSet>(TypeSetPath);
            if (typeSet == null)
            {
                typeSet = ScriptableObject.CreateInstance<TypeSet>();
                AssetDatabase.CreateAsset(typeSet, TypeSetPath);
            }
            var serialized = new SerializedObject(typeSet);
            var faces = serialized.FindProperty("faces");
            faces.arraySize = manifest.faces.Length;
            for (var at = 0; at < manifest.faces.Length; at++)
            {
                var face = manifest.faces[at];
                var element = faces.GetArrayElementAtIndex(at);
                element.FindPropertyRelative("family").enumValueIndex = (int)FamilyToken(face.family);
                element.FindPropertyRelative("weight").intValue = face.weight;
                element.FindPropertyRelative("font").objectReferenceValue = fonts[face.id];
            }
            serialized.ApplyModifiedPropertiesWithoutUndo();
            EditorUtility.SetDirty(typeSet);
        }

        static void PointTextMeshProAt(TMP_FontAsset font)
        {
            var settings = TMP_Settings.instance;
            if (settings == null) throw new InvalidOperationException("TMP Settings is missing");
            var serialized = new SerializedObject(settings);
            serialized.FindProperty("m_defaultFontAsset").objectReferenceValue = font;
            serialized.FindProperty("m_fallbackFontAssets").ClearArray();
            serialized.FindProperty("m_defaultSpriteAsset").objectReferenceValue = null;
            serialized.ApplyModifiedPropertiesWithoutUndo();
            EditorUtility.SetDirty(settings);
        }
    }
}
