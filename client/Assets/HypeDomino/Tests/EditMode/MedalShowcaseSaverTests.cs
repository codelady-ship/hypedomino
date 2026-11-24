using System;
using System.Collections.Generic;
using HypeDomino.Game.Medals;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Medals;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MedalShowcaseSaverTests
    {
        const string CollectionJson = @"{""version"":7,""slots"":3,""medals"":[{""id"":""medal.first_win"",""revision"":1,""nameKey"":""medal.first_win.name"",""descriptionKey"":""medal.first_win.description"",""rarityId"":""common"",""staticAddress"":""medal/first_win/static"",""sources"":[{""type"":""mission"",""ref"":""mission.first_win""}],""availability"":""live""},{""id"":""medal.wins_50"",""revision"":1,""nameKey"":""medal.wins_50.name"",""descriptionKey"":""medal.wins_50.description"",""rarityId"":""rare"",""staticAddress"":""medal/wins_50/static"",""sources"":[{""type"":""mission"",""ref"":""mission.win_50""}],""availability"":""live""}],""owned"":[{""medal"":""medal.first_win"",""earned_at"":""2026-11-01T10:00:00Z""},{""medal"":""medal.wins_50"",""earned_at"":""2026-11-05T10:00:00Z""}],""showcase"":[""medal.first_win""]}";

        static readonly string[] ShownKeys =
        {
            MedalShowcaseKeys.Invalid,
            MedalShowcaseKeys.NotOwned,
            MedalShowcaseKeys.Offline,
            MedalShowcaseKeys.RateLimited,
            MedalLoadKeys.Offline,
            MedalLoadKeys.RateLimited,
        };

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
        MedalShowcaseSaver saver;
        MedalCollection collection;
        List<MedalShowcaseResult> results;

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
            saver = new MedalShowcaseSaver(new ApiClient(api, session));
            collection = new MedalCollection();
            collection.Show(MedalCollectionReply.FromJson(CollectionJson));
            results = new List<MedalShowcaseResult>();
        }

        MedalShowcaseRequest AddWins50()
        {
            Assert.IsTrue(collection.TryAddToShowcase("medal.wins_50", out var request));
            return request;
        }

        [Test]
        public void SaveSendsTheIntentAndTheShowcaseChangesOnlyFromTheServersReply()
        {
            Assert.IsTrue(saver.Save(AddWins50(), results.Add));
            Assert.IsTrue(saver.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("PUT", api.Requests[0].Method);
            Assert.AreEqual("/medals/showcase", api.Requests[0].PathAndQuery);
            Assert.AreEqual(@"{""medals"":[""medal.first_win"",""medal.wins_50""]}", api.Requests[0].Body);
            Assert.IsEmpty(results);
            Assert.AreEqual(1, collection.ShowcaseCount);

            api.Requests[0].Done(new HttpReply(200, @"{""showcase"":[""medal.wins_50"",""medal.first_win""]}"));
            Assert.IsFalse(saver.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(MedalShowcaseOutcome.Saved, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);

            collection.Apply(results[0].Showcase);
            Assert.AreEqual(2, collection.ShowcaseCount);
            Assert.AreEqual("medal.wins_50", collection.ShowcaseAt(0).id);
            Assert.AreEqual("medal.first_win", collection.ShowcaseAt(1).id);
        }

        [Test]
        public void ASecondSaveWhileOneIsInFlightSendsNothing()
        {
            var request = AddWins50();
            Assert.IsTrue(saver.Save(request, results.Add));
            Assert.IsFalse(saver.Save(request, results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(409, "{\"error\":\"medal.not_owned\"}"));
            Assert.IsTrue(saver.Save(request, results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void NothingIsSentWithoutAShowcaseRequest()
        {
            Assert.IsFalse(saver.Save(null, results.Add));
            Assert.IsFalse(saver.Save(new MedalShowcaseRequest(), results.Add));

            Assert.IsFalse(saver.Busy);
            Assert.IsEmpty(api.Requests);
            Assert.IsEmpty(results);
        }

        [TestCase(HttpReply.NoResponse, null, MedalShowcaseOutcome.Offline, "medal.showcase.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", MedalShowcaseOutcome.RateLimited, "medal.showcase.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", MedalShowcaseOutcome.SignInRequired, null)]
        [TestCase(400, "{\"error\":\"medal.showcase_invalid\"}", MedalShowcaseOutcome.Invalid, "medal.showcase.invalid")]
        [TestCase(409, "{\"error\":\"medal.not_owned\"}", MedalShowcaseOutcome.NotOwned, "medal.showcase.not_owned")]
        [TestCase(400, "{\"error\":\"malformed\"}", MedalShowcaseOutcome.Failed, "notice.generic")]
        [TestCase(503, "{\"error\":\"medal.unavailable\"}", MedalShowcaseOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", MedalShowcaseOutcome.Failed, "notice.generic")]
        [TestCase(200, "", MedalShowcaseOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", MedalShowcaseOutcome.Failed, "notice.generic")]
        public void ARefusedSaveEndsWithoutAShowcase(int status, string body, MedalShowcaseOutcome outcome, string key)
        {
            saver.Save(AddWins50(), results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(saver.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Showcase);
            Assert.AreEqual(1, collection.ShowcaseCount);
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
