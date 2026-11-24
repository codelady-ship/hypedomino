using System;
using System.Collections.Generic;
using HypeDomino.Core.Analytics;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Game.Privacy;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ConsentStartupTests
    {
        const string PolicyVersion = "2026-11-01";
        const string AppVersion = "0.4.2";

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

        sealed class AnalyticsSender : IAnalyticsSender
        {
            public readonly List<string> Ids = new();

            public void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done)
            {
                Ids.Add(analyticsId);
                done(true);
            }
        }

        sealed class ErrorSender : IErrorSender
        {
            public void Send(ErrorReport report)
            {
            }
        }

        Api api;
        ApiClient client;
        AnalyticsSender analyticsSender;
        AnalyticsQueue analytics;
        ErrorReports diagnostics;
        ConsentStartup startup;
        int issuedIds;

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
            client = new ApiClient(api, session);
            analyticsSender = new AnalyticsSender();
            analytics = new AnalyticsQueue(analyticsSender, () => 1000, 2, 4);
            diagnostics = new ErrorReports(new ErrorSender());
            issuedIds = 0;
            startup = new ConsentStartup(client, new ConsentGates(analytics, diagnostics, NextAnalyticsId));
        }

        string NextAnalyticsId() => "analytics-" + ++issuedIds;

        static string Json(bool analyticsOn, bool diagnosticsOn) =>
            "{\"analytics\":" + (analyticsOn ? "true" : "false") + ",\"diagnostics\":" + (diagnosticsOn ? "true" : "false") + "}";

        Sent Last => api.Requests[api.Requests.Count - 1];

        FirstLaunchConsents Chosen(bool analyticsOn, bool diagnosticsOn)
        {
            var choices = new FirstLaunchConsents(PolicyVersion, AppVersion);
            choices.Choose(ConsentPurpose.Diagnostics, diagnosticsOn);
            choices.Choose(ConsentPurpose.Analytics, analyticsOn);
            startup.Hold(choices);
            return choices;
        }

        string SentAnalyticsId()
        {
            analytics.Track(AnalyticsEvents.AppOpen);
            analytics.Flush();
            return analyticsSender.Ids[analyticsSender.Ids.Count - 1];
        }

        [Test]
        public void ItIsBuiltWithAnApiClientAndGates()
        {
            Assert.Throws<ArgumentNullException>(() => new ConsentStartup(null, new ConsentGates(analytics, diagnostics, NextAnalyticsId)));
            Assert.Throws<ArgumentNullException>(() => new ConsentStartup(client, null));
            Assert.Throws<ArgumentNullException>(() => new ConsentGates(null, diagnostics, NextAnalyticsId));
            Assert.Throws<ArgumentNullException>(() => new ConsentGates(analytics, null, NextAnalyticsId));
            Assert.Throws<ArgumentNullException>(() => new ConsentGates(analytics, diagnostics, null));
            Assert.Throws<ArgumentNullException>(() => startup.Hold(null));
        }

        [Test]
        public void EveryPurposeStaysOffUntilTheServerAnswers()
        {
            Assert.IsFalse(analytics.Consented);
            Assert.IsFalse(diagnostics.Consented);
            Assert.IsTrue(startup.SignedIn("acc-1"));
            Assert.IsTrue(startup.Busy);
            Assert.IsFalse(analytics.Consented);
            Assert.IsFalse(diagnostics.Consented);
            Assert.IsFalse(analytics.Track(AnalyticsEvents.AppOpen));
        }

        [TestCase(true, false)]
        [TestCase(false, true)]
        [TestCase(true, true)]
        [TestCase(false, false)]
        public void ASignInWithoutFirstLaunchChoicesAppliesTheStoredConsents(bool analyticsOn, bool diagnosticsOn)
        {
            Assert.IsTrue(startup.SignedIn("acc-1"));
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", Last.Method);
            Assert.AreEqual("/consents", Last.PathAndQuery);
            Assert.IsNull(Last.Body);

            Last.Done(new HttpReply(200, Json(analyticsOn, diagnosticsOn)));
            Assert.IsFalse(startup.Busy);
            Assert.AreEqual("acc-1", startup.AppliedAccount);
            Assert.AreEqual(analyticsOn, analytics.Consented);
            Assert.AreEqual(diagnosticsOn, diagnostics.Consented);
            if (analyticsOn) Assert.AreEqual("analytics-1", SentAnalyticsId());
        }

        [Test]
        public void FirstLaunchChoicesAreWrittenAtSignInInTheOrderTheyWereMade()
        {
            var choices = Chosen(true, true);
            Assert.IsTrue(startup.SignedIn("acc-1"));
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", Last.Method);
            Assert.AreEqual("/consents/first-launch", Last.PathAndQuery);
            Assert.AreEqual("{\"choices\":[{\"purpose\":\"diagnostics\",\"granted\":true},{\"purpose\":\"analytics\",\"granted\":true}],\"policyVersion\":\"2026-11-01\",\"appVersion\":\"0.4.2\"}", Last.Body);
            Assert.IsFalse(analytics.Consented);
            Assert.IsFalse(diagnostics.Consented);

            Last.Done(new HttpReply(200, Json(true, true)));
            Assert.AreEqual(0, choices.Held);
            Assert.IsFalse(startup.Busy);
            Assert.AreEqual("acc-1", startup.AppliedAccount);
            Assert.IsTrue(analytics.Consented);
            Assert.IsTrue(diagnostics.Consented);
            Assert.AreEqual("analytics-1", SentAnalyticsId());
        }

        [Test]
        public void TheServerRecordAppliesWhenTheAccountAlreadyHadOne()
        {
            Chosen(true, true);
            startup.SignedIn("acc-1");
            Last.Done(new HttpReply(200, Json(false, true)));
            Assert.IsFalse(analytics.Consented);
            Assert.IsTrue(diagnostics.Consented);
        }

        [TestCase(HttpReply.NoResponse, null)]
        [TestCase(429, "{\"error\":\"rate_limited\"}")]
        [TestCase(500, "{\"error\":\"internal\"}")]
        [TestCase(200, "not json")]
        public void AFailedWriteOpensNothingAndIsSentAgainAtTheNextSignIn(int status, string body)
        {
            var choices = Chosen(true, false);
            startup.SignedIn("acc-1");
            Last.Done(new HttpReply(status, body));
            Assert.IsFalse(startup.Busy);
            Assert.AreEqual(2, choices.Held);
            Assert.IsNull(startup.AppliedAccount);
            Assert.IsFalse(analytics.Consented);
            Assert.IsFalse(diagnostics.Consented);

            Assert.IsTrue(startup.SignedIn("acc-1"));
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("/consents/first-launch", Last.PathAndQuery);
            Last.Done(new HttpReply(200, Json(true, false)));
            Assert.AreEqual(0, choices.Held);
            Assert.IsTrue(analytics.Consented);
        }

        [Test]
        public void AReconnectOfTheSameAccountSendsNothing()
        {
            Chosen(true, false);
            startup.SignedIn("acc-1");
            Last.Done(new HttpReply(200, Json(true, false)));
            Assert.IsFalse(startup.SignedIn("acc-1"));
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("analytics-1", SentAnalyticsId());
        }

        [Test]
        public void AnotherAccountLoadsItsOwnRecordAndClosesWhatItDidNotGrant()
        {
            startup.SignedIn("acc-1");
            Last.Done(new HttpReply(200, Json(true, true)));

            Assert.IsTrue(startup.SignedIn("acc-2"));
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("/consents", Last.PathAndQuery);
            Last.Done(new HttpReply(200, Json(false, false)));
            Assert.AreEqual("acc-2", startup.AppliedAccount);
            Assert.IsFalse(analytics.Consented);
            Assert.IsFalse(diagnostics.Consented);
        }

        [TestCase(HttpReply.NoResponse, null)]
        [TestCase(429, "{\"error\":\"rate_limited\"}")]
        [TestCase(500, "{\"error\":\"internal\"}")]
        [TestCase(200, "")]
        public void AFailedLoadOpensNothingAndIsRetriedAtTheNextSignIn(int status, string body)
        {
            startup.SignedIn("acc-1");
            Last.Done(new HttpReply(status, body));
            Assert.IsFalse(startup.Busy);
            Assert.IsNull(startup.AppliedAccount);
            Assert.IsFalse(analytics.Consented);
            Assert.IsFalse(diagnostics.Consented);

            Assert.IsTrue(startup.SignedIn("acc-1"));
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("/consents", Last.PathAndQuery);
        }

        [Test]
        public void ASignInWhileARequestIsInFlightSendsNothing()
        {
            startup.SignedIn("acc-1");
            Assert.IsFalse(startup.SignedIn("acc-1"));
            Assert.IsFalse(startup.SignedIn("acc-2"));
            Assert.AreEqual(1, api.Requests.Count);
        }

        [Test]
        public void AnAnswerWithoutARequestLeavesNothingInFlight()
        {
            var signedOut = new ApiClient(api, new AuthSession(new Auth(), new Identity(), new Store()));
            startup = new ConsentStartup(signedOut, new ConsentGates(analytics, diagnostics, NextAnalyticsId));
            var choices = Chosen(true, true);

            Assert.IsTrue(startup.SignedIn("acc-1"));
            Assert.IsFalse(startup.Busy);
            Assert.AreEqual(2, choices.Held);
            Assert.AreEqual(0, api.Requests.Count);
            Assert.IsFalse(analytics.Consented);

            startup.Hold(new FirstLaunchConsents(PolicyVersion, AppVersion));
            Assert.IsTrue(startup.SignedIn("acc-1"));
            Assert.IsFalse(startup.Busy);
            Assert.IsNull(startup.AppliedAccount);
        }

        [TestCase(null)]
        [TestCase("")]
        public void NoAccountSendsNothing(string accountId)
        {
            Chosen(true, true);
            Assert.IsFalse(startup.SignedIn(accountId));
            Assert.AreEqual(0, api.Requests.Count);
        }

        [Test]
        public void TheGatesWithdrawOnlyThePurposesTurnedOff()
        {
            var gates = new ConsentGates(analytics, diagnostics, NextAnalyticsId);
            gates.Apply(ConsentChoicesReply.FromJson(Json(true, true)));
            Assert.IsTrue(analytics.Consented);
            Assert.IsTrue(diagnostics.Consented);
            gates.Withdraw(false, true);
            Assert.IsFalse(analytics.Consented);
            Assert.IsTrue(diagnostics.Consented);
            gates.Withdraw(true, false);
            Assert.IsFalse(diagnostics.Consented);
            Assert.Throws<ArgumentNullException>(() => gates.Apply(null));
        }
    }
}
