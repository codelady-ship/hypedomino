using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using System.Text.RegularExpressions;
using HypeDomino.Content;
using HypeDomino.UI.Localization;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.TestTools;

namespace HypeDomino.Tests.EditMode
{
    public sealed class StringBundleUpdaterTests
    {
        const string Cdn = "https://cdn.example.com";
        const string Builtin = @"{""entries"":[{""key"":""k"",""en"":""E"",""az"":""B""}]}";
        const string AzContent = @"{""language"":""az"",""strings"":{""k"":""A""}}";
        const string NewerAzContent = @"{""language"":""az"",""strings"":{""k"":""A2""}}";
        const string EnContent = @"{""language"":""en"",""strings"":{""k"":""N""}}";
        const string RuContent = @"{""language"":""ru"",""strings"":{""k"":""R""}}";

        static readonly string[] NoLocales = { };

        sealed class ServedSource : IStringBundleSource
        {
            readonly UnityStringBundleSource disk;

            public ServedSource(string folder) => disk = new UnityStringBundleSource(folder);

            public Dictionary<string, byte[]> Served { get; } = new();

            public List<string> Downloads { get; } = new();

            public void Serve(string language, string hash, string content) => Served[ContentUrls.Strings(Cdn, language, hash)] = Encoding.UTF8.GetBytes(content);

            public void Download(string url, Action<byte[]> done)
            {
                Downloads.Add(url);
                done(Served.TryGetValue(url, out var content) ? content : null);
            }

            public bool TryLoadStored(string language, out int version, out string hash, out byte[] content) =>
                disk.TryLoadStored(language, out version, out hash, out content);

            public void Store(string language, int version, string hash, byte[] content) => disk.Store(language, version, hash, content);
        }

        string folder;

        [SetUp]
        public void CreateFolder() => folder = Path.Combine(Path.GetTempPath(), "hd-strings-" + Guid.NewGuid().ToString("N"));

        [TearDown]
        public void DeleteFolder()
        {
            if (Directory.Exists(folder)) Directory.Delete(folder, true);
        }

        static string Hash(string content) => StringBundle.Sha256Hex(Encoding.UTF8.GetBytes(content));

        static GameStrings Strings(string language) =>
            new GameStrings(BuiltinStrings.Parse(Builtin), LanguageChoice.Load(language, () => NoLocales), false);

        static StringBundleUpdater Updater(ServedSource source, GameStrings strings) => new StringBundleUpdater(source, strings);

        GameStrings NextStart(string language, out ServedSource source)
        {
            source = new ServedSource(folder);
            var strings = Strings(language);
            Updater(source, strings).LoadStored();
            return strings;
        }

        GameStrings FetchedOnce(string language, string content, int version)
        {
            var source = new ServedSource(folder);
            source.Serve(language, Hash(content), content);
            var strings = Strings(language);
            Updater(source, strings).Fetch(Cdn, language, version, Hash(content));
            return strings;
        }

        [Test]
        public void TheAnnouncedBundleIsDownloadedFromItsHashAndAppliedAtTheNextScreen()
        {
            var source = new ServedSource(folder);
            source.Serve("az", Hash(AzContent), AzContent);
            var strings = Strings("az");
            Updater(source, strings).Fetch(Cdn, "az", 3, Hash(AzContent));
            CollectionAssert.AreEqual(new[] { Cdn + "/strings/az/" + Hash(AzContent) + ".json" }, source.Downloads);
            Assert.IsTrue(strings.Holds("az", Hash(AzContent)));
            Assert.AreEqual("B", strings.Text("k"));
            Assert.IsFalse(strings.ApplyStaged(true));
            Assert.IsTrue(strings.ApplyStaged(false));
            Assert.AreEqual("A", strings.Text("k"));
        }

        [Test]
        public void ABundleWhoseContentDoesNotMatchTheAnnouncedHashIsRefusedAndNotKept()
        {
            var source = new ServedSource(folder);
            source.Serve("az", Hash(AzContent), NewerAzContent);
            var strings = Strings("az");
            LogAssert.Expect(LogType.Warning, new Regex("^Strings fetch refused language=az version=3$"));
            Updater(source, strings).Fetch(Cdn, "az", 3, Hash(AzContent));
            Assert.IsFalse(strings.Holds("az", Hash(AzContent)));
            Assert.IsFalse(strings.ApplyStaged(false));
            Assert.AreEqual("B", strings.Text("k"));
            Assert.IsFalse(Directory.Exists(Path.Combine(folder, "strings", "az")));
        }

        [Test]
        public void TheKeptBundleIsUsedOnTheNextStartWithoutAFetch()
        {
            FetchedOnce("az", AzContent, 3);
            FetchedOnce("en", EnContent, 4);
            var strings = NextStart("az", out var source);
            Assert.AreEqual("A", strings.Text("k"));
            Assert.IsTrue(strings.Holds("az", Hash(AzContent)));
            Assert.IsTrue(strings.Holds("en", Hash(EnContent)));
            Assert.IsFalse(strings.ApplyStaged(false));
            Updater(source, strings).Fetch(Cdn, "az", 3, Hash(AzContent));
            CollectionAssert.IsEmpty(source.Downloads);
        }

        [Test]
        public void ANewerBundleReplacesTheKeptOne()
        {
            FetchedOnce("az", AzContent, 3);
            FetchedOnce("az", NewerAzContent, 4);
            Assert.AreEqual(1, Directory.GetFiles(Path.Combine(folder, "strings", "az")).Length);
            Assert.AreEqual("A2", NextStart("az", out _).Text("k"));
        }

        [Test]
        public void ADamagedKeptBundleIsRefusedAndTheBuiltinStays()
        {
            FetchedOnce("az", AzContent, 3);
            var kept = Directory.GetFiles(Path.Combine(folder, "strings", "az"))[0];
            File.WriteAllText(kept, NewerAzContent);
            LogAssert.Expect(LogType.Warning, new Regex("^Strings stored refused language=az version=3$"));
            Assert.AreEqual("B", NextStart("az", out _).Text("k"));
        }

        [Test]
        public void ABundleNamingAnotherLanguageIsIgnored()
        {
            var source = new ServedSource(folder);
            source.Serve("az", Hash(RuContent), RuContent);
            var strings = Strings("az");
            LogAssert.Expect(LogType.Warning, new Regex("^Strings fetch refused language=az version=2$"));
            Updater(source, strings).Fetch(Cdn, "az", 2, Hash(RuContent));
            Assert.IsFalse(strings.ApplyStaged(false));
            Assert.AreEqual("B", strings.Text("k"));
        }

        [Test]
        public void AnUnknownLanguageIsNeverDownloaded()
        {
            var source = new ServedSource(folder);
            Updater(source, Strings("az")).Fetch(Cdn, "../de", 2, Hash(RuContent));
            CollectionAssert.IsEmpty(source.Downloads);
        }

        [Test]
        public void AKeptBundleOfAnotherLanguageIsNotLoaded()
        {
            FetchedOnce("ru", RuContent, 2);
            var strings = NextStart("az", out _);
            Assert.IsFalse(strings.Holds("ru", Hash(RuContent)));
            Assert.AreEqual("B", strings.Text("k"));
        }

        [Test]
        public void TheKeptBundleOfALanguageChosenLaterIsStagedForTheNextScreen()
        {
            FetchedOnce("ru", RuContent, 2);
            var source = new ServedSource(folder);
            var strings = Strings("az");
            var updater = Updater(source, strings);
            updater.LoadStored();
            updater.Restore("ru");
            Assert.IsTrue(strings.Holds("ru", Hash(RuContent)));
            Assert.IsTrue(strings.ApplyStaged(false));
            CollectionAssert.IsEmpty(source.Downloads);
        }

        [Test]
        public void AKeptBundleAlreadyHeldIsNotStagedAgain()
        {
            FetchedOnce("az", AzContent, 3);
            var source = new ServedSource(folder);
            var strings = Strings("az");
            var updater = Updater(source, strings);
            updater.LoadStored();
            updater.Restore("az");
            Assert.IsTrue(strings.Holds("az", Hash(AzContent)));
            Assert.IsFalse(strings.ApplyStaged(false));
        }

        [Test]
        public void OfflineTheBuiltinStaysAndTheNextHelloTriesAgain()
        {
            var source = new ServedSource(folder);
            var strings = Strings("az");
            var updater = Updater(source, strings);
            LogAssert.Expect(LogType.Warning, new Regex("^Strings fetch failed language=az version=3$"));
            updater.Fetch(Cdn, "az", 3, Hash(AzContent));
            Assert.IsFalse(strings.ApplyStaged(false));
            Assert.AreEqual("B", strings.Text("k"));
            Assert.AreEqual("B", NextStart("az", out _).Text("k"));
            source.Serve("az", Hash(AzContent), AzContent);
            updater.Fetch(Cdn, "az", 3, Hash(AzContent));
            Assert.AreEqual(2, source.Downloads.Count);
            Assert.IsTrue(strings.ApplyStaged(false));
            Assert.AreEqual("A", strings.Text("k"));
        }
    }
}
