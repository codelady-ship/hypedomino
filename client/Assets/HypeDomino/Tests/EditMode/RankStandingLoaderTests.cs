using System;
using System.Collections.Generic;
using HypeDomino.Game.Ranking;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class RankStandingLoaderTests
    {
        const string StandingJson = @"{""season"":{""id"":""season.2026_11"",""name_key"":""season.2026_11.name"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":true,""placement_played"":5,""placement_matches"":5,""rank"":""Diamond I"",""tier"":""Diamond"",""lp"":12,""division_lp"":100,""wins"":40}]}";

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
        RankStandingLoader loader;
        List<RankStandingLoadResult> results;

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
            loader = new RankStandingLoader(new ApiClient(api, session));
            results = new List<RankStandingLoadResult>();
        }

        [Test]
        public void LoadReadsThePlayersOwnStanding()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsTrue(loader.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/ranked/me", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, StandingJson));
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(RankStandingLoadOutcome.Loaded, results[0].Outcome);
            Assert.AreEqual("season.2026_11", results[0].Standing.SeasonId);
            Assert.IsTrue(results[0].Standing.TryFind("1v1", out var solo));
            Assert.AreEqual(RankTier.Diamond, solo.Badge.Tier);
            Assert.AreEqual(12, solo.LP);
        }

        [Test]
        public void ASecondLoadWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsFalse(loader.Load(results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(200, StandingJson));
            Assert.AreEqual(1, results.Count);
            Assert.IsTrue(loader.Load(results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void ALoadWithoutACallbackSendsNothing()
        {
            Assert.IsFalse(loader.Load(null));
            Assert.IsFalse(loader.Busy);
            Assert.IsEmpty(api.Requests);
        }

        [TestCase(HttpReply.NoResponse, null, RankStandingLoadOutcome.Offline)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", RankStandingLoadOutcome.RateLimited)]
        [TestCase(401, "{\"error\":\"access_rejected\"}", RankStandingLoadOutcome.SignInRequired)]
        [TestCase(404, "{\"error\":\"rating.season_unknown\"}", RankStandingLoadOutcome.NoSeason)]
        [TestCase(500, "{\"error\":\"internal\"}", RankStandingLoadOutcome.Failed)]
        [TestCase(200, "", RankStandingLoadOutcome.Failed)]
        [TestCase(200, "not json", RankStandingLoadOutcome.Failed)]
        public void AFailedReplyEndsTheLoadWithoutAStanding(int status, string body, RankStandingLoadOutcome outcome)
        {
            loader.Load(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.IsNull(results[0].Standing);
        }
    }
}
