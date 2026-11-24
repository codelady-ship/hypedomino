using System;
using System.Collections.Generic;
using HypeDomino.Game.Push;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PushSettingsTests
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

        Api api;
        ApiClient client;
        PushSettings settings;
        List<PushSettingsResult> results;
        int changes;

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
            settings = new PushSettings(client, PolicyVersion, AppVersion);
            settings.Changed += () => changes++;
            results = new List<PushSettingsResult>();
            changes = 0;
        }

        static string Json(bool gameplay, bool promotional) =>
            "{\"gameplay\":" + (gameplay ? "true" : "false") + ",\"promotional\":" + (promotional ? "true" : "false") + "}";

        void LoadWith(bool gameplay, bool promotional)
        {
            Assert.IsTrue(settings.Load(results.Add));
            api.Requests[api.Requests.Count - 1].Done(new HttpReply(200, Json(gameplay, promotional)));
            results.Clear();
            changes = 0;
        }

        [TestCase(null, AppVersion)]
        [TestCase("", AppVersion)]
        [TestCase(PolicyVersion, null)]
        [TestCase(PolicyVersion, "")]
        public void AConsentContextIsRequiredToBuildTheModel(string policyVersion, string appVersion)
        {
            Assert.Throws<ArgumentException>(() => new PushSettings(client, policyVersion, appVersion));
        }

        [Test]
        public void LoadReadsTheServerSettings()
        {
            Assert.IsFalse(settings.Loaded);
            Assert.IsTrue(settings.Load(results.Add));
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/push/settings", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsTrue(settings.Busy);

            api.Requests[0].Done(new HttpReply(200, Json(true, false)));
            Assert.IsFalse(settings.Busy);
            Assert.IsTrue(settings.Loaded);
            Assert.IsTrue(settings.Gameplay);
            Assert.IsFalse(settings.Promotional);
            Assert.AreEqual(PushSettingsOutcome.Done, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void NothingIsSavedBeforeTheSettingsAreLoaded()
        {
            Assert.IsFalse(settings.SetGameplay(false, results.Add));
            Assert.IsFalse(settings.SetPromotional(true, results.Add));
            Assert.AreEqual(0, api.Requests.Count);
            Assert.IsEmpty(results);
        }

        [Test]
        public void TogglingGameplaySendsBothCategoriesWithTheConsentContext()
        {
            LoadWith(true, true);
            Assert.IsTrue(settings.SetGameplay(false, results.Add));
            Assert.AreEqual("PUT", api.Requests[1].Method);
            Assert.AreEqual("/push/settings", api.Requests[1].PathAndQuery);
            Assert.AreEqual("{\"gameplay\":false,\"promotional\":true,\"policyVersion\":\"2026-11-01\",\"appVersion\":\"0.4.2\"}", api.Requests[1].Body);
            Assert.IsTrue(settings.Gameplay);

            api.Requests[1].Done(new HttpReply(200, Json(false, true)));
            Assert.IsFalse(settings.Gameplay);
            Assert.IsTrue(settings.Promotional);
            Assert.AreEqual(PushSettingsOutcome.Done, results[0].Outcome);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void TogglingPromotionalSendsThePolicyAndAppVersionTheConsentIsRecordedWith()
        {
            LoadWith(true, false);
            Assert.IsTrue(settings.SetPromotional(true, results.Add));
            Assert.AreEqual("{\"gameplay\":true,\"promotional\":true,\"policyVersion\":\"2026-11-01\",\"appVersion\":\"0.4.2\"}", api.Requests[1].Body);
            Assert.IsFalse(settings.Promotional);

            api.Requests[1].Done(new HttpReply(200, Json(true, true)));
            Assert.IsTrue(settings.Promotional);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void TheValuesShownAreTheOnesTheServerAnswered()
        {
            LoadWith(true, false);
            settings.SetPromotional(true, results.Add);
            api.Requests[1].Done(new HttpReply(200, Json(false, false)));
            Assert.IsFalse(settings.Gameplay);
            Assert.IsFalse(settings.Promotional);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void AnAnswerWithoutAChangeRaisesNothing()
        {
            LoadWith(true, false);
            settings.SetGameplay(true, results.Add);
            api.Requests[1].Done(new HttpReply(200, Json(true, false)));
            Assert.AreEqual(PushSettingsOutcome.Done, results[0].Outcome);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void ASecondRequestWhileOneIsInFlightSendsNothing()
        {
            LoadWith(true, false);
            Assert.IsTrue(settings.SetGameplay(false, results.Add));
            Assert.IsTrue(settings.Busy);
            Assert.IsFalse(settings.SetPromotional(true, results.Add));
            Assert.IsFalse(settings.Load(results.Add));
            Assert.AreEqual(2, api.Requests.Count);

            api.Requests[1].Done(new HttpReply(200, Json(false, false)));
            Assert.IsFalse(settings.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.IsTrue(settings.SetPromotional(true, results.Add));
            Assert.AreEqual(3, api.Requests.Count);
        }

        [TestCase(400, "{\"error\":\"push.malformed\"}", PushSettingsOutcome.Failed, "notice.generic")]
        [TestCase(400, "{\"error\":\"push.consent_context_required\"}", PushSettingsOutcome.Failed, "notice.generic")]
        [TestCase(400, "{\"error\":\"push.reason_from_a_newer_server\"}", PushSettingsOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", PushSettingsOutcome.Failed, "notice.generic")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", PushSettingsOutcome.RateLimited, "push.settings.rate_limited")]
        [TestCase(HttpReply.NoResponse, null, PushSettingsOutcome.Offline, "push.settings.offline")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", PushSettingsOutcome.SignInRequired, null)]
        [TestCase(200, "not json", PushSettingsOutcome.Failed, "notice.generic")]
        [TestCase(200, "", PushSettingsOutcome.Failed, "notice.generic")]
        public void AFailedSaveKeepsTheLastServerValues(int status, string body, PushSettingsOutcome outcome, string key)
        {
            LoadWith(true, false);
            settings.SetPromotional(true, results.Add);
            api.Requests[1].Done(new HttpReply(status, body));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsTrue(settings.Gameplay);
            Assert.IsFalse(settings.Promotional);
            Assert.IsFalse(settings.Busy);
            Assert.AreEqual(0, changes);
        }

        [TestCase(HttpReply.NoResponse, null, PushSettingsOutcome.Offline)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", PushSettingsOutcome.RateLimited)]
        [TestCase(500, "{\"error\":\"internal\"}", PushSettingsOutcome.Failed)]
        public void AFailedLoadLeavesTheSettingsUnloaded(int status, string body, PushSettingsOutcome outcome)
        {
            settings.Load(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.IsFalse(settings.Loaded);
            Assert.IsFalse(settings.SetGameplay(false, results.Add));
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void EveryOutcomeButDoneAndSignInHasAMessageKeyWithBuiltInTexts()
        {
            var strings = BuiltinStrings.Load();
            foreach (PushSettingsOutcome outcome in Enum.GetValues(typeof(PushSettingsOutcome)))
            {
                var key = PushSettingsKeys.Of(outcome);
                if (outcome == PushSettingsOutcome.Done || outcome == PushSettingsOutcome.SignInRequired)
                {
                    Assert.IsNull(key);
                    continue;
                }
                StringAssert.IsMatch("^[a-z0-9_]+(\\.[a-z0-9_]+)+$", key, outcome.ToString());
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
