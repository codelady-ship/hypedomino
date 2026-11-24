using HypeDomino.Content;
using HypeDomino.Editor;
using NUnit.Framework;
using UnityEditor.AddressableAssets;
using UnityEditor.AddressableAssets.Settings;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ReleaseBuildTests
    {
        [TestCase("2026.11.08-1")]
        [TestCase("v7_hotfix")]
        [TestCase("a")]
        [TestCase("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")]
        public void ACatalogVersionTheServerAcceptsIsAccepted(string version)
        {
            Assert.IsTrue(ReleaseBuild.IsCatalogVersion(version));
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase(".hidden")]
        [TestCase("-1")]
        [TestCase("a/b")]
        [TestCase("a b")]
        [TestCase("a?b")]
        [TestCase("1\n")]
        [TestCase("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")]
        public void ACatalogVersionTheServerRefusesIsRefused(string version)
        {
            Assert.IsFalse(ReleaseBuild.IsCatalogVersion(version));
        }

        [TestCase("0.0.1")]
        [TestCase("1.2.3")]
        [TestCase("10.0.999999")]
        public void AnAppVersionTheServerComparesIsAccepted(string version)
        {
            Assert.IsTrue(ReleaseBuild.IsAppVersion(version));
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("1.2")]
        [TestCase("1.2.3.4")]
        [TestCase("v1.2.3")]
        [TestCase("01.2.3")]
        [TestCase("1.2.3-beta")]
        [TestCase("1.2.3\n")]
        [TestCase("1.2.1000000")]
        public void AnAppVersionTheServerCannotCompareIsRefused(string version)
        {
            Assert.IsFalse(ReleaseBuild.IsAppVersion(version));
        }

        [TestCase("1", 1)]
        [TestCase("57", 57)]
        [TestCase("2100000000", 2100000000)]
        public void ABuildNumberPlayAcceptsIsParsed(string text, int expected)
        {
            Assert.IsTrue(ReleaseBuild.TryParseBuildNumber(text, out var number));
            Assert.AreEqual(expected, number);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("0")]
        [TestCase("-1")]
        [TestCase("+5")]
        [TestCase(" 5")]
        [TestCase("1.5")]
        [TestCase("2100000001")]
        [TestCase("99999999999")]
        public void ABuildNumberPlayRefusesIsRefused(string text)
        {
            Assert.IsFalse(ReleaseBuild.TryParseBuildNumber(text, out _));
        }

        [Test]
        public void ATaggedReleaseCarriesItsVersionNumberIdentifierAndSigning()
        {
            var release = ReleaseBuild.ParseRelease("1.2.3", "57", "com.example.game", "{\"keystore\":\"Signing/release.keystore\",\"keystorePass\":\"a\",\"keyAlias\":\"upload\",\"keyAliasPass\":\"b\"}");
            Assert.AreEqual("1.2.3", release.AppVersion);
            Assert.AreEqual(57, release.BuildNumber);
            Assert.AreEqual("com.example.game", release.ApplicationId);
            Assert.AreEqual("Signing/release.keystore", release.Signing.keystore);
            Assert.AreEqual("upload", release.Signing.keyAlias);
        }

        [Test]
        public void AnUntaggedReleaseKeepsTheProjectSettings()
        {
            var release = ReleaseBuild.ParseRelease(null, null, null, null);
            Assert.IsNull(release.AppVersion);
            Assert.AreEqual(0, release.BuildNumber);
            Assert.IsNull(release.ApplicationId);
            Assert.IsNull(release.Signing);
        }

        [TestCase("1.2", null, null, null)]
        [TestCase(null, "0", null, null)]
        [TestCase(null, null, "", null)]
        [TestCase(null, null, null, "{}")]
        [TestCase(null, null, null, "{\"keystore\":\"k\",\"keystorePass\":\"a\",\"keyAlias\":\"upload\"}")]
        public void AReleaseWithAnInvalidPartIsRefused(string appVersion, string buildNumber, string applicationId, string signing)
        {
            Assert.Throws<System.ArgumentException>(() => ReleaseBuild.ParseRelease(appVersion, buildNumber, applicationId, signing));
        }

        [Test]
        public void TheBuiltCatalogHasTheNameTheClientDownloads()
        {
            Assert.AreEqual("catalog_2026.11.08-1.bin", ContentUrls.CatalogFile("2026.11.08-1"));
            StringAssert.EndsWith("/Android/" + ContentUrls.CatalogFile("2026.11.08-1"), ContentUrls.Catalog("https://content.example.com", "Android", "2026.11.08-1"));
        }

        [Test]
        public void RemoteContentIsBuiltIntoTheFolderTheWorkflowUploads()
        {
            var settings = AddressableAssetSettingsDefaultObject.Settings;
            Assert.AreEqual(AddressableAssetSettings.kRemoteBuildPathValue, settings.profileSettings.GetValueByName(settings.activeProfileId, AddressableAssetSettings.kRemoteBuildPath));
            Assert.AreEqual(ReleaseBuild.AndroidRemoteFolder, AddressableAssetSettings.kRemoteBuildPathValue.Replace("[BuildTarget]", "Android"));
        }
    }
}
