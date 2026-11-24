using HypeDomino.Editor;
using NUnit.Framework;
using UnityEditor.AddressableAssets;
using UnityEditor.AddressableAssets.Settings;
using UnityEditor.AddressableAssets.Settings.GroupSchemas;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ContentSetupTests
    {
        AddressableAssetSettings settings;

        [SetUp]
        public void LoadSettings()
        {
            settings = AddressableAssetSettingsDefaultObject.Settings;
            Assert.IsNotNull(settings, "Assets/AddressableAssetsData is missing; run ContentSetup.Apply");
        }

        [Test]
        public void RemoteContentLoadsThroughThePrefixTheClientRewrites()
        {
            Assert.AreEqual(ContentSetup.RemoteLoadPath, settings.profileSettings.GetValueByName(settings.activeProfileId, AddressableAssetSettings.kRemoteLoadPath));
            Assert.AreEqual("hd-content:/[BuildTarget]", ContentSetup.RemoteLoadPath);
        }

        [Test]
        public void TheRemoteCatalogIsBinaryAndNeverCheckedAtStart()
        {
            Assert.IsTrue(settings.BuildRemoteCatalog);
            Assert.IsTrue(settings.DisableCatalogUpdateOnStartup);
            Assert.IsFalse(settings.EnableJsonCatalog);
            Assert.AreEqual(AddressableAssetSettings.kRemoteLoadPath, settings.RemoteCatalogLoadPath.GetName(settings));
            Assert.AreEqual(AddressableAssetSettings.kRemoteBuildPath, settings.RemoteCatalogBuildPath.GetName(settings));
        }

        [Test]
        public void ContentIsBuiltWithEveryPlayerBuild()
        {
            Assert.AreEqual(AddressableAssetSettings.PlayerBuildOption.BuildWithPlayer, settings.BuildAddressablesWithPlayerBuild);
        }

        [Test]
        public void TheBuiltinGroupIsInstalledWithTheAppAndIsTheDefault()
        {
            var group = settings.FindGroup(ContentSetup.BuiltinGroupName);
            Assert.IsNotNull(group);
            Assert.AreSame(group, settings.DefaultGroup);
            var bundles = group.GetSchema<BundledAssetGroupSchema>();
            Assert.AreEqual(AddressableAssetSettings.kLocalLoadPath, bundles.LoadPath.GetName(settings));
            Assert.AreEqual(AddressableAssetSettings.kLocalBuildPath, bundles.BuildPath.GetName(settings));
            Assert.IsTrue(group.GetSchema<ContentUpdateGroupSchema>().StaticContent);
        }

        [Test]
        public void ItemGroupsAreRemoteOneBundlePerEntryWithContentHashedNames()
        {
            var template = ContentSetup.FindTemplate(settings);
            Assert.IsNotNull(template);
            var bundles = (BundledAssetGroupSchema)template.GetSchemaByType(typeof(BundledAssetGroupSchema));
            Assert.AreEqual(AddressableAssetSettings.kRemoteLoadPath, bundles.LoadPath.GetName(settings));
            Assert.AreEqual(AddressableAssetSettings.kRemoteBuildPath, bundles.BuildPath.GetName(settings));
            Assert.AreEqual(BundledAssetGroupSchema.BundlePackingMode.PackSeparately, bundles.BundleMode);
            Assert.AreEqual(BundledAssetGroupSchema.BundleNamingStyle.AppendHash, bundles.BundleNaming);
            Assert.AreEqual(ContentSetup.BundleRetries, bundles.RetryCount);
            Assert.AreEqual(ContentSetup.BundleTimeoutSeconds, bundles.Timeout);
            Assert.IsTrue(bundles.UseAssetBundleCache);
        }
    }
}
