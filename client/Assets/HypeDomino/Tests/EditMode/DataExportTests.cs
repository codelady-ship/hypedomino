using System;
using System.Collections.Generic;
using HypeDomino.Game.Privacy;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class DataExportTests
    {
        const string Link = "https://store.example/exports/abc.json?X-Amz-Signature=1";
        const string ExpiresAt = "2026-11-24T10:00:00.123456789Z";
        const ulong ExpiresAtUnixMs = 1795514400123;
        const string RetryAt = "2026-11-28T10:00:00Z";
        const ulong RetryAtUnixMs = 1795860000000;

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
        DataExport export;
        List<DataExportResult> results;

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
            export = new DataExport(new ApiClient(api, session));
            results = new List<DataExportResult>();
        }

        static string ReadyJson(string url, string expiresAt) => "{\"url\":\"" + url + "\",\"expires_at\":\"" + expiresAt + "\"}";

        [Test]
        public void TheModelNeedsAnApiClient()
        {
            Assert.Throws<ArgumentNullException>(() => new DataExport(null));
        }

        [Test]
        public void ARequestPostsASignedEmptyExportRequest()
        {
            Assert.IsTrue(export.Request(results.Add));
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual("/account/export", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsNotEmpty(api.Requests[0].Mac);
            Assert.IsTrue(export.Busy);
            Assert.AreEqual(0, results.Count);
        }

        [Test]
        public void AReadyExportCarriesTheLinkAndWhenItExpires()
        {
            export.Request(results.Add);
            api.Requests[0].Done(new HttpReply(201, ReadyJson(Link, ExpiresAt)));
            Assert.IsFalse(export.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(DataExportOutcome.Ready, results[0].Outcome);
            Assert.AreEqual(Link, results[0].Url);
            Assert.AreEqual(ExpiresAtUnixMs, results[0].ExpiresAtUnixMs);
            Assert.IsNull(results[0].MessageKey);
        }

        [Test]
        public void ACooldownCarriesWhenTheNextRequestIsAllowed()
        {
            export.Request(results.Add);
            api.Requests[0].Done(new HttpReply(429, "{\"error\":\"export_cooldown\",\"retry_at\":\"" + RetryAt + "\"}"));
            Assert.AreEqual(DataExportOutcome.Cooldown, results[0].Outcome);
            Assert.AreEqual(RetryAtUnixMs, results[0].RetryAtUnixMs);
            Assert.IsNull(results[0].Url);
            Assert.AreEqual("account.export.cooldown", results[0].MessageKey);
        }

        [Test]
        public void ACooldownWithoutARetryTimeIsStillACooldown()
        {
            export.Request(results.Add);
            api.Requests[0].Done(new HttpReply(429, "{\"error\":\"export_cooldown\"}"));
            Assert.AreEqual(DataExportOutcome.Cooldown, results[0].Outcome);
            Assert.AreEqual(0UL, results[0].RetryAtUnixMs);
        }

        [TestCase(503, "{\"error\":\"export_unavailable\"}", DataExportOutcome.Unavailable, "account.export.unavailable")]
        [TestCase(500, "", DataExportOutcome.Unavailable, "account.export.unavailable")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", DataExportOutcome.RateLimited, "account.export.rate_limited")]
        [TestCase(HttpReply.NoResponse, null, DataExportOutcome.Offline, "account.export.offline")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", DataExportOutcome.SignInRequired, null)]
        [TestCase(404, "404 page not found", DataExportOutcome.Failed, "notice.generic")]
        [TestCase(201, "not json", DataExportOutcome.Failed, "notice.generic")]
        [TestCase(201, "", DataExportOutcome.Failed, "notice.generic")]
        [TestCase(201, "{\"expires_at\":\"2026-11-24T10:00:00Z\"}", DataExportOutcome.Failed, "notice.generic")]
        [TestCase(201, "{\"url\":\"https://store.example/exports/abc.json\"}", DataExportOutcome.Failed, "notice.generic")]
        [TestCase(201, "{\"url\":\"https://store.example/exports/abc.json\",\"expires_at\":\"soon\"}", DataExportOutcome.Failed, "notice.generic")]
        public void EveryOtherAnswerHasItsOutcomeAndNoLink(int status, string body, DataExportOutcome outcome, string key)
        {
            export.Request(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Url);
            Assert.IsFalse(export.Busy);
        }

        [TestCase("javascript:alert(1)")]
        [TestCase("intent://store.example/exports/abc.json")]
        [TestCase("file:///data/exports/abc.json")]
        [TestCase("exports/abc.json")]
        public void ALinkThatIsNotAWebAddressIsNeverHandedOn(string url)
        {
            export.Request(results.Add);
            api.Requests[0].Done(new HttpReply(201, ReadyJson(url, ExpiresAt)));
            Assert.AreEqual(DataExportOutcome.Failed, results[0].Outcome);
            Assert.IsNull(results[0].Url);
        }

        [Test]
        public void ASecondRequestWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(export.Request(results.Add));
            Assert.IsFalse(export.Request(results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(503, "{\"error\":\"export_unavailable\"}"));
            Assert.IsTrue(export.Request(results.Add));
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual(1, results.Count);
        }

        [Test]
        public void EveryOutcomeButReadyAndSignInHasAMessageKeyWithBuiltInTexts()
        {
            var strings = BuiltinStrings.Load();
            foreach (DataExportOutcome outcome in Enum.GetValues(typeof(DataExportOutcome)))
            {
                var key = DataExportKeys.Of(outcome);
                if (outcome == DataExportOutcome.Ready || outcome == DataExportOutcome.SignInRequired)
                {
                    Assert.IsNull(key);
                    continue;
                }
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
