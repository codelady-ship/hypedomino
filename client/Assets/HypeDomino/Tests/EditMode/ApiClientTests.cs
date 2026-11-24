using System;
using System.Collections.Generic;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ApiClientTests
    {
        static readonly byte[] KeyOfBytes0To31 = BytesUpTo(32);

        static byte[] BytesUpTo(int count)
        {
            var bytes = new byte[count];
            for (var at = 0; at < count; at++) bytes[at] = (byte)at;
            return bytes;
        }

        static string Tokens(string access) =>
            "{\"access_token\":\"" + access + "\",\"refresh_token\":\"refresh-" + access + "\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(KeyOfBytes0To31) + "\",\"account_id\":\"acc-1\"}";

        sealed class Sent
        {
            public string Method;
            public string PathAndQuery;
            public string Body;
            public string Authorization;
            public string Mac;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, PathAndQuery = pathAndQuery, Body = body, Authorization = authorization, Mac = mac, Done = done });

            public void Answer(int at, int status, string body) => Requests[at].Done(new HttpReply(status, body));
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
        Auth auth;
        Store store;
        AuthSession session;
        ApiClient client;
        List<ApiReply> replies;

        [SetUp]
        public void Build()
        {
            api = new Api();
            auth = new Auth();
            store = new Store();
            session = new AuthSession(auth, new Identity(), store);
            client = new ApiClient(api, session);
            replies = new List<ApiReply>();
        }

        void SignIn(string access)
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            Refresh(200, Tokens(access));
        }

        void Refresh(int status, string body)
        {
            var done = auth.Refreshes[auth.Refreshes.Count - 1];
            done(new HttpReply(status, body));
        }

        void Keep(ApiReply reply) => replies.Add(reply);

        [Test]
        public void WithoutASessionNothingIsSent()
        {
            client.Get("/leaderboard/1v1", Keep);
            Assert.AreEqual(0, api.Requests.Count);
            Assert.AreEqual(ApiOutcome.SignInRequired, replies[0].Outcome);
        }

        [Test]
        public void AReadCarriesTheAccessTokenAndNoMac()
        {
            SignIn("access-1");
            client.Get("/leaderboard/1v1?board=friends", Keep);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("/leaderboard/1v1?board=friends", api.Requests[0].PathAndQuery);
            Assert.AreEqual("Bearer access-1", api.Requests[0].Authorization);
            Assert.IsNull(api.Requests[0].Mac);
            Assert.IsNull(api.Requests[0].Body);

            api.Answer(0, 200, "{\"top\":[]}");
            Assert.AreEqual(ApiOutcome.Ok, replies[0].Outcome);
            Assert.AreEqual("{\"top\":[]}", replies[0].Body);
        }

        [Test]
        public void AWriteIsSignedWithTheSessionRequestKeyAsTheServerChecksIt()
        {
            SignIn("access-1");
            client.Put("/push/settings?x=1", "{\"gameplay\":true,\"promotional\":false}", Keep);
            Assert.AreEqual("eCGAZTyVp8Jhxnh47f6vmwLa_kZa6-cld7IxBi945vg", api.Requests[0].Mac);
        }

        [Test]
        public void TheMacCoversTheBodyAsUtf8()
        {
            Assert.AreEqual("AIsiKhnaPaLK4Qvm4hcaf2X9qYyYyzkj2TgdtaDyd-s", RequestMac.Sign(KeyOfBytes0To31, "POST", "/shop", "acc", "{\"name\":\"Gün\"}"));
        }

        [Test]
        public void AnExpiredAccessTokenIsRefreshedOnceAndTheRequestResentWithTheNewOne()
        {
            SignIn("access-1");
            client.Post("/daily-login", "{}", Keep);
            api.Answer(0, 401, "{\"error\":\"access_expired\"}");
            Assert.AreEqual(0, replies.Count);
            Assert.AreEqual(2, auth.Refreshes.Count);

            Refresh(200, Tokens("access-2"));
            Assert.AreEqual("Bearer access-2", api.Requests[1].Authorization);
            Assert.AreNotEqual(api.Requests[0].Mac, api.Requests[1].Mac, "the MAC covers the access token");

            api.Answer(1, 200, "{}");
            Assert.AreEqual(ApiOutcome.Ok, replies[0].Outcome);
        }

        [Test]
        public void ASecondExpiryAfterTheRefreshEndsInSignInRequired()
        {
            SignIn("access-1");
            client.Get("/pass", Keep);
            api.Answer(0, 401, "{\"error\":\"access_expired\"}");
            Refresh(200, Tokens("access-2"));
            api.Answer(1, 401, "{\"error\":\"access_expired\"}");
            Assert.AreEqual(2, auth.Refreshes.Count);
            Assert.AreEqual(ApiOutcome.SignInRequired, replies[0].Outcome);
            Assert.AreEqual("access_expired", replies[0].Error);
        }

        [Test]
        public void RequestsDuringARefreshWaitForItAndShareIt()
        {
            SignIn("access-1");
            client.Get("/missions", Keep);
            client.Get("/medals", Keep);
            api.Answer(0, 401, "{\"error\":\"access_expired\"}");
            client.Put("/profile/country", "{\"country\":\"AZ\"}", Keep);
            api.Answer(1, 401, "{\"error\":\"access_expired\"}");
            Assert.AreEqual(2, auth.Refreshes.Count, "one refresh for every expired request");
            Assert.AreEqual(2, api.Requests.Count, "the new request waits for the refresh");

            Refresh(200, Tokens("access-2"));
            Assert.AreEqual(5, api.Requests.Count);
            for (var at = 2; at < 5; at++) Assert.AreEqual("Bearer access-2", api.Requests[at].Authorization);
            Assert.AreEqual("/missions", api.Requests[2].PathAndQuery);
            Assert.AreEqual("/profile/country", api.Requests[3].PathAndQuery);
            Assert.AreEqual("/medals", api.Requests[4].PathAndQuery);
        }

        [Test]
        public void AFailedRefreshAnswersEveryWaitingRequest()
        {
            SignIn("access-1");
            client.Get("/missions", Keep);
            client.Get("/medals", Keep);
            api.Answer(0, 401, "{\"error\":\"access_expired\"}");
            api.Answer(1, 401, "{\"error\":\"access_expired\"}");
            Refresh(HttpReply.NoResponse, null);
            Assert.AreEqual(2, replies.Count);
            Assert.AreEqual(ApiOutcome.Offline, replies[0].Outcome);
            Assert.AreEqual(ApiOutcome.Offline, replies[1].Outcome);
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void ARejectedRefreshAsksForSignIn()
        {
            SignIn("access-1");
            client.Get("/missions", Keep);
            api.Answer(0, 401, "{\"error\":\"access_expired\"}");
            Refresh(401, "{\"error\":\"refresh_rejected\"}");
            Assert.AreEqual(ApiOutcome.SignInRequired, replies[0].Outcome);
            Assert.IsFalse(session.SignedIn);
        }

        [TestCase(HttpReply.NoResponse, null, ApiOutcome.Offline)]
        [TestCase(204, null, ApiOutcome.Ok)]
        [TestCase(401, "{\"error\":\"access_rejected\"}", ApiOutcome.SignInRequired)]
        [TestCase(401, "{\"error\":\"bad_mac\"}", ApiOutcome.Rejected)]
        [TestCase(409, "{\"error\":\"ranked.emblem_not_owned\"}", ApiOutcome.Rejected)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", ApiOutcome.RateLimited)]
        [TestCase(503, "not json", ApiOutcome.Unavailable)]
        public void EachAnswerHasOneOutcome(int status, string body, ApiOutcome expected)
        {
            SignIn("access-1");
            client.Get("/wallet", Keep);
            api.Answer(0, status, body);
            Assert.AreEqual(expected, replies[0].Outcome);
            Assert.AreEqual(status, replies[0].Status);
        }

        [Test]
        public void TheErrorCodeOfARefusalIsKept()
        {
            SignIn("access-1");
            client.Put("/ranked/emblems/showcase", "{\"emblems\":[]}", Keep);
            api.Answer(0, 409, "{\"error\":\"ranked.emblem_not_owned\"}");
            Assert.AreEqual("ranked.emblem_not_owned", replies[0].Error);
        }
    }
}
