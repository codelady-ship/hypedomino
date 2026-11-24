using System;
using System.Collections.Generic;
using HypeDomino.Game;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class NetStartTests
    {
        const string TokensJson = "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-2\",\"expires_in\":900,\"request_key\":\"AAECAw==\",\"account_id\":\"acc-1\",\"created\":false}";

        sealed class Transport : IAuthTransport
        {
            public readonly List<string> Paths = new List<string>();
            public readonly Queue<HttpReply> Replies = new Queue<HttpReply>();

            public void Post(string path, string json, Action<HttpReply> done)
            {
                Paths.Add(path);
                done(Replies.Dequeue());
            }
        }

        Transport transport;
        MemorySecureStore store;
        AuthSession auth;
        AppFlow flow;
        int dials;
        readonly List<AuthOutcome> stalls = new List<AuthOutcome>();
        NetStart start;

        [SetUp]
        public void Build()
        {
            transport = new Transport();
            store = new MemorySecureStore();
            auth = new AuthSession(transport, new NoIdentityProvider(), store);
            flow = new AppFlow();
            dials = 0;
            stalls.Clear();
            start = new NetStart(auth, flow, () => dials++, stalls.Add);
        }

        [Test]
        public void WithoutAStoredSessionTheFlowAsksForSignInAndNothingIsSent()
        {
            start.Begin();

            Assert.AreEqual(AppState.Login, flow.State);
            Assert.AreEqual(0, dials);
            CollectionAssert.IsEmpty(transport.Paths);
        }

        [Test]
        public void AStoredSessionResumesAndThenDials()
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(200, TokensJson));

            start.Begin();

            CollectionAssert.AreEqual(new[] { AuthSession.RefreshPath }, transport.Paths);
            Assert.AreEqual(1, dials);
            Assert.AreEqual("access-1", auth.AccessToken);
            Assert.AreEqual(AppState.Boot, flow.State);
        }

        [Test]
        public void ARefusedRefreshAsksForSignIn()
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(401, "{\"error\":\"refresh_rejected\"}"));

            start.Begin();

            Assert.AreEqual(AppState.Login, flow.State);
            Assert.AreEqual(0, dials);
        }

        [TestCase(0, AuthOutcome.Offline)]
        [TestCase(503, AuthOutcome.Unavailable)]
        [TestCase(429, AuthOutcome.RateLimited)]
        public void AFailedRefreshKeepsTheStateAndReportsTheOutcome(int status, AuthOutcome outcome)
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(status, null));

            start.Begin();

            Assert.AreEqual(AppState.Boot, flow.State);
            Assert.AreEqual(0, dials);
            CollectionAssert.AreEqual(new[] { outcome }, stalls);
            Assert.IsTrue(auth.HasStoredSession);
        }

        [Test]
        public void UnauthenticatedResumesAndDialsAgain()
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(200, TokensJson));
            transport.Replies.Enqueue(new HttpReply(200, TokensJson));
            start.Begin();

            start.Unauthenticated();

            CollectionAssert.AreEqual(new[] { AuthSession.RefreshPath, AuthSession.RefreshPath }, transport.Paths);
            Assert.AreEqual(2, dials);
            Assert.AreNotEqual(AppState.Login, flow.State);
        }

        [Test]
        public void UnauthenticatedWithARefusedRefreshAsksForSignIn()
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(200, TokensJson));
            transport.Replies.Enqueue(new HttpReply(401, "{\"error\":\"refresh_rejected\"}"));
            start.Begin();

            start.Unauthenticated();

            Assert.AreEqual(AppState.Login, flow.State);
            Assert.AreEqual(1, dials);
        }

        [Test]
        public void TheStandInsKeepNothingAndSignInNobody()
        {
            Assert.IsFalse(auth.HasStoredSession);
            var result = default(AuthResult);
            auth.SignIn(AuthProvider.Google, r => result = r);
            Assert.AreEqual(AuthOutcome.Unsupported, result.Outcome);
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            store.Delete(AuthSession.RefreshTokenKey);
            Assert.IsNull(store.Read(AuthSession.RefreshTokenKey));
        }
    }
}
