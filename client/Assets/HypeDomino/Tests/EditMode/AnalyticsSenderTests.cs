using System;
using System.Collections.Generic;
using System.Text;
using HypeDomino.Core.Analytics;
using HypeDomino.Game.Analytics;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AnalyticsSenderTests
    {
        const string AnalyticsId = "3f2b8c1e-5a4d-4e8f-9b7a-1c2d3e4f5a6b";
        const int ServerMaxBodyBytes = 64 * 1024;

        sealed class Sent
        {
            public string Method;
            public string PathAndQuery;
            public string Body;
            public string Mac;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, PathAndQuery = pathAndQuery, Body = body, Mac = mac, Done = done });
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
        ApiAnalyticsSender sender;
        AnalyticsQueue queue;
        long now;

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
            sender = new ApiAnalyticsSender(new ApiClient(api, session));
            now = 1700000000000;
            queue = new AnalyticsQueue(sender, Clock, 2, 4);
            queue.Consent(AnalyticsId);
        }

        long Clock() => now;

        static string Error(string code) => "{\"error\":\"" + code + "\"}";

        void Answer(int status, string body) => api.Requests[api.Requests.Count - 1].Done(new HttpReply(status, body));

        [Test]
        public void AFlushPostsTheBatchSignedToTheIngestRoute()
        {
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Flush();
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual("/analytics/events", api.Requests[0].PathAndQuery);
            Assert.IsNotEmpty(api.Requests[0].Mac);
        }

        [Test]
        public void TheBodyIsTheBatchFormatTheServerReads()
        {
            queue.Track(AnalyticsEvents.AppOpen);
            now += 5;
            queue.Track(AnalyticsEvents.SignIn, "google");
            queue.Flush();
            var body = api.Requests[0].Body;
            Assert.AreEqual(
                "{\"analyticsId\":\"" + AnalyticsId + "\",\"events\":[" +
                "{\"name\":\"app_open\",\"label\":\"\",\"value\":0,\"clientTimeMs\":1700000000000}," +
                "{\"name\":\"sign_in\",\"label\":\"google\",\"value\":0,\"clientTimeMs\":1700000000005}]}",
                body);
        }

        [Test]
        public void TheBodyHoldsOnlyTheCurrentBatch()
        {
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Track(AnalyticsEvents.NicknameSet);
            queue.Track(AnalyticsEvents.ScreenView, "shop");
            queue.Flush();
            Answer(200, "{\"accepted\":2}");
            queue.Flush();
            var read = JsonUtility.FromJson<AnalyticsBatchRequest>(api.Requests[1].Body);
            Assert.AreEqual(1, read.events.Count);
            Assert.AreEqual("screen_view", read.events[0].name);
            Assert.AreEqual("shop", read.events[0].label);
        }

        [TestCase(200, "{\"accepted\":1}", true)]
        [TestCase(400, "{\"error\":\"analytics.malformed\"}", true)]
        [TestCase(403, "{\"error\":\"analytics.consent_required\"}", true)]
        [TestCase(409, "{\"error\":\"analytics.id_taken\"}", true)]
        [TestCase(403, "{\"error\":\"forbidden\"}", false)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", false)]
        [TestCase(500, "{\"error\":\"internal\"}", false)]
        [TestCase(503, "", false)]
        [TestCase(HttpReply.NoResponse, null, false)]
        public void AnAcceptedOrRefusedForGoodBatchLeavesTheQueueAndTheRestIsKept(int status, string body, bool settled)
        {
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Flush();
            Answer(status, body);
            Assert.AreEqual(settled ? 0 : 1, queue.Count);
        }

        [Test]
        public void AKeptBatchIsSentAgainOnTheNextFlush()
        {
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Flush();
            Answer(HttpReply.NoResponse, null);
            queue.Flush();
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual(api.Requests[0].Body, api.Requests[1].Body);
        }

        [Test]
        public void ASignedOutSessionKeepsTheBatchWithoutARequest()
        {
            var signedOut = new ApiAnalyticsSender(new ApiClient(api, new AuthSession(new Auth(), new Identity(), new Store())));
            var held = new AnalyticsQueue(signedOut, Clock, 2, 4);
            held.Consent(AnalyticsId);
            held.Track(AnalyticsEvents.AppOpen);
            held.Flush();
            Assert.IsEmpty(api.Requests);
            Assert.AreEqual(1, held.Count);
        }

        [Test]
        public void SettledReadsTheErrorCodeOfARefusal()
        {
            Assert.IsTrue(ApiAnalyticsSender.Settled(new ApiReply(ApiOutcome.Rejected, 409, Error(ApiAnalyticsSender.IdTaken), ApiAnalyticsSender.IdTaken)));
            Assert.IsFalse(ApiAnalyticsSender.Settled(new ApiReply(ApiOutcome.Rejected, 409, Error("conflict"), "conflict")));
        }

        [Test]
        public void AFullBatchOfTheLongestValidEventsFitsTheServerBodyLimit()
        {
            var label = new string('a', 64);
            var batch = new List<AnalyticsEvent>();
            for (var i = 0; i < AnalyticsSettings.MaxBatchSize; i++) batch.Add(new AnalyticsEvent(AnalyticsEvents.PurchaseCancel, label, 0, long.MaxValue));
            var body = sender.Body(AnalyticsId, batch);
            Assert.LessOrEqual(Encoding.UTF8.GetByteCount(body), ServerMaxBodyBytes);
        }
    }
}
