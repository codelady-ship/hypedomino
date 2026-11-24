using System;
using System.Collections.Generic;
using System.Text;
using System.Text.RegularExpressions;
using Google.Protobuf;
using HypeDomino.Content;
using HypeDomino.Game;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Audio;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Localization;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.TestTools;
using AnnouncedBundle = Hypedomino.V1.StringBundle;
using DownloadedBundle = HypeDomino.UI.Localization.StringBundle;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AppBootTests
    {
        const int ChosenRate = 120;
        const int MenuRate = 60;
        const string Cdn = "https://cdn.example.com";
        const string Version = "2026.10.26-1";
        const ulong Nonce = 0x5EED_CAFE_F00D_0001;
        const string Builtin = @"{""entries"":[{""key"":""k"",""en"":""E"",""tr"":""T""}]}";
        const string AzContent = @"{""language"":""az"",""strings"":{""k"":""A""}}";
        const string RuContent = @"{""language"":""ru"",""strings"":{""k"":""R""}}";
        const string EnHash = "e0";
        const string ManifestHash = "c9";

        static readonly string[] NoLocales = { };
        static readonly Regex MissingK = new Regex("^Strings missing key=k language=az version=0$");

        sealed class IdleCatalogSource : ICatalogSource
        {
            public List<string> Downloads { get; } = new();

            public string StoredVersion => null;

            public string StoredBaseUrl => null;

            public void LoadStored(Action<bool> done) { }

            public void Download(string url, Action<bool> done) => Downloads.Add(url);

            public void Commit(string version, string baseUrl, Action<bool> done) { }
        }

        sealed class IdleStringSource : IStringBundleSource
        {
            public List<string> Downloads { get; } = new();

            public string KeptLanguage { get; set; }

            public int KeptReads { get; private set; }

            public void Download(string url, Action<byte[]> done) => Downloads.Add(url);

            public bool TryLoadStored(string language, out int version, out string hash, out byte[] content)
            {
                KeptReads++;
                var kept = language == KeptLanguage;
                version = kept ? 2 : 0;
                hash = kept ? RuHash : null;
                content = kept ? Encoding.UTF8.GetBytes(RuContent) : null;
                return kept;
            }

            public void Store(string language, int version, string hash, byte[] content) { }
        }

        sealed class IdleCosmeticSource : ICosmeticManifestSource
        {
            public List<string> Downloads { get; } = new();

            public void Download(string url, Action<byte[]> done) => Downloads.Add(url);

            public bool TryLoadStored(out int version, out string hash, out byte[] content)
            {
                version = 0;
                hash = null;
                content = null;
                return false;
            }

            public void Store(int version, string hash, byte[] content) { }
        }

        sealed class ServerFrames : IFrameTransport
        {
            readonly List<byte[]> incoming = new(4);
            int next;

            public void Send(ReadOnlySpan<byte> frame) { }

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                if (next == incoming.Count)
                {
                    incoming.Clear();
                    next = 0;
                    return false;
                }
                frame = incoming[next++];
                return true;
            }

            public void Deliver(byte[] frame) => incoming.Add(frame);
        }

        sealed class Listened
        {
            public readonly ServerFrames Server = new();
            public readonly Session Session = new();
            public readonly AppFlow Flow = new();
            public readonly IdleCatalogSource Source = new();
            public readonly FrameRatePolicy FrameRate = AppBootTests.FrameRate();
            public readonly LanguageChoice Choice = LanguageChoice.Load("az", () => NoLocales);
            public readonly IdleStringSource StringSource = new();
            public readonly IdleCosmeticSource CosmeticSource = new();
            public readonly GameStrings Strings;
            public readonly CatalogUpdater Catalog;
            public HapticCue Felt;
            public MatchSounds Heard;

            public Listened()
            {
                Strings = new GameStrings(BuiltinStrings.Parse(Builtin), Choice, false);
                Catalog = new CatalogUpdater(Source, "Android");
                Flow.Changed += state => AppBoot.FollowMatch(state, FrameRate, Catalog);
                AppBoot.Listen(Session, Flow, Catalog, Strings, new StringBundleUpdater(StringSource, Strings), new CosmeticManifestUpdater(CosmeticSource, new ActiveCosmeticManifest(null)), cue => Felt = cue, sounds => Heard = sounds);
                Session.Hello(Server, new ClientHello());
            }

            public void Receive(Envelope.PayloadOneofCase payload, IMessage message)
            {
                Server.Deliver(FrameOf(payload, message));
                Session.Poll();
            }
        }

        static byte[] FrameOf(Envelope.PayloadOneofCase payload, IMessage message) =>
            new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, Nonce), payload, message).ToArray();

        static ServerHello Hello(HelloStatus status, bool inMatch) =>
            new ServerHello { Status = status, InMatch = inMatch, SessionNonce = Nonce, CatalogVersion = Version, ContentBaseUrl = Cdn };

        static MatchView View(MatchState state) => new MatchView { State = state };

        static MatchView Seated(Seat current, int boardTiles)
        {
            var view = new MatchView { State = MatchState.Playing, Seat = Seat.South, Mode = Mode._2V2, Current = current, Board = new BoardView() };
            for (uint tile = 0; tile < boardTiles; tile++) view.Board.Line.Add(new OrientedTile { First = tile, Second = tile });
            return view;
        }

        static string CatalogUrl => ContentUrls.Catalog(Cdn, "Android", Version);

        static FrameRatePolicy FrameRate() =>
            new FrameRatePolicy(new GraphicsQuality(DeviceTier.High, GraphicsLevel.High, 0, false), new[] { 60, 120 }, ChosenRate, false);

        static CatalogUpdater Catalog() => new CatalogUpdater(new IdleCatalogSource(), "Android");

        static GameStrings Strings(string language) =>
            new GameStrings(BuiltinStrings.Parse(Builtin), LanguageChoice.Load(language, () => NoLocales), false);

        static string AzHash => DownloadedBundle.Sha256Hex(Encoding.UTF8.GetBytes(AzContent));

        static string RuHash => DownloadedBundle.Sha256Hex(Encoding.UTF8.GetBytes(RuContent));

        static string StringsUrl(string language, string hash) => ContentUrls.Strings(Cdn, language, hash);

        static ServerHello Announcing(HelloStatus status, params FallbackChain[] chains)
        {
            var hello = Hello(status, false);
            hello.StringBundles.Add(new AnnouncedBundle { Language = "az", Version = 3, Hash = AzHash });
            hello.StringBundles.Add(new AnnouncedBundle { Language = "en", Version = 4, Hash = EnHash });
            hello.StringBundles.Add(new AnnouncedBundle { Language = "ru", Version = 2, Hash = RuHash });
            hello.FallbackChains.Add(chains);
            hello.CosmeticsManifest = new CosmeticsManifest { Version = 9, Hash = ManifestHash };
            return hello;
        }

        static FallbackChain Chain(string language, params string[] languages)
        {
            var chain = new FallbackChain { Language = language };
            chain.Languages.Add(languages);
            return chain;
        }

        static string[] LanguagesOf(List<AnnouncedBundle> bundles)
        {
            var languages = new string[bundles.Count];
            for (var i = 0; i < languages.Length; i++) languages[i] = bundles[i].Language;
            return languages;
        }

        [TestCase(AppState.Boot, false)]
        [TestCase(AppState.Login, false)]
        [TestCase(AppState.Lobby, false)]
        [TestCase(AppState.Match, true)]
        [TestCase(AppState.UpdateRequired, false)]
        [TestCase(AppState.Maintenance, false)]
        public void OnlyTheMatchStateCountsAsInMatch(AppState state, bool inMatch)
        {
            var frameRate = FrameRate();
            var catalog = Catalog();
            AppBoot.FollowMatch(state, frameRate, catalog);
            Assert.AreEqual(inMatch, frameRate.InMatch);
            Assert.AreEqual(inMatch, catalog.InMatch);
        }

        [Test]
        public void TheChosenRateAndTheCatalogHoldLastOnlyWhileInTheMatch()
        {
            var frameRate = FrameRate();
            var catalog = Catalog();
            AppBoot.FollowMatch(AppState.Match, frameRate, catalog);
            Assert.AreEqual(ChosenRate, frameRate.Target);
            AppBoot.FollowMatch(AppState.Lobby, frameRate, catalog);
            Assert.AreEqual(MenuRate, frameRate.Target);
            Assert.IsFalse(catalog.InMatch);
        }

        [Test]
        public void TheEditorHasNoCatalogAndStillFollowsTheMatch()
        {
            var frameRate = FrameRate();
            AppBoot.FollowMatch(AppState.Match, frameRate, null);
            Assert.IsTrue(frameRate.InMatch);
        }

        static GameStrings StagedAz()
        {
            var strings = Strings("az");
            Assert.IsTrue(strings.Stage("az", 3, AzHash, Encoding.UTF8.GetBytes(AzContent)));
            return strings;
        }

        [TestCase(AppState.Login)]
        [TestCase(AppState.Lobby)]
        [TestCase(AppState.Room)]
        [TestCase(AppState.UpdateRequired)]
        [TestCase(AppState.Maintenance)]
        public void ABundleDownloadedAfterStartIsAppliedAtTheNextStateOutsideTheMatch(AppState state)
        {
            var strings = StagedAz();
            var changes = 0;
            strings.Changed += () => changes++;
            AppBoot.ApplyStagedStrings(state, strings);
            Assert.AreEqual("A", strings.Text("k"));
            Assert.AreEqual(1, changes);
            Assert.IsFalse(strings.ApplyStaged(false));
        }

        [Test]
        public void ABundleDownloadedBeforeTheMatchWaitsUntilTheMatchIsLeft()
        {
            var strings = StagedAz();
            var changes = 0;
            strings.Changed += () => changes++;
            AppBoot.ApplyStagedStrings(AppState.Match, strings);
            LogAssert.Expect(LogType.Warning, MissingK);
            Assert.AreEqual("T", strings.Text("k"));
            Assert.AreEqual(0, changes);
            Assert.IsTrue(strings.Holds("az", AzHash));

            AppBoot.ApplyStagedStrings(AppState.Lobby, strings);
            Assert.AreEqual("A", strings.Text("k"));
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void AStateChangeWithNothingStagedLeavesTheStringsUnchanged()
        {
            var strings = Strings("az");
            var changes = 0;
            strings.Changed += () => changes++;
            AppBoot.ApplyStagedStrings(AppState.Lobby, strings);
            LogAssert.Expect(LogType.Warning, MissingK);
            Assert.AreEqual("T", strings.Text("k"));
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void AStateChangeBeforeTheStringsExistAppliesNothing() =>
            Assert.DoesNotThrow(() => AppBoot.ApplyStagedStrings(AppState.Lobby, null));

        [Test]
        public void AnAcceptedHelloFromTheSessionOpensTheLobbyAndFetchesTheAnnouncedCatalog()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, false));
            Assert.AreEqual(AppState.Lobby, app.Flow.State);
            CollectionAssert.AreEqual(new[] { CatalogUrl }, app.Source.Downloads);
            Assert.AreEqual(Cdn, app.Catalog.BaseUrl);
        }

        [Test]
        public void AHelloIntoARunningMatchHoldsTheAnnouncedCatalogUntilTheMatchIsLeft()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, true));
            Assert.AreEqual(AppState.Match, app.Flow.State);
            Assert.IsTrue(app.FrameRate.InMatch);
            CollectionAssert.IsEmpty(app.Source.Downloads);

            app.Receive(Envelope.PayloadOneofCase.MatchView, View(MatchState.Finished));
            Assert.IsTrue(app.Flow.LeaveMatch());
            CollectionAssert.AreEqual(new[] { CatalogUrl }, app.Source.Downloads);
        }

        [TestCase(HelloStatus.UpdateRequired, AppState.UpdateRequired)]
        [TestCase(HelloStatus.Maintenance, AppState.Maintenance)]
        public void AHelloThatBlocksPlayShowsItsScreenAndFetchesNoCatalog(HelloStatus status, AppState shown)
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(status, false));
            Assert.AreEqual(shown, app.Flow.State);
            CollectionAssert.IsEmpty(app.Source.Downloads);
            Assert.IsNull(app.Catalog.BaseUrl);
        }

        [Test]
        public void AMatchViewFromTheSessionTakesTheLobbyIntoTheMatch()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, false));
            app.Receive(Envelope.PayloadOneofCase.MatchView, View(MatchState.Loading));
            Assert.AreEqual(AppState.Match, app.Flow.State);
            Assert.IsTrue(app.Catalog.InMatch);
            Assert.IsFalse(app.Flow.LeaveMatch());
        }

        [Test]
        public void TheEditorWithoutACatalogStillFollowsTheHello()
        {
            var session = new Session();
            var flow = new AppFlow();
            var server = new ServerFrames();
            AppBoot.Listen(session, flow, null, Strings("az"), null, null, _ => { }, _ => { });
            session.Hello(server, new ClientHello());
            server.Deliver(FrameOf(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, false)));
            session.Poll();
            Assert.AreEqual(AppState.Lobby, flow.State);
        }

        [Test]
        public void TheBundlesToFetchAreTheChosenLanguageAndEnglishNotYetHeld()
        {
            var fetch = AppBoot.FollowStrings(Announcing(HelloStatus.Accepted), Strings("az"));
            CollectionAssert.AreEqual(new[] { "az", "en" }, LanguagesOf(fetch));
            Assert.AreEqual(3u, fetch[0].Version);
            Assert.AreEqual(AzHash, fetch[0].Hash);
            Assert.AreEqual(4u, fetch[1].Version);
            Assert.AreEqual(EnHash, fetch[1].Hash);
        }

        [Test]
        public void EnglishAloneIsFetchedWhenEnglishIsChosen() =>
            CollectionAssert.AreEqual(new[] { "en" }, LanguagesOf(AppBoot.FollowStrings(Announcing(HelloStatus.Accepted), Strings("en"))));

        [Test]
        public void ABundleWhoseHashIsAlreadyHeldIsNotFetchedAgain()
        {
            var strings = Strings("az");
            Assert.IsTrue(strings.Stage("az", 3, AzHash, Encoding.UTF8.GetBytes(AzContent)));
            CollectionAssert.AreEqual(new[] { "en" }, LanguagesOf(AppBoot.FollowStrings(Announcing(HelloStatus.Accepted), strings)));
            Assert.IsTrue(strings.ApplyStaged(false));
            CollectionAssert.AreEqual(new[] { "en" }, LanguagesOf(AppBoot.FollowStrings(Announcing(HelloStatus.Accepted), strings)));
        }

        [Test]
        public void AHelloWithoutBundlesFetchesNothing() =>
            CollectionAssert.IsEmpty(AppBoot.FollowStrings(Hello(HelloStatus.Accepted, false), Strings("az")));

        [Test]
        public void TheAnnouncedChainsReplaceTheDefaultOnes()
        {
            var strings = Strings("az");
            LogAssert.Expect(LogType.Warning, MissingK);
            Assert.AreEqual("T", strings.Text("k"));
            AppBoot.FollowStrings(Announcing(HelloStatus.Accepted, Chain("az", "az", "en")), strings);
            Assert.AreEqual("E", strings.Text("k"));
        }

        [Test]
        public void AChainNotEndingInEnglishIsRefusedAndLogged()
        {
            var strings = Strings("az");
            LogAssert.Expect(LogType.Warning, new Regex("^Strings chain refused language=az$"));
            AppBoot.FollowStrings(Announcing(HelloStatus.Accepted, Chain("az", "az", "ru")), strings);
            LogAssert.Expect(LogType.Warning, MissingK);
            Assert.AreEqual("T", strings.Text("k"));
        }

        [Test]
        public void AnAcceptedHelloFromTheSessionFollowsTheAnnouncedStrings()
        {
            var app = new Listened();
            LogAssert.Expect(LogType.Log, new Regex("^Strings fetch language=az version=3$"));
            LogAssert.Expect(LogType.Log, new Regex("^Strings fetch language=en version=4$"));
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Announcing(HelloStatus.Accepted, Chain("az", "az", "en")));
            CollectionAssert.AreEqual(new[] { ContentUrls.Strings(Cdn, "az", AzHash), ContentUrls.Strings(Cdn, "en", EnHash) }, app.StringSource.Downloads);
            CollectionAssert.AreEqual(new[] { ContentUrls.Cosmetics(Cdn, ManifestHash) }, app.CosmeticSource.Downloads);
            LogAssert.Expect(LogType.Warning, MissingK);
            Assert.AreEqual("E", app.Strings.Text("k"));
        }

        [Test]
        public void ALanguageChosenAfterTheHelloFetchesItsAnnouncedBundle()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Announcing(HelloStatus.Accepted));
            LogAssert.Expect(LogType.Log, new Regex("^Strings fetch language=ru version=2$"));
            Assert.IsTrue(app.Choice.Choose("ru"));
            CollectionAssert.AreEqual(new[] { StringsUrl("az", AzHash), StringsUrl("en", EnHash), StringsUrl("ru", RuHash) }, app.StringSource.Downloads);
        }

        [Test]
        public void ALanguageChosenBeforeAnyHelloFetchesNothing()
        {
            var app = new Listened();
            Assert.IsTrue(app.Choice.Choose("ru"));
            CollectionAssert.IsEmpty(app.StringSource.Downloads);
        }

        [Test]
        public void ALanguageTheHelloDidNotAnnounceFetchesNothing()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Announcing(HelloStatus.Accepted));
            Assert.IsTrue(app.Choice.Choose("tr"));
            Assert.AreEqual(2, app.StringSource.Downloads.Count);
        }

        [Test]
        public void ALanguageChosenAfterABlockingHelloFetchesNothing()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Announcing(HelloStatus.Maintenance));
            Assert.IsTrue(app.Choice.Choose("ru"));
            CollectionAssert.IsEmpty(app.StringSource.Downloads);
        }

        [Test]
        public void AKeptBundleOfTheChosenLanguageIsUsedInsteadOfAFetch()
        {
            var app = new Listened();
            app.StringSource.KeptLanguage = "ru";
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Announcing(HelloStatus.Accepted));
            Assert.IsTrue(app.Choice.Choose("ru"));
            Assert.AreEqual(2, app.StringSource.Downloads.Count);
            Assert.IsTrue(app.Strings.Holds("ru", RuHash));
            Assert.IsTrue(app.Strings.ApplyStaged(false));
            Assert.AreEqual("R", app.Strings.Text("k"));
        }

        [Test]
        public void AppliedBundlesInTheSameLanguageReadAndFetchNothingAgain()
        {
            var app = new Listened();
            app.StringSource.KeptLanguage = "ru";
            Assert.IsTrue(app.Choice.Choose("ru"));
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Announcing(HelloStatus.Accepted));
            CollectionAssert.AreEqual(new[] { StringsUrl("en", EnHash) }, app.StringSource.Downloads);
            Assert.IsTrue(app.Strings.ApplyStaged(false));
            Assert.AreEqual(1, app.StringSource.KeptReads);
            Assert.AreEqual(1, app.StringSource.Downloads.Count);
        }

        [Test]
        public void ARoomFromTheSessionTakesTheMemberBackToItAfterTheCustomMatch()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, false));
            app.Receive(Envelope.PayloadOneofCase.LobbyState, new LobbyState { Code = "K7QM3X", LeaderAccountId = "a", Phase = LobbyPhase.Match });
            Assert.AreEqual(AppState.Room, app.Flow.State);
            app.Receive(Envelope.PayloadOneofCase.MatchView, View(MatchState.Loading));
            Assert.AreEqual(AppState.Match, app.Flow.State);
            app.Receive(Envelope.PayloadOneofCase.MatchView, View(MatchState.Finished));
            app.Receive(Envelope.PayloadOneofCase.LobbyState, new LobbyState { Code = "K7QM3X", LeaderAccountId = "a", Phase = LobbyPhase.Lobby });
            Assert.AreEqual(AppState.Room, app.Flow.State);
        }

        [Test]
        public void AHelloThatBlocksPlayLeavesTheStringsAlone()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Announcing(HelloStatus.Maintenance, Chain("az", "az", "en")));
            CollectionAssert.IsEmpty(app.StringSource.Downloads);
            CollectionAssert.IsEmpty(app.CosmeticSource.Downloads);
            LogAssert.Expect(LogType.Warning, MissingK);
            Assert.AreEqual("T", app.Strings.Text("k"));
        }

        [Test]
        public void TheLocalPlayersPlacementInAViewFromTheSessionPlaysTheTileCue()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, true));
            app.Receive(Envelope.PayloadOneofCase.MatchView, Seated(Seat.South, 1));
            Assert.AreEqual(HapticCue.None, app.Felt);
            app.Receive(Envelope.PayloadOneofCase.MatchView, Seated(Seat.West, 2));
            Assert.AreEqual(HapticCue.TilePlace, app.Felt);
        }

        [Test]
        public void APlacementInAViewFromTheSessionPlaysTheTileSoundForAnySeat()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, true));
            app.Receive(Envelope.PayloadOneofCase.MatchView, Seated(Seat.South, 1));
            Assert.AreEqual(MatchSounds.None, app.Heard);
            app.Receive(Envelope.PayloadOneofCase.MatchView, Seated(Seat.North, 2));
            Assert.AreEqual(MatchSounds.TilePlace, app.Heard);
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, true));
            app.Receive(Envelope.PayloadOneofCase.MatchView, Seated(Seat.East, 3));
            Assert.AreEqual(MatchSounds.None, app.Heard, "first view after a reconnect");
        }

        [TestCase(AppState.Match, Music.Match)]
        [TestCase(AppState.Lobby, Music.Lobby)]
        [TestCase(AppState.Room, Music.Lobby)]
        [TestCase(AppState.Login, Music.Lobby)]
        [TestCase(AppState.Boot, Music.None)]
        [TestCase(AppState.UpdateRequired, Music.None)]
        [TestCase(AppState.Maintenance, Music.None)]
        public void EachScreenHasItsMusic(AppState state, Music music) => Assert.AreEqual(music, AudioBoot.MusicFor(state));

        [Test]
        public void TheFirstViewAfterAReconnectPlaysNothing()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, true));
            app.Receive(Envelope.PayloadOneofCase.MatchView, Seated(Seat.South, 1));
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, true));
            app.Receive(Envelope.PayloadOneofCase.MatchView, Seated(Seat.West, 2));
            Assert.AreEqual(HapticCue.None, app.Felt);
            app.Receive(Envelope.PayloadOneofCase.MatchView, Seated(Seat.South, 3));
            Assert.AreEqual(HapticCue.None, app.Felt, "West placed");
        }

        [Test]
        public void AMatchViewPassedToTheFlowAllocatesNothing()
        {
            var app = new Listened();
            app.Receive(Envelope.PayloadOneofCase.ServerHello, Hello(HelloStatus.Accepted, true));
            var view = FrameOf(Envelope.PayloadOneofCase.MatchView, View(MatchState.Playing));

            TestDelegate step = () =>
            {
                app.Server.Deliver(view);
                app.Session.Poll();
            };
            step();

            Assert.That(step, Allocates.Nothing);
            Assert.AreEqual(AppState.Match, app.Flow.State);
        }
    }
}
