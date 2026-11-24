using System;
using System.Collections.Generic;
using HypeDomino.Game.Profile;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LanguageSyncTests
    {
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
        LanguageChoice choice;
        LanguageSync sync;

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
            choice = LanguageChoice.Load("az", Array.Empty<string>);
            sync = new LanguageSync(new ApiClient(api, session), choice);
        }

        static string Json(string language) => "{\"language\":\"" + language + "\"}";

        Sent Last => api.Requests[api.Requests.Count - 1];

        void AnswerRead(string language)
        {
            Assert.AreEqual("GET", Last.Method);
            Assert.AreEqual(LanguageSync.Path, Last.PathAndQuery);
            Last.Done(new HttpReply(200, Json(language)));
        }

        void AssertSaved(string language)
        {
            Assert.AreEqual("PUT", Last.Method);
            Assert.AreEqual(LanguageSync.Path, Last.PathAndQuery);
            Assert.AreEqual(Json(language), Last.Body);
        }

        [Test]
        public void AnAccountWithoutALanguageGetsTheDeviceChoice()
        {
            sync.SignedIn("acc-1");
            AnswerRead("");
            Assert.AreEqual(2, api.Requests.Count);
            AssertSaved("az");
            Last.Done(new HttpReply(200, Json("az")));
            Assert.IsFalse(sync.Busy);
            Assert.AreEqual("az", choice.Language);
        }

        [Test]
        public void AnAccountLanguageAndTheDeviceChoiceAreBothLeftAsTheyAre()
        {
            sync.SignedIn("acc-1");
            AnswerRead("ru");
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("az", choice.Language);
            Assert.IsFalse(sync.Busy);
        }

        [Test]
        public void ALanguageThatIsNotALaunchLanguageIsReplacedByTheDeviceChoice()
        {
            sync.SignedIn("acc-1");
            AnswerRead("de");
            AssertSaved("az");
        }

        [Test]
        public void AChangeAfterSignInIsSentToTheAccount()
        {
            sync.SignedIn("acc-1");
            AnswerRead("ru");
            Assert.IsTrue(choice.Choose("tr"));
            Assert.AreEqual(2, api.Requests.Count);
            AssertSaved("tr");
        }

        [Test]
        public void AChangeBeforeSignInIsNotSent()
        {
            Assert.IsTrue(choice.Choose("tr"));
            sync.SignedIn(null);
            sync.SignedIn(string.Empty);
            Assert.IsEmpty(api.Requests);
            sync.SignedIn("acc-1");
            AnswerRead("ru");
            Assert.AreEqual(1, api.Requests.Count);
        }

        [Test]
        public void AChangeDuringTheReadIsSentAfterIt()
        {
            sync.SignedIn("acc-1");
            Assert.IsTrue(choice.Choose("en"));
            Assert.AreEqual(1, api.Requests.Count);
            AnswerRead("ru");
            Assert.AreEqual(2, api.Requests.Count);
            AssertSaved("en");
        }

        [Test]
        public void ChangesDuringASaveAreSentAfterItAsTheLatestChoice()
        {
            sync.SignedIn("acc-1");
            AnswerRead("ru");
            choice.Choose("tr");
            var first = Last;
            choice.Choose("en");
            choice.Choose("ru");
            Assert.AreEqual(2, api.Requests.Count);
            first.Done(new HttpReply(200, Json("tr")));
            Assert.AreEqual(3, api.Requests.Count);
            AssertSaved("ru");
            Last.Done(new HttpReply(200, Json("ru")));
            Assert.AreEqual(3, api.Requests.Count);
        }

        [Test]
        public void ASignInOfTheSameAccountDoesNotReadAgain()
        {
            sync.SignedIn("acc-1");
            AnswerRead("ru");
            sync.SignedIn("acc-1");
            Assert.AreEqual(1, api.Requests.Count);
        }

        [Test]
        public void AFailedReadIsRetriedAtTheNextSignIn()
        {
            sync.SignedIn("acc-1");
            Last.Done(new HttpReply(HttpReply.NoResponse, null));
            Assert.IsFalse(sync.Busy);
            choice.Choose("tr");
            Assert.AreEqual(2, api.Requests.Count);
            Last.Done(new HttpReply(503, null));
            sync.SignedIn("acc-1");
            Assert.AreEqual(3, api.Requests.Count);
            AnswerRead("");
            AssertSaved("tr");
        }

        [Test]
        public void AFailedSaveIsRetriedAtTheNextSignIn()
        {
            sync.SignedIn("acc-1");
            AnswerRead("ru");
            choice.Choose("tr");
            Last.Done(new HttpReply(HttpReply.NoResponse, null));
            Assert.AreEqual(2, api.Requests.Count);
            sync.SignedIn("acc-1");
            Assert.AreEqual(3, api.Requests.Count);
            AssertSaved("tr");
        }

        [Test]
        public void ARejectedSaveIsNotRetried()
        {
            sync.SignedIn("acc-1");
            AnswerRead("ru");
            choice.Choose("tr");
            Last.Done(new HttpReply(400, "{\"error\":\"language.invalid\"}"));
            sync.SignedIn("acc-1");
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void AnotherAccountIsReadAgainAndAnAnswerForThePreviousOneIsIgnored()
        {
            sync.SignedIn("acc-1");
            var stale = Last;
            sync.SignedIn("acc-2");
            Assert.AreEqual(1, api.Requests.Count);
            stale.Done(new HttpReply(200, Json("")));
            Assert.AreEqual(2, api.Requests.Count);
            AnswerRead("ru");
            Assert.AreEqual(2, api.Requests.Count);
        }

        [TestCase(null, null)]
        [TestCase("", null)]
        [TestCase("not json", null)]
        [TestCase("{}", "")]
        [TestCase("{\"language\":\"tr\"}", "tr")]
        public void TheAccountLanguageIsReadFromTheReply(string body, string expected) =>
            Assert.AreEqual(expected, AccountLanguage.Read(body));
    }
}
