using System;
using System.Collections.Generic;
using HypeDomino.Game.Pass;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PassLoaderTests
    {
        const string PassJson = @"{""season"":""season_1"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z"",""config_version"":4,""levels"":100,""xp"":1700,""level"":5,""level_xp"":1500,""next_level_xp"":2000,""premium"":false,""tracks"":{""free"":{""1"":{""id"":""pass.season_1.free.1"",""revision"":1,""coins"":30,""cosmetics"":null,""medals"":null,""passXp"":0}},""premium"":{""1"":{""id"":""pass.season_1.premium.1"",""revision"":1,""coins"":0,""crystals"":6,""cosmetics"":[""avatar.s1_navigator""],""medals"":null,""passXp"":0}}}}";
        const string LevelOutsideSeasonJson = @"{""season"":""season_1"",""ends_at"":""2027-02-01T00:00:00Z"",""levels"":100,""xp"":1700,""level"":5,""level_xp"":1500,""next_level_xp"":2000,""premium"":false,""tracks"":{""free"":{""101"":{""id"":""pass.season_1.free.101"",""coins"":30}}}}";
        const string NoSeasonFieldJson = @"{""levels"":100,""xp"":0,""level"":1,""level_xp"":0,""next_level_xp"":100,""ends_at"":""2027-02-01T00:00:00Z"",""tracks"":{}}";

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
        PassLoader loader;
        List<PassLoadResult> results;

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
            loader = new PassLoader(new ApiClient(api, session));
            results = new List<PassLoadResult>();
        }

        [Test]
        public void LoadReadsTheStandingAndTracksOfOneReply()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsTrue(loader.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/pass", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, PassJson));
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(PassLoadOutcome.Loaded, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.AreEqual("season_1", results[0].Standing.Season);
            Assert.AreEqual(5, results[0].Standing.Level);
            Assert.IsTrue(results[0].Tracks.TryRewardAt(PassTrack.Free, 1, out var free));
            Assert.AreEqual(30, free.Coins);
            Assert.IsTrue(results[0].Tracks.TryRewardAt(PassTrack.Premium, 1, out var premium));
            Assert.AreEqual("avatar.s1_navigator", premium.Cosmetics[0]);
        }

        [Test]
        public void ASecondLoadWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsFalse(loader.Load(results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(200, PassJson));
            Assert.IsTrue(loader.Load(results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [TestCase(404, "{\"error\":\"pass.no_season\"}", PassLoadOutcome.NoSeason, "pass.no_season")]
        [TestCase(HttpReply.NoResponse, null, PassLoadOutcome.Offline, "pass.load.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", PassLoadOutcome.RateLimited, "pass.load.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", PassLoadOutcome.SignInRequired, null)]
        [TestCase(404, "{\"error\":\"not_found\"}", PassLoadOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", PassLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "", PassLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", PassLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, NoSeasonFieldJson, PassLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, LevelOutsideSeasonJson, PassLoadOutcome.Failed, "notice.generic")]
        public void AFailedReplyEndsTheLoadWithoutAStanding(int status, string body, PassLoadOutcome outcome, string key)
        {
            loader.Load(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Standing);
            Assert.IsNull(results[0].Tracks);
        }
    }
}
