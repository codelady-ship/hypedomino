using System;
using System.IO;
using HypeDomino.Bench;
using UnityEditor;
using UnityEditor.Build;
using UnityEditor.Build.Reporting;
using UnityEditor.SceneManagement;
using UnityEngine;

namespace HypeDomino.Editor
{
    public static class CanvasBenchBuild
    {
        const string SceneFolder = "Assets/HypeDomino/Bench/Generated";
        const string ScenePath = SceneFolder + "/CanvasBench.unity";
        const string DefaultOutputPath = "Builds/canvas-bench.apk";
        const string OutputPathArgument = "-customBuildPath";
        const string BenchIdentifier = "com.hypedomino.canvasbench";

        public static void BuildAndroid() => EditorMethod.Run(() => Build(OutputPath()));

        static void Build(string output)
        {
            Directory.CreateDirectory(SceneFolder);
            Directory.CreateDirectory(Path.GetDirectoryName(Path.GetFullPath(output)));
            var scene = EditorSceneManager.NewScene(NewSceneSetup.EmptyScene, NewSceneMode.Single);
            new GameObject("Bench").AddComponent<CanvasBench>();
            EditorSceneManager.SaveScene(scene, ScenePath);

            PlayerSettings.SetApplicationIdentifier(NamedBuildTarget.Android, BenchIdentifier);
            PlayerSettings.productName = "Canvas Bench";
            EditorUserBuildSettings.buildAppBundle = false;
            var report = BuildPipeline.BuildPlayer(new BuildPlayerOptions
            {
                scenes = new[] { ScenePath },
                locationPathName = output,
                target = BuildTarget.Android,
                options = BuildOptions.Development,
            });
            Debug.Log($"CanvasBenchBuild result={report.summary.result} bytes={report.summary.totalSize}");
            if (report.summary.result != BuildResult.Succeeded) throw new BuildFailedException($"canvas bench build {report.summary.result}");
        }

        static string OutputPath()
        {
            var arguments = Environment.GetCommandLineArgs();
            var at = Array.IndexOf(arguments, OutputPathArgument);
            return at >= 0 && at + 1 < arguments.Length ? arguments[at + 1] : DefaultOutputPath;
        }
    }
}
