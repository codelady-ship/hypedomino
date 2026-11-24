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
    public sealed class MissionRerollerTests
    {
        const string BoardJson = @"{""version"":7,""day"":""2026-11-14"",""week"":""2026-W46"",""missions"":[{""mission"":{""id"":""mission.daily_points_150"",""type"":""daily"",""nameKey"":""mission.daily_points_150.name"",""target"":150},""period"":""2026-11-14"",""progress"":45,""completed_at"":null},{""mission"":{""id"":""mission.weekly_wins_15"",""type"":""weekly"",""nameKey"":""mission.weekly_wins_15.name"",""target"":15},""period"":""2026-W46"",""progress"":15,""completed_at"":""2026-11-13T09:00:00Z""}],""daily_rerolls"":1,""weekly_rerolls"":1}";
        const string RerolledJson = @"{""version"":8,""day"":""2026-11-14"",""week"":""2026-W46"",""missions"":[{""mission"":{""id"":""mission.daily_win_3"",""type"":""daily"",""nameKey"":""mission.daily_win_3.name"",""target"":3},""period"":""2026-11-14"",""progress"":0,""completed_at"":null},{""mission"":{""id"":""mission.weekly_wins_15"",""type"":""weekly"",""nameKey"":""mission.weekly_wins_15.name"",""target"":15},""period"":""2026-W46"",""progress"":15,""completed_at"":""2026-11-13T09:00:00Z""}],""daily_rerolls"":0,""weekly_rerolls"":1}";
        const string NoRerollsJson = @"{""version"":7,""day"":""2026-11-14"",""week"":""2026-W46"",""missions"":[{""mission"":{""id"":""mission.daily_points_150"",""type"":""daily"",""nameKey"":""mission.daily_points_150.name"",""target"":150},""period"":""2026-11-14"",""progress"":45,""completed_at"":null}],""daily_rerolls"":0,""weekly_rerolls"":0}";
        const string NoWeekJson = @"{""version"":8,""day"":""2026-11-14"",""missions"":[],""daily_rerolls"":0,""weekly_rerolls"":1}";

        static readonly string[] ShownKeys =
        {
            MissionRerollKeys.RerollUsed,
            MissionRerollKeys.NotAssigned,
            MissionRerollKeys.NoReplacement,
            MissionRerollKeys.Offline,
            MissionRerollKeys.RateLimited,
        };

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
        MissionReroller reroller;
        MissionBoard board;
        List<MissionRerollResult> results;

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
            reroller = new MissionReroller(new ApiClient(api, session));
            board = Board(BoardJson);
            results = new List<MissionRerollResult>();
        }

        static MissionBoard Board(string json)
        {
            Assert.IsTrue(MissionBoard.TryFrom(MissionBoardReply.FromJson(json), out var read));
            return read;
        }

        MissionCard Daily => board.Card(MissionTab.Daily, 0);

        [Test]
        public void RerollSendsTheIntentAndShowsTheServersNewBoard()
        {
            Assert.IsTrue(reroller.Reroll(board, Daily, results.Add));
            Assert.IsTrue(reroller.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual("/missions/reroll", api.Requests[0].PathAndQuery);
            Assert.AreEqual(@"{""mission"":""mission.daily_points_150""}", api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, RerolledJson));
            Assert.IsFalse(reroller.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(MissionRerollOutcome.Rerolled, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            var rerolled = results[0].Board;
            Assert.AreEqual(8, rerolled.Version);
            Assert.AreEqual("mission.daily_win_3", rerolled.Card(MissionTab.Daily, 0).Id);
            Assert.AreEqual(0, rerolled.Card(MissionTab.Daily, 0).Progress);
            Assert.AreEqual(0, rerolled.RerollsLeft(MissionTab.Daily));
            Assert.AreEqual(1, rerolled.RerollsLeft(MissionTab.Weekly));
        }

        [Test]
        public void ASecondRerollWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(reroller.Reroll(board, Daily, results.Add));
            Assert.IsFalse(reroller.Reroll(board, Daily, results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(409, "{\"error\":\"mission.reroll_used\"}"));
            Assert.IsTrue(reroller.Reroll(board, Daily, results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void NothingIsSentForACardTheBoardDoesNotOfferToReroll()
        {
            Assert.IsFalse(reroller.Reroll(board, board.Card(MissionTab.Weekly, 0), results.Add));
            Assert.IsFalse(reroller.Reroll(null, Daily, results.Add));
            Assert.IsFalse(reroller.Reroll(board, default, results.Add));
            var spent = Board(NoRerollsJson);
            Assert.IsFalse(reroller.Reroll(spent, spent.Card(MissionTab.Daily, 0), results.Add));

            Assert.IsFalse(reroller.Busy);
            Assert.IsEmpty(api.Requests);
            Assert.IsEmpty(results);
        }

        [TestCase(HttpReply.NoResponse, null, MissionRerollOutcome.Offline, "mission.reroll.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", MissionRerollOutcome.RateLimited, "mission.reroll.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", MissionRerollOutcome.SignInRequired, null)]
        [TestCase(409, "{\"error\":\"mission.reroll_used\"}", MissionRerollOutcome.RerollUsed, "mission.reroll.used")]
        [TestCase(409, "{\"error\":\"mission.not_assigned\"}", MissionRerollOutcome.NotAssigned, "mission.reroll.not_assigned")]
        [TestCase(409, "{\"error\":\"mission.no_replacement\"}", MissionRerollOutcome.NoReplacement, "mission.reroll.no_replacement")]
        [TestCase(400, "{\"error\":\"malformed\"}", MissionRerollOutcome.Failed, "notice.generic")]
        [TestCase(503, "{\"error\":\"mission.unavailable\"}", MissionRerollOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", MissionRerollOutcome.Failed, "notice.generic")]
        [TestCase(200, "", MissionRerollOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", MissionRerollOutcome.Failed, "notice.generic")]
        [TestCase(200, NoWeekJson, MissionRerollOutcome.Failed, "notice.generic")]
        public void ARefusedRerollEndsWithoutABoard(int status, string body, MissionRerollOutcome outcome, string key)
        {
            reroller.Reroll(board, Daily, results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(reroller.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Board);
        }

        [Test]
        public void TheShippedBundleHasEveryShownKeyInAzerbaijaniAndEnglish()
        {
            Assert.AreEqual(ShownKeys.Length, new HashSet<string>(ShownKeys).Count);
            var strings = BuiltinStrings.Load();
            foreach (var key in ShownKeys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
