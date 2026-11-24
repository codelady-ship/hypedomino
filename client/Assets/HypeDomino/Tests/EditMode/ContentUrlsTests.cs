using HypeDomino.Content;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ContentUrlsTests
    {
        const string Cdn = "https://content.example.com/production";

        [Test]
        public void TheCatalogOfAVersionLivesUnderItsPlatform()
        {
            Assert.AreEqual(Cdn + "/addressables/iOS/catalog_2026.11.01-2.bin", ContentUrls.Catalog(Cdn, "iOS", "2026.11.01-2"));
        }

        [Test]
        public void ALanguageBundleLivesUnderItsLanguageByItsHash()
        {
            Assert.AreEqual(Cdn + "/strings/az/0a1b.json", ContentUrls.Strings(Cdn, "az", "0a1b"));
        }

        [Test]
        public void ACosmeticsManifestLivesByItsHash()
        {
            Assert.AreEqual(Cdn + "/cosmetics/0a1b.json", ContentUrls.Cosmetics(Cdn, "0a1b"));
        }

        [Test]
        public void ARemoteBundleResolvesUnderTheAnnouncedRoot()
        {
            Assert.AreEqual(Cdn + "/addressables/Android/cosmetic-avatar-comet_fox_static_1a2b.bundle", ContentUrls.Resolve("hd-content:/Android/cosmetic-avatar-comet_fox_static_1a2b.bundle", Cdn));
        }

        [Test]
        public void InstallTimeContentIsLeftAlone()
        {
            const string local = "{UnityEngine.AddressableAssets.Addressables.RuntimePath}/Android/cosmetic-builtin.bundle";
            Assert.AreEqual(local, ContentUrls.Resolve(local, Cdn));
        }

        [Test]
        public void ARemoteBundleWithoutARootStaysUnresolved()
        {
            const string remote = "hd-content:/Android/a.bundle";
            Assert.AreEqual(remote, ContentUrls.Resolve(remote, null));
            Assert.AreEqual(remote, ContentUrls.Resolve(remote, ""));
        }
    }
}
