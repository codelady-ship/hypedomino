using System;
using System.Collections.Generic;
using HypeDomino.Game.Privacy;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AccountDeletionTests
    {
        const string NonceJson = "{\"nonce\":\"nonce-1\"}";
        const string DeleteAt = "2026-12-21T09:00:00.5Z";
        const ulong DeleteAtUnixMs = 1797843600500;

        sealed class Sent
        {
            public string Path;
            public string Body;
            public Action<HttpReply> Done;
        }

        sealed class Transport : IAuthTransport
        {
            public readonly List<Sent> Requests = new();

            public void Post(string path, string json, Action<HttpReply> done) => Requests.Add(new Sent { Path = path, Body = json, Done = done });
        }

        sealed class Identity : IIdentityProvider
        {
            public bool AppleSupported = true;
            public readonly List<string> Nonces = new();
            public readonly List<Action<IdTokenResult>> Pending = new();

            public bool Supports(AuthProvider provider) => provider == AuthProvider.Google || AppleSupported;

            public void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done)
            {
                Nonces.Add(nonce);
                Pending.Add(done);
            }
        }

        Transport transport;
        Identity identity;
        AccountDeletion deletion;
        List<AccountDeletionResult> results;

        [SetUp]
        public void Build()
        {
            transport = new Transport();
            identity = new Identity();
            deletion = new AccountDeletion(transport, identity);
            results = new List<AccountDeletionResult>();
        }

        void SignInFreshly(string token = "id-token")
        {
            transport.Requests[0].Done(new HttpReply(200, NonceJson));
            identity.Pending[0](new IdTokenResult(IdTokenOutcome.Token, token));
        }

        AccountDeletionResult Answer(Func<AuthProvider, Action<AccountDeletionResult>, bool> start, int status, string body)
        {
            start(AuthProvider.Google, results.Add);
            SignInFreshly();
            transport.Requests[1].Done(new HttpReply(status, body));
            return results[0];
        }

        [Test]
        public void TheModelNeedsATransportAndAnIdentityProvider()
        {
            Assert.Throws<ArgumentNullException>(() => new AccountDeletion(null, identity));
            Assert.Throws<ArgumentNullException>(() => new AccountDeletion(transport, null));
        }

        [Test]
        public void ARequestAsksForANonceThenAFreshIdTokenThenPostsBoth()
        {
            Assert.IsTrue(deletion.Request(AuthProvider.Google, results.Add));
            Assert.IsTrue(deletion.Busy);
            Assert.AreEqual(1, transport.Requests.Count);
            Assert.AreEqual("/auth/nonce", transport.Requests[0].Path);

            transport.Requests[0].Done(new HttpReply(200, NonceJson));
            Assert.AreEqual("nonce-1", identity.Nonces[0]);
            Assert.AreEqual(1, transport.Requests.Count);

            identity.Pending[0](new IdTokenResult(IdTokenOutcome.Token, "id-token"));
            Assert.AreEqual("/auth/deletion", transport.Requests[1].Path);
            var sent = JsonUtility.FromJson<AccountDeletionRequest>(transport.Requests[1].Body);
            Assert.AreEqual("google", sent.provider);
            Assert.AreEqual("id-token", sent.id_token);
            Assert.AreEqual("nonce-1", sent.nonce);
            Assert.IsTrue(deletion.Busy);
            Assert.AreEqual(0, results.Count);
        }

        [Test]
        public void AppleGetsTheHashedNonceAndTheServerTheRawOne()
        {
            deletion.Cancel(AuthProvider.Apple, results.Add);
            transport.Requests[0].Done(new HttpReply(200, NonceJson));
            Assert.AreEqual(AuthSession.ProviderNonce(AuthProvider.Apple, "nonce-1"), identity.Nonces[0]);
            identity.Pending[0](new IdTokenResult(IdTokenOutcome.Token, "apple-token"));
            Assert.AreEqual("/auth/deletion/cancel", transport.Requests[1].Path);
            var sent = JsonUtility.FromJson<AccountDeletionRequest>(transport.Requests[1].Body);
            Assert.AreEqual("apple", sent.provider);
            Assert.AreEqual("nonce-1", sent.nonce);
        }

        [Test]
        public void AnAcceptedRequestIsPendingUntilTheServersDate()
        {
            var result = Answer(deletion.Request, 202, "{\"delete_at\":\"" + DeleteAt + "\"}");
            Assert.AreEqual(AccountDeletionOutcome.Pending, result.Outcome);
            Assert.AreEqual(DeleteAtUnixMs, result.DeleteAtUnixMs);
            Assert.AreEqual(DateTimeOffset.FromUnixTimeMilliseconds((long)DeleteAtUnixMs), result.DeleteAt);
            Assert.AreEqual("account.deletion.pending", result.MessageKey);
            Assert.IsFalse(deletion.Busy);
        }

        [TestCase("")]
        [TestCase("{}")]
        [TestCase("{\"delete_at\":\"soon\"}")]
        [TestCase("not json")]
        public void AnAcceptedRequestWithoutAReadableDateIsStillPending(string body)
        {
            var result = Answer(deletion.Request, 202, body);
            Assert.AreEqual(AccountDeletionOutcome.Pending, result.Outcome);
            Assert.AreEqual(0UL, result.DeleteAtUnixMs);
            Assert.AreEqual("account.deletion.pending_undated", result.MessageKey);
        }

        [Test]
        public void ACancelledDeletionRestoresTheAccount()
        {
            var result = Answer(deletion.Cancel, 204, "");
            Assert.AreEqual(AccountDeletionOutcome.Restored, result.Outcome);
            Assert.AreEqual("account.deletion.restored", result.MessageKey);
            Assert.IsFalse(deletion.Busy);
        }

        [Test]
        public void AnAcceptedReplyToACancelIsNotARestore()
        {
            Assert.AreEqual(AccountDeletionOutcome.Failed, Answer(deletion.Cancel, 202, "{\"delete_at\":\"" + DeleteAt + "\"}").Outcome);
        }

        [Test]
        public void ANoContentReplyToARequestIsNotPending()
        {
            Assert.AreEqual(AccountDeletionOutcome.Failed, Answer(deletion.Request, 204, "").Outcome);
        }

        [TestCase(401, "{\"error\":\"token_rejected\"}", AccountDeletionOutcome.SignInFailed, "account.deletion.sign_in_failed")]
        [TestCase(401, "{\"error\":\"token_expired\"}", AccountDeletionOutcome.SignInFailed, "account.deletion.sign_in_failed")]
        [TestCase(401, "{\"error\":\"nonce_rejected\"}", AccountDeletionOutcome.SignInFailed, "account.deletion.sign_in_failed")]
        [TestCase(409, "{\"error\":\"in_match\"}", AccountDeletionOutcome.InMatch, "account.deletion.in_match")]
        [TestCase(404, "{\"error\":\"account_unknown\"}", AccountDeletionOutcome.UnknownAccount, "account.deletion.unknown_account")]
        [TestCase(403, "{\"error\":\"banned\"}", AccountDeletionOutcome.Banned, "account.deletion.banned")]
        [TestCase(410, "{\"error\":\"deleted\"}", AccountDeletionOutcome.Deleted, "account.deletion.deleted")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", AccountDeletionOutcome.RateLimited, "account.deletion.rate_limited")]
        [TestCase(503, "{\"error\":\"provider_unavailable\"}", AccountDeletionOutcome.Unavailable, "account.deletion.unavailable")]
        [TestCase(500, "{\"error\":\"internal\"}", AccountDeletionOutcome.Unavailable, "account.deletion.unavailable")]
        [TestCase(HttpReply.NoResponse, null, AccountDeletionOutcome.Offline, "account.deletion.offline")]
        [TestCase(400, "{\"error\":\"malformed\"}", AccountDeletionOutcome.Failed, "notice.generic")]
        [TestCase(400, "{\"error\":\"unknown_provider\"}", AccountDeletionOutcome.Failed, "notice.generic")]
        [TestCase(404, "404 page not found", AccountDeletionOutcome.Failed, "notice.generic")]
        [TestCase(409, "{\"error\":\"something_new\"}", AccountDeletionOutcome.Failed, "notice.generic")]
        public void EveryRequestRefusalHasItsOutcomeAndKey(int status, string body, AccountDeletionOutcome outcome, string key)
        {
            var result = Answer(deletion.Request, status, body);
            Assert.AreEqual(outcome, result.Outcome);
            Assert.AreEqual(key, result.MessageKey);
            Assert.AreEqual(0UL, result.DeleteAtUnixMs);
            Assert.IsFalse(deletion.Busy);
        }

        [TestCase(409, "{\"error\":\"no_deletion_pending\"}", AccountDeletionOutcome.NotPending, "account.deletion.not_pending")]
        [TestCase(401, "{\"error\":\"token_rejected\"}", AccountDeletionOutcome.SignInFailed, "account.deletion.sign_in_failed")]
        [TestCase(410, "{\"error\":\"deleted\"}", AccountDeletionOutcome.Deleted, "account.deletion.deleted")]
        [TestCase(HttpReply.NoResponse, null, AccountDeletionOutcome.Offline, "account.deletion.offline")]
        public void EveryCancelRefusalHasItsOutcomeAndKey(int status, string body, AccountDeletionOutcome outcome, string key)
        {
            var result = Answer(deletion.Cancel, status, body);
            Assert.AreEqual(outcome, result.Outcome);
            Assert.AreEqual(key, result.MessageKey);
        }

        [TestCase(HttpReply.NoResponse, "", AccountDeletionOutcome.Offline)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", AccountDeletionOutcome.RateLimited)]
        [TestCase(503, "", AccountDeletionOutcome.Unavailable)]
        [TestCase(200, "{}", AccountDeletionOutcome.Unavailable)]
        [TestCase(200, "not json", AccountDeletionOutcome.Unavailable)]
        public void ANonceFailureEndsBeforeTheSignIn(int status, string body, AccountDeletionOutcome outcome)
        {
            deletion.Request(AuthProvider.Google, results.Add);
            transport.Requests[0].Done(new HttpReply(status, body));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(0, identity.Pending.Count);
            Assert.AreEqual(1, transport.Requests.Count);
            Assert.IsFalse(deletion.Busy);
        }

        [Test]
        public void ClosingTheSignInSendsNothingAndShowsNoMessage()
        {
            deletion.Request(AuthProvider.Google, results.Add);
            transport.Requests[0].Done(new HttpReply(200, NonceJson));
            identity.Pending[0](new IdTokenResult(IdTokenOutcome.Cancelled, null));
            Assert.AreEqual(AccountDeletionOutcome.SignInCancelled, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.AreEqual(1, transport.Requests.Count);
            Assert.IsFalse(deletion.Busy);
        }

        [TestCase(IdTokenOutcome.Failed, null)]
        [TestCase(IdTokenOutcome.Token, "")]
        public void AFailedSignInSendsNothing(IdTokenOutcome outcome, string token)
        {
            deletion.Request(AuthProvider.Google, results.Add);
            transport.Requests[0].Done(new HttpReply(200, NonceJson));
            identity.Pending[0](new IdTokenResult(outcome, token));
            Assert.AreEqual(AccountDeletionOutcome.SignInFailed, results[0].Outcome);
            Assert.AreEqual(1, transport.Requests.Count);
        }

        [Test]
        public void AnUnsupportedProviderSendsNothing()
        {
            identity.AppleSupported = false;
            Assert.IsTrue(deletion.Request(AuthProvider.Apple, results.Add));
            Assert.AreEqual(AccountDeletionOutcome.SignInFailed, results[0].Outcome);
            Assert.AreEqual(0, transport.Requests.Count);
            Assert.IsFalse(deletion.Busy);
        }

        [Test]
        public void ASecondRequestOrCancelWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(deletion.Request(AuthProvider.Google, results.Add));
            Assert.IsFalse(deletion.Request(AuthProvider.Google, results.Add));
            Assert.IsFalse(deletion.Cancel(AuthProvider.Google, results.Add));
            Assert.AreEqual(1, transport.Requests.Count);

            SignInFreshly();
            transport.Requests[1].Done(new HttpReply(202, "{\"delete_at\":\"" + DeleteAt + "\"}"));
            Assert.IsTrue(deletion.Cancel(AuthProvider.Google, results.Add));
            Assert.AreEqual(3, transport.Requests.Count);
            Assert.AreEqual("/auth/nonce", transport.Requests[2].Path);
            Assert.AreEqual(1, results.Count);
        }

        [Test]
        public void ASignInDuringTheGraceGivesThePendingStateAndDate()
        {
            var signIn = new AuthResult(AuthOutcome.DeletionPending, error: "deletion_pending", deleteAt: DeleteAt);
            Assert.IsTrue(AccountDeletionResult.TryPending(signIn, out var pending));
            Assert.AreEqual(AccountDeletionOutcome.Pending, pending.Outcome);
            Assert.AreEqual(DeleteAtUnixMs, pending.DeleteAtUnixMs);
            Assert.AreEqual("account.deletion.pending", pending.MessageKey);
        }

        [TestCase(AuthOutcome.SignedIn)]
        [TestCase(AuthOutcome.Deleted)]
        [TestCase(AuthOutcome.Banned)]
        public void AnyOtherSignInIsNotPending(AuthOutcome outcome)
        {
            Assert.IsFalse(AccountDeletionResult.TryPending(new AuthResult(outcome, deleteAt: DeleteAt), out _));
        }

        [Test]
        public void ThePendingTextShowsTheServersDate()
        {
            var strings = BuiltinStrings.Load();
            var args = new Dictionary<string, object> { [AccountDeletionKeys.DeleteAtArgument] = DateTimeOffset.FromUnixTimeMilliseconds((long)DeleteAtUnixMs) };
            Assert.IsTrue(MessageFormat.TryFormat(strings.Value(AccountDeletionKeys.Pending, GameLanguage.English), GameLanguage.English, GameLanguage.English, args, out var text));
            StringAssert.DoesNotContain("{", text);
        }

        [Test]
        public void EveryOutcomeButAClosedSignInHasAMessageKeyWithBuiltInTexts()
        {
            var strings = BuiltinStrings.Load();
            var keys = new List<string> { AccountDeletionKeys.PendingUndated };
            foreach (AccountDeletionOutcome outcome in Enum.GetValues(typeof(AccountDeletionOutcome)))
            {
                var key = AccountDeletionKeys.Of(outcome);
                if (outcome == AccountDeletionOutcome.SignInCancelled)
                {
                    Assert.IsNull(key);
                    continue;
                }
                keys.Add(key);
            }
            foreach (var key in keys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
