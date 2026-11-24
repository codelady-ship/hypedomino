using System;
using System.Collections.Generic;
using HypeDomino.Game.Ranking;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LeaderboardLoaderTests
    {
        const string BoardJson = @"{""season"":""season.s1"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""top"",""rank"":""Diamond I"",""tier"":""Diamond"",""wins"":40}],""me"":{""position"":1,""account_id"":""top"",""rank"":""Diamond I"",""tier"":""Diamond"",""wins"":40},""around"":[{""position"":1,""account_id"":""top"",""rank"":""Diamond I"",""tier"":""Diamond"",""wins"":40}]}";

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
        LeaderboardLoader loader;
        List<LeaderboardLoadResult> results;

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
            loader = new LeaderboardLoader(new ApiClient(api, session));
            results = new List<LeaderboardLoadResult>();
        }

        [Test]
        public void LoadReadsTheBoardOfTheMode()
        {
            Assert.IsTrue(loader.Load("1v1", LeaderboardStanding.BoardGlobal, results.Add));
            Assert.IsTrue(loader.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/leaderboard/1v1?board=global", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, BoardJson));
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(LeaderboardLoadOutcome.Loaded, results[0].Outcome);
            Assert.AreEqual(1, results[0].Standing.TopCount);
            Assert.AreEqual(RankTier.Diamond, results[0].Standing.TopAt(0).Badge.Tier);
            Assert.IsTrue(results[0].Standing.Listed);
        }

        [Test]
        public void ASecondLoadWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(loader.Load("1v1", LeaderboardStanding.BoardGlobal, results.Add));
            Assert.IsFalse(loader.Load("2v2", LeaderboardStanding.BoardFriends, results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(200, BoardJson));
            Assert.IsTrue(loader.Load("2v2", LeaderboardStanding.BoardFriends, results.Add));
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("/leaderboard/2v2?board=friends", api.Requests[1].PathAndQuery);
        }

        [TestCase(null, "global")]
        [TestCase("", "global")]
        [TestCase("1v1", null)]
        [TestCase("1v1", "")]
        public void AnEmptyModeOrBoardSendsNothing(string mode, string board)
        {
            Assert.IsFalse(loader.Load(mode, board, results.Add));
            Assert.IsFalse(loader.Busy);
            Assert.IsEmpty(api.Requests);
        }

        [TestCase(HttpReply.NoResponse, null, LeaderboardLoadOutcome.Offline)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", LeaderboardLoadOutcome.RateLimited)]
        [TestCase(401, "{\"error\":\"access_rejected\"}", LeaderboardLoadOutcome.SignInRequired)]
        [TestCase(404, "{\"error\":\"rating.season_unknown\"}", LeaderboardLoadOutcome.NoSeason)]
        [TestCase(404, "{\"error\":\"leaderboard.unknown_board\"}", LeaderboardLoadOutcome.UnknownBoard)]
        [TestCase(400, "{\"error\":\"mode_invalid\"}", LeaderboardLoadOutcome.Failed)]
        [TestCase(503, "{\"error\":\"leaderboard.unavailable\"}", LeaderboardLoadOutcome.Failed)]
        [TestCase(500, "{\"error\":\"internal\"}", LeaderboardLoadOutcome.Failed)]
        [TestCase(200, "", LeaderboardLoadOutcome.Failed)]
        [TestCase(200, "not json", LeaderboardLoadOutcome.Failed)]
        public void AFailedReplyEndsTheLoadWithoutAStanding(int status, string body, LeaderboardLoadOutcome outcome)
        {
            loader.Load("1v1", LeaderboardStanding.BoardGlobal, results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.IsNull(results[0].Standing);
        }
    }
}
