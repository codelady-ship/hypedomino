using System;
using System.Collections.Generic;
using HypeDomino.Game.Missions;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Missions;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MissionLoaderTests
    {
        const string BoardJson = @"{""version"":7,""day"":""2026-11-14"",""week"":""2026-W46"",""missions"":[{""mission"":{""id"":""mission.daily_points_150"",""revision"":1,""type"":""daily"",""nameKey"":""mission.daily_points_150.name"",""event"":""board_points"",""target"":150,""filters"":{},""slot"":3,""weight"":2,""availability"":""live""},""period"":""2026-11-14"",""progress"":45,""completed_at"":null},{""mission"":{""id"":""mission.weekly_wins_15"",""revision"":1,""type"":""weekly"",""nameKey"":""mission.weekly_wins_15.name"",""event"":""match_won"",""target"":15,""filters"":{},""slot"":2,""weight"":1,""availability"":""live""},""period"":""2026-W46"",""progress"":15,""completed_at"":""2026-11-13T09:00:00Z""}],""daily_rerolls"":1,""weekly_rerolls"":0}";
        const string NoWeekJson = @"{""version"":7,""day"":""2026-11-14"",""missions"":[],""daily_rerolls"":1,""weekly_rerolls"":1}";

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
        MissionLoader loader;
        List<MissionLoadResult> results;

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
            loader = new MissionLoader(new ApiClient(api, session));
            results = new List<MissionLoadResult>();
        }

        [Test]
        public void LoadReadsTheBoardOfOneReply()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsTrue(loader.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/missions", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, BoardJson));
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(MissionLoadOutcome.Loaded, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            var board = results[0].Board;
            Assert.AreEqual(7, board.Version);
            Assert.AreEqual(1, board.Count(MissionTab.Daily));
            Assert.AreEqual(45, board.Card(MissionTab.Daily, 0).Progress);
            Assert.IsTrue(board.Card(MissionTab.Weekly, 0).Completed);
            Assert.AreEqual(1, board.RerollsLeft(MissionTab.Daily));
        }

        [Test]
        public void ASecondLoadWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsFalse(loader.Load(results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(200, BoardJson));
            Assert.IsTrue(loader.Load(results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [TestCase(HttpReply.NoResponse, null, MissionLoadOutcome.Offline, "mission.load.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", MissionLoadOutcome.RateLimited, "mission.load.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", MissionLoadOutcome.SignInRequired, null)]
        [TestCase(503, "{\"error\":\"mission.unavailable\"}", MissionLoadOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", MissionLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "", MissionLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", MissionLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, NoWeekJson, MissionLoadOutcome.Failed, "notice.generic")]
        public void AFailedReplyEndsTheLoadWithoutABoard(int status, string body, MissionLoadOutcome outcome, string key)
        {
            loader.Load(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Board);
        }

        [Test]
        public void TheShippedBundleHasEveryLoadKeyInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            foreach (MissionLoadOutcome outcome in Enum.GetValues(typeof(MissionLoadOutcome)))
            {
                var key = MissionLoadKeys.Of(outcome);
                if (key == null)
                {
                    continue;
                }
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
