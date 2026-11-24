using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchLoadingTests
    {
        const string NeonTable = "cosmetic/table/neon/static";
        const string FoxAvatar = "cosmetic/avatar/fox/static";
        const string Catalog = @"{""schemaVersion"":1,""manifestVersion"":1,""items"":[
{""id"":""avatar.fox"",""type"":""avatar"",""rarityId"":""epic"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/fox/static""},
{""id"":""table.neon"",""type"":""table"",""rarityId"":""epic"",""availability"":""live"",""staticAddress"":""cosmetic/table/neon/static""}]}";

        sealed class ShownScreen : ILoadingScreen
        {
            public int CardsShown;
            public int Closed;
            public LoadingStatus FirstStatus;
            public LoadingStatus SecondStatus;
            public float Progress;

            public void ShowCards(MatchSnapshot view, LoadingCards cards)
            {
                CardsShown++;
                FirstStatus = cards[0].Status;
                SecondStatus = cards[1].Status;
                Progress = cards.Progress;
            }

            public void Close() => Closed++;
        }

        sealed class Source : ICosmeticArtSource<Sprite>
        {
            readonly Dictionary<string, Action<Sprite>> pending = new();

            public readonly List<string> Loads = new();
            public readonly List<string> Releases = new();

            public void Load(string address, Action<Sprite> done)
            {
                Loads.Add(address);
                pending[address] = done;
            }

            public void Release(string address) => Releases.Add(address);

            public Sprite Art;

            public void Finish(string address)
            {
                var done = pending[address];
                pending.Remove(address);
                done(Art);
            }
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

        ShownScreen screen;
        Source source;
        ServerFrames frames;
        Session session;
        MatchLoading loading;
        CosmeticManifest manifest;
        Texture2D texture;

        [SetUp]
        public void CreateLoading()
        {
            screen = new ShownScreen();
            texture = new Texture2D(4, 4);
            source = new Source { Art = Sprite.Create(texture, new Rect(0, 0, 4, 4), Vector2.zero) };
            frames = new ServerFrames();
            session = new Session();
            Assert.IsTrue(CosmeticManifest.TryParse(Catalog, out manifest));
            loading = new MatchLoading(screen, new MatchPrefetch(new CosmeticArtCache<Sprite>(source)), new ActiveCosmeticManifest(manifest));
            loading.Bind(session);
            session.Hello(frames, new ClientHello());
            frames.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = 1 });
            session.Poll();
        }

        [TearDown]
        public void UnbindLoading()
        {
            loading.Unbind();
            Object.DestroyImmediate(source.Art);
            Object.DestroyImmediate(texture);
        }

        static MatchView View(MatchState state, bool opponentReady = false)
        {
            var view = new MatchView { State = state, Mode = Mode._1V1, Seat = Seat.X, Table = "table.neon" };
            view.Roster.Add(new RosterEntry { Seat = Seat.X, Avatar = "avatar.fox" });
            view.Roster.Add(new RosterEntry { Seat = Seat.Y, Avatar = "avatar.fox" });
            view.Seats.Add(new SeatView { Seat = Seat.X, LoadingProgress = 40 });
            view.Seats.Add(new SeatView { Seat = Seat.Y, Ready = opponentReady });
            return view;
        }

        void Receive(MatchView view)
        {
            frames.Deliver(Envelope.PayloadOneofCase.MatchView, view);
            session.Poll();
        }

        void TickNotReady() => loading.Tick(false);

        [Test]
        public void ALoadingViewOpensTheScreenAndPrefetchesTheMatchArtOnce()
        {
            Receive(View(MatchState.Loading));
            Receive(View(MatchState.Loading));

            Assert.IsTrue(loading.Open);
            Assert.AreEqual(2, screen.CardsShown);
            Assert.AreEqual(LoadingStatus.Loading, screen.FirstStatus);
            Assert.AreEqual(0.2f, screen.Progress, 1e-6f);
            CollectionAssert.AreEqual(new[] { NeonTable, FoxAvatar }, source.Loads);
        }

        [Test]
        public void AViewPastTheLoadingPhaseWithoutAnOpenScreenShowsNothing()
        {
            Receive(View(MatchState.Playing));

            Assert.IsFalse(loading.Open);
            Assert.AreEqual(0, screen.CardsShown);
            Assert.AreEqual(0, screen.Closed);
            Assert.IsEmpty(source.Loads);
        }

        [Test]
        public void TheScreenClosesWhenTheServerLeavesTheLoadingState()
        {
            Receive(View(MatchState.Loading));

            Receive(View(MatchState.Opening));
            Receive(View(MatchState.Playing));

            Assert.IsFalse(loading.Open);
            Assert.AreEqual(1, screen.Closed);
            Assert.AreEqual(2, screen.CardsShown);
            Assert.AreEqual(LoadingStatus.BotPlaying, screen.FirstStatus);
        }

        [Test]
        public void AReadyOpponentShowsReadyWhileTheOwnSeatLoads()
        {
            Receive(View(MatchState.Loading, opponentReady: true));

            Assert.AreEqual(LoadingStatus.Loading, screen.FirstStatus);
            Assert.AreEqual(LoadingStatus.Ready, screen.SecondStatus);
        }

        [Test]
        public void ProgressIsReportedOnlyOnANewStep()
        {
            Receive(View(MatchState.Loading));

            loading.Tick(false);
            loading.Tick(false);
            source.Finish(NeonTable);
            loading.Tick(false);
            loading.Tick(false);

            Assert.AreEqual(2, frames.Reports.Count);
            Assert.AreEqual(0u, frames.Reports[0].Progress);
            Assert.AreEqual(50u, frames.Reports[1].Progress);
            Assert.IsFalse(frames.Reports[1].Ready);
        }

        [Test]
        public void ReadyWaitsForTheMatchSceneEvenWhenEverythingIsLoaded()
        {
            Receive(View(MatchState.Loading));
            source.Finish(NeonTable);
            source.Finish(FoxAvatar);

            loading.Tick(false);
            Assert.AreEqual(90u, frames.Reports[^1].Progress);
            Assert.IsFalse(frames.Reports[^1].Ready);

            loading.Tick(true);
            loading.Tick(true);

            Assert.AreEqual(2, frames.Reports.Count);
            Assert.AreEqual(100u, frames.Reports[1].Progress);
            Assert.IsTrue(frames.Reports[1].Ready);
        }

        [Test]
        public void ReadyIsNotSentBeforeTheArtIsLoaded()
        {
            Receive(View(MatchState.Loading));
            source.Finish(NeonTable);

            loading.Tick(true);

            Assert.IsFalse(frames.Reports[^1].Ready);
        }

        [Test]
        public void NothingIsReportedAfterTheScreenCloses()
        {
            Receive(View(MatchState.Loading));
            Receive(View(MatchState.Opening));

            loading.Tick(true);

            Assert.IsEmpty(frames.Reports);
        }

        [Test]
        public void NothingIsReportedWithoutAnAcceptedSession()
        {
            var unbound = new MatchLoading(screen, new MatchPrefetch(new CosmeticArtCache<Sprite>(source)), new ActiveCosmeticManifest(manifest));
            var snapshot = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(View(MatchState.Loading).ToByteArray()));
            unbound.Show(snapshot);

            unbound.Tick(true);

            Assert.IsEmpty(frames.Reports);
        }

        [Test]
        public void ANewMatchReportsFromTheStartAgain()
        {
            Receive(View(MatchState.Loading));
            source.Finish(NeonTable);
            source.Finish(FoxAvatar);
            loading.Tick(true);
            Receive(View(MatchState.Opening));

            Receive(View(MatchState.Loading));
            loading.Tick(false);

            Assert.AreEqual(2, frames.Reports.Count);
            Assert.AreEqual(0u, frames.Reports[1].Progress);
            CollectionAssert.AreEqual(new[] { NeonTable, FoxAvatar }, source.Releases);
        }

        [Test]
        public void ReleaseArtFreesThePrefetchedArt()
        {
            Receive(View(MatchState.Loading));
            source.Finish(NeonTable);
            source.Finish(FoxAvatar);
            Receive(View(MatchState.Opening));

            loading.ReleaseArt();

            CollectionAssert.AreEqual(new[] { NeonTable, FoxAvatar }, source.Releases);
        }

        [Test]
        public void TickingWhileLoadingAllocatesNothing()
        {
            Receive(View(MatchState.Loading));
            loading.Tick(false);

            Assert.That(TickNotReady, Allocates.Nothing);
        }
    }
}
