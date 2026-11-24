using System;
using System.Collections.Generic;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.Net.Integrity;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class IntegrityProofsTests
    {
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

        sealed class Attestor : IDeviceAttestor
        {
            public readonly List<string> Nonces = new();

            public bool CanAttest { get; set; } = true;

            public void Attest(string nonce, Action<IntegrityProof> done)
            {
                Nonces.Add(nonce);
                done(new IntegrityProof { PlayIntegrityToken = "token-for-" + nonce });
            }
        }

        Api api;
        Attestor attestor;
        IntegrityProofs proofs;
        List<IntegrityProof> answers;

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
            attestor = new Attestor();
            proofs = new IntegrityProofs(new ApiClient(api, session), attestor);
            answers = new List<IntegrityProof>();
        }

        void Keep(IntegrityProof proof) => answers.Add(proof);

        [TestCase(IntegrityPurpose.SessionStart, "session_start")]
        [TestCase(IntegrityPurpose.Purchase, "purchase")]
        [TestCase(IntegrityPurpose.RankedMatch, "ranked_match")]
        [TestCase(IntegrityPurpose.CasualMatch, "casual_match")]
        public void ANonceIsAskedForThePurposeTheServerKnows(IntegrityPurpose purpose, string wire)
        {
            proofs.Request(purpose, Keep);

            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual(IntegrityProofs.NoncePath, api.Requests[0].PathAndQuery);
            Assert.AreEqual("{\"purpose\":\"" + wire + "\"}", api.Requests[0].Body);
            Assert.IsNotNull(api.Requests[0].Mac);
            Assert.AreEqual(0, answers.Count);
        }

        [Test]
        public void TheIssuedNonceIsAttestedIntoTheProof()
        {
            proofs.Request(IntegrityPurpose.SessionStart, Keep);
            api.Requests[0].Done(new HttpReply(200, "{\"nonce\":\"n-1\",\"expires_in\":300}"));

            CollectionAssert.AreEqual(new[] { "n-1" }, attestor.Nonces);
            Assert.AreEqual(1, answers.Count);
            Assert.AreEqual("token-for-n-1", answers[0].PlayIntegrityToken);
        }

        [Test]
        public void EveryRequestAttestsItsOwnNonce()
        {
            proofs.Request(IntegrityPurpose.RankedMatch, Keep);
            proofs.Request(IntegrityPurpose.Purchase, Keep);
            api.Requests[1].Done(new HttpReply(200, "{\"nonce\":\"n-2\",\"expires_in\":300}"));
            api.Requests[0].Done(new HttpReply(200, "{\"nonce\":\"n-1\",\"expires_in\":300}"));

            CollectionAssert.AreEqual(new[] { "n-2", "n-1" }, attestor.Nonces);
            Assert.AreEqual("token-for-n-2", answers[0].PlayIntegrityToken);
            Assert.AreEqual("token-for-n-1", answers[1].PlayIntegrityToken);
        }

        [Test]
        public void ADeviceThatCannotAttestAsksForNoNonceAndHasNoProof()
        {
            attestor.CanAttest = false;

            proofs.Request(IntegrityPurpose.SessionStart, Keep);

            Assert.AreEqual(0, api.Requests.Count);
            Assert.AreEqual(1, answers.Count);
            Assert.IsNull(answers[0]);
        }

        [TestCase(HttpReply.NoResponse, "")]
        [TestCase(400, "{\"error\":\"unknown_purpose\"}")]
        [TestCase(503, "{\"error\":\"unavailable\"}")]
        [TestCase(200, "")]
        [TestCase(200, "{\"nonce\":\"\",\"expires_in\":300}")]
        [TestCase(200, "{\"expires_in\":300}")]
        [TestCase(200, "not json")]
        public void WithoutAnIssuedNonceThereIsNoProofAndNothingIsAttested(int status, string body)
        {
            proofs.Request(IntegrityPurpose.CasualMatch, Keep);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.AreEqual(0, attestor.Nonces.Count);
            Assert.AreEqual(1, answers.Count);
            Assert.IsNull(answers[0]);
        }

        [Test]
        public void WithoutASignedInSessionThereIsNoProof()
        {
            var signedOut = new IntegrityProofs(new ApiClient(api, new AuthSession(new Auth(), new Identity(), new Store())), attestor);

            signedOut.Request(IntegrityPurpose.SessionStart, Keep);

            Assert.AreEqual(0, api.Requests.Count);
            Assert.AreEqual(0, attestor.Nonces.Count);
            Assert.IsNull(answers[0]);
        }

        [TestCase(QueueKind.Ranked, IntegrityPurpose.RankedMatch)]
        [TestCase(QueueKind.Casual, IntegrityPurpose.CasualMatch)]
        [TestCase(QueueKind.Unspecified, IntegrityPurpose.CasualMatch)]
        public void ARankedQueueNeedsARankedNonceAndAnyOtherACasualOne(QueueKind kind, IntegrityPurpose expected)
        {
            Assert.AreEqual(expected, IntegrityPurposes.ForQueue(kind));
        }

        [Test]
        public void TheDefaultAttestorHasNoProof()
        {
            var none = new NoDeviceAttestor();
            var proof = new IntegrityProof();
            none.Attest("n-1", answer => proof = answer);

            Assert.IsFalse(none.CanAttest);
            Assert.IsNull(proof);
        }
    }
}
