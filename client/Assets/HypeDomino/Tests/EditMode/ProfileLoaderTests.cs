using System;
using System.Collections.Generic;
using HypeDomino.Game.Profile;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Profile;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ProfileLoaderTests
    {
        const string OwnJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":""Leyla"",""setup_required"":false,""rename_required"":false,""rename_available_at"":""2026-12-01T10:00:00Z"",""level"":7,""xp"":1250,""loadout"":{""avatar"":""avatar.comet_fox"",""frame"":""frame.season_1_top""},""stats"":{""games"":40,""wins"":22,""losses"":18,""win_rate_percent"":55,""streak"":3,""best_streak"":6,""modes"":{""1v1"":{""games"":30,""wins"":17,""losses"":13}}}}";
        const string SetupJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":"""",""setup_required"":true,""rename_required"":false,""suggested_nickname"":""Player4821"",""level"":1,""xp"":0,""loadout"":{},""stats"":{""games"":0,""wins"":0,""losses"":0,""win_rate_percent"":0,""streak"":0,""best_streak"":0,""modes"":{}}}";
        const string PublicJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":""Leyla"",""level"":7,""loadout"":{""avatar"":""avatar.comet_fox"",""frame"":""frame.season_1_top""},""stats"":{""games"":40,""win_rate_percent"":55,""best_streak"":6},""ranked"":{""ranks"":{""1v1"":""Gold II""}}}";
        const string NamelessJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":"""",""setup_required"":false,""level"":1,""xp"":0,""loadout"":{},""stats"":{""modes"":{}}}";

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
        ProfileLoader loader;
        List<ProfileLoadResult> results;

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
            loader = new ProfileLoader(new ApiClient(api, session));
            results = new List<ProfileLoadResult>();
        }

        [Test]
        public void LoadReadsTheOwnProfileOfOneReply()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsTrue(loader.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/profile/me", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, OwnJson));
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(ProfileLoadOutcome.Loaded, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            var profile = results[0].Profile;
            Assert.IsTrue(profile.IsOwn);
            Assert.AreEqual("Leyla", profile.Nickname);
            Assert.AreEqual(7, profile.Level);
            Assert.AreEqual(1250, profile.Xp);
            Assert.AreEqual("avatar.comet_fox", profile.AvatarId);
            Assert.AreEqual(1, profile.ModeStatsCount);
            Assert.AreEqual(17, profile.ModeStats(0).Wins);
        }

        [Test]
        public void ASecondLoadWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsFalse(loader.Load(results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(200, OwnJson));
            Assert.IsTrue(loader.Load(results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void AnAccountWithoutAProfileAsksForSetupWithoutAProfile()
        {
            loader.Load(results.Add);
            api.Requests[0].Done(new HttpReply(200, SetupJson));

            Assert.AreEqual(ProfileLoadOutcome.SetupRequired, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.IsNull(results[0].Profile);
        }

        [TestCase(HttpReply.NoResponse, null, ProfileLoadOutcome.Offline, "profile.load.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", ProfileLoadOutcome.RateLimited, "profile.load.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", ProfileLoadOutcome.SignInRequired, null)]
        [TestCase(500, "{\"error\":\"internal\"}", ProfileLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "", ProfileLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", ProfileLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, NamelessJson, ProfileLoadOutcome.Failed, "notice.generic")]
        public void AFailedReplyEndsTheLoadWithoutAProfile(int status, string body, ProfileLoadOutcome outcome, string key)
        {
            loader.Load(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Profile);
        }

        [Test]
        public void LoadPublicReadsThePublicProfileOfTheAccount()
        {
            Assert.IsTrue(loader.LoadPublic("8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10", results.Add));
            Assert.IsTrue(loader.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/profile/8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10", api.Requests[0].PathAndQuery);
            Assert.IsNull(api.Requests[0].Body);

            api.Requests[0].Done(new HttpReply(200, PublicJson));
            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(ProfileLoadOutcome.Loaded, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            var profile = results[0].Profile;
            Assert.IsFalse(profile.IsOwn);
            Assert.AreEqual("Leyla", profile.Nickname);
            Assert.AreEqual(7, profile.Level);
            Assert.AreEqual(3, profile.StatCount);
            Assert.AreEqual(1, profile.RankCount);
            Assert.AreEqual("Gold II", profile.Rank(0).Label);
        }

        [Test]
        public void LoadPublicEscapesTheAccountAndSkipsAnEmptyOne()
        {
            Assert.IsFalse(loader.LoadPublic("", results.Add));
            Assert.IsFalse(loader.LoadPublic(null, results.Add));
            Assert.IsEmpty(api.Requests);

            loader.LoadPublic("a/b", results.Add);
            Assert.AreEqual("/profile/a%2Fb", api.Requests[0].PathAndQuery);
        }

        [Test]
        public void LoadPublicWhileALoadIsInFlightSendsNothing()
        {
            Assert.IsTrue(loader.Load(results.Add));
            Assert.IsFalse(loader.LoadPublic("acc-2", results.Add));
            Assert.AreEqual(1, api.Requests.Count);
        }

        [TestCase(404, "{\"error\":\"profile.unknown\"}", ProfileLoadOutcome.Unknown, "profile.load.unknown")]
        [TestCase(HttpReply.NoResponse, null, ProfileLoadOutcome.Offline, "profile.load.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", ProfileLoadOutcome.RateLimited, "profile.load.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", ProfileLoadOutcome.SignInRequired, null)]
        [TestCase(404, "{\"error\":\"not_found\"}", ProfileLoadOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", ProfileLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", ProfileLoadOutcome.Failed, "notice.generic")]
        [TestCase(200, NamelessJson, ProfileLoadOutcome.Failed, "notice.generic")]
        public void AFailedPublicReplyEndsTheLoadWithoutAProfile(int status, string body, ProfileLoadOutcome outcome, string key)
        {
            loader.LoadPublic("acc-2", results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(loader.Busy);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Profile);
        }

        [Test]
        public void TheShippedBundleHasEveryProfileLoadKeyInAzerbaijaniAndEnglish()
        {
            var keys = new HashSet<string>();
            foreach (ProfileLoadOutcome outcome in Enum.GetValues(typeof(ProfileLoadOutcome)))
            {
                var key = ProfileLoadKeys.Of(outcome);
                if (key != null) keys.Add(key);
            }
            Assert.AreEqual(4, keys.Count);
            var strings = BuiltinStrings.Load();
            foreach (var key in keys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
