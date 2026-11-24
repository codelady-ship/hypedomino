using System;
using System.Collections.Generic;
using HypeDomino.Content;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class CatalogUpdaterTests
    {
        const string Cdn = "https://content.example.com/production";

        sealed class FakeSource : ICatalogSource
        {
            public readonly List<string> Downloads = new();
            public readonly List<string> Commits = new();
            public Action<bool> PendingDownload;
            public Action<bool> PendingCommit;
            public Action<bool> PendingStored;

            public string StoredVersion { get; set; }

            public string StoredBaseUrl { get; set; }

            public void LoadStored(Action<bool> done) => PendingStored = done;

            public void Download(string url, Action<bool> done)
            {
                Downloads.Add(url);
                PendingDownload = done;
            }

            public void Commit(string version, string baseUrl, Action<bool> done)
            {
                Commits.Add(version + " " + baseUrl);
                PendingCommit = done;
            }

            public void FinishDownload(bool ok) => Take(ref PendingDownload)(ok);

            public void FinishCommit(bool ok) => Take(ref PendingCommit)(ok);

            static Action<bool> Take(ref Action<bool> pending)
            {
                var done = pending;
                pending = null;
                return done;
            }
        }

        FakeSource source;
        CatalogUpdater updater;

        [SetUp]
        public void Create()
        {
            source = new FakeSource();
            updater = new CatalogUpdater(source, "Android");
        }

        static string CatalogUrl(string version) => Cdn + "/addressables/Android/catalog_" + version + ".bin";

        void Install(string version)
        {
            updater.Announce(version, Cdn);
            source.FinishDownload(true);
            source.FinishCommit(true);
        }

        [Test]
        public void StartLoadsTheStoredCatalogWithoutTheNetwork()
        {
            source.StoredVersion = "v1";
            source.StoredBaseUrl = Cdn;
            updater = new CatalogUpdater(source, "Android");

            updater.Start();
            source.PendingStored(true);

            Assert.AreEqual("v1", updater.LoadedVersion);
            Assert.AreEqual(Cdn, updater.BaseUrl);
            Assert.IsEmpty(source.Downloads);
        }

        [Test]
        public void StartWithoutAStoredCatalogDoesNothing()
        {
            updater.Start();

            Assert.IsNull(source.PendingStored);
            Assert.IsNull(updater.LoadedVersion);
        }

        [Test]
        public void ANewVersionIsDownloadedFromTheAnnouncedRootAndCommitted()
        {
            updater.Announce("v2", Cdn);

            Assert.AreEqual(new[] { CatalogUrl("v2") }, source.Downloads);
            source.FinishDownload(true);
            Assert.AreEqual(new[] { "v2 " + Cdn }, source.Commits);
            Assert.IsNull(updater.LoadedVersion);

            source.FinishCommit(true);
            Assert.AreEqual("v2", updater.LoadedVersion);
            Assert.AreEqual(Cdn, updater.BaseUrl);
        }

        [Test]
        public void TheLoadedVersionIsNotDownloadedAgain()
        {
            Install("v2");

            updater.Announce("v2", Cdn);

            Assert.AreEqual(1, source.Downloads.Count);
        }

        [Test]
        public void AnEmptyVersionOrRootKeepsWhatIsLoaded()
        {
            updater.Announce("", Cdn);
            updater.Announce("v2", "");
            updater.Announce(null, null);

            Assert.IsEmpty(source.Downloads);
        }

        [Test]
        public void AVersionAnnouncedInAMatchWaitsForTheMatchToEnd()
        {
            updater.SetInMatch(true);
            updater.Announce("v2", Cdn);
            Assert.IsEmpty(source.Downloads);

            updater.SetInMatch(false);
            Assert.AreEqual(new[] { CatalogUrl("v2") }, source.Downloads);
        }

        [Test]
        public void AFailedDownloadIsRetriedAfterEachDelayThenGivenUp()
        {
            updater.Announce("v2", Cdn);
            foreach (var delayMs in new[] { 2000f, 8000f, 30000f })
            {
                source.FinishDownload(false);
                Assert.AreEqual(delayMs, updater.RetryInMs);
                var before = source.Downloads.Count;
                updater.Advance(delayMs - 1f);
                Assert.AreEqual(before, source.Downloads.Count);
                updater.Advance(1f);
                Assert.AreEqual(before + 1, source.Downloads.Count);
            }

            source.FinishDownload(false);
            updater.Advance(60000f);

            Assert.AreEqual(4, source.Downloads.Count);
            Assert.AreEqual(0f, updater.RetryInMs);
            Assert.IsNull(updater.LoadedVersion);
        }

        [Test]
        public void TheNextAnnouncementTriesAgainAfterGivingUp()
        {
            updater.Announce("v2", Cdn);
            for (var attempt = 0; attempt < 4; attempt++)
            {
                source.FinishDownload(false);
                updater.Advance(30000f);
            }

            updater.Announce("v2", Cdn);

            Assert.AreEqual(5, source.Downloads.Count);
        }

        [Test]
        public void AFailedCommitKeepsTheLoadedVersionAndRetries()
        {
            Install("v1");
            updater.Announce("v2", Cdn);
            source.FinishDownload(true);

            source.FinishCommit(false);

            Assert.AreEqual("v1", updater.LoadedVersion);
            Assert.AreEqual(2000f, updater.RetryInMs);
        }

        [Test]
        public void ANewerVersionDuringADownloadReplacesIt()
        {
            updater.Announce("v2", Cdn);
            updater.Announce("v3", Cdn);
            Assert.AreEqual(1, source.Downloads.Count);

            source.FinishDownload(true);

            Assert.IsEmpty(source.Commits);
            Assert.AreEqual(new[] { CatalogUrl("v2"), CatalogUrl("v3") }, source.Downloads);
            source.FinishDownload(true);
            source.FinishCommit(true);
            Assert.AreEqual("v3", updater.LoadedVersion);
        }

        [Test]
        public void ARollbackDuringACommitIsLoadedAfterIt()
        {
            Install("v1");
            updater.Announce("v2", Cdn);
            source.FinishDownload(true);

            updater.Announce("v1", Cdn);
            source.FinishCommit(true);

            Assert.AreEqual("v2", updater.LoadedVersion);
            Assert.AreEqual(CatalogUrl("v1"), source.Downloads[source.Downloads.Count - 1]);
            source.FinishDownload(true);
            source.FinishCommit(true);
            Assert.AreEqual("v1", updater.LoadedVersion);
        }

        [Test]
        public void ANewRootIsUsedAtOnceForTheLoadedVersion()
        {
            Install("v1");

            updater.Announce("v1", "https://mirror.example.com");

            Assert.AreEqual("https://mirror.example.com", updater.BaseUrl);
            Assert.AreEqual(1, source.Downloads.Count);
        }
    }
}
