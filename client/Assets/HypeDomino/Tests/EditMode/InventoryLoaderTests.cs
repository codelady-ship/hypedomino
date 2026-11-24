using System;
using System.Collections.Generic;
using HypeDomino.Game.Inventory;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class InventoryLoaderTests
    {
        const string InventoryJson = @"{""items"":[{""id"":""avatar.default"",""type"":""avatar"",""source_type"":""default""},{""id"":""frame.s1_gold"",""type"":""frame"",""source_type"":""shop"",""source_ref"":""offer.frame_gold"",""granted_at"":""2026-11-02T10:00:00Z""}],""loadout"":{""avatar"":""avatar.default"",""frame"":""frame.s1_gold"",""emojis"":[""emoji.wave""]},""emoji_slots"":6}";

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
        InventoryLoader loader;
        List<InventoryLoadResult> results;

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
            loader = new InventoryLoader(new ApiClient(api, session));
            results = new List<InventoryLoadResult>();
        }

        [Test]
        public void LoadReadsTheHeldItemsAndLoadoutOfTheReply()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsTrue(loader.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/inventory", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, InventoryJson));
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(InventoryLoadOutcome.Loaded, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.AreEqual(2, results[0].Inventory.items.Length);
            Assert.AreEqual("frame.s1_gold", results[0].Inventory.items[1].id);
            Assert.AreEqual("frame.s1_gold", results[0].Inventory.loadout.frame);
            Assert.AreEqual("emoji.wave", results[0].Inventory.loadout.emojis[0]);
            Assert.AreEqual(6, results[0].Inventory.emoji_slots);
        }

        [Test]
        public void ASecondLoadWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsFalse(loader.Load(results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(200, InventoryJson));
            Assert.IsTrue(loader.Load(results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [TestCase(HttpReply.NoResponse, null, InventoryLoadOutcome.Offline, "inventory.load.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", InventoryLoadOutcome.RateLimited, "inventory.load.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", InventoryLoadOutcome.SignInRequired, null)]
        [TestCase(404, "{\"error\":\"not_found\"}", InventoryLoadOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", InventoryLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "", InventoryLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", InventoryLoadOutcome.Failed, "notice.generic")]
        public void AFailedReplyEndsTheLoadWithoutAnInventory(int status, string body, InventoryLoadOutcome outcome, string key)
        {
            loader.Load(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Inventory);
        }
    }
}
