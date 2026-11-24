using System;
using System.Collections.Generic;
using HypeDomino.Game.FriendLobby;
using HypeDomino.Game.Friends;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class InvitePanelTests
    {
        const ulong Nonce = 0x5EED_0000_0000_0279;
        const string Leader = "acc-leader";
        const string Member = "acc-member";
        const string Code = "K7QM3X";

        static readonly Friend OnlineFriend = new("acc-online", "Nara", FriendPresence.Online);
        static readonly Friend PlayingFriend = new("acc-playing", "Tural", FriendPresence.InMatch);
        static readonly Friend OfflineFriend = new("acc-offline", "Lale", FriendPresence.Offline);
        static readonly Friend MemberFriend = new(Member, "Aysel", FriendPresence.Online);

        const string ListedFriends =
            "{\"friends\":["
            + "{\"account_id\":\"acc-online\",\"nickname\":\"Nara\",\"since\":\"2026-11-01T10:00:00Z\",\"presence\":\"online\"},"
            + "{\"account_id\":\"acc-playing\",\"nickname\":\"Tural\",\"since\":\"2026-11-02T10:00:00Z\",\"presence\":\"in_match\"},"
            + "{\"account_id\":\"acc-offline\",\"nickname\":\"Lale\",\"since\":\"2026-11-03T10:00:00Z\",\"presence\":\"offline\"},"
            + "{\"account_id\":\"acc-unknown\",\"nickname\":\"Orxan\",\"since\":\"2026-11-04T10:00:00Z\",\"presence\":\"away\"},"
            + "{\"account_id\":\"acc-silent\",\"since\":\"2026-11-05T10:00:00Z\"}],"
            + "\"incoming\":[{\"account_id\":\"acc-asking\",\"nickname\":\"Nigar\",\"since\":\"2026-11-06T10:00:00Z\"}],"
            + "\"outgoing\":[{\"account_id\":\"acc-asked\",\"nickname\":\"Elvin\",\"since\":\"2026-11-07T10:00:00Z\"}],"
            + "\"blocked\":[{\"account_id\":\"acc-blocked\",\"nickname\":\"Kamal\",\"since\":\"2026-11-08T10:00:00Z\"}]}";

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

            public void Deliver(byte[] frame) => incoming.Enqueue(frame);
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

        Api api;
        ApiClient client;
        FriendRoom room;
        InvitePanel panel;
        FakeTransport transport;
        Session session;
        int changes;

        [SetUp]
        public void SignIn()
        {
            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var signedIn = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            signedIn.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            client = new ApiClient(api, signedIn);
            room = new FriendRoom();
            panel = new InvitePanel(room, new LobbyCommands(), client);
            panel.Changed += Count;
            transport = new FakeTransport();
            session = AcceptedSession(transport);
            changes = 0;
        }

        void Count() => changes++;

        static Session AcceptedSession(FakeTransport transport)
        {
            var accepted = new Session();
            accepted.Hello(transport, new ClientHello());
            var hello = new ServerHello { Status = HelloStatus.Accepted, SessionNonce = Nonce };
            transport.Deliver(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, Nonce), Envelope.PayloadOneofCase.ServerHello, hello).ToArray());
            accepted.Poll();
            Assert.IsTrue(accepted.Accepted);
            transport.Sent.Clear();
            return accepted;
        }

        static LobbyState Lobby(LobbyPhase phase, params string[] accounts)
        {
            var state = new LobbyState { Code = Code, LeaderAccountId = Leader, Phase = phase, Mode = Mode._1V1 };
            state.MemberAccountIds.Add(accounts);
            return state;
        }

        void EnterRoom() => room.Read(Lobby(LobbyPhase.Lobby, Leader, Member), Leader);

        void AnswerFriends(string body) => api.Requests[api.Requests.Count - 1].Done(new HttpReply(200, body));

        LobbyIntent SentIntent(int index) => Envelope.Parser.ParseFrom(transport.Sent[index]).LobbyIntent;

        [Test]
        public void OnlyTheFriendsListIsKeptWithEachPresence()
        {
            var friends = new List<Friend>();

            Assert.IsTrue(FriendList.TryRead(ListedFriends, friends));

            Assert.AreEqual(5, friends.Count);
            Assert.AreEqual("acc-online", friends[0].Account);
            Assert.AreEqual("Nara", friends[0].Nickname);
            Assert.AreEqual(FriendPresence.Online, friends[0].Presence);
            Assert.AreEqual(FriendPresence.InMatch, friends[1].Presence);
            Assert.AreEqual(FriendPresence.Offline, friends[2].Presence);
            Assert.AreEqual(FriendPresence.Offline, friends[3].Presence, "an unknown presence reads as offline");
            Assert.AreEqual(FriendPresence.Offline, friends[4].Presence, "a missing presence reads as offline");
            Assert.AreEqual(string.Empty, friends[4].Nickname);
        }

        [Test]
        public void NoFriendsReadsAsAnEmptyList()
        {
            var friends = new List<Friend> { OnlineFriend };

            Assert.IsTrue(FriendList.TryRead("{\"friends\":[],\"incoming\":[],\"outgoing\":[],\"blocked\":[]}", friends));
            Assert.AreEqual(0, friends.Count);
            friends.Add(OnlineFriend);
            Assert.IsTrue(FriendList.TryRead("{\"friends\":null}", friends));
            Assert.AreEqual(0, friends.Count);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("not json")]
        [TestCase("{\"friends\":[{\"nickname\":\"Nara\",\"presence\":\"online\"}]}")]
        [TestCase("{\"friends\":[{\"account_id\":\"\",\"nickname\":\"Nara\"}]}")]
        public void AnUnreadableReplyKeepsTheEarlierList(string body)
        {
            var friends = new List<Friend> { OnlineFriend };

            Assert.IsFalse(FriendList.TryRead(body, friends));
            Assert.AreEqual(1, friends.Count);
            Assert.AreEqual(OnlineFriend.Account, friends[0].Account);
        }

        [Test]
        public void TheLoaderAsksForTheFriendsAndKeepsThem()
        {
            var loader = new FriendListLoader(client);
            var outcomes = new List<FriendListLoadOutcome>();

            Assert.IsTrue(loader.Load(outcomes.Add));
            Assert.IsTrue(loader.Busy);
            Assert.IsFalse(loader.Load(outcomes.Add), "one load at a time");
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual(FriendList.Path, api.Requests[0].PathAndQuery);

            AnswerFriends(ListedFriends);

            Assert.IsFalse(loader.Busy);
            CollectionAssert.AreEqual(new[] { FriendListLoadOutcome.Loaded }, outcomes);
            Assert.AreEqual(5, loader.Friends.Count);
        }

        [TestCase(HttpReply.NoResponse, "", FriendListLoadOutcome.Offline)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", FriendListLoadOutcome.RateLimited)]
        [TestCase(500, "{\"error\":\"internal\"}", FriendListLoadOutcome.Failed)]
        [TestCase(200, "not json", FriendListLoadOutcome.Failed)]
        public void AFailedLoadNamesItsOutcomeAndKeepsTheEarlierFriends(int status, string body, FriendListLoadOutcome expected)
        {
            var loader = new FriendListLoader(client);
            var outcome = FriendListLoadOutcome.Loaded;
            loader.Load(_ => { });
            AnswerFriends(ListedFriends);

            loader.Load(answer => outcome = answer);
            api.Requests[1].Done(new HttpReply(status, body));

            Assert.AreEqual(expected, outcome);
            Assert.AreEqual(5, loader.Friends.Count);
            Assert.IsFalse(loader.Busy);
        }

        [Test]
        public void ThePanelShowsTheLoadedFriendsAndTheRoomCode()
        {
            EnterRoom();
            var outcome = FriendListLoadOutcome.Failed;

            Assert.IsTrue(panel.Load(answer => outcome = answer));
            Assert.IsTrue(panel.Loading);
            Assert.IsFalse(panel.Load(_ => { }));
            AnswerFriends(ListedFriends);

            Assert.AreEqual(FriendListLoadOutcome.Loaded, outcome);
            Assert.AreEqual(5, panel.Friends.Count);
            Assert.AreEqual(Code, panel.Code);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void EachFriendGetsTheStateOfWhatTheRoomAndTheirPresenceAllow()
        {
            EnterRoom();

            Assert.AreEqual(FriendInviteState.Invitable, panel.StateOf(OnlineFriend));
            Assert.AreEqual(FriendInviteState.InMatch, panel.StateOf(PlayingFriend));
            Assert.AreEqual(FriendInviteState.Offline, panel.StateOf(OfflineFriend));
            Assert.AreEqual(FriendInviteState.AlreadyInRoom, panel.StateOf(MemberFriend));
        }

        [Test]
        public void AFriendOutsideTheRoomCannotBeInvitedWhileTheRoomTakesNoInvites()
        {
            Assert.AreEqual(FriendInviteState.RoomClosed, panel.StateOf(OnlineFriend), "no room");

            room.Read(Lobby(LobbyPhase.Lobby, Leader, Member, "acc-3", "acc-4"), Leader);
            Assert.AreEqual(FriendInviteState.RoomClosed, panel.StateOf(OnlineFriend), "full room");
            Assert.AreEqual(FriendInviteState.RoomClosed, panel.StateOf(OfflineFriend), "no code to share into a full room");
            Assert.AreEqual(FriendInviteState.AlreadyInRoom, panel.StateOf(MemberFriend));

            room.Read(Lobby(LobbyPhase.Match, Leader, Member), Leader);
            Assert.AreEqual(FriendInviteState.RoomClosed, panel.StateOf(OnlineFriend), "running match");

            room.Read(Lobby(LobbyPhase.Teams, Leader, Member), Member);
            Assert.AreEqual(FriendInviteState.Invitable, panel.StateOf(OnlineFriend), "any member invites in team setup");
        }

        [Test]
        public void InvitingAnOnlineFriendSendsOneInviteUntilTheRoomChanges()
        {
            EnterRoom();

            Assert.IsTrue(panel.TryInvite(session, OnlineFriend));
            Assert.AreEqual(FriendInviteState.Invited, panel.StateOf(OnlineFriend));
            Assert.IsFalse(panel.TryInvite(session, OnlineFriend));
            Assert.AreEqual(1, transport.Sent.Count);
            Assert.AreEqual(new LobbyIntent { Command = LobbyCommand.Invite, AccountId = OnlineFriend.Account }, SentIntent(0));
            Assert.AreEqual(1, changes);

            EnterRoom();
            panel.RoomChanged();

            Assert.AreEqual(2, changes);
            Assert.AreEqual(FriendInviteState.Invitable, panel.StateOf(OnlineFriend));
            Assert.IsTrue(panel.TryInvite(session, OnlineFriend));
            Assert.AreEqual(2, transport.Sent.Count);
        }

        [Test]
        public void OnlyAnInvitableFriendIsInvited()
        {
            EnterRoom();

            Assert.IsFalse(panel.TryInvite(session, PlayingFriend));
            Assert.IsFalse(panel.TryInvite(session, OfflineFriend));
            Assert.IsFalse(panel.TryInvite(session, MemberFriend));
            room.Read(Lobby(LobbyPhase.Match, Leader, Member), Leader);
            Assert.IsFalse(panel.TryInvite(session, OnlineFriend));

            Assert.AreEqual(0, transport.Sent.Count);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void AnInviteThatCannotBeSentIsNotMarked()
        {
            EnterRoom();

            Assert.IsFalse(panel.TryInvite(new Session(), OnlineFriend));

            Assert.AreEqual(FriendInviteState.Invitable, panel.StateOf(OnlineFriend));
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void ReadingAFriendsStateAllocatesNothing()
        {
            EnterRoom();
            panel.TryInvite(session, OnlineFriend);
            var state = FriendInviteState.RoomClosed;
            TestDelegate read = () =>
            {
                state = panel.StateOf(OnlineFriend);
                state = panel.StateOf(PlayingFriend);
                state = panel.StateOf(OfflineFriend);
                state = panel.StateOf(MemberFriend);
            };
            read();

            Assert.That(read, Allocates.Nothing);
            Assert.AreEqual(FriendInviteState.AlreadyInRoom, state);
        }
    }
}
