using System;
using System.IO;
using System.Linq;
using HypeDomino.Bench;
using HypeDomino.Content;
using HypeDomino.UI.Board;
using UnityEditor;
using UnityEditor.Build;
using UnityEditor.Build.Reporting;
using UnityEditor.SceneManagement;
using UnityEngine;

namespace HypeDomino.Editor
{
    public static class ArtBenchBuild
    {
        const string SceneFolder = "Assets/HypeDomino/Bench/Generated";
        const string ArtRoot = SceneFolder + "/Art";
        const string ScenePath = SceneFolder + "/ArtBench.unity";
        const string CosmeticRoot = ArtRoot + "/cosmetic";
        const string MedalRoot = ArtRoot + "/medal";
        const string DefaultOutputPath = "Builds/art-bench.apk";
        const string OutputPathArgument = "-customBuildPath";
        const string BenchIdentifier = "com.hypedomino.artbench";

        public static void BuildAndroid() => EditorMethod.Run(() => Build(OutputPath()));

        static void Build(string output)
        {
            if (!AssetDatabase.IsValidFolder(ArtRoot)) throw new InvalidOperationException($"{ArtRoot} is missing; the Bench APK workflow stages it from assets-src");
            Directory.CreateDirectory(Path.GetDirectoryName(Path.GetFullPath(output)));
            var content = Content();
            Debug.Log($"ArtBenchBuild avatars={content.avatars.Length} frames={content.frames.Length} tables={content.tables.Length} emoji={content.emoji.Length} " +
                      $"tileBacks={content.tileBacks.Length} tileFaces={content.tileFaces.Length} medals={content.medals.Length} effects={content.effectSprites.Length} " +
                      $"kit={content.kit.Length} sfx={content.sfx.Length} music={content.music.Length}");

            var scene = EditorSceneManager.NewScene(NewSceneSetup.EmptyScene, NewSceneMode.Single);
            new GameObject("Bench").AddComponent<ArtBench>().Configure(content);
            EditorSceneManager.SaveScene(scene, ScenePath);

            PlayerSettings.SetApplicationIdentifier(NamedBuildTarget.Android, BenchIdentifier);
            PlayerSettings.productName = "Art Bench";
            EditorUserBuildSettings.buildAppBundle = false;
            var report = BuildPipeline.BuildPlayer(new BuildPlayerOptions
            {
                scenes = new[] { ScenePath },
                locationPathName = output,
                target = BuildTarget.Android,
                options = BuildOptions.Development,
            });
            Debug.Log($"ArtBenchBuild result={report.summary.result} bytes={report.summary.totalSize}");
            if (report.summary.result != BuildResult.Succeeded) throw new BuildFailedException($"art bench build {report.summary.result}");
        }

        static ArtBenchContent Content()
        {
            return new ArtBenchContent
            {
                avatars = Items("avatar"),
                frames = Items("frame"),
                tables = Tables(),
                emoji = Items("emoji"),
                tileBacks = Items("tile"),
                tileFaces = Loaded<Sprite>(PngPaths(TileFaceContent.FacesFolder(TileFaceContent.DefaultSet))),
                tileFaceSet = AssetDatabase.LoadAssetAtPath<TileFaceSet>(TileFaceContent.SetPath(TileFaceContent.DefaultSet)),
                medals = SubFolders(MedalRoot).Select(Item).Where(item => item != null).ToArray(),
                effectSprites = Loaded<Sprite>(EffectsContent.SpritePaths()),
                kit = Loaded<Sprite>(UiKitContent.SpritePaths()),
                additive = AssetDatabase.LoadAssetAtPath<Material>(EffectsContent.AdditiveMaterialPath),
                alpha = AssetDatabase.LoadAssetAtPath<Material>(EffectsContent.AlphaMaterialPath),
                effectTable = AssetDatabase.LoadAssetAtPath<TextAsset>(EffectsContent.TablePath),
                sfx = Loaded<AudioClip>(SoundContent.SfxPaths()),
                music = Loaded<AudioClip>(SoundContent.MusicPaths()),
            };
        }

        static ArtBenchLoop[] Items(string type) => SubFolders($"{CosmeticRoot}/{type}").Select(Item).Where(item => item != null).ToArray();

        static ArtBenchLoop Item(string folder)
        {
            var still = $"{folder}/{CosmeticContent.StaticFile}";
            if (!File.Exists(still)) return null;
            var anim = $"{folder}/{CosmeticContent.AnimSlot}";
            return new ArtBenchLoop
            {
                name = Path.GetFileName(folder),
                still = CosmeticContent.ImportSprite(still),
                frames = CosmeticContent.HasFrames(anim) ? LoopFrames(anim) : Array.Empty<Sprite>(),
            };
        }

        static ArtBenchTable[] Tables() =>
            SubFolders($"{CosmeticRoot}/table").Select(Table).Where(table => table != null).ToArray();

        static ArtBenchTable Table(string folder)
        {
            var still = $"{folder}/{CosmeticContent.StaticFile}";
            if (!File.Exists(still)) return null;
            var border = CosmeticContent.SliceBorder(folder, still);
            var layers = $"{folder}/{CosmeticContent.LayerSlot}";
            return new ArtBenchTable
            {
                name = Path.GetFileName(folder),
                still = CosmeticContent.ImportSprite(still, border),
                layers = SubFolders(layers).Where(CosmeticContent.HasFrames).Select(layer => Layer(layer, border)).ToArray(),
            };
        }

        static ArtBenchLoop Layer(string folder, Vector4 border)
        {
            var frames = LoopFrames(folder, border);
            return new ArtBenchLoop { name = Path.GetFileName(folder), still = frames[0], frames = frames };
        }

        static Sprite[] LoopFrames(string folder, Vector4 border = default) => AssetDatabase.LoadAssetAtPath<SpriteLoop>(CosmeticContent.BuildLoop(folder, border)).Frames;

        static T[] Loaded<T>(string[] paths) where T : UnityEngine.Object =>
            paths.Select(AssetDatabase.LoadAssetAtPath<T>).Where(asset => asset != null).ToArray();

        static string[] SubFolders(string folder) =>
            AssetDatabase.IsValidFolder(folder) ? AssetDatabase.GetSubFolders(folder).OrderBy(path => path, StringComparer.Ordinal).ToArray() : Array.Empty<string>();

        static string[] PngPaths(string folder) =>
            Directory.Exists(folder)
                ? Directory.GetFiles(folder, "*.png").Select(path => path.Replace('\\', '/')).OrderBy(path => path, StringComparer.Ordinal).ToArray()
                : Array.Empty<string>();

        static string OutputPath()
        {
            var arguments = Environment.GetCommandLineArgs();
            var at = Array.IndexOf(arguments, OutputPathArgument);
            return at >= 0 && at + 1 < arguments.Length ? arguments[at + 1] : DefaultOutputPath;
        }
    }
}
