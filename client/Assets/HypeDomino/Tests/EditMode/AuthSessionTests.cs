using System;
using System.Collections.Generic;
using HypeDomino.Net.Auth;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AuthSessionTests
    {
        const string TokensJson = "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-2\",\"expires_in\":900,\"request_key\":\"AAECAw==\",\"account_id\":\"acc-1\",\"created\":true}";

        sealed class Transport : IAuthTransport
        {
            public readonly List<string> Paths = new List<string>();
            public readonly List<string> Bodies = new List<string>();
            public readonly Queue<HttpReply> Replies = new Queue<HttpReply>();

            public void Post(string path, string json, Action<HttpReply> done)
            {
                Paths.Add(path);
                Bodies.Add(json);
                done(Replies.Dequeue());
            }
        }

        sealed class Identity : IIdentityProvider
        {
            public IdTokenResult Result = new IdTokenResult(IdTokenOutcome.Token, "id-token");
            public bool AppleSupported = true;
            public string SeenNonce;

            public bool Supports(AuthProvider provider) => provider == AuthProvider.Google || AppleSupported;

            public void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done)
            {
                SeenNonce = nonce;
                done(Result);
            }
        }

        sealed class Store : ISecureStore
        {
            public readonly Dictionary<string, string> Values = new Dictionary<string, string>();

            public string Read(string key) => Values.TryGetValue(key, out var value) ? value : null;

            public void Write(string key, string value) => Values[key] = value;

            public void Delete(string key) => Values.Remove(key);
        }

        Transport transport;
        Identity identity;
        Store store;
        AuthSession session;

        [SetUp]
        public void Build()
        {
            transport = new Transport();
            identity = new Identity();
            store = new Store();
            session = new AuthSession(transport, identity, store);
        }

        AuthResult SignIn(AuthProvider provider)
        {
            var result = default(AuthResult);
            session.SignIn(provider, r => result = r);
            return result;
        }

        AuthResult Resume()
        {
            var result = default(AuthResult);
            session.Resume(r => result = r);
            return result;
        }

        [Test]
        public void GoogleSignInSendsTheRawNonceAndKeepsTheTokens()
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(200, TokensJson));

            var result = SignIn(AuthProvider.Google);

            Assert.AreEqual(AuthOutcome.SignedIn, result.Outcome);
            Assert.IsTrue(result.NewAccount);
            Assert.AreEqual("n-1", identity.SeenNonce);
            CollectionAssert.AreEqual(new[] { AuthSession.NoncePath, AuthSession.SignInPath }, transport.Paths);
            var request = JsonUtility.FromJson<SignInRequest>(transport.Bodies[1]);
            Assert.AreEqual("google", request.provider);
            Assert.AreEqual("id-token", request.id_token);
            Assert.AreEqual("n-1", request.nonce);
            Assert.AreEqual(store.Read(AuthSession.DeviceIdKey), request.device_id);
            Assert.AreEqual("refresh-2", store.Read(AuthSession.RefreshTokenKey));
            Assert.AreEqual("access-1", session.AccessToken);
            Assert.AreEqual("acc-1", session.AccountId);
            Assert.AreEqual(900, session.ExpiresInSeconds);
            CollectionAssert.AreEqual(new byte[] { 0, 1, 2, 3 }, session.RequestKey);
        }

        [Test]
        public void AppleGetsTheSha256OfTheNonceAndTheServerTheRawNonce()
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"abc\"}"));
            transport.Replies.Enqueue(new HttpReply(200, TokensJson));

            Assert.AreEqual(AuthOutcome.SignedIn, SignIn(AuthProvider.Apple).Outcome);

            Assert.AreEqual("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", identity.SeenNonce);
            var request = JsonUtility.FromJson<SignInRequest>(transport.Bodies[1]);
            Assert.AreEqual("apple", request.provider);
            Assert.AreEqual("abc", request.nonce);
        }

        [Test]
        public void AnUnsupportedProviderAsksNothing()
        {
            identity.AppleSupported = false;

            Assert.AreEqual(AuthOutcome.Unsupported, SignIn(AuthProvider.Apple).Outcome);
            Assert.IsEmpty(transport.Paths);
        }

        [Test]
        public void CancellingTheAccountPickerSignsNothingIn()
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            identity.Result = new IdTokenResult(IdTokenOutcome.Cancelled, null);

            Assert.AreEqual(AuthOutcome.Cancelled, SignIn(AuthProvider.Google).Outcome);
            Assert.AreEqual(1, transport.Paths.Count);
            Assert.IsFalse(session.SignedIn);
            Assert.IsNull(store.Read(AuthSession.RefreshTokenKey));
        }

        [TestCase(IdTokenOutcome.Failed, "x")]
        [TestCase(IdTokenOutcome.Token, "")]
        public void AProviderFailureIsReported(IdTokenOutcome outcome, string token)
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            identity.Result = new IdTokenResult(outcome, token);

            Assert.AreEqual(AuthOutcome.ProviderFailed, SignIn(AuthProvider.Google).Outcome);
        }

        [TestCase(HttpReply.NoResponse, "", AuthOutcome.Offline)]
        [TestCase(503, "{\"error\":\"unavailable\"}", AuthOutcome.Unavailable)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", AuthOutcome.RateLimited)]
        [TestCase(200, "not json", AuthOutcome.Unavailable)]
        public void ANonceFailureStopsBeforeTheProvider(int status, string body, AuthOutcome expected)
        {
            transport.Replies.Enqueue(new HttpReply(status, body));

            Assert.AreEqual(expected, SignIn(AuthProvider.Google).Outcome);
            Assert.IsNull(identity.SeenNonce);
        }

        [TestCase(401, "{\"error\":\"token_rejected\"}", AuthOutcome.Rejected)]
        [TestCase(401, "{\"error\":\"nonce_rejected\"}", AuthOutcome.Rejected)]
        [TestCase(403, "{\"error\":\"banned\"}", AuthOutcome.Banned)]
        [TestCase(410, "{\"error\":\"deleted\"}", AuthOutcome.Deleted)]
        [TestCase(503, "{\"error\":\"provider_unavailable\"}", AuthOutcome.Unavailable)]
        [TestCase(HttpReply.NoResponse, "", AuthOutcome.Offline)]
        [TestCase(200, "{\"access_token\":\"a\"}", AuthOutcome.Unavailable)]
        public void ASignInRefusalIsReportedWithItsCode(int status, string body, AuthOutcome expected)
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(status, body));

            var result = SignIn(AuthProvider.Google);

            Assert.AreEqual(expected, result.Outcome);
            Assert.IsFalse(session.SignedIn);
            Assert.IsNull(store.Read(AuthSession.RefreshTokenKey));
        }

        [TestCase("cheating")]
        [TestCase("collusion")]
        [TestCase("afk_leaving")]
        [TestCase("offensive_name")]
        [TestCase("payment_fraud")]
        [TestCase("other")]
        [TestCase("a_future_category")]
        public void ABanCarriesTheCategoryTheServerNamed(string category)
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(403, "{\"error\":\"banned\",\"category\":\"" + category + "\"}"));

            var result = SignIn(AuthProvider.Google);

            Assert.AreEqual(AuthOutcome.Banned, result.Outcome);
            Assert.AreEqual(category, result.BanCategory);
        }

        [Test]
        public void ABanWithoutACategoryCarriesNone()
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(403, "{\"error\":\"banned\"}"));

            var result = SignIn(AuthProvider.Google);

            Assert.AreEqual(AuthOutcome.Banned, result.Outcome);
            Assert.IsTrue(string.IsNullOrEmpty(result.BanCategory));
            Assert.IsFalse(result.Appeal.Offered);
        }

        [Test]
        public void ABanCarriesItsAppealGrant()
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(403, "{\"error\":\"banned\",\"category\":\"cheating\",\"banned_until\":\"2026-12-01T00:00:00Z\",\"appeal\":{\"sanction_id\":42,\"token\":\"appeal-1\",\"expires_at\":\"2026-11-22T10:30:00Z\"}}"));

            var result = SignIn(AuthProvider.Google);

            Assert.AreEqual(AuthOutcome.Banned, result.Outcome);
            Assert.IsTrue(result.Appeal.Offered);
            Assert.AreEqual(42, result.Appeal.SanctionId);
            Assert.AreEqual("appeal-1", result.Appeal.Token);
            Assert.AreEqual((ulong)new DateTimeOffset(2026, 11, 22, 10, 30, 0, TimeSpan.Zero).ToUnixTimeMilliseconds(), result.Appeal.ExpiresUnixMs);
            CollectionAssert.DoesNotContain(store.Values.Values, "appeal-1");
        }

        [Test]
        public void ARefusedRefreshForABanCarriesItsCategory()
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(403, "{\"error\":\"banned\",\"category\":\"collusion\"}"));

            var result = Resume();

            Assert.AreEqual(AuthOutcome.Banned, result.Outcome);
            Assert.AreEqual("collusion", result.BanCategory);
        }

        [Test]
        public void ATemporaryBanCarriesItsEnd()
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(403, "{\"error\":\"banned\",\"category\":\"cheating\",\"banned_until\":\"2026-12-01T00:00:00Z\"}"));

            var result = SignIn(AuthProvider.Google);

            Assert.AreEqual(AuthOutcome.Banned, result.Outcome);
            Assert.AreEqual((ulong)new DateTimeOffset(2026, 12, 1, 0, 0, 0, TimeSpan.Zero).ToUnixTimeMilliseconds(), result.BannedUntilUnixMs);
        }

        [TestCase("{\"error\":\"banned\",\"category\":\"cheating\"}")]
        [TestCase("{\"error\":\"banned\",\"category\":\"cheating\",\"banned_until\":\"\"}")]
        [TestCase("{\"error\":\"banned\",\"category\":\"cheating\",\"banned_until\":\"not-a-time\"}")]
        public void APermanentOrUnreadableBanCarriesNoEnd(string body)
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(403, body));

            var result = SignIn(AuthProvider.Google);

            Assert.AreEqual(AuthOutcome.Banned, result.Outcome);
            Assert.AreEqual(0UL, result.BannedUntilUnixMs);
        }

        [Test]
        public void ARefusedRefreshForABanCarriesItsEnd()
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(403, "{\"error\":\"banned\",\"category\":\"collusion\",\"banned_until\":\"2026-12-01T00:00:00Z\"}"));

            var result = Resume();

            Assert.AreEqual(AuthOutcome.Banned, result.Outcome);
            Assert.AreEqual((ulong)new DateTimeOffset(2026, 12, 1, 0, 0, 0, TimeSpan.Zero).ToUnixTimeMilliseconds(), result.BannedUntilUnixMs);
        }

        [Test]
        public void OnlyABanCarriesACategory()
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(401, "{\"error\":\"token_rejected\",\"category\":\"cheating\"}"));

            Assert.IsNull(SignIn(AuthProvider.Google).BanCategory);
        }

        [Test]
        public void APendingDeletionCarriesItsDate()
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(409, "{\"error\":\"deletion_pending\",\"delete_at\":\"2026-12-01T00:00:00Z\"}"));

            var result = SignIn(AuthProvider.Google);

            Assert.AreEqual(AuthOutcome.DeletionPending, result.Outcome);
            Assert.AreEqual("2026-12-01T00:00:00Z", result.DeleteAt);
        }

        [Test]
        public void ResumeWithoutAStoredSessionAsksForSignIn()
        {
            Assert.AreEqual(AuthOutcome.SignInRequired, Resume().Outcome);
            Assert.IsEmpty(transport.Paths);
        }

        [Test]
        public void ResumeRefreshesSilentlyWithTheSameDevice()
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            store.Write(AuthSession.DeviceIdKey, "device-1");
            transport.Replies.Enqueue(new HttpReply(200, TokensJson.Replace(",\"created\":true", string.Empty)));

            var result = Resume();

            Assert.AreEqual(AuthOutcome.SignedIn, result.Outcome);
            Assert.IsFalse(result.NewAccount);
            CollectionAssert.AreEqual(new[] { AuthSession.RefreshPath }, transport.Paths);
            var request = JsonUtility.FromJson<RefreshRequest>(transport.Bodies[0]);
            Assert.AreEqual("refresh-1", request.refresh_token);
            Assert.AreEqual("device-1", request.device_id);
            Assert.AreEqual("refresh-2", store.Read(AuthSession.RefreshTokenKey));
            Assert.IsTrue(session.SignedIn);
        }

        [TestCase(401, "{\"error\":\"refresh_expired\"}", AuthOutcome.SignInRequired)]
        [TestCase(401, "{\"error\":\"refresh_reused\"}", AuthOutcome.SignInRequired)]
        [TestCase(403, "{\"error\":\"banned\"}", AuthOutcome.Banned)]
        public void ARefusedRefreshForgetsTheStoredSession(int status, string body, AuthOutcome expected)
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(status, body));

            Assert.AreEqual(expected, Resume().Outcome);
            Assert.IsFalse(session.HasStoredSession);
        }

        [TestCase(HttpReply.NoResponse, "", AuthOutcome.Offline)]
        [TestCase(503, "{\"error\":\"unavailable\"}", AuthOutcome.Unavailable)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", AuthOutcome.RateLimited)]
        public void AnUnreachableServerKeepsTheStoredSession(int status, string body, AuthOutcome expected)
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(status, body));

            Assert.AreEqual(expected, Resume().Outcome);
            Assert.AreEqual("refresh-1", store.Read(AuthSession.RefreshTokenKey));
        }

        [Test]
        public void TheDeviceIdIsMadeOnceAndKept()
        {
            transport.Replies.Enqueue(new HttpReply(200, "{\"nonce\":\"n-1\"}"));
            transport.Replies.Enqueue(new HttpReply(200, TokensJson));
            SignIn(AuthProvider.Google);
            var first = store.Read(AuthSession.DeviceIdKey);

            transport.Replies.Enqueue(new HttpReply(200, TokensJson));
            Resume();

            Assert.IsNotEmpty(first);
            Assert.AreEqual(first, JsonUtility.FromJson<RefreshRequest>(transport.Bodies[2]).device_id);
        }

        [Test]
        public void ASecondRequestWhileOneRunsIsBusy()
        {
            var pending = new PendingTransport();
            session = new AuthSession(pending, identity, store);
            session.SignIn(AuthProvider.Google, _ => { });
            var result = default(AuthResult);

            session.SignIn(AuthProvider.Google, r => result = r);

            Assert.AreEqual(AuthOutcome.Busy, result.Outcome);
            Assert.AreEqual(1, pending.Posts);
            pending.Done(new HttpReply(HttpReply.NoResponse, string.Empty));
            session.SignIn(AuthProvider.Google, _ => { });
            Assert.AreEqual(2, pending.Posts);
        }

        [Test]
        public void SignOutForgetsLocallyAndRevokesOnTheServer()
        {
            store.Write(AuthSession.RefreshTokenKey, "refresh-1");
            transport.Replies.Enqueue(new HttpReply(200, TokensJson));
            Resume();
            transport.Replies.Enqueue(new HttpReply(HttpReply.NoResponse, string.Empty));
            var finished = false;

            session.SignOut(() => finished = true);

            Assert.IsTrue(finished);
            Assert.IsFalse(session.SignedIn);
            Assert.IsNull(session.RequestKey);
            Assert.IsFalse(session.HasStoredSession);
            Assert.AreEqual(AuthSession.LogoutPath, transport.Paths[1]);
            Assert.AreEqual("refresh-2", JsonUtility.FromJson<LogoutRequest>(transport.Bodies[1]).refresh_token);
        }

        sealed class PendingTransport : IAuthTransport
        {
            Action<HttpReply> waiting;

            public int Posts { get; private set; }

            public void Post(string path, string json, Action<HttpReply> done)
            {
                Posts++;
                waiting = done;
            }

            public void Done(HttpReply reply)
            {
                var done = waiting;
                waiting = null;
                done(reply);
            }
        }
    }
}
