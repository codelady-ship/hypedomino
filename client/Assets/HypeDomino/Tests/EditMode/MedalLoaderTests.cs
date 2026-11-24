using System;
using System.Collections.Generic;
using HypeDomino.Game.Medals;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Medals;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MedalLoaderTests
    {
        const string CollectionJson = @"{""version"":7,""slots"":3,""medals"":[{""id"":""medal.first_win"",""revision"":1,""nameKey"":""medal.first_win.name"",""descriptionKey"":""medal.first_win.description"",""rarityId"":""common"",""staticAddress"":""medal/first_win/static"",""sources"":[{""type"":""mission"",""ref"":""mission.first_win""}],""availability"":""live""}],""owned"":[{""medal"":""medal.first_win"",""earned_at"":""2026-11-01T10:00:00Z""}],""showcase"":[""medal.first_win""]}";

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
        MedalLoader loader;
        List<MedalLoadResult> results;

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
            loader = new MedalLoader(new ApiClient(api, session));
            results = new List<MedalLoadResult>();
        }

        [Test]
        public void LoadReadsTheCollectionTheScreenShows()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsTrue(loader.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/medals", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, CollectionJson));
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(MedalLoadOutcome.Loaded, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);

            var collection = new MedalCollection();
            collection.Show(results[0].Collection);
            Assert.AreEqual(1, collection.ShownCount);
            Assert.AreEqual(1, collection.EarnedCount);
            Assert.AreEqual(3, collection.Slots);
            Assert.AreEqual("medal.first_win", collection.ShowcaseAt(0).id);
        }

        [Test]
        public void ASecondLoadWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsFalse(loader.Load(results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(200, CollectionJson));
            Assert.IsTrue(loader.Load(results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [TestCase(HttpReply.NoResponse, null, MedalLoadOutcome.Offline, "medal.load.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", MedalLoadOutcome.RateLimited, "medal.load.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", MedalLoadOutcome.SignInRequired, null)]
        [TestCase(503, "{\"error\":\"medal.unavailable\"}", MedalLoadOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", MedalLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "", MedalLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", MedalLoadOutcome.Failed, "notice.generic")]
        public void AFailedReplyEndsTheLoadWithoutACollection(int status, string body, MedalLoadOutcome outcome, string key)
        {
            loader.Load(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Collection);
        }
    }
}
