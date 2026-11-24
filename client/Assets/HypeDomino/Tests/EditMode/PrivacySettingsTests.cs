using System;
using System.Collections.Generic;
using HypeDomino.Core.Analytics;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Game.Privacy;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PrivacySettingsTests
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
        PrivacySettings settings;
        List<PrivacySettingsResult> results;
        int changes;
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
            settings = new PrivacySettings(client, analytics, diagnostics, NextAnalyticsId, PolicyVersion, AppVersion);
            settings.Changed += () => changes++;
            results = new List<PrivacySettingsResult>();
            changes = 0;
        }

        string NextAnalyticsId() => "analytics-" + ++issuedIds;

        static string Json(bool analyticsOn, bool diagnosticsOn) =>
            "{\"analytics\":" + (analyticsOn ? "true" : "false") + ",\"diagnostics\":" + (diagnosticsOn ? "true" : "false") + "}";

        void LoadWith(bool analyticsOn, bool diagnosticsOn)
        {
            Assert.IsTrue(settings.Load(results.Add));
            api.Requests[api.Requests.Count - 1].Done(new HttpReply(200, Json(analyticsOn, diagnosticsOn)));
            results.Clear();
            changes = 0;
        }

        string SentAnalyticsId()
        {
            analytics.Track(AnalyticsEvents.AppOpen);
            analytics.Flush();
            return analyticsSender.Ids[analyticsSender.Ids.Count - 1];
        }

        [TestCase(null, AppVersion)]
        [TestCase("", AppVersion)]
        [TestCase(PolicyVersion, null)]
        [TestCase(PolicyVersion, "")]
        public void AConsentContextIsRequiredToBuildTheModel(string policyVersion, string appVersion)
        {
            Assert.Throws<ArgumentException>(() => new PrivacySettings(client, analytics, diagnostics, NextAnalyticsId, policyVersion, appVersion));
        }

        [Test]
        public void ARandomAnalyticsIdIsAUuidInItsLongForm()
        {
            var id = PrivacySettings.RandomAnalyticsId();
            StringAssert.IsMatch("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", id);
            Assert.AreNotEqual(id, PrivacySettings.RandomAnalyticsId());
        }

        [Test]
        public void LoadReadsTheServerChoicesAndOpensOnlyTheGrantedPurposes()
        {
            Assert.IsFalse(settings.Loaded);
            Assert.IsTrue(settings.Load(results.Add));
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/consents", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsTrue(settings.Busy);

            api.Requests[0].Done(new HttpReply(200, Json(true, false)));
            Assert.IsFalse(settings.Busy);
            Assert.IsTrue(settings.Loaded);
            Assert.IsTrue(settings.Analytics);
            Assert.IsFalse(settings.Diagnostics);
            Assert.IsTrue(analytics.Consented);
            Assert.IsFalse(diagnostics.Consented);
            Assert.AreEqual("analytics-1", SentAnalyticsId());
            Assert.AreEqual(PrivacySettingsOutcome.Done, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void NothingIsSavedBeforeTheChoicesAreLoaded()
        {
            Assert.IsFalse(settings.SetAnalytics(true, results.Add));
            Assert.IsFalse(settings.SetDiagnostics(true, results.Add));
            Assert.AreEqual(0, api.Requests.Count);
            Assert.IsFalse(analytics.Consented);
            Assert.IsFalse(diagnostics.Consented);
        }

        [Test]
        public void GrantingSendsBothPurposesWithTheConsentContextAndOpensOnlyOnTheServerAnswer()
        {
            LoadWith(false, false);
            Assert.IsTrue(settings.SetAnalytics(true, results.Add));
            Assert.AreEqual("PUT", api.Requests[1].Method);
            Assert.AreEqual("/consents", api.Requests[1].PathAndQuery);
            Assert.AreEqual("{\"analytics\":true,\"diagnostics\":false,\"policyVersion\":\"2026-11-01\",\"appVersion\":\"0.4.2\"}", api.Requests[1].Body);
            Assert.IsFalse(settings.Analytics);
            Assert.IsFalse(analytics.Consented);

            api.Requests[1].Done(new HttpReply(200, Json(true, false)));
            Assert.IsTrue(settings.Analytics);
            Assert.IsTrue(analytics.Consented);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void WithdrawingStopsAtOnceBeforeTheServerAnswers()
        {
            LoadWith(true, true);
            Assert.IsTrue(settings.SetAnalytics(false, results.Add));
            Assert.AreEqual("{\"analytics\":false,\"diagnostics\":true,\"policyVersion\":\"2026-11-01\",\"appVersion\":\"0.4.2\"}", api.Requests[1].Body);
            Assert.IsFalse(analytics.Consented);
            Assert.IsTrue(diagnostics.Consented);

            api.Requests[1].Done(new HttpReply(200, Json(false, true)));
            Assert.IsFalse(settings.Analytics);
            Assert.IsTrue(settings.Diagnostics);
            Assert.IsFalse(analytics.Consented);

            settings.SetDiagnostics(false, results.Add);
            Assert.IsFalse(diagnostics.Consented);
            Assert.AreEqual("{\"analytics\":false,\"diagnostics\":false,\"policyVersion\":\"2026-11-01\",\"appVersion\":\"0.4.2\"}", api.Requests[2].Body);
        }

        [Test]
        public void GrantingAgainAfterAWithdrawalUsesANewAnalyticsId()
        {
            LoadWith(true, false);
            Assert.AreEqual("analytics-1", SentAnalyticsId());

            settings.SetAnalytics(false, results.Add);
            api.Requests[1].Done(new HttpReply(200, Json(false, false)));
            settings.SetAnalytics(true, results.Add);
            api.Requests[2].Done(new HttpReply(200, Json(true, false)));

            Assert.AreEqual("analytics-2", SentAnalyticsId());
        }

        [Test]
        public void AnAnswerThatKeepsAnalyticsOnKeepsItsId()
        {
            LoadWith(true, false);
            settings.SetDiagnostics(true, results.Add);
            api.Requests[1].Done(new HttpReply(200, Json(true, true)));
            Assert.IsTrue(diagnostics.Consented);
            Assert.AreEqual(1, issuedIds);
            Assert.AreEqual("analytics-1", SentAnalyticsId());
        }

        [Test]
        public void TheGatesFollowTheServerAnswerNotTheTap()
        {
            LoadWith(false, false);
            settings.SetAnalytics(true, results.Add);
            api.Requests[1].Done(new HttpReply(200, Json(false, false)));
            Assert.IsFalse(settings.Analytics);
            Assert.IsFalse(analytics.Consented);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void ASecondRequestWhileOneIsInFlightSendsNothing()
        {
            LoadWith(false, false);
            Assert.IsTrue(settings.SetAnalytics(true, results.Add));
            Assert.IsTrue(settings.Busy);
            Assert.IsFalse(settings.SetDiagnostics(true, results.Add));
            Assert.IsFalse(settings.Load(results.Add));
            Assert.AreEqual(2, api.Requests.Count);

            api.Requests[1].Done(new HttpReply(200, Json(true, false)));
            Assert.IsFalse(settings.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.IsTrue(settings.SetDiagnostics(true, results.Add));
            Assert.AreEqual(3, api.Requests.Count);
        }

        [TestCase(400, "{\"error\":\"consent.malformed\"}", PrivacySettingsOutcome.Failed, "notice.generic")]
        [TestCase(400, "{\"error\":\"consent.context_required\"}", PrivacySettingsOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", PrivacySettingsOutcome.Failed, "notice.generic")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", PrivacySettingsOutcome.RateLimited, "privacy.settings.rate_limited")]
        [TestCase(HttpReply.NoResponse, null, PrivacySettingsOutcome.Offline, "privacy.settings.offline")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", PrivacySettingsOutcome.SignInRequired, null)]
        [TestCase(200, "not json", PrivacySettingsOutcome.Failed, "notice.generic")]
        [TestCase(200, "", PrivacySettingsOutcome.Failed, "notice.generic")]
        public void AFailedGrantKeepsTheLastServerChoicesAndOpensNothing(int status, string body, PrivacySettingsOutcome outcome, string key)
        {
            LoadWith(false, true);
            settings.SetAnalytics(true, results.Add);
            api.Requests[1].Done(new HttpReply(status, body));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsFalse(settings.Analytics);
            Assert.IsTrue(settings.Diagnostics);
            Assert.IsFalse(analytics.Consented);
            Assert.IsTrue(diagnostics.Consented);
            Assert.IsFalse(settings.Busy);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void AFailedWithdrawalStaysStoppedUntilTheServerSaysOtherwise()
        {
            LoadWith(true, false);
            settings.SetAnalytics(false, results.Add);
            api.Requests[1].Done(new HttpReply(HttpReply.NoResponse, null));
            Assert.AreEqual(PrivacySettingsOutcome.Offline, results[0].Outcome);
            Assert.IsTrue(settings.Analytics);
            Assert.IsFalse(analytics.Consented);
        }

        [TestCase(HttpReply.NoResponse, null, PrivacySettingsOutcome.Offline)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", PrivacySettingsOutcome.RateLimited)]
        [TestCase(500, "{\"error\":\"internal\"}", PrivacySettingsOutcome.Failed)]
        public void AFailedLoadLeavesTheChoicesUnloadedAndEveryPurposeOff(int status, string body, PrivacySettingsOutcome outcome)
        {
            settings.Load(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.IsFalse(settings.Loaded);
            Assert.IsFalse(analytics.Consented);
            Assert.IsFalse(diagnostics.Consented);
            Assert.IsFalse(settings.SetAnalytics(true, results.Add));
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void EveryOutcomeButDoneAndSignInHasAMessageKeyWithBuiltInTexts()
        {
            var strings = BuiltinStrings.Load();
            foreach (PrivacySettingsOutcome outcome in Enum.GetValues(typeof(PrivacySettingsOutcome)))
            {
                var key = PrivacySettingsKeys.Of(outcome);
                if (outcome == PrivacySettingsOutcome.Done || outcome == PrivacySettingsOutcome.SignInRequired)
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
