using System;
using System.Collections.Generic;
using HypeDomino.Game.Profile;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PlayerReportTests
    {
        const string Target = "8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10";
        const string Match = "3b0e7c1a-9f2d-4c55-8a61-2d4e9b7f0c13";

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
        PlayerReport report;
        List<PlayerReportResult> results;

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
            report = new PlayerReport(new ApiClient(api, session));
            results = new List<PlayerReportResult>();
        }

        [TestCase(PlayerReportReason.OffensiveNickname, "offensive_nickname")]
        [TestCase(PlayerReportReason.OffensiveRoomName, "offensive_room_name")]
        [TestCase(PlayerReportReason.Impersonation, "impersonation")]
        [TestCase(PlayerReportReason.Other, "other")]
        public void SendPostsTheTargetTheReasonAndTheMatch(PlayerReportReason reason, string wire)
        {
            Assert.IsTrue(report.Send(Target, reason, Match, results.Add));
            Assert.IsTrue(report.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual("/reports", api.Requests[0].PathAndQuery);
            Assert.AreEqual("{\"target\":\"" + Target + "\",\"reason\":\"" + wire + "\",\"match_id\":\"" + Match + "\"}", api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(202, "{\"status\":\"received\"}"));
            Assert.IsFalse(report.Busy);
            Assert.AreEqual(PlayerReportOutcome.Received, results[0].Outcome);
            Assert.AreEqual("report.send.received", results[0].MessageKey);
        }

        [Test]
        public void AReportOutsideAMatchSendsAnEmptyMatch()
        {
            report.Send(Target, PlayerReportReason.Other, null, results.Add);

            Assert.AreEqual("{\"target\":\"" + Target + "\",\"reason\":\"other\",\"match_id\":\"\"}", api.Requests[0].Body);
        }

        [Test]
        public void AnEmptyTargetOrASecondReportInFlightSendsNothing()
        {
            Assert.IsFalse(report.Send("", PlayerReportReason.Other, null, results.Add));
            Assert.IsFalse(report.Send(null, PlayerReportReason.Other, null, results.Add));
            Assert.IsEmpty(api.Requests);

            Assert.IsTrue(report.Send(Target, PlayerReportReason.Other, null, results.Add));
            Assert.IsFalse(report.Send(Target, PlayerReportReason.Impersonation, null, results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(202, "{\"status\":\"received\"}"));
            Assert.IsTrue(report.Send(Target, PlayerReportReason.Impersonation, null, results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [TestCase(404, "{\"error\":\"report.unknown_player\"}", PlayerReportOutcome.UnknownPlayer, "report.send.unknown_player")]
        [TestCase(400, "{\"error\":\"report.self\"}", PlayerReportOutcome.Self, "report.send.self")]
        [TestCase(HttpReply.NoResponse, null, PlayerReportOutcome.Offline, "report.send.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", PlayerReportOutcome.RateLimited, "report.send.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", PlayerReportOutcome.SignInRequired, null)]
        [TestCase(400, "{\"error\":\"report.reason\"}", PlayerReportOutcome.Failed, "notice.generic")]
        [TestCase(400, "{\"error\":\"report.match\"}", PlayerReportOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", PlayerReportOutcome.Failed, "notice.generic")]
        public void ARefusedReportMapsToItsKey(int status, string body, PlayerReportOutcome outcome, string key)
        {
            report.Send(Target, PlayerReportReason.OffensiveNickname, null, results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(report.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
        }

        [Test]
        public void TheShippedBundleHasEveryReportKeyInAzerbaijaniAndEnglish()
        {
            var keys = new HashSet<string>();
            foreach (PlayerReportOutcome outcome in Enum.GetValues(typeof(PlayerReportOutcome)))
            {
                var key = PlayerReportKeys.Of(outcome);
                if (key != null) keys.Add(key);
            }
            Assert.AreEqual(6, keys.Count);
            var strings = BuiltinStrings.Load();
            foreach (var key in keys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
