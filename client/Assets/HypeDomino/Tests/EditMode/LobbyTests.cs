using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game.Lobby;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.Net.Integrity;
using HypeDomino.UI.Shop;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LobbyTests
    {
        const ulong Nonce = 0x5EED_0074_0000_0002;
        const ulong JoinedMs = 1_800_000_000_000;
        const int Ranked1V1 = 0;
        const int Ranked2V2 = 1;
        const int Casual1V1 = 2;
        const int Casual2V2 = 3;
        const string WalletJson = "{\"coins\":1500,\"crystals\":-20,\"entries\":[]}";
        const string OwnJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":""Leyla"",""setup_required"":false,""rename_required"":false,""level"":7,""xp"":1250,""loadout"":{},""stats"":{""games"":40,""wins"":22,""losses"":18,""win_rate_percent"":55,""streak"":2,""best_streak"":6,""modes"":{}}}";
        const string SetupJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":"""",""setup_required"":true,""rename_required"":false,""level"":1,""xp"":0,""loadout"":{},""stats"":{""modes"":{}}}";

        sealed class FakeTransport : IFrameTransport
        {
            readonly Queue<byte[]> incoming = new();

            public List<byte[]> Sent { get; } = new();

            public void Send(ReadOnlySpan<byte> frame) => Sent.Add(frame.ToArray());

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                if (incoming.Count == 0) return false;
                frame = incoming.Dequeue();
                return true;
            }

            public void Deliver(Envelope.PayloadOneofCase payload, IMessage message) =>
                incoming.Enqueue(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, Nonce), payload, message).ToArray());
        }

        sealed class ProofDesk
        {
            public readonly List<IntegrityPurpose> Purposes = new();
            public readonly List<Action<IntegrityProof>> Answers = new();
            public bool AnswerAtOnce;

            public void Request(IntegrityPurpose purpose, Action<IntegrityProof> done)
            {
                Purposes.Add(purpose);
                if (AnswerAtOnce) done(null);
                else Answers.Add(done);
            }
        }

        sealed class Sent
        {
            public string Method;
            public string PathAndQuery;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, PathAndQuery = pathAndQuery, Done = done });

            public Sent To(string path) => Requests.Find(request => request.PathAndQuery == path);
        }

        sealed class Auth : IAuthTransport
        {
            public readonly List<Action<HttpReply>> Refreshes = new();

            public void Post(string path, string json, Action<HttpReply> done) => Refreshes.Add(done);
        }

        sealed class Identity : IIdentityProvider
        {
            public bool Supports(AuthProvider provider) => false;

            public void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done) => throw new NotSupportedException();
        }

        sealed class Store : ISecureStore
        {
            readonly Dictionary<string, string> values = new();

            public string Read(string key) => values.TryGetValue(key, out var value) ? value : null;

            public void Write(string key, string value) => values[key] = value;

            public void Delete(string key) => values.Remove(key);
        }

        FakeTransport transport;
        Session session;
        MatchQueue queue;
        ProofDesk proofs;
        LobbyPlay play;
        LobbyHeader header;
        LobbyScreen screen;
        int playChanges;

        [SetUp]
        public void Build()
        {
            playChanges = 0;
            transport = new FakeTransport();
            session = new Session();
            queue = new MatchQueue();
            proofs = new ProofDesk();
            play = new LobbyPlay(queue, proofs.Request);
            play.Changed += () => playChanges++;
            header = new LobbyHeader();
            screen = new LobbyScreen(header, play, queue);
        }

        void Accept()
        {
            session.Hello(transport, new ClientHello());
            transport.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = Nonce });
            session.Poll();
            Assert.IsTrue(session.Accepted);
            transport.Sent.Clear();
        }

        Envelope SentAt(int index) => Envelope.Parser.ParseFrom(transport.Sent[index]);

        void AssertCards(params PlayCardState[] states)
        {
            for (var card = 0; card < PlayModes.Count; card++) Assert.AreEqual(states[card], play.StateOf(card), $"card {card}");
        }

        static QueueState Searching(Mode mode, QueueKind kind, ulong searchStartsMs = JoinedMs) => new()
        {
            Status = QueueStatus.Searching,
            Mode = mode,
            Queue = kind,
            JoinedUnixMs = JoinedMs,
            SearchStartsUnixMs = searchStartsMs,
        };

        static QueueState Refused(QueueRefusal refusal, ulong untilMs = 0) => new()
        {
            Status = QueueStatus.Refused,
            Mode = Mode._1V1,
            Queue = QueueKind.Ranked,
            Refusal = refusal,
            RefusedUntilUnixMs = untilMs,
        };

        static ServerClock ClockAt(ulong serverMs)
        {
            var clock = new ServerClock();
            clock.Sync(serverMs, 0, 0);
            return clock;
        }

        static Balances Wallet(long coins, long crystals)
        {
            Assert.IsTrue(Balances.TryRead("{\"coins\":" + coins + ",\"crystals\":" + crystals + "}", out var balances));
            return balances;
        }

        [Test]
        public void TheCardsAreTheFourQueuesTheProtocolNames()
        {
            Assert.AreEqual(4, PlayModes.Count);
            Assert.IsTrue(PlayModes.At(Ranked1V1).Is(Mode._1V1, QueueKind.Ranked));
            Assert.IsTrue(PlayModes.At(Ranked2V2).Is(Mode._2V2, QueueKind.Ranked));
            Assert.IsTrue(PlayModes.At(Casual1V1).Is(Mode._1V1, QueueKind.Casual));
            Assert.IsTrue(PlayModes.At(Casual2V2).Is(Mode._2V2, QueueKind.Casual));
            Assert.AreEqual(IntegrityPurpose.RankedMatch, PlayModes.At(Ranked2V2).Purpose);
            Assert.AreEqual(IntegrityPurpose.CasualMatch, PlayModes.At(Casual1V1).Purpose);
            Assert.IsFalse(PlayModes.IsCard(-1));
            Assert.IsFalse(PlayModes.IsCard(4));
        }

        [Test]
        public void PlayProvesTheDeviceForTheCardsQueueThenJoins()
        {
            Accept();

            Assert.IsTrue(play.Play(session, Casual2V2));
            Assert.AreEqual(new[] { IntegrityPurpose.CasualMatch }, proofs.Purposes);
            Assert.IsEmpty(transport.Sent);
            Assert.IsTrue(play.ProvingDevice);
            Assert.IsTrue(play.CanCancel);
            AssertCards(PlayCardState.Locked, PlayCardState.Locked, PlayCardState.Locked, PlayCardState.Starting);

            proofs.Answers[0](new IntegrityProof { PlayIntegrityToken = "token" });

            Assert.IsFalse(play.ProvingDevice);
            Assert.AreEqual(1, transport.Sent.Count);
            var join = SentAt(0).JoinQueueIntent;
            Assert.AreEqual(Mode._2V2, join.Mode);
            Assert.AreEqual(QueueKind.Casual, join.Queue);
            Assert.AreEqual("token", join.Integrity.PlayIntegrityToken);
            AssertCards(PlayCardState.Locked, PlayCardState.Locked, PlayCardState.Locked, PlayCardState.Starting);
        }

        [Test]
        public void ARankedCardAsksForARankedProof()
        {
            Accept();
            proofs.AnswerAtOnce = true;

            Assert.IsTrue(play.Play(session, Ranked1V1));

            Assert.AreEqual(new[] { IntegrityPurpose.RankedMatch }, proofs.Purposes);
            var join = SentAt(0).JoinQueueIntent;
            Assert.AreEqual(Mode._1V1, join.Mode);
            Assert.AreEqual(QueueKind.Ranked, join.Queue);
            Assert.IsNull(join.Integrity);
        }

        [Test]
        public void NothingStartsBeforeAnAcceptedHelloOrForACardThatIsNotOffered()
        {
            Assert.IsFalse(play.Play(session, Ranked1V1));
            Accept();
            Assert.IsFalse(play.Play(session, -1));
            Assert.IsFalse(play.Play(session, PlayModes.Count));

            Assert.IsEmpty(proofs.Purposes);
            Assert.AreEqual(0, playChanges);
        }

        [Test]
        public void ASecondTapWhileTheDeviceIsProvedAsksNothing()
        {
            Accept();
            Assert.IsTrue(play.Play(session, Ranked1V1));

            Assert.IsFalse(play.Play(session, Ranked1V1));
            Assert.IsFalse(play.Play(session, Casual1V1));

            Assert.AreEqual(1, proofs.Purposes.Count);
        }

        [Test]
        public void NoTapWhileTheJoinWaitsForItsReply()
        {
            Accept();
            proofs.AnswerAtOnce = true;
            Assert.IsTrue(play.Play(session, Ranked1V1));

            Assert.IsFalse(play.Play(session, Casual1V1));
            Assert.AreEqual(1, transport.Sent.Count);
        }

        [Test]
        public void CancelWhileProvingDropsTheLateProof()
        {
            Accept();
            Assert.IsTrue(play.Play(session, Ranked1V1));

            Assert.IsTrue(play.Cancel(session));
            AssertCards(PlayCardState.Ready, PlayCardState.Ready, PlayCardState.Ready, PlayCardState.Ready);
            proofs.Answers[0](new IntegrityProof());

            Assert.IsEmpty(transport.Sent);
            Assert.IsFalse(play.CanCancel);
        }

        [Test]
        public void AProofOfAnEarlierTapNeverJoinsTheLaterCard()
        {
            Accept();
            Assert.IsTrue(play.Play(session, Ranked1V1));
            Assert.IsTrue(play.Cancel(session));
            Assert.IsTrue(play.Play(session, Casual2V2));

            proofs.Answers[0](new IntegrityProof { PlayIntegrityToken = "ranked" });
            Assert.IsEmpty(transport.Sent);
            proofs.Answers[1](new IntegrityProof { PlayIntegrityToken = "casual" });

            Assert.AreEqual(1, transport.Sent.Count);
            Assert.AreEqual(QueueKind.Casual, SentAt(0).JoinQueueIntent.Queue);
            Assert.AreEqual("casual", SentAt(0).JoinQueueIntent.Integrity.PlayIntegrityToken);
        }

        [Test]
        public void TheSearchingCardShowsItAndAnotherCardMovesTheEntry()
        {
            Accept();
            proofs.AnswerAtOnce = true;
            Assert.IsTrue(play.Play(session, Ranked2V2));

            queue.Read(Searching(Mode._2V2, QueueKind.Ranked));

            AssertCards(PlayCardState.Ready, PlayCardState.Searching, PlayCardState.Ready, PlayCardState.Ready);
            Assert.IsFalse(play.Play(session, Ranked2V2));
            Assert.IsTrue(play.Play(session, Casual1V1));
            Assert.AreEqual(2, transport.Sent.Count);
            Assert.AreEqual(QueueKind.Casual, SentAt(1).JoinQueueIntent.Queue);
            AssertCards(PlayCardState.Locked, PlayCardState.Locked, PlayCardState.Starting, PlayCardState.Locked);
        }

        [Test]
        public void CancelWhileSearchingLeavesOnceAndLocksUntilTheServerAnswers()
        {
            Accept();
            proofs.AnswerAtOnce = true;
            Assert.IsTrue(play.Play(session, Casual1V1));
            queue.Read(Searching(Mode._1V1, QueueKind.Casual));

            Assert.IsTrue(play.CanCancel);
            Assert.IsTrue(play.Cancel(session));
            Assert.IsFalse(play.Cancel(session));

            Assert.AreEqual(2, transport.Sent.Count);
            Assert.AreEqual(Envelope.PayloadOneofCase.LeaveQueueIntent, SentAt(1).PayloadCase);
            AssertCards(PlayCardState.Locked, PlayCardState.Locked, PlayCardState.Locked, PlayCardState.Locked);
            queue.Read(new QueueState { Status = QueueStatus.Left });
            AssertCards(PlayCardState.Ready, PlayCardState.Ready, PlayCardState.Ready, PlayCardState.Ready);
        }

        [Test]
        public void AFoundMatchLocksEveryCardUntilTheLobbyIsShownAgain()
        {
            Accept();
            proofs.AnswerAtOnce = true;
            Assert.IsTrue(play.Play(session, Ranked1V1));
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked));

            queue.Read(new QueueState { Status = QueueStatus.MatchFound });

            AssertCards(PlayCardState.Locked, PlayCardState.Locked, PlayCardState.Locked, PlayCardState.Locked);
            Assert.IsFalse(play.Play(session, Casual1V1));
            queue.Reset();
            AssertCards(PlayCardState.Ready, PlayCardState.Ready, PlayCardState.Ready, PlayCardState.Ready);
        }

        [Test]
        public void ARefusalFreesTheCardsAgain()
        {
            Accept();
            proofs.AnswerAtOnce = true;
            Assert.IsTrue(play.Play(session, Ranked1V1));

            queue.Read(Refused(QueueRefusal.RankedClosed));

            AssertCards(PlayCardState.Ready, PlayCardState.Ready, PlayCardState.Ready, PlayCardState.Ready);
            Assert.IsTrue(play.Play(session, Casual1V1));
        }

        [Test]
        public void EveryStepOfThePlayFlowIsRaised()
        {
            Accept();
            Assert.IsTrue(play.Play(session, Ranked1V1));
            Assert.AreEqual(1, playChanges);
            proofs.Answers[0](null);
            Assert.AreEqual(2, playChanges);
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked));
            Assert.AreEqual(3, playChanges);
            Assert.IsTrue(play.Cancel(session));
            Assert.AreEqual(4, playChanges);
        }

        [Test]
        public void TheHeaderRaisesOnlyWhatChanged()
        {
            var changes = 0;
            header.Changed += () => changes++;
            Assert.IsFalse(header.HasBalances);
            Assert.IsFalse(header.HasLevel);

            header.Show(Wallet(1500, -20));
            header.Show(Wallet(1500, -20));
            header.ShowLevel(7);
            header.ShowLevel(7);

            Assert.AreEqual(2, changes);
            Assert.AreEqual(1500, header.Balances.Coins.Amount);
            Assert.AreEqual(-20, header.Balances.Crystals.Amount);
            Assert.AreEqual(7, header.Level);
            header.Show(Wallet(1500, 30));
            header.ShowLevel(8);
            Assert.AreEqual(4, changes);
            header.Clear();
            header.Clear();
            Assert.AreEqual(5, changes);
            Assert.IsFalse(header.HasBalances);
            Assert.IsFalse(header.HasLevel);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("not json")]
        public void AnUnreadableWalletIsNotABalance(string body)
        {
            Assert.IsFalse(Balances.TryRead(body, out _));
        }

        sealed class HeaderLoad
        {
            public Api Api;
            public LobbyHeaderLoader Loader;
            public readonly List<LobbyHeaderLoadOutcome> Outcomes = new();
        }

        HeaderLoad SignedIn(bool signedIn = true)
        {
            var load = new HeaderLoad { Api = new Api() };
            var auth = new Auth();
            var store = new Store();
            var authSession = new AuthSession(auth, new Identity(), store);
            if (signedIn)
            {
                store.Write(AuthSession.RefreshTokenKey, "refresh-0");
                authSession.Resume(_ => { });
                auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            }
            load.Loader = new LobbyHeaderLoader(new ApiClient(load.Api, authSession), header);
            return load;
        }

        [Test]
        public void TheHeaderLoadsTheWalletAndTheOwnLevel()
        {
            var load = SignedIn();

            Assert.IsTrue(load.Loader.Load(load.Outcomes.Add));
            Assert.IsTrue(load.Loader.Busy);
            Assert.AreEqual(2, load.Api.Requests.Count);
            Assert.AreEqual("GET", load.Api.To("/wallet").Method);
            Assert.AreEqual("GET", load.Api.To("/profile/me").Method);

            load.Api.To("/wallet").Done(new HttpReply(200, WalletJson));
            Assert.IsTrue(header.HasBalances);
            Assert.AreEqual(1500, header.Balances.Coins.Amount);
            Assert.IsEmpty(load.Outcomes);
            load.Api.To("/profile/me").Done(new HttpReply(200, OwnJson));

            Assert.AreEqual(7, header.Level);
            Assert.IsFalse(load.Loader.Busy);
            Assert.AreEqual(new[] { LobbyHeaderLoadOutcome.Loaded }, load.Outcomes);
        }

        [Test]
        public void ASecondHeaderLoadWhileOneIsInFlightSendsNothing()
        {
            var load = SignedIn();
            Assert.IsTrue(load.Loader.Load(load.Outcomes.Add));
            Assert.IsFalse(load.Loader.Load(load.Outcomes.Add));
            load.Api.To("/wallet").Done(new HttpReply(200, WalletJson));
            Assert.IsFalse(load.Loader.Load(load.Outcomes.Add));
            load.Api.To("/profile/me").Done(new HttpReply(200, OwnJson));

            Assert.IsTrue(load.Loader.Load(load.Outcomes.Add));
            Assert.AreEqual(4, load.Api.Requests.Count);
        }

        [Test]
        public void OnlyTheFirstFailureIsAnswered()
        {
            var load = SignedIn();
            Assert.IsTrue(load.Loader.Load(load.Outcomes.Add));

            load.Api.To("/profile/me").Done(new HttpReply(HttpReply.NoResponse, null));
            load.Api.To("/wallet").Done(new HttpReply(429, "{\"error\":\"rate_limited\"}"));

            Assert.AreEqual(new[] { LobbyHeaderLoadOutcome.Offline }, load.Outcomes);
            Assert.IsFalse(header.HasBalances);
            Assert.IsFalse(header.HasLevel);
        }

        [TestCase(500, "{}", LobbyHeaderLoadOutcome.Failed)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", LobbyHeaderLoadOutcome.RateLimited)]
        [TestCase(200, "not json", LobbyHeaderLoadOutcome.Failed)]
        public void AWalletFailureKeepsTheLevel(int status, string body, LobbyHeaderLoadOutcome outcome)
        {
            var load = SignedIn();
            Assert.IsTrue(load.Loader.Load(load.Outcomes.Add));

            load.Api.To("/wallet").Done(new HttpReply(status, body));
            load.Api.To("/profile/me").Done(new HttpReply(200, OwnJson));

            Assert.AreEqual(new[] { outcome }, load.Outcomes);
            Assert.IsFalse(header.HasBalances);
            Assert.AreEqual(7, header.Level);
        }

        [TestCase(SetupJson, LobbyHeaderLoadOutcome.SetupRequired)]
        [TestCase("{}", LobbyHeaderLoadOutcome.Failed)]
        public void AProfileThatDoesNotReadShowsNoLevel(string body, LobbyHeaderLoadOutcome outcome)
        {
            var load = SignedIn();
            Assert.IsTrue(load.Loader.Load(load.Outcomes.Add));

            load.Api.To("/profile/me").Done(new HttpReply(200, body));
            load.Api.To("/wallet").Done(new HttpReply(200, WalletJson));

            Assert.AreEqual(new[] { outcome }, load.Outcomes);
            Assert.IsFalse(header.HasLevel);
            Assert.IsTrue(header.HasBalances);
        }

        [Test]
        public void WithoutASessionTheHeaderLoadAsksForSignIn()
        {
            var load = SignedIn(false);

            Assert.IsTrue(load.Loader.Load(load.Outcomes.Add));

            Assert.IsEmpty(load.Api.Requests);
            Assert.IsFalse(load.Loader.Busy);
            Assert.AreEqual(new[] { LobbyHeaderLoadOutcome.SignInRequired }, load.Outcomes);
        }

        [Test]
        public void TheFirstTickShowsEverythingAndAnIdleTickNothing()
        {
            var clock = ClockAt(JoinedMs);

            Assert.AreEqual(LobbyChanges.All, screen.Tick(clock, 0));
            Assert.AreEqual(LobbyChanges.None, screen.Tick(clock, 1));
            Assert.AreEqual(LobbyChanges.None, screen.Tick(clock, 2));
            Assert.AreEqual(QueueCountdown.None, screen.Countdown);
            screen.ShowAll();
            Assert.AreEqual(LobbyChanges.All, screen.Tick(clock, 3));
        }

        [Test]
        public void HeaderAndPlayChangesReachTheNextTick()
        {
            var clock = ClockAt(JoinedMs);
            screen.Tick(clock, 0);

            header.ShowLevel(3);
            Assert.AreEqual(LobbyChanges.Header, screen.Tick(clock, 0));
            queue.Read(Refused(QueueRefusal.Unavailable));
            Assert.AreEqual(LobbyChanges.Play, screen.Tick(clock, 0));
            Assert.AreEqual(LobbyChanges.None, screen.Tick(clock, 0));
        }

        [Test]
        public void TheSearchTimeChangesOncePerWholeSecond()
        {
            var clock = ClockAt(JoinedMs);
            screen.Tick(clock, 0);
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked));

            Assert.AreEqual(LobbyChanges.Play | LobbyChanges.Countdown, screen.Tick(clock, 0.2));
            Assert.AreEqual(QueueCountdown.Searching, screen.Countdown);
            Assert.AreEqual(0, screen.CountdownSeconds);
            Assert.AreEqual(LobbyChanges.None, screen.Tick(clock, 0.9));
            Assert.AreEqual(LobbyChanges.Countdown, screen.Tick(clock, 1.1));
            Assert.AreEqual(1, screen.CountdownSeconds);
            Assert.AreEqual(LobbyChanges.None, screen.Tick(clock, 1.5));
            Assert.AreEqual(LobbyChanges.Countdown, screen.Tick(clock, 65.0));
            Assert.AreEqual(65, screen.CountdownSeconds);
        }

        [Test]
        public void APenaltyDelayCountsDownToTheSearchStart()
        {
            var clock = ClockAt(JoinedMs);
            queue.Read(Searching(Mode._2V2, QueueKind.Casual, JoinedMs + 30_000));

            screen.Tick(clock, 0.5);
            Assert.AreEqual(QueueCountdown.SearchStartsIn, screen.Countdown);
            Assert.AreEqual(30, screen.CountdownSeconds);
            Assert.AreEqual(LobbyChanges.Countdown, screen.Tick(clock, 1.5));
            Assert.AreEqual(29, screen.CountdownSeconds);

            Assert.AreEqual(LobbyChanges.Countdown, screen.Tick(clock, 31));
            Assert.AreEqual(QueueCountdown.Searching, screen.Countdown);
            Assert.AreEqual(31, screen.CountdownSeconds);
        }

        [Test]
        public void ABanCountsDownToItsEndAndAnOpenBanCountsNothing()
        {
            var clock = ClockAt(JoinedMs);
            queue.Read(Refused(QueueRefusal.MatchmakingBan, JoinedMs + 90_000));

            screen.Tick(clock, 0);
            Assert.AreEqual(QueueCountdown.BanEndsIn, screen.Countdown);
            Assert.AreEqual(90, screen.CountdownSeconds);
            screen.Tick(clock, 90);
            Assert.AreEqual(QueueCountdown.None, screen.Countdown);
            Assert.AreEqual(0, screen.CountdownSeconds);

            queue.Read(Refused(QueueRefusal.MatchmakingBan));
            screen.Tick(clock, 0);
            Assert.AreEqual(QueueCountdown.None, screen.Countdown);
        }

        [Test]
        public void NoCountdownRunsBeforeTheClockIsSynced()
        {
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked, JoinedMs + 30_000));

            screen.Tick(new ServerClock(), 10);

            Assert.AreEqual(QueueCountdown.Searching, screen.Countdown);
            Assert.AreEqual(0, screen.CountdownSeconds);
        }

        [Test]
        public void TickingAnIdleOrSearchingLobbyAllocatesNothing()
        {
            var clock = ClockAt(JoinedMs);
            var now = 0d;
            var shown = LobbyChanges.None;
            TestDelegate tick = () =>
            {
                now += 0.25;
                shown |= screen.Tick(clock, now);
                for (var card = 0; card < PlayModes.Count; card++) play.StateOf(card);
            };
            tick();
            Assert.That(tick, Allocates.Nothing);

            queue.Read(Searching(Mode._1V1, QueueKind.Ranked));
            tick();
            Assert.That(tick, Allocates.Nothing);
            Assert.AreNotEqual(LobbyChanges.None, shown);
        }
    }
}
