using System;
using System.Collections.Generic;
using HypeDomino.Game.FriendLobby;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class RoomMembersTests
    {
        const string Leader = "acc-leader";
        const string Friend = "acc-friend";
        const string Third = "acc-third";

        sealed class Sent
        {
            public string PathAndQuery;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { PathAndQuery = pathAndQuery, Done = done });
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
        RoomMembers members;
        FriendRoom room;
        int changes;

        [SetUp]
        public void SignIn()
        {
            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            members = new RoomMembers(new ApiClient(api, session));
            members.Changed += Count;
            room = new FriendRoom();
            changes = 0;
        }

        void Count() => changes++;

        static string PublicProfile(string account, string nickname, string avatar, string frame) =>
            "{\"account_id\":\"" + account + "\",\"nickname\":\"" + nickname + "\",\"level\":3,\"loadout\":{\"avatar\":\"" + avatar + "\",\"frame\":\"" + frame
            + "\"},\"stats\":{\"games\":4,\"win_rate_percent\":50,\"best_streak\":2},\"ranked\":{\"ranks\":{\"1v1\":\"Gold II\"}}}";

        static LobbyState Lobby(string leader, params string[] accounts)
        {
            var state = new LobbyState { Code = "K7QM3X", LeaderAccountId = leader, Phase = LobbyPhase.Lobby, Mode = Mode._1V1 };
            state.MemberAccountIds.Add(accounts);
            return state;
        }

        static LobbyState OneVersusOne()
        {
            var state = Lobby(Leader, Leader, Friend, Third);
            state.Phase = LobbyPhase.Teams;
            state.Seats.Add(new LobbySeat { Seat = Seat.X, AccountId = Leader });
            state.Seats.Add(new LobbySeat { Seat = Seat.Y, AccountId = Third });
            return state;
        }

        void Read(LobbyState state, string local)
        {
            room.Read(state, local);
            members.Read(room);
        }

        void Answer(int request, string account, string nickname) =>
            api.Requests[request].Done(new HttpReply(200, PublicProfile(account, nickname, "avatar." + nickname, "frame." + nickname)));

        [Test]
        public void MembersFollowTheJoinOrderWithTheLeaderTheLocalPlayerAndTheirSeats()
        {
            Read(OneVersusOne(), Friend);

            Assert.AreEqual(3, members.All.Count);
            Assert.AreEqual(Leader, members.All[0].Account);
            Assert.IsTrue(members.All[0].IsLeader);
            Assert.IsFalse(members.All[0].IsLocal);
            Assert.AreEqual(Seat.X, members.All[0].Seat);
            Assert.IsTrue(members.All[0].Seated);
            Assert.AreEqual(Friend, members.All[1].Account);
            Assert.IsFalse(members.All[1].IsLeader);
            Assert.IsTrue(members.All[1].IsLocal);
            Assert.AreEqual(Seat.Unspecified, members.All[1].Seat);
            Assert.IsFalse(members.All[1].Seated);
            Assert.AreEqual(Seat.Y, members.All[2].Seat);
            Assert.AreEqual(2, members.IndexOf(Third));
            Assert.AreEqual(-1, members.IndexOf("acc-stranger"));
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void AMemberShowsNoLookUntilTheirPublicProfileArrives()
        {
            Read(Lobby(Leader, Leader, Friend), Leader);

            Assert.IsFalse(members.All[0].ProfileLoaded);
            Assert.IsNull(members.All[0].Nickname);
            Assert.IsNull(members.All[0].AvatarId);
            Assert.IsNull(members.All[0].FrameId);
            Assert.AreEqual(1, api.Requests.Count, "one public profile at a time");
            Assert.AreEqual("/profile/" + Leader, api.Requests[0].PathAndQuery);
        }

        [Test]
        public void EachProfileFillsItsMemberAndAsksForTheNext()
        {
            Read(Lobby(Leader, Leader, Friend), Leader);

            Answer(0, Leader, "Leyla");

            Assert.IsTrue(members.All[0].ProfileLoaded);
            Assert.AreEqual("Leyla", members.All[0].Nickname);
            Assert.AreEqual("avatar.Leyla", members.All[0].AvatarId);
            Assert.AreEqual("frame.Leyla", members.All[0].FrameId);
            Assert.AreEqual(1, members.All[0].Profile.RankCount);
            Assert.IsTrue(members.All[0].IsLeader, "a loaded profile keeps the room facts");
            Assert.IsTrue(members.All[0].IsLocal);
            Assert.AreEqual(2, changes);
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("/profile/" + Friend, api.Requests[1].PathAndQuery);

            Answer(1, Friend, "Murad");

            Assert.AreEqual("Murad", members.All[1].Nickname);
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual(3, changes);
        }

        [Test]
        public void ANewStateKeepsTheLoadedProfilesWithoutAskingAgain()
        {
            Read(Lobby(Leader, Leader, Friend), Leader);
            Answer(0, Leader, "Leyla");
            Answer(1, Friend, "Murad");

            Read(Lobby(Friend, Friend, Leader), Leader);

            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual(Friend, members.All[0].Account);
            Assert.IsTrue(members.All[0].IsLeader);
            Assert.AreEqual("Murad", members.All[0].Nickname);
            Assert.IsFalse(members.All[1].IsLeader);
            Assert.AreEqual("Leyla", members.All[1].Nickname);
        }

        [Test]
        public void AFailedProfileMovesOnAndIsAskedAgainOnTheNextState()
        {
            Read(Lobby(Leader, Leader, Friend), Leader);

            api.Requests[0].Done(new HttpReply(500, "{\"error\":\"internal\"}"));

            Assert.IsFalse(members.All[0].ProfileLoaded);
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("/profile/" + Friend, api.Requests[1].PathAndQuery);
            Answer(1, Friend, "Murad");
            Assert.AreEqual(2, api.Requests.Count);

            Read(Lobby(Leader, Leader, Friend), Leader);

            Assert.AreEqual(3, api.Requests.Count);
            Assert.AreEqual("/profile/" + Leader, api.Requests[2].PathAndQuery);
        }

        [Test]
        public void AProfileThatArrivesAfterItsMemberLeftIsDropped()
        {
            Read(Lobby(Leader, Leader, Friend), Leader);

            Read(Lobby(Friend, Friend), Friend);
            Assert.AreEqual(1, api.Requests.Count, "the next profile waits for the one in flight");

            Answer(0, Leader, "Leyla");

            Assert.AreEqual(1, members.All.Count);
            Assert.AreEqual(Friend, members.All[0].Account);
            Assert.IsFalse(members.All[0].ProfileLoaded);
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("/profile/" + Friend, api.Requests[1].PathAndQuery);
        }

        [Test]
        public void LeavingTheRoomEmptiesTheMembersAndForgetsTheirProfiles()
        {
            Read(Lobby(Leader, Leader), Leader);
            Answer(0, Leader, "Leyla");

            Read(new LobbyState(), Leader);

            Assert.AreEqual(0, members.All.Count);
            Assert.AreEqual(1, api.Requests.Count);

            Read(Lobby(Leader, Leader), Leader);

            Assert.IsFalse(members.All[0].ProfileLoaded);
            Assert.AreEqual(2, api.Requests.Count, "a room entered again shows the current look");
        }
    }
}
