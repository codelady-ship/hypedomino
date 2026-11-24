using System;
using System.Collections.Generic;
using HypeDomino.Game.Privacy;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class FirstLaunchConsentsTests
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
        FirstLaunchConsents consents;
        List<FirstLaunchConsentsResult> results;

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
            consents = new FirstLaunchConsents(PolicyVersion, AppVersion);
            results = new List<FirstLaunchConsentsResult>();
        }

        const string AnalyticsOnThenDiagnosticsOnThenAnalyticsOff =
            "{\"choices\":[{\"purpose\":\"analytics\",\"granted\":true},{\"purpose\":\"diagnostics\",\"granted\":true},{\"purpose\":\"analytics\",\"granted\":false}],\"policyVersion\":\"2026-11-01\",\"appVersion\":\"0.4.2\"}";

        void ChooseThree()
        {
            consents.Choose(ConsentPurpose.Analytics, true);
            consents.Choose(ConsentPurpose.Diagnostics, true);
            consents.Choose(ConsentPurpose.Analytics, false);
        }

        [TestCase(null, AppVersion)]
        [TestCase("", AppVersion)]
        [TestCase(PolicyVersion, null)]
        [TestCase(PolicyVersion, "")]
        public void AConsentContextIsRequiredToHoldChoices(string policyVersion, string appVersion)
        {
            Assert.Throws<ArgumentException>(() => new FirstLaunchConsents(policyVersion, appVersion));
        }

        [Test]
        public void NothingIsSentWithoutAChoice()
        {
            Assert.IsFalse(consents.Write(client, results.Add));
            Assert.AreEqual(0, api.Requests.Count);
            Assert.IsEmpty(results);
        }

        [Test]
        public void TheChoicesAreSentInTheOrderTheyWereMadeWithTheConsentContext()
        {
            ChooseThree();
            Assert.AreEqual(3, consents.Held);
            Assert.IsTrue(consents.Write(client, results.Add));
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual("/consents/first-launch", api.Requests[0].PathAndQuery);
            Assert.AreEqual(AnalyticsOnThenDiagnosticsOnThenAnalyticsOff, api.Requests[0].Body);
            Assert.IsTrue(consents.Busy);
        }

        [Test]
        public void AWrittenAnswerDropsTheHeldChoicesAndCarriesTheServerState()
        {
            ChooseThree();
            consents.Write(client, results.Add);
            api.Requests[0].Done(new HttpReply(200, "{\"analytics\":false,\"diagnostics\":true}"));
            Assert.IsFalse(consents.Busy);
            Assert.AreEqual(0, consents.Held);
            Assert.AreEqual(FirstLaunchConsentsOutcome.Written, results[0].Outcome);
            Assert.IsFalse(results[0].Choices.analytics);
            Assert.IsTrue(results[0].Choices.diagnostics);
            Assert.IsFalse(consents.Write(client, results.Add));
        }

        [Test]
        public void AChoiceMadeWhileWritingIsKeptForTheNextWrite()
        {
            consents.Choose(ConsentPurpose.Analytics, true);
            consents.Write(client, results.Add);
            consents.Choose(ConsentPurpose.Diagnostics, true);
            Assert.IsFalse(consents.Write(client, results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(200, "{\"analytics\":true,\"diagnostics\":false}"));
            Assert.AreEqual(1, consents.Held);
            Assert.IsTrue(consents.Write(client, results.Add));
            Assert.AreEqual("{\"choices\":[{\"purpose\":\"diagnostics\",\"granted\":true}],\"policyVersion\":\"2026-11-01\",\"appVersion\":\"0.4.2\"}", api.Requests[1].Body);
        }

        [TestCase(400, "{\"error\":\"consent.malformed\"}", FirstLaunchConsentsOutcome.Failed)]
        [TestCase(400, "{\"error\":\"consent.context_required\"}", FirstLaunchConsentsOutcome.Failed)]
        [TestCase(500, "{\"error\":\"internal\"}", FirstLaunchConsentsOutcome.Failed)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", FirstLaunchConsentsOutcome.RateLimited)]
        [TestCase(HttpReply.NoResponse, null, FirstLaunchConsentsOutcome.Offline)]
        [TestCase(401, "{\"error\":\"access_rejected\"}", FirstLaunchConsentsOutcome.SignInRequired)]
        [TestCase(200, "not json", FirstLaunchConsentsOutcome.Failed)]
        [TestCase(200, "", FirstLaunchConsentsOutcome.Failed)]
        public void AFailedWriteKeepsEveryChoiceForTheNextWrite(int status, string body, FirstLaunchConsentsOutcome outcome)
        {
            ChooseThree();
            consents.Write(client, results.Add);
            api.Requests[0].Done(new HttpReply(status, body));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.IsNull(results[0].Choices);
            Assert.IsFalse(consents.Busy);
            Assert.AreEqual(3, consents.Held);

            Assert.IsTrue(consents.Write(client, results.Add));
            Assert.AreEqual(AnalyticsOnThenDiagnosticsOnThenAnalyticsOff, api.Requests[api.Requests.Count - 1].Body);
        }
    }
}
