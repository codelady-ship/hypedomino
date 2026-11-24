using System;
using System.Collections.Generic;
using HypeDomino.Core.Analytics;
using HypeDomino.Game.Profile;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class NicknameSaverTests
    {
        const string RenamedJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":""Leyla07"",""setup_required"":false,""rename_required"":false,
""rename_available_at"":""2026-12-01T10:00:00Z"",""level"":7,""xp"":1250,""loadout"":{""avatar"":""avatar.comet_fox"",""frame"":""frame.season_1_top""},
""stats"":{""games"":40,""wins"":22,""losses"":18,""win_rate_percent"":55,""streak"":3,""best_streak"":6,""modes"":{}}}";

        const ulong RetryAtUnixMs = 1796119200000;

        static readonly string[] ShownKeys =
        {
            NicknameSaveKeys.InvalidLength,
            NicknameSaveKeys.InvalidCharacters,
            NicknameSaveKeys.InvalidSeparators,
            NicknameSaveKeys.MixedScripts,
            NicknameSaveKeys.DigitsOnly,
            NicknameSaveKeys.Taken,
            NicknameSaveKeys.Unchanged,
            NicknameSaveKeys.NotAllowed,
            NicknameSaveKeys.Cooldown,
            NicknameSaveKeys.Offline,
            NicknameSaveKeys.RateLimited,
            NicknameSaveKeys.Failed,
        };

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

        sealed class AnalyticsSink : IAnalyticsSender
        {
            public readonly List<AnalyticsEvent> Sent = new();

            public void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done)
            {
                for (var i = 0; i < batch.Count; i++) Sent.Add(batch[i]);
                done(true);
            }
        }

        Api api;
        AnalyticsSink analytics;
        AnalyticsQueue events;
        NicknameSaver saver;
        List<NicknameSaveResult> results;

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
            analytics = new AnalyticsSink();
            events = new AnalyticsQueue(analytics, () => 1700000000000, 10, 10);
            events.Consent("3f2b8c1e-5a4d-4e8f-9b7a-1c2d3e4f5a6b");
            saver = new NicknameSaver(new ApiClient(api, session), events);
            results = new List<NicknameSaveResult>();
        }

        [Test]
        public void SaveSendsTheTypedNicknameAndTheNewNameComesFromTheServersProfile()
        {
            Assert.IsTrue(saver.Save(" Leyla07 ", results.Add));
            Assert.IsTrue(saver.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("PUT", api.Requests[0].Method);
            Assert.AreEqual("/profile/nickname", api.Requests[0].PathAndQuery);
            Assert.AreEqual(@"{""nickname"":"" Leyla07 ""}", api.Requests[0].Body);
            Assert.IsNotEmpty(api.Requests[0].Mac);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, RenamedJson));
            Assert.IsFalse(saver.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(NicknameSaveOutcome.Saved, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.AreEqual("Leyla07", results[0].Profile.Nickname);
            Assert.IsTrue(results[0].Profile.IsOwn);
            Assert.AreEqual(RetryAtUnixMs, results[0].Profile.RenameAvailableUnixMs);
        }

        [Test]
        public void AnAcceptedSaveTracksNicknameSetWithoutTheNickname()
        {
            Assert.IsTrue(saver.Save("Leyla07", results.Add));
            Assert.AreEqual(0, events.Count);

            api.Requests[0].Done(new HttpReply(200, RenamedJson));
            events.Flush();

            Assert.AreEqual(1, analytics.Sent.Count);
            Assert.AreEqual(AnalyticsEvents.NicknameSet, analytics.Sent[0].Name);
            Assert.IsNull(analytics.Sent[0].Label);
            Assert.AreEqual(0, analytics.Sent[0].Value);
        }

        [TestCase(409, "{\"error\":\"nickname.taken\"}")]
        [TestCase(200, "not json")]
        [TestCase(500, "{\"error\":\"internal\"}")]
        public void ARefusedOrUnreadableSaveTracksNothing(int status, string body)
        {
            Assert.IsTrue(saver.Save("Leyla07", results.Add));

            api.Requests[0].Done(new HttpReply(status, body));

            Assert.AreNotEqual(NicknameSaveOutcome.Saved, results[0].Outcome);
            Assert.AreEqual(0, events.Count);
        }

        [Test]
        public void ASaveWithoutAnalyticsConsentTracksNothing()
        {
            events.Withdraw();
            Assert.IsTrue(saver.Save("Leyla07", results.Add));

            api.Requests[0].Done(new HttpReply(200, RenamedJson));

            Assert.AreEqual(NicknameSaveOutcome.Saved, results[0].Outcome);
            Assert.AreEqual(0, events.Count);
        }

        [Test]
        public void ASecondSaveWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(saver.Save("Leyla07", results.Add));
            Assert.IsFalse(saver.Save("Leyla08", results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(409, "{\"error\":\"nickname.taken\"}"));
            Assert.IsTrue(saver.Save("Leyla08", results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("   ")]
        public void ABlankNicknameIsNotSent(string nickname)
        {
            Assert.IsFalse(saver.Save(nickname, results.Add));

            Assert.IsFalse(saver.Busy);
            Assert.IsEmpty(api.Requests);
            Assert.IsEmpty(results);
        }

        [Test]
        public void ACooldownRefusalCarriesTheServersRetryTime()
        {
            saver.Save("Leyla07", results.Add);
            api.Requests[0].Done(new HttpReply(429, @"{""error"":""nickname.cooldown"",""retry_at"":""2026-12-01T10:00:00Z""}"));

            Assert.AreEqual(NicknameSaveOutcome.Cooldown, results[0].Outcome);
            Assert.AreEqual("nickname.rename.cooldown", results[0].MessageKey);
            Assert.AreEqual(RetryAtUnixMs, results[0].RetryAtUnixMs);
            Assert.IsNull(results[0].Profile);
        }

        [TestCase(HttpReply.NoResponse, null, NicknameSaveOutcome.Offline, "nickname.rename.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", NicknameSaveOutcome.RateLimited, "nickname.rename.rate_limited")]
        [TestCase(429, "{\"error\":\"nickname.cooldown\"}", NicknameSaveOutcome.Cooldown, "nickname.rename.cooldown")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", NicknameSaveOutcome.SignInRequired, null)]
        [TestCase(400, "{\"error\":\"nickname.invalid_length\"}", NicknameSaveOutcome.InvalidLength, "nickname.rename.invalid_length")]
        [TestCase(400, "{\"error\":\"nickname.invalid_characters\"}", NicknameSaveOutcome.InvalidCharacters, "nickname.rename.invalid_characters")]
        [TestCase(400, "{\"error\":\"nickname.invalid_separators\"}", NicknameSaveOutcome.InvalidSeparators, "nickname.rename.invalid_separators")]
        [TestCase(400, "{\"error\":\"nickname.mixed_scripts\"}", NicknameSaveOutcome.MixedScripts, "nickname.rename.mixed_scripts")]
        [TestCase(400, "{\"error\":\"nickname.digits_only\"}", NicknameSaveOutcome.DigitsOnly, "nickname.rename.digits_only")]
        [TestCase(400, "{\"error\":\"nickname.unchanged\"}", NicknameSaveOutcome.Unchanged, "nickname.rename.unchanged")]
        [TestCase(400, "{\"error\":\"nickname.not_allowed\"}", NicknameSaveOutcome.NotAllowed, "nickname.rename.not_allowed")]
        [TestCase(409, "{\"error\":\"nickname.taken\"}", NicknameSaveOutcome.Taken, "nickname.rename.taken")]
        [TestCase(400, "{\"error\":\"malformed\"}", NicknameSaveOutcome.Failed, "notice.generic")]
        [TestCase(400, "{\"error\":\"nickname.newer_reason\"}", NicknameSaveOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", NicknameSaveOutcome.Failed, "notice.generic")]
        [TestCase(200, "", NicknameSaveOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", NicknameSaveOutcome.Failed, "notice.generic")]
        [TestCase(200, "{\"account_id\":\"acc-1\",\"nickname\":\"\",\"setup_required\":true}", NicknameSaveOutcome.Failed, "notice.generic")]
        public void ARefusedSaveEndsWithoutAProfile(int status, string body, NicknameSaveOutcome outcome, string key)
        {
            saver.Save("Leyla07", results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(saver.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Profile);
        }

        [Test]
        public void TheShippedBundleHasEveryShownKeyInAzerbaijaniAndEnglish()
        {
            Assert.AreEqual(ShownKeys.Length, new HashSet<string>(ShownKeys).Count);
            var strings = BuiltinStrings.Load();
            foreach (var key in ShownKeys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
