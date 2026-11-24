using HypeDomino.Content;
using UnityEditor;
using UnityEditor.AddressableAssets;
using UnityEditor.AddressableAssets.Settings;
using UnityEditor.AddressableAssets.Settings.GroupSchemas;
using UnityEngine;

namespace HypeDomino.Editor
{
    public static class ContentSetup
    {
        public const string BuiltinGroupName = "cosmetic-builtin";
        public const string ItemTemplateName = "Cosmetic Item";
        public const string RemoteLoadPath = ContentUrls.RemotePrefix + "[BuildTarget]";
        public const int BundleRetries = 2;
        public const int BundleTimeoutSeconds = 20;
        const string DefaultLocalGroupName = "Default Local Group";

        [MenuItem("HypeDomino/Apply Content Setup")]
        public static void Apply()
        {
            var settings = AddressableAssetSettingsDefaultObject.GetSettings(true);
            settings.profileSettings.SetValue(settings.activeProfileId, AddressableAssetSettings.kRemoteLoadPath, RemoteLoadPath);
            settings.profileSettings.SetValue(settings.activeProfileId, AddressableAssetSettings.kRemoteBuildPath, AddressableAssetSettings.kRemoteBuildPathValue);
            settings.BuildRemoteCatalog = true;
            settings.RemoteCatalogBuildPath.SetVariableByName(settings, AddressableAssetSettings.kRemoteBuildPath);
            settings.RemoteCatalogLoadPath.SetVariableByName(settings, AddressableAssetSettings.kRemoteLoadPath);
            settings.DisableCatalogUpdateOnStartup = true;
            settings.EnableJsonCatalog = false;
            settings.BuildAddressablesWithPlayerBuild = AddressableAssetSettings.PlayerBuildOption.BuildWithPlayer;
            ApplyBuiltinGroup(settings);
            ApplyItemTemplate(settings);
            EditorUtility.SetDirty(settings);
            AssetDatabase.SaveAssets();
            Debug.Log("HypeDomino content setup applied");
        }

        public static void ApplyAndExit() => EditorMethod.Run(Apply);

        public static void ApplyRemoteItem(AddressableAssetSettings settings, BundledAssetGroupSchema bundles)
        {
            bundles.BuildPath.SetVariableByName(settings, AddressableAssetSettings.kRemoteBuildPath);
            bundles.LoadPath.SetVariableByName(settings, AddressableAssetSettings.kRemoteLoadPath);
            bundles.BundleMode = BundledAssetGroupSchema.BundlePackingMode.PackSeparately;
            bundles.BundleNaming = BundledAssetGroupSchema.BundleNamingStyle.AppendHash;
            bundles.RetryCount = BundleRetries;
            bundles.Timeout = BundleTimeoutSeconds;
            bundles.UseAssetBundleCache = true;
        }

        static void ApplyBuiltinGroup(AddressableAssetSettings settings)
        {
            var group = settings.FindGroup(BuiltinGroupName)
                ?? settings.CreateGroup(BuiltinGroupName, false, false, false, null, typeof(BundledAssetGroupSchema), typeof(ContentUpdateGroupSchema));
            var bundles = group.GetSchema<BundledAssetGroupSchema>();
            bundles.BuildPath.SetVariableByName(settings, AddressableAssetSettings.kLocalBuildPath);
            bundles.LoadPath.SetVariableByName(settings, AddressableAssetSettings.kLocalLoadPath);
            bundles.BundleMode = BundledAssetGroupSchema.BundlePackingMode.PackTogether;
            group.GetSchema<ContentUpdateGroupSchema>().StaticContent = true;
            settings.DefaultGroup = group;
            var unused = settings.FindGroup(DefaultLocalGroupName);
            if (unused != null && unused != group && unused.entries.Count == 0) settings.RemoveGroup(unused);
            EditorUtility.SetDirty(group);
        }

        static void ApplyItemTemplate(AddressableAssetSettings settings)
        {
            var template = FindTemplate(settings);
            if (template == null)
            {
                settings.CreateAndAddGroupTemplate(ItemTemplateName, "One cosmetic item, downloaded on demand (docs/cosmetics.md 8.2)", typeof(BundledAssetGroupSchema), typeof(ContentUpdateGroupSchema));
                template = FindTemplate(settings);
            }
            ApplyRemoteItem(settings, (BundledAssetGroupSchema)template.GetSchemaByType(typeof(BundledAssetGroupSchema)));
            ((ContentUpdateGroupSchema)template.GetSchemaByType(typeof(ContentUpdateGroupSchema))).StaticContent = false;
            EditorUtility.SetDirty(template);
        }

        public static AddressableAssetGroupTemplate FindTemplate(AddressableAssetSettings settings)
        {
            foreach (var candidate in settings.GroupTemplateObjects)
                if (candidate is AddressableAssetGroupTemplate template && template.name == ItemTemplateName) return template;
            return null;
        }
    }
}
