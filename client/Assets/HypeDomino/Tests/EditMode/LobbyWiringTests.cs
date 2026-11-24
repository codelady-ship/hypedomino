using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game;
using HypeDomino.Game.Lobby;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.Net.Integrity;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LobbyWiringTests
    {
        const ulong Nonce = 0x5EED_0074_0000_0003;
        const int Ranked1V1 = 0;
        const int Casual2V2 = 3;
        const string WalletPath = "/wallet";
        const string OwnPath = "/profile/me";
        const string WalletJson = "{\"coins\":1500,\"crystals\":20,\"entries\":[]}";
        const string OwnJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":""Leyla"",""setup_required"":false,""rename_required"":false,""level"":7,""xp"":1250,""loadout"":{},""stats"":{""games"":40,""wins"":22,""losses"":18,""win_rate_percent"":55,""streak"":2,""best_streak"":6,""modes"":{}}}";

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

        sealed class Sent
        {
            public string Method;
            public string PathAndQuery;
            public string Body;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, PathAndQuery = pathAndQuery, Body = body, Done = done });

            public int CountTo(string path) => Requests.FindAll(request => request.PathAndQuery == path).Count;

            public Sent LastTo(string path) => Requests.FindLast(request => request.PathAndQuery == path);
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

        sealed class Attestor : IDeviceAttestor
        {
            public bool CanAttest { get; set; } = true;

            public void Attest(string nonce, Action<IntegrityProof> done) => done(new IntegrityProof { PlayIntegrityToken = "token-for-" + nonce });
        }

        Api api;
        ApiClient client;
        LobbyHeader header;
        LobbyHeaderLoader loader;
        AppFlow flow;
        List<LobbyHeaderLoadOutcome> outcomes;

        [SetUp]
        public void Build()
        {
            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var authSession = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            authSession.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            client = new ApiClient(api, authSession);
            header = new LobbyHeader();
            loader = new LobbyHeaderLoader(client, header);
            flow = new AppFlow();
            outcomes = new List<LobbyHeaderLoadOutcome>();
            flow.Changed += Reload;
        }

        void Reload(AppState state) => NetBoot.ReloadHeader(state, loader, outcomes.Add);

        void AnswerHeader()
        {
            api.LastTo(WalletPath).Done(new HttpReply(200, WalletJson));
            api.LastTo(OwnPath).Done(new HttpReply(200, OwnJson));
        }

        static ServerHello Accepted(bool inMatch) => new() { Status = HelloStatus.Accepted, InMatch = inMatch };

        [Test]
        public void EnteringTheLobbyLoadsTheHeader()
        {
            flow.Hello(Accepted(false));

            Assert.AreEqual(1, api.CountTo(WalletPath));
            Assert.AreEqual(1, api.CountTo(OwnPath));
            AnswerHeader();
            Assert.AreEqual(new[] { LobbyHeaderLoadOutcome.Loaded }, outcomes);
            Assert.AreEqual(1500, header.Balances.Coins.Amount);
            Assert.AreEqual(7, header.Level);
        }

        [Test]
        public void ReturningFromAMatchReloadsTheHeader()
        {
            flow.Hello(Accepted(true));
            Assert.IsEmpty(api.Requests);

            flow.View(MatchState.Finished);
            Assert.IsTrue(flow.LeaveMatch());

            Assert.AreEqual(1, api.CountTo(WalletPath));
            Assert.AreEqual(1, api.CountTo(OwnPath));
        }

        [Test]
        public void LeavingAFriendRoomReloadsTheHeader()
        {
            flow.Hello(Accepted(false));
            AnswerHeader();
            flow.Room(new LobbyState { Code = "ABCDEF" });
            Assert.AreEqual(1, api.CountTo(WalletPath));

            flow.Room(new LobbyState());

            Assert.AreEqual(AppState.Lobby, flow.State);
            Assert.AreEqual(2, api.CountTo(WalletPath));
            Assert.AreEqual(2, api.CountTo(OwnPath));
        }

        [Test]
        public void AHelloThatKeepsTheLobbyDoesNotLoadAgain()
        {
            flow.Hello(Accepted(false));
            AnswerHeader();

            flow.Hello(Accepted(false));

            Assert.AreEqual(1, api.CountTo(WalletPath));
        }

        [TestCase(AppState.Boot)]
        [TestCase(AppState.Login)]
        [TestCase(AppState.Match)]
        [TestCase(AppState.UpdateRequired)]
        [TestCase(AppState.Maintenance)]
        [TestCase(AppState.Room)]
        public void NoOtherStateLoadsTheHeader(AppState state)
        {
            Assert.IsFalse(NetBoot.ReloadHeader(state, loader, outcomes.Add));
            Assert.IsEmpty(api.Requests);
        }

        [Test]
        public void ALoadInFlightIsNotSentTwice()
        {
            Assert.IsTrue(NetBoot.ReloadHeader(AppState.Lobby, loader, outcomes.Add));
            Assert.IsFalse(NetBoot.ReloadHeader(AppState.Lobby, loader, outcomes.Add));

            Assert.AreEqual(2, api.Requests.Count);
        }

        [TestCase(Ranked1V1, "ranked_match", QueueKind.Ranked)]
        [TestCase(Casual2V2, "casual_match", QueueKind.Casual)]
        public void ACardTapProvesTheDeviceThroughTheNonceThenJoins(int card, string purpose, QueueKind kind)
        {
            var transport = new FakeTransport();
            var session = new Session();
            var queue = new MatchQueue();
            var screen = NetBoot.NewLobbyScreen(queue, new IntegrityProofs(client, new Attestor()), header);
            session.Hello(transport, new ClientHello());
            transport.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = Nonce });
            session.Poll();
            transport.Sent.Clear();

            Assert.AreSame(header, screen.Header);
            Assert.IsTrue(screen.Play.Play(session, card));

            var nonce = api.LastTo(IntegrityProofs.NoncePath);
            Assert.AreEqual("POST", nonce.Method);
            StringAssert.Contains(purpose, nonce.Body);
            Assert.IsEmpty(transport.Sent);

            nonce.Done(new HttpReply(200, "{\"nonce\":\"n-1\"}"));

            Assert.AreEqual(1, transport.Sent.Count);
            var join = Envelope.Parser.ParseFrom(transport.Sent[0]).JoinQueueIntent;
            Assert.AreEqual(kind, join.Queue);
            Assert.AreEqual("token-for-n-1", join.Integrity.PlayIntegrityToken);
            Assert.AreEqual(PlayCardState.Starting, screen.Play.StateOf(card));
        }

        [Test]
        public void WithoutAnAttestorACardTapJoinsWithoutAProof()
        {
            var transport = new FakeTransport();
            var session = new Session();
            var queue = new MatchQueue();
            var screen = NetBoot.NewLobbyScreen(queue, new IntegrityProofs(client, new NoDeviceAttestor()), header);
            session.Hello(transport, new ClientHello());
            transport.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = Nonce });
            session.Poll();
            transport.Sent.Clear();

            Assert.IsTrue(screen.Play.Play(session, Ranked1V1));

            Assert.IsEmpty(api.Requests);
            Assert.AreEqual(1, transport.Sent.Count);
            Assert.IsNull(Envelope.Parser.ParseFrom(transport.Sent[0]).JoinQueueIntent.Integrity);
        }
    }
}
