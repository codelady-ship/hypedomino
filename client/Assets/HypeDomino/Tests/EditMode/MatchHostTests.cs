using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Emoji;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Match;
using HypeDomino.UI.Motion;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchHostTests
    {
        const ulong ServerNowMs = 1_000_000;
        const string NeonTable = "cosmetic/table/neon/static";
        const string FoxAvatar = "cosmetic/avatar/fox/static";
        const string Builtin = @"{""entries"":[]}";
        static readonly string[] NoLocales = Array.Empty<string>();
        const string Catalog = @"{""schemaVersion"":1,""manifestVersion"":1,""items"":[
{""id"":""avatar.fox"",""type"":""avatar"",""rarityId"":""epic"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/fox/static""},
{""id"":""table.neon"",""type"":""table"",""rarityId"":""epic"",""availability"":""live"",""staticAddress"":""cosmetic/table/neon/static""}]}";

        sealed class Table : IMatchTable
        {
            public Vector2 BoardSize => new(400f, 300f);
            public float BoardMaxScale => 1f;
            public int BackgroundsShown;

            public void ShowBackground(MatchSnapshot view) => BackgroundsShown++;

            public void ShowSeats(MatchSnapshot view)
            {
            }

            public void ShowScore(ScoreBoard score)
            {
            }

            public void ShowBoard(ChainLayout board)
            {
            }
        }

        sealed class LoadingCardsScreen : ILoadingScreen
        {
            public int CardsShown;
            public int Closed;

            public void ShowCards(MatchSnapshot view, LoadingCards cards) => CardsShown++;

            public void Close() => Closed++;
        }

        sealed class Source : ICosmeticArtSource<Sprite>
        {
            readonly Dictionary<string, Action<Sprite>> pending = new();

            public readonly List<string> Releases = new();
            public Sprite Art;

            public void Load(string address, Action<Sprite> done) => pending[address] = done;

            public void Release(string address) => Releases.Add(address);

            public void FinishAll()
            {
                var done = new List<Action<Sprite>>(pending.Values);
                pending.Clear();
                foreach (var finish in done) finish(Art);
            }
        }

        sealed class NoLoops : ICosmeticArtSource<LoopArt>
        {
            public void Load(string address, Action<LoopArt> done)
            {
            }

            public void Release(string address)
            {
            }
        }

        sealed class Views : IMatchViews
        {
            public bool Ready { get; set; } = true;
            public IMatchTable Table { get; set; }
            public ILoadingScreen LoadingScreen { get; set; }
            public CosmeticArtCache<Sprite> Stills { get; set; }
            public MatchEmoji Emoji { get; set; }
        }

        sealed class ServerFrames : IFrameTransport
        {
            readonly Queue<byte[]> pending = new();

            public readonly List<LoadingIntent> Reports = new();

            public void Send(ReadOnlySpan<byte> frame)
            {
                if (!FrameReader.TryRead(frame, out var read) || read.Payload != Envelope.PayloadOneofCase.LoadingIntent) return;
                Reports.Add(LoadingIntent.Parser.ParseFrom(read.PayloadBytes));
            }

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                var has = pending.Count > 0;
                frame = has ? pending.Dequeue() : null;
                return has;
            }

            public void Deliver(Envelope.PayloadOneofCase payload, IMessage message) =>
                pending.Enqueue(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, 1), payload, message).ToArray());
        }

        Table table;
        LoadingCardsScreen loadingScreen;
        Source source;
        Views views;
        ServerFrames frames;
        Connection connection;
        Session session;
        AppFlow flow;
        MatchHost host;
        Texture2D texture;
        double now;

        [SetUp]
        public void CreateHost()
        {
            table = new Table();
            loadingScreen = new LoadingCardsScreen();
            texture = new Texture2D(4, 4);
            source = new Source { Art = Sprite.Create(texture, new Rect(0, 0, 4, 4), Vector2.zero) };
            Assert.IsTrue(CosmeticManifest.TryParse(Catalog, out var manifest));
            views = new Views { Table = table, LoadingScreen = loadingScreen, Stills = new CosmeticArtCache<Sprite>(source), Emoji = CreateEmoji(manifest) };
            frames = new ServerFrames();
            connection = new Connection(() => throw new InvalidOperationException(), () => new ClientHello(), () => 0d, new System.Random(1));
            session = connection.Session;
            flow = new AppFlow();
            session.HelloReceived += flow.Hello;
            session.ViewReceived += FollowView;
            host = new MatchHost(flow, connection, new ActiveCosmeticManifest(manifest));
            now = 0d;
            session.Hello(frames, new ClientHello());
            frames.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = 1, ServerTimeUnixMs = ServerNowMs });
            session.Poll();
        }

        [TearDown]
        public void DestroyArt()
        {
            host.Detach(views);
            Object.DestroyImmediate(source.Art);
            Object.DestroyImmediate(texture);
        }

        void FollowView(MatchSnapshot view) => flow.View(view.State);

        static MatchEmoji CreateEmoji(CosmeticManifest manifest)
        {
            var strings = new GameStrings(BuiltinStrings.Parse(Builtin), LanguageChoice.Load("az", () => NoLocales), false);
            var stage = new CosmeticStage(new Source(), new NoLoops(), null, null, null, EffectLevel.Medium, false);
            return new MatchEmoji(new EmojiWheel(0), new SeatMessages(), strings, new EmojiArtShelf(manifest, stage, IgnoreEmoji), () => 0);
        }

        static void IgnoreEmoji(TableSide side, LoopArt art)
        {
        }

        static MatchView View(MatchState state)
        {
            var view = new MatchView
            {
                State = state,
                Mode = Mode._1V1,
                Seat = Seat.X,
                Current = Seat.X,
                Table = "table.neon",
                TurnDeadlineUnixMs = ServerNowMs + 10_000,
                TurnSeconds = 15,
                TurnWarningSeconds = 5,
            };
            view.Roster.Add(new RosterEntry { Seat = Seat.X, Avatar = "avatar.fox" });
            view.Roster.Add(new RosterEntry { Seat = Seat.Y, Avatar = "avatar.fox" });
            view.Seats.Add(new SeatView { Seat = Seat.X });
            view.Seats.Add(new SeatView { Seat = Seat.Y });
            return view;
        }

        void Receive(MatchView view)
        {
            frames.Deliver(Envelope.PayloadOneofCase.MatchView, view);
            session.Poll();
        }

        void Tick() => host.Tick(now += 0.1d, false);

        void FinishMatchAndLeave()
        {
            Receive(View(MatchState.Finished));
            Assert.IsTrue(flow.LeaveMatch());
        }

        [Test]
        public void NothingIsBuiltOutsideAMatch()
        {
            host.Attach(views);

            Assert.AreEqual(AppState.Lobby, flow.State);
            Assert.IsFalse(host.Live);
        }

        [Test]
        public void WithoutViewsEnteringAMatchBuildsNothingAndATickDoesNothing()
        {
            Receive(View(MatchState.Loading));
            Tick();

            Assert.AreEqual(AppState.Match, flow.State);
            Assert.IsFalse(host.Live);
            Assert.IsEmpty(frames.Reports);
        }

        [Test]
        public void TheViewThatEntersTheMatchReachesTheNewRootOnce()
        {
            host.Attach(views);

            Receive(View(MatchState.Loading));

            Assert.IsTrue(host.Live);
            Assert.IsTrue(host.Root.Loading.Open);
            Assert.AreEqual(1, loadingScreen.CardsShown);
            Assert.AreEqual(1, table.BackgroundsShown);
            Assert.AreSame(host.Root.Screen.Hand, host.Hud.Hand);
        }

        [Test]
        public void LaterViewsReachTheRootThroughItsBinding()
        {
            host.Attach(views);
            Receive(View(MatchState.Loading));

            Receive(View(MatchState.Loading));

            Assert.AreEqual(2, loadingScreen.CardsShown);
            Assert.AreEqual(2, table.BackgroundsShown);
        }

        [Test]
        public void ViewsAttachedDuringAMatchShowTheCurrentView()
        {
            Receive(View(MatchState.Playing));

            host.Attach(views);

            Assert.IsTrue(host.Live);
            Assert.AreEqual(1, table.BackgroundsShown);
            Assert.AreEqual(0, loadingScreen.CardsShown);
        }

        [Test]
        public void ATickReportsLoadingWhenTheViewsAreReadyAndRunsTheHudTimer()
        {
            host.Attach(views);
            Receive(View(MatchState.Loading));
            source.FinishAll();

            Tick();

            Assert.AreEqual(1, frames.Reports.Count);
            Assert.IsTrue(frames.Reports[0].Ready);
            Assert.AreSame(host.Root.Screen.Timer, host.Hud.Timer);
        }

        [Test]
        public void ATickWaitsForTheViewsBeforeReportingReady()
        {
            views.Ready = false;
            host.Attach(views);
            Receive(View(MatchState.Loading));
            source.FinishAll();

            Tick();

            Assert.IsFalse(frames.Reports.Count > 0 && frames.Reports[frames.Reports.Count - 1].Ready);
        }

        [Test]
        public void LeavingTheMatchTearsTheRootDownAndReleasesTheLoadingArt()
        {
            host.Attach(views);
            Receive(View(MatchState.Loading));
            source.FinishAll();

            FinishMatchAndLeave();

            Assert.AreNotEqual(AppState.Match, flow.State);
            Assert.IsFalse(host.Live);
            Assert.IsNull(host.Hud.Hand);
            CollectionAssert.AreEquivalent(new[] { NeonTable, FoxAvatar }, source.Releases);
        }

        [Test]
        public void ViewsAfterLeavingReachNothing()
        {
            host.Attach(views);
            Receive(View(MatchState.Playing));
            FinishMatchAndLeave();
            var shown = table.BackgroundsShown;

            Receive(View(MatchState.Finished));

            Assert.AreEqual(shown, table.BackgroundsShown);
        }

        [Test]
        public void TheNextMatchBuildsAFreshRoot()
        {
            host.Attach(views);
            Receive(View(MatchState.Playing));
            var first = host.Root;
            FinishMatchAndLeave();

            Receive(View(MatchState.Loading));

            Assert.IsTrue(host.Live);
            Assert.AreNotSame(first, host.Root);
            Assert.IsTrue(host.Root.Loading.Open);
        }

        [Test]
        public void DetachingTheViewsTearsTheRootDown()
        {
            host.Attach(views);
            Receive(View(MatchState.Playing));

            host.Detach(views);
            Receive(View(MatchState.Playing));

            Assert.IsFalse(host.Live);
            Assert.AreEqual(1, table.BackgroundsShown);
        }

        [Test]
        public void DetachingOtherViewsKeepsTheRoot()
        {
            host.Attach(views);
            Receive(View(MatchState.Playing));

            host.Detach(new Views());

            Assert.IsTrue(host.Live);
        }

        [Test]
        public void TickingWhileLoadingAllocatesNothing()
        {
            host.Attach(views);
            Receive(View(MatchState.Loading));
            Tick();

            Assert.That(Tick, Allocates.Nothing);
        }

        [Test]
        public void TickingDuringPlayAllocatesNothing()
        {
            host.Attach(views);
            Receive(View(MatchState.Playing));
            Tick();

            Assert.That(Tick, Allocates.Nothing);
        }

        [Test]
        public void TickingWithoutAMatchAllocatesNothing()
        {
            host.Attach(views);

            Assert.That(Tick, Allocates.Nothing);
        }
    }
}
