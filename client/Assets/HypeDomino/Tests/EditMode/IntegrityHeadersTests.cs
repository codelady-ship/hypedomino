using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.Net.Integrity;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class IntegrityHeadersTests
    {
        sealed class Sent
        {
            public string PathAndQuery;
            public string Authorization;
            public string Mac;
            public IReadOnlyList<ApiHeader> Headers;
            public Action<HttpReply> Done;
        }

        sealed class HeaderApi : IApiHeaderTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { PathAndQuery = pathAndQuery, Authorization = authorization, Mac = mac, Done = done });

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, IReadOnlyList<ApiHeader> headers, Action<HttpReply> done) =>
                Requests.Add(new Sent { PathAndQuery = pathAndQuery, Authorization = authorization, Mac = mac, Headers = headers, Done = done });
        }

        sealed class PlainApi : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { PathAndQuery = pathAndQuery, Authorization = authorization, Mac = mac, Done = done });
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

        static string Tokens(int n) => "{\"access_token\":\"access-" + n + "\",\"refresh_token\":\"refresh-" + n + "\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}";

        Auth auth;
        AuthSession session;
        List<ApiReply> replies;

        [SetUp]
        public void SignIn()
        {
            auth = new Auth();
            var store = new Store();
            session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, Tokens(1)));
            replies = new List<ApiReply>();
        }

        [Test]
        public void APlayIntegrityProofIsOneHeader()
        {
            var headers = IntegrityHeaders.Of(new IntegrityProof { PlayIntegrityToken = "play-token" });
            Assert.AreEqual(1, headers.Length);
            Assert.AreEqual("X-HD-Play-Integrity", headers[0].Name);
            Assert.AreEqual("play-token", headers[0].Value);
        }

        [Test]
        public void AnAppAttestAssertionIsTheKeyTheNonceAndTheAssertionInStandardBase64()
        {
            var proof = new IntegrityProof
            {
                AppAttest = new AppAttestAssertion
                {
                    KeyId = ByteString.CopyFrom(0xfb, 0xff, 0x01),
                    Nonce = "nonce-1",
                    Assertion = ByteString.CopyFrom(0x3e, 0x3f, 0x40, 0x41),
                },
            };
            var headers = IntegrityHeaders.Of(proof);
            Assert.AreEqual(3, headers.Length);
            Assert.AreEqual("X-HD-Attest-Key", headers[0].Name);
            Assert.AreEqual("+/8B", headers[0].Value);
            Assert.AreEqual("X-HD-Attest-Nonce", headers[1].Name);
            Assert.AreEqual("nonce-1", headers[1].Value);
            Assert.AreEqual("X-HD-Attest-Assertion", headers[2].Name);
            Assert.AreEqual("Pj9AQQ==", headers[2].Value);
        }

        [Test]
        public void NoProofIsNoHeaders()
        {
            Assert.IsNull(IntegrityHeaders.Of(null));
            Assert.IsNull(IntegrityHeaders.Of(new IntegrityProof()));
            Assert.IsNull(IntegrityHeaders.Of(new IntegrityProof { PlayIntegrityToken = "" }));
        }

        [Test]
        public void ExtraHeadersGoWithTheRequestAndAgainAfterARefresh()
        {
            var api = new HeaderApi();
            var client = new ApiClient(api, session);
            var headers = new[] { new ApiHeader("X-HD-Play-Integrity", "play-token") };
            client.Post("/shop/complete", "{}", headers, replies.Add);
            Assert.AreSame(headers, api.Requests[0].Headers);
            Assert.IsNotEmpty(api.Requests[0].Mac);

            api.Requests[0].Done(new HttpReply(401, "{\"error\":\"access_expired\"}"));
            auth.Refreshes[1](new HttpReply(200, Tokens(2)));
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("Bearer access-2", api.Requests[1].Authorization);
            Assert.AreSame(headers, api.Requests[1].Headers);

            api.Requests[1].Done(new HttpReply(200, "{}"));
            Assert.AreEqual(ApiOutcome.Ok, replies[0].Outcome);
        }

        [Test]
        public void ARequestWithoutExtraHeadersUsesThePlainSend()
        {
            var api = new HeaderApi();
            new ApiClient(api, session).Post("/shop/begin", "{}", replies.Add);
            Assert.IsNull(api.Requests[0].Headers);
        }

        [Test]
        public void ATransportWithoutExtraHeadersStillSendsTheRequest()
        {
            var api = new PlainApi();
            new ApiClient(api, session).Post("/shop/complete", "{}", new[] { new ApiHeader("X-HD-Play-Integrity", "play-token") }, replies.Add);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("/shop/complete", api.Requests[0].PathAndQuery);
            Assert.IsNotEmpty(api.Requests[0].Mac);
        }
    }
}
