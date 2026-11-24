using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Core.Analytics;
using HypeDomino.Game.Analytics;
using HypeDomino.Game.Lobby;
using HypeDomino.Net;
using HypeDomino.Net.Auth;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AppAnalyticsTests
    {
        const string AnalyticsId = "3f2b8c1e-5a4d-4e8f-9b7a-1c2d3e4f5a6b";
        const string TokensJson = "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-2\",\"expires_in\":900,\"request_key\":\"AAECAw==\",\"account_id\":\"acc-1\",\"created\":true}";
        const ulong Nonce = 0x5EED_0181_0000_0001;

        sealed class Sender : IAnalyticsSender
        {
            public readonly List<AnalyticsEvent> Sent = new();

            public void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done)
            {
                for (var i = 0; i < batch.Count; i++) Sent.Add(batch[i]);
                done(true);
            }
        }

        sealed class AuthTransport : IAuthTransport
        {
            public readonly Queue<HttpReply> Replies = new();

            public void Post(string path, string json, Action<HttpReply> done) => done(Replies.Dequeue());
        }

        sealed class Identity : IIdentityProvider
        {
            public IdTokenResult Result = new(IdTokenOutcome.Token, "id-token");

            public bool Supports(AuthProvider provider) => true;

            public void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done) => done(Result);
        }

        sealed class Store : ISecureStore
        {
            readonly Dictionary<string, string> values = new();

            public string Read(string key) => values.TryGetValue(key, out var value) ? value : null;

            public void Write(string key, string value) => values[key] = value;

            public void Delete(string key) => values.Remove(key);
        }

        sealed class FrameTransport : IFrameTransport
        {
            readonly Queue<byte[]> incoming = new();

            public void Send(ReadOnlySpan<byte> frame)
            {
            }

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                if (incoming.Count == 0) return false;
                frame = incoming.Dequeue();
                return true;
            }

            public void Deliver(Envelope.PayloadOneofCase payload, IMessage message) =>
                incoming.Enqueue(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, Nonce), payload, message).ToArray());
        }

        Sender sender;
        AnalyticsQueue queue;
        AuthTransport authTransport;
        Identity identity;
        Store store;
        AuthSession auth;
        MatchQueue matches;
        Session session;
        AppAnalytics app;

        [SetUp]
        public void Build()
        {
            sender = new Sender();
            queue = new AnalyticsQueue(sender, () => 1700000000000, 10, 10);
            authTransport = new AuthTransport();
            identity = new Identity();
            store = new Store();
            auth = new AuthSession(authTransport, identity, store);
            matches = new MatchQueue();
            session = new Session();
            app = new AppAnalytics(queue);
            app.Follow(auth, matches);
        }

        List<AnalyticsEvent> Sent()
        {
            queue.Flush();
            return sender.Sent;
        }

        void AcceptSession()
        {
            var frames = new FrameTransport();
            session.Hello(frames, new ClientHello());
            frames.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = Nonce });
            session.Poll();
            Assert.IsTrue(session.Accepted);
        }

        AuthOutcome SignIn(AuthProvider provider)
        {
            authTransport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            authTransport.Replies.Enqueue(new HttpReply(200, TokensJson));
            var outcome = AuthOutcome.Busy;
            auth.SignIn(provider, result => outcome = result.Outcome);
            return outcome;
        }

        static void AssertOne(List<AnalyticsEvent> sent, string name, string label)
        {
            Assert.AreEqual(1, sent.Count);
            Assert.AreEqual(name, sent[0].Name);
            Assert.AreEqual(label, sent[0].Label);
            Assert.AreEqual(0, sent[0].Value);
        }

        [Test]
        public void AppOpenIsTrackedOnceOnTheFirstTickWithConsent()
        {
            app.Tick();
            Assert.IsFalse(app.AppOpenTracked);
            Assert.AreEqual(0, queue.Count);

            queue.Consent(AnalyticsId);
            app.Tick();
            app.Tick();

            Assert.IsTrue(app.AppOpenTracked);
            AssertOne(Sent(), AnalyticsEvents.AppOpen, null);
        }

        [Test]
        public void AppOpenIsNotTrackedAgainAfterAWithdrawalAndANewConsentInTheSameRun()
        {
            queue.Consent(AnalyticsId);
            app.Tick();
            queue.Withdraw();
            queue.Consent("9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d");

            app.Tick();

            Assert.AreEqual(0, queue.Count);
        }

        [Test]
        public void TickingAfterAppOpenAllocatesNothing()
        {
            queue.Consent(AnalyticsId);
            app.Tick();
            Assert.That(app.Tick, Allocates.Nothing);
        }

        [Test]
        public void TickingWithoutConsentAllocatesNothing()
        {
            app.Tick();
            Assert.That(app.Tick, Allocates.Nothing);
        }

        [TestCase(AuthProvider.Google, "google")]
        [TestCase(AuthProvider.Apple, "apple")]
        public void AProviderSignInIsTrackedWithTheProvider(AuthProvider provider, string label)
        {
            queue.Consent(AnalyticsId);

            Assert.AreEqual(AuthOutcome.SignedIn, SignIn(provider));

            AssertOne(Sent(), AnalyticsEvents.SignIn, label);
        }

        [Test]
        public void ASignInBeforeConsentIsDropped()
        {
            Assert.AreEqual(AuthOutcome.SignedIn, SignIn(AuthProvider.Google));
            queue.Consent(AnalyticsId);

            Assert.IsEmpty(Sent());
        }

        [Test]
        public void ACancelledOrRefusedSignInIsNotTracked()
        {
            queue.Consent(AnalyticsId);
            identity.Result = new IdTokenResult(IdTokenOutcome.Cancelled, null);
            authTransport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            auth.SignIn(AuthProvider.Google, _ => { });

            identity.Result = new IdTokenResult(IdTokenOutcome.Token, "id-token");
            authTransport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-2\"}"));
            authTransport.Replies.Enqueue(new HttpReply(403, "{\"error\":\"banned\"}"));
            auth.SignIn(AuthProvider.Google, _ => { });

            Assert.IsFalse(auth.SignedIn);
            Assert.IsEmpty(Sent());
        }

        [Test]
        public void ResumingAStoredSessionIsNotASignIn()
        {
            queue.Consent(AnalyticsId);
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            authTransport.Replies.Enqueue(new HttpReply(200, TokensJson));
            var outcome = AuthOutcome.Busy;

            auth.Resume(result => outcome = result.Outcome);

            Assert.AreEqual(AuthOutcome.SignedIn, outcome);
            Assert.IsEmpty(Sent());
        }

        [TestCase(QueueKind.Ranked, Mode._1V1, "ranked_1v1")]
        [TestCase(QueueKind.Ranked, Mode._2V2, "ranked_2v2")]
        [TestCase(QueueKind.Casual, Mode._1V1, "casual_1v1")]
        [TestCase(QueueKind.Casual, Mode._2V2, "casual_2v2")]
        public void ASentQueueJoinIsTrackedWithTheQueueAndMode(QueueKind kind, Mode mode, string label)
        {
            queue.Consent(AnalyticsId);
            AcceptSession();

            Assert.IsTrue(matches.TryJoin(session, mode, kind, new IntegrityProof()));

            AssertOne(Sent(), AnalyticsEvents.QueueJoin, label);
        }

        [Test]
        public void AJoinThatIsNotSentIsNotTracked()
        {
            queue.Consent(AnalyticsId);

            Assert.IsFalse(matches.TryJoin(session, Mode._1V1, QueueKind.Ranked, new IntegrityProof()));
            AcceptSession();
            Assert.IsFalse(matches.TryJoin(session, Mode.Unspecified, QueueKind.Ranked, new IntegrityProof()));
            Assert.IsTrue(matches.TryJoin(session, Mode._1V1, QueueKind.Ranked, new IntegrityProof()));
            Assert.IsFalse(matches.TryJoin(session, Mode._1V1, QueueKind.Ranked, new IntegrityProof()));

            Assert.AreEqual(1, Sent().Count);
        }
    }
}
