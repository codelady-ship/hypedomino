using System;
using System.Collections.Generic;
using HypeDomino.Game.Shop;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ShopLoaderTests
    {
        const string ShopJson = "{\"version\":3,\"sections\":[{\"section\":\"featured\",\"offers\":[{\"id\":\"offer.x\",\"kind\":\"bundle\",\"name_key\":\"shop.offer.x\",\"price_coins\":100,\"cosmetics\":[],\"owned\":[],\"purchasable\":true}]}]}";
        const string WalletJson = "{\"coins\":1500,\"crystals\":-20,\"entries\":[]}";

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
        ShopLoader loader;
        List<ShopLoadResult> results;

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
            loader = new ShopLoader(new ApiClient(api, session));
            results = new List<ShopLoadResult>();
        }

        [Test]
        public void LoadReadsTheShopThenTheWallet()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsTrue(loader.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/shop", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);

            api.Requests[0].Done(new HttpReply(200, ShopJson));
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[1].Method);
            Assert.AreEqual("/wallet", api.Requests[1].PathAndQuery);
            Assert.IsTrue(loader.Busy);
            Assert.IsEmpty(results);

            api.Requests[1].Done(new HttpReply(200, WalletJson));
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(ShopLoadOutcome.Loaded, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.AreEqual(3, results[0].Showcase.Version);
            Assert.AreEqual("offer.x", results[0].Showcase.Sections[0].Offers[0].OfferId);
            Assert.AreEqual(1500, results[0].Balances.Coins.Amount);
            Assert.AreEqual(-20, results[0].Balances.Crystals.Amount);
        }

        [Test]
        public void ASecondLoadWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsFalse(loader.Load(results.Add));
            api.Requests[0].Done(new HttpReply(200, ShopJson));
            Assert.IsFalse(loader.Load(results.Add));
            Assert.AreEqual(2, api.Requests.Count);

            api.Requests[1].Done(new HttpReply(200, WalletJson));
            Assert.IsTrue(loader.Load(results.Add));
            Assert.AreEqual(3, api.Requests.Count);
        }

        [TestCase(HttpReply.NoResponse, null, ShopLoadOutcome.Offline, "shop.load.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", ShopLoadOutcome.RateLimited, "shop.load.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", ShopLoadOutcome.SignInRequired, null)]
        [TestCase(503, "{\"error\":\"shop.unavailable\"}", ShopLoadOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", ShopLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", ShopLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "{\"version\":1}", ShopLoadOutcome.Failed, "notice.generic")]
        public void AFailedShopReplyEndsTheLoadWithoutTheWallet(int status, string body, ShopLoadOutcome outcome, string key)
        {
            loader.Load(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.AreEqual(1, api.Requests.Count);
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Showcase);
        }

        [TestCase(HttpReply.NoResponse, null, ShopLoadOutcome.Offline, "shop.load.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", ShopLoadOutcome.RateLimited, "shop.load.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", ShopLoadOutcome.SignInRequired, null)]
        [TestCase(500, "{\"error\":\"internal\"}", ShopLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "", ShopLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", ShopLoadOutcome.Failed, "notice.generic")]
        public void AFailedWalletReplyEndsTheLoadWithoutAShowcase(int status, string body, ShopLoadOutcome outcome, string key)
        {
            loader.Load(results.Add);
            api.Requests[0].Done(new HttpReply(200, ShopJson));
            api.Requests[1].Done(new HttpReply(status, body));

            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Showcase);
        }
    }
}
