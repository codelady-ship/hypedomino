using System;
using System.Globalization;
using System.IO;
using System.Text.RegularExpressions;
using HypeDomino.Content;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Net;
using UnityEditor;
using UnityEditor.AddressableAssets;
using UnityEditor.Build;
using UnityEditor.Build.Reporting;
using UnityEngine;

namespace HypeDomino.Editor
{
    public static class ReleaseBuild
    {
        public const string CatalogVersionArgument = "-catalogVersion";
        public const string AppVersionArgument = "-appVersion";
        public const string BuildNumberArgument = "-buildNumber";
        public const string ApplicationIdArgument = "-applicationId";
        public const string SigningArgument = "-androidSigning";
        public const string ServerUrlArgument = "-serverUrl";
        public const string HttpTimeoutArgument = "-httpTimeoutSeconds";
        public const string CrashReportingArgument = "-crashReporting";
        public const string EndpointFolder = "Assets/HypeDomino/Net/Resources";
        public const string AndroidRemoteFolder = "ServerData/Android";
        public const int MaxAndroidVersionCode = 2100000000;
        const string OutputPathArgument = "-customBuildPath";
        const string DefaultAndroidOutput = "Builds/hypedomino.aab";

        static readonly Regex CatalogVersionPattern = new Regex(@"^[A-Za-z0-9][A-Za-z0-9._-]{0,63}\z");
        static readonly Regex AppVersionPattern = new Regex(@"^(0|[1-9][0-9]{0,5})\.(0|[1-9][0-9]{0,5})\.(0|[1-9][0-9]{0,5})\z");

        [Serializable]
        public sealed class AndroidSigning
        {
            public string keystore;
            public string keystorePass;
            public string keyAlias;
            public string keyAliasPass;
        }

        public sealed class AndroidRelease
        {
            public string AppVersion;
            public int BuildNumber;
            public string ApplicationId;
            public AndroidSigning Signing;
            public ServerEndpoint Endpoint;
            public CrashReportSettings CrashReporting;
        }

        public static void BuildAndroid() => EditorMethod.Run(() => BuildAndroid(Argument(CatalogVersionArgument), Argument(OutputPathArgument) ?? DefaultAndroidOutput, ReleaseFromArguments()));

        public static void BuildAndroid(string catalogVersion, string output, AndroidRelease release)
        {
            if (!IsCatalogVersion(catalogVersion)) throw new ArgumentException($"{CatalogVersionArgument} must be 1 to 64 letters, digits, '.', '_' or '-' starting with a letter or digit, got '{catalogVersion}'");
            var settings = AddressableAssetSettingsDefaultObject.Settings ?? throw new BuildFailedException("Addressables settings are missing; run ContentSetup.Apply");
            var previousVersion = settings.OverridePlayerVersion;
            var previousAppBundle = EditorUserBuildSettings.buildAppBundle;
            var restorePlayer = SnapshotAndroidPlayer();
            try
            {
                settings.OverridePlayerVersion = catalogVersion;
                EditorUserBuildSettings.buildAppBundle = true;
                Apply(release);
                WriteEndpoint(release.Endpoint);
                WriteCrashReporting(release.CrashReporting);
                if (Directory.Exists(AndroidRemoteFolder)) Directory.Delete(AndroidRemoteFolder, true);
                Directory.CreateDirectory(Path.GetDirectoryName(Path.GetFullPath(output)));
                var report = BuildPipeline.BuildPlayer(new BuildPlayerOptions
                {
                    scenes = EditorBuildSettingsScene.GetActiveSceneList(EditorBuildSettings.scenes),
                    locationPathName = output,
                    target = BuildTarget.Android,
                    options = BuildOptions.None,
                });
                if (report.summary.result != BuildResult.Succeeded) throw new BuildFailedException($"Android release build {report.summary.result}");
                var catalog = Path.Combine(AndroidRemoteFolder, ContentUrls.CatalogFile(catalogVersion));
                if (!File.Exists(catalog)) throw new BuildFailedException($"{catalog} was not built");
                Debug.Log($"ReleaseBuild version={PlayerSettings.bundleVersion} versionCode={PlayerSettings.Android.bundleVersionCode} catalog={catalogVersion} signed={PlayerSettings.Android.useCustomKeystore} server={release.Endpoint?.serverUrl ?? "none"} crashReports={release.CrashReporting != null}");
            }
            finally
            {
                AssetDatabase.DeleteAsset(EndpointFolder);
                settings.OverridePlayerVersion = previousVersion;
                EditorUserBuildSettings.buildAppBundle = previousAppBundle;
                restorePlayer();
            }
        }

        public static bool IsCatalogVersion(string version) => version != null && CatalogVersionPattern.IsMatch(version);

        public static bool IsAppVersion(string version) => version != null && AppVersionPattern.IsMatch(version);

        public static bool TryParseBuildNumber(string text, out int number) =>
            int.TryParse(text, NumberStyles.None, CultureInfo.InvariantCulture, out number) && number >= 1 && number <= MaxAndroidVersionCode;

        public static AndroidRelease ParseRelease(string appVersion, string buildNumber, string applicationId, string signingJson)
        {
            if (appVersion != null && !IsAppVersion(appVersion)) throw new ArgumentException($"{AppVersionArgument} must be major.minor.patch, got '{appVersion}'");
            var number = 0;
            if (buildNumber != null && !TryParseBuildNumber(buildNumber, out number)) throw new ArgumentException($"{BuildNumberArgument} must be a whole number from 1 to {MaxAndroidVersionCode}, got '{buildNumber}'");
            if (applicationId != null && applicationId.Length == 0) throw new ArgumentException($"{ApplicationIdArgument} is empty");
            return new AndroidRelease
            {
                AppVersion = appVersion,
                BuildNumber = number,
                ApplicationId = applicationId,
                Signing = signingJson == null ? null : ParseSigning(signingJson),
            };
        }

        public static AndroidSigning ParseSigning(string json)
        {
            var signing = JsonUtility.FromJson<AndroidSigning>(json);
            if (signing == null || string.IsNullOrEmpty(signing.keystore) || string.IsNullOrEmpty(signing.keystorePass) || string.IsNullOrEmpty(signing.keyAlias) || string.IsNullOrEmpty(signing.keyAliasPass))
                throw new ArgumentException("the Android signing file needs keystore, keystorePass, keyAlias and keyAliasPass");
            return signing;
        }

        public static ServerEndpoint ParseEndpoint(string serverUrl, string httpTimeoutSeconds)
        {
            if (serverUrl == null && httpTimeoutSeconds == null) return null;
            if (serverUrl == null || httpTimeoutSeconds == null) throw new ArgumentException($"{ServerUrlArgument} and {HttpTimeoutArgument} are given together or not at all");
            return ServerEndpoint.Parse(serverUrl, httpTimeoutSeconds);
        }

        static AndroidRelease ReleaseFromArguments()
        {
            var signingFile = Argument(SigningArgument);
            var release = ParseRelease(Argument(AppVersionArgument), Argument(BuildNumberArgument), Argument(ApplicationIdArgument), signingFile == null ? null : File.ReadAllText(signingFile));
            release.Endpoint = ParseEndpoint(Argument(ServerUrlArgument), Argument(HttpTimeoutArgument));
            var crashReportingFile = Argument(CrashReportingArgument);
            release.CrashReporting = crashReportingFile == null ? null : ParseCrashReporting(File.ReadAllText(crashReportingFile));
            return release;
        }

        public static CrashReportSettings ParseCrashReporting(string json) =>
            CrashReportSettings.FromJson(json) ?? throw new ArgumentException($"{CrashReportingArgument} has no dsn; leave the argument out to build without crash reports");

        static void WriteCrashReporting(CrashReportSettings settings)
        {
            if (settings == null) return;
            Directory.CreateDirectory(EndpointFolder);
            var path = $"{EndpointFolder}/{CrashReportSettings.ResourceName}.json";
            File.WriteAllText(path, JsonUtility.ToJson(settings));
            AssetDatabase.ImportAsset(path);
        }

        static void WriteEndpoint(ServerEndpoint endpoint)
        {
            if (endpoint == null) return;
            Directory.CreateDirectory(EndpointFolder);
            var path = $"{EndpointFolder}/{ServerEndpoint.ResourceName}.json";
            File.WriteAllText(path, JsonUtility.ToJson(endpoint));
            AssetDatabase.ImportAsset(path);
        }

        static void Apply(AndroidRelease release)
        {
            if (release.AppVersion != null) PlayerSettings.bundleVersion = release.AppVersion;
            if (release.BuildNumber > 0) PlayerSettings.Android.bundleVersionCode = release.BuildNumber;
            if (release.ApplicationId != null) PlayerSettings.SetApplicationIdentifier(NamedBuildTarget.Android, release.ApplicationId);
            if (release.Signing == null) return;
            PlayerSettings.Android.useCustomKeystore = true;
            PlayerSettings.Android.keystoreName = Path.GetFullPath(release.Signing.keystore);
            PlayerSettings.Android.keystorePass = release.Signing.keystorePass;
            PlayerSettings.Android.keyaliasName = release.Signing.keyAlias;
            PlayerSettings.Android.keyaliasPass = release.Signing.keyAliasPass;
        }

        static Action SnapshotAndroidPlayer()
        {
            var bundleVersion = PlayerSettings.bundleVersion;
            var versionCode = PlayerSettings.Android.bundleVersionCode;
            var applicationId = PlayerSettings.GetApplicationIdentifier(NamedBuildTarget.Android);
            var useCustomKeystore = PlayerSettings.Android.useCustomKeystore;
            var keystoreName = PlayerSettings.Android.keystoreName;
            var keystorePass = PlayerSettings.Android.keystorePass;
            var keyaliasName = PlayerSettings.Android.keyaliasName;
            var keyaliasPass = PlayerSettings.Android.keyaliasPass;
            return () =>
            {
                PlayerSettings.bundleVersion = bundleVersion;
                PlayerSettings.Android.bundleVersionCode = versionCode;
                PlayerSettings.SetApplicationIdentifier(NamedBuildTarget.Android, applicationId);
                PlayerSettings.Android.useCustomKeystore = useCustomKeystore;
                PlayerSettings.Android.keystoreName = keystoreName;
                PlayerSettings.Android.keystorePass = keystorePass;
                PlayerSettings.Android.keyaliasName = keyaliasName;
                PlayerSettings.Android.keyaliasPass = keyaliasPass;
            };
        }

        static string Argument(string name)
        {
            var arguments = Environment.GetCommandLineArgs();
            var at = Array.IndexOf(arguments, name);
            return at >= 0 && at + 1 < arguments.Length ? arguments[at + 1] : null;
        }
    }
}
