using System;
using HypeDomino.Net.Auth;
using UnityEngine;

namespace HypeDomino.Game.Privacy
{
    public sealed class AccountDeletion
    {
        public const string RequestPath = "/auth/deletion";
        public const string CancelPath = "/auth/deletion/cancel";

        readonly IAuthTransport transport;
        readonly IIdentityProvider identity;
        AuthProvider provider;
        string path;
        string nonce;
        Action<AccountDeletionResult> done;

        public AccountDeletion(IAuthTransport transport, IIdentityProvider identity)
        {
            this.transport = transport ?? throw new ArgumentNullException(nameof(transport));
            this.identity = identity ?? throw new ArgumentNullException(nameof(identity));
        }

        public bool Busy => done != null;

        public bool Request(AuthProvider provider, Action<AccountDeletionResult> done) => Begin(provider, RequestPath, done);

        public bool Cancel(AuthProvider provider, Action<AccountDeletionResult> done) => Begin(provider, CancelPath, done);

        bool Begin(AuthProvider provider, string path, Action<AccountDeletionResult> done)
        {
            if (done == null) throw new ArgumentNullException(nameof(done));
            if (Busy) return false;
            if (!identity.Supports(provider))
            {
                done(AccountDeletionResult.SignInFailed);
                return true;
            }
            this.provider = provider;
            this.path = path;
            this.done = done;
            transport.Post(AuthSession.NoncePath, "{}", OnNonce);
            return true;
        }

        void OnNonce(HttpReply reply)
        {
            if (reply.Status != 200)
            {
                Finish(AccountDeletionResult.Refused(reply));
                return;
            }
            nonce = ReadNonce(reply.Body);
            if (string.IsNullOrEmpty(nonce))
            {
                Finish(AccountDeletionResult.Unavailable);
                return;
            }
            identity.RequestIdToken(provider, AuthSession.ProviderNonce(provider, nonce), OnIdToken);
        }

        void OnIdToken(IdTokenResult token)
        {
            if (token.Outcome == IdTokenOutcome.Cancelled)
            {
                Finish(AccountDeletionResult.SignInCancelled);
                return;
            }
            if (token.Outcome != IdTokenOutcome.Token || string.IsNullOrEmpty(token.Token))
            {
                Finish(AccountDeletionResult.SignInFailed);
                return;
            }
            var request = new AccountDeletionRequest { provider = AuthSession.ProviderName(provider), id_token = token.Token, nonce = nonce };
            transport.Post(path, JsonUtility.ToJson(request), OnAnswer);
        }

        void OnAnswer(HttpReply reply) =>
            Finish(path == RequestPath ? AccountDeletionResult.FromRequest(reply) : AccountDeletionResult.FromCancel(reply));

        void Finish(AccountDeletionResult result)
        {
            var callback = done;
            done = null;
            path = null;
            nonce = null;
            callback(result);
        }

        static string ReadNonce(string body)
        {
            if (string.IsNullOrEmpty(body)) return null;
            try
            {
                return JsonUtility.FromJson<NonceReply>(body)?.nonce;
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
