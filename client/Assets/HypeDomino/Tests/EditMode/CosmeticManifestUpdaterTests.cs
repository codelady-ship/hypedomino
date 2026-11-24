using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using System.Text.RegularExpressions;
using HypeDomino.Content;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Localization;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.TestTools;

namespace HypeDomino.Tests.EditMode
{
    public sealed class CosmeticManifestUpdaterTests
    {
        const string Cdn = "https://cdn.example.com";

        static readonly string Fourth = Manifest(4, "avatar.fox");
        static readonly string Fifth = Manifest(5, "avatar.owl");

        static string Manifest(int version, string id) =>
            "{\"schemaVersion\":1,\"manifestVersion\":" + version + ",\"items\":[{\"id\":\"" + id + "\",\"type\":\"avatar\",\"rarityId\":\"rare\",\"availability\":\"live\",\"staticAddress\":\"cosmetic/avatar/x/static\"}]}";

        sealed class ServedSource : ICosmeticManifestSource
        {
            readonly UnityCosmeticManifestSource disk;

            public ServedSource(string folder) => disk = new UnityCosmeticManifestSource(folder);

            public Dictionary<string, byte[]> Served { get; } = new();

            public List<string> Downloads { get; } = new();

            public void Serve(string hash, string content) => Served[ContentUrls.Cosmetics(Cdn, hash)] = Encoding.UTF8.GetBytes(content);

            public void Download(string url, Action<byte[]> done)
            {
                Downloads.Add(url);
                done(Served.TryGetValue(url, out var content) ? content : null);
            }

            public bool TryLoadStored(out int version, out string hash, out byte[] content) => disk.TryLoadStored(out version, out hash, out content);

            public void Store(int version, string hash, byte[] content) => disk.Store(version, hash, content);
        }

        string folder;

        [SetUp]
        public void CreateFolder() => folder = Path.Combine(Path.GetTempPath(), "hd-cosmetics-" + Guid.NewGuid().ToString("N"));

        [TearDown]
        public void DeleteFolder()
        {
            if (Directory.Exists(folder)) Directory.Delete(folder, true);
        }

        static string Hash(string content) => StringBundle.Sha256Hex(Encoding.UTF8.GetBytes(content));

        static ActiveCosmeticManifest Builtin() => new ActiveCosmeticManifest(null);

        ActiveCosmeticManifest NextStart(out ServedSource source)
        {
            source = new ServedSource(folder);
            var manifest = Builtin();
            new CosmeticManifestUpdater(source, manifest).LoadStored();
            return manifest;
        }

        void FetchedOnce(string content, int version)
        {
            var source = new ServedSource(folder);
            source.Serve(Hash(content), content);
            new CosmeticManifestUpdater(source, Builtin()).Fetch(Cdn, version, Hash(content));
        }

        [Test]
        public void TheAnnouncedManifestIsDownloadedFromItsHashAndSwappedAtTheNextScreenOutsideAMatch()
        {
            var source = new ServedSource(folder);
            source.Serve(Hash(Fourth), Fourth);
            var manifest = Builtin();
            var changed = 0;
            manifest.Changed += () => changed++;
            new CosmeticManifestUpdater(source, manifest).Fetch(Cdn, 4, Hash(Fourth));
            CollectionAssert.AreEqual(new[] { Cdn + "/cosmetics/" + Hash(Fourth) + ".json" }, source.Downloads);
            Assert.IsTrue(manifest.Holds(Hash(Fourth)));
            Assert.IsNull(manifest.Current);
            Assert.IsFalse(manifest.ApplyStaged(true));
            Assert.IsNull(manifest.Current);
            Assert.IsTrue(manifest.ApplyStaged(false));
            Assert.AreEqual(4, manifest.Current.ManifestVersion);
            Assert.AreEqual(Hash(Fourth), manifest.Hash);
            Assert.AreEqual(1, changed);
        }

        [Test]
        public void AManifestWhoseContentDoesNotMatchTheAnnouncedHashIsRefusedAndNotKept()
        {
            var source = new ServedSource(folder);
            source.Serve(Hash(Fourth), Fifth);
            var manifest = Builtin();
            LogAssert.Expect(LogType.Warning, new Regex("^Cosmetics fetch refused version=4$"));
            new CosmeticManifestUpdater(source, manifest).Fetch(Cdn, 4, Hash(Fourth));
            Assert.IsFalse(manifest.Holds(Hash(Fourth)));
            Assert.IsFalse(manifest.ApplyStaged(false));
            Assert.IsFalse(Directory.Exists(Path.Combine(folder, "cosmetics")));
        }

        [Test]
        public void AManifestOfAnotherVersionThanAnnouncedIsRefused()
        {
            var source = new ServedSource(folder);
            source.Serve(Hash(Fifth), Fifth);
            var manifest = Builtin();
            LogAssert.Expect(LogType.Warning, new Regex("^Cosmetics fetch refused version=4$"));
            new CosmeticManifestUpdater(source, manifest).Fetch(Cdn, 4, Hash(Fifth));
            Assert.IsFalse(manifest.ApplyStaged(false));
        }

        [Test]
        public void AnUnreadableManifestIsRefused()
        {
            const string broken = "{\"schemaVersion\":2,\"manifestVersion\":4,\"items\":[]}";
            var source = new ServedSource(folder);
            source.Serve(Hash(broken), broken);
            var manifest = Builtin();
            LogAssert.Expect(LogType.Warning, new Regex("^Cosmetics fetch refused version=4$"));
            new CosmeticManifestUpdater(source, manifest).Fetch(Cdn, 4, Hash(broken));
            Assert.IsFalse(manifest.ApplyStaged(false));
        }

        [Test]
        public void TheKeptManifestIsUsedOnTheNextStartWithoutAFetch()
        {
            FetchedOnce(Fourth, 4);
            var manifest = NextStart(out var source);
            Assert.AreEqual(4, manifest.Current.ManifestVersion);
            Assert.IsTrue(manifest.Holds(Hash(Fourth)));
            new CosmeticManifestUpdater(source, manifest).Fetch(Cdn, 4, Hash(Fourth));
            CollectionAssert.IsEmpty(source.Downloads);
        }

        [Test]
        public void ANewerManifestReplacesTheKeptOne()
        {
            FetchedOnce(Fourth, 4);
            FetchedOnce(Fifth, 5);
            Assert.AreEqual(1, Directory.GetFiles(Path.Combine(folder, "cosmetics")).Length);
            Assert.AreEqual(5, NextStart(out _).Current.ManifestVersion);
        }

        [Test]
        public void ADamagedKeptManifestIsRefusedAndTheBuiltinStays()
        {
            FetchedOnce(Fourth, 4);
            var kept = Directory.GetFiles(Path.Combine(folder, "cosmetics"))[0];
            File.WriteAllText(kept, Fifth);
            LogAssert.Expect(LogType.Warning, new Regex("^Cosmetics stored refused version=4$"));
            Assert.IsNull(NextStart(out _).Current);
        }

        [Test]
        public void OfflineThePreviousManifestStaysAndTheNextHelloTriesAgain()
        {
            var source = new ServedSource(folder);
            var manifest = Builtin();
            var updater = new CosmeticManifestUpdater(source, manifest);
            LogAssert.Expect(LogType.Warning, new Regex("^Cosmetics fetch failed version=4$"));
            updater.Fetch(Cdn, 4, Hash(Fourth));
            Assert.IsFalse(manifest.ApplyStaged(false));
            source.Serve(Hash(Fourth), Fourth);
            updater.Fetch(Cdn, 4, Hash(Fourth));
            Assert.AreEqual(2, source.Downloads.Count);
            Assert.IsTrue(manifest.ApplyStaged(false));
            Assert.AreEqual(4, manifest.Current.ManifestVersion);
        }

        [Test]
        public void NoContentRootOrHashDownloadsNothing()
        {
            var source = new ServedSource(folder);
            var updater = new CosmeticManifestUpdater(source, Builtin());
            updater.Fetch("", 4, Hash(Fourth));
            updater.Fetch(Cdn, 4, "");
            CollectionAssert.IsEmpty(source.Downloads);
        }

        [Test]
        public void TheBuiltinManifestShipsAsAResource()
        {
            var builtin = CosmeticManifest.LoadBuiltin();
            Assert.IsNotNull(builtin);
            Assert.IsNotNull(builtin.DefaultOf(CosmeticAddresses.Avatar));
        }
    }
}
