using System;
using System.Security.Cryptography;
using System.Text;
using UnityEngine;

namespace HypeDomino.Net.Auth
{
    public sealed class AuthSession
    {
        public const string RefreshTokenKey = "auth.refreshToken";
        public const string DeviceIdKey = "auth.deviceId";
        public const string NoncePath = "/auth/nonce";
        public const string SignInPath = "/auth/signin";
        public const string RefreshPath = "/auth/refresh";
        public const string LogoutPath = "/auth/logout";

        readonly IAuthTransport transport;
        readonly IIdentityProvider identity;
        readonly ISecureStore store;
        bool busy;

        public AuthSession(IAuthTransport transport, IIdentityProvider identity, ISecureStore store)
        {
            this.transport = transport;
            this.identity = identity;
            this.store = store;
        }

        public string AccountId { get; private set; }

        public string AccessToken { get; private set; }

        public byte[] RequestKey { get; private set; }

        public int ExpiresInSeconds { get; private set; }

        public bool SignedIn => AccessToken != null;

        public bool HasStoredSession => !string.IsNullOrEmpty(store.Read(RefreshTokenKey));

        public event Action<AuthProvider> ProviderSignedIn;

        public void Resume(Action<AuthResult> done)
        {
            var refresh = store.Read(RefreshTokenKey);
            if (string.IsNullOrEmpty(refresh))
            {
                done(new AuthResult(AuthOutcome.SignInRequired));
                return;
            }
            if (!Begin(done)) return;
            var request = JsonUtility.ToJson(new RefreshRequest { refresh_token = refresh, device_id = DeviceId() });
            transport.Post(RefreshPath, request, reply => End(done, Tokens(reply, true)));
        }

        public void SignIn(AuthProvider provider, Action<AuthResult> done)
        {
            if (!identity.Supports(provider))
            {
                done(new AuthResult(AuthOutcome.Unsupported));
                return;
            }
            if (!Begin(done)) return;
            transport.Post(NoncePath, "{}", reply => Nonce(provider, reply, done));
        }

        public void SignOut(Action done)
        {
            var refresh = store.Read(RefreshTokenKey);
            Forget();
            if (string.IsNullOrEmpty(refresh))
            {
                done();
                return;
            }
            transport.Post(LogoutPath, JsonUtility.ToJson(new LogoutRequest { refresh_token = refresh }), _ => done());
        }

        public static string ProviderNonce(AuthProvider provider, string nonce)
        {
            if (provider != AuthProvider.Apple) return nonce;
            using var sha = SHA256.Create();
            var digest = sha.ComputeHash(Encoding.UTF8.GetBytes(nonce));
            var hex = new StringBuilder(digest.Length * 2);
            for (var i = 0; i < digest.Length; i++) hex.Append(digest[i].ToString("x2"));
            return hex.ToString();
        }

        public static string ProviderName(AuthProvider provider) => provider == AuthProvider.Apple ? "apple" : "google";

        void Nonce(AuthProvider provider, HttpReply reply, Action<AuthResult> done)
        {
            var nonce = reply.Status == 200 ? AuthJson.Read<NonceReply>(reply.Body)?.nonce : null;
            if (string.IsNullOrEmpty(nonce))
            {
                End(done, reply.Status == 200 ? new AuthResult(AuthOutcome.Unavailable) : Failure(reply, false));
                return;
            }
            identity.RequestIdToken(provider, ProviderNonce(provider, nonce), token => IdToken(provider, nonce, token, done));
        }

        void IdToken(AuthProvider provider, string nonce, IdTokenResult token, Action<AuthResult> done)
        {
            if (token.Outcome == IdTokenOutcome.Cancelled)
            {
                End(done, new AuthResult(AuthOutcome.Cancelled));
                return;
            }
            if (token.Outcome != IdTokenOutcome.Token || string.IsNullOrEmpty(token.Token))
            {
                End(done, new AuthResult(AuthOutcome.ProviderFailed));
                return;
            }
            var request = JsonUtility.ToJson(new SignInRequest { provider = ProviderName(provider), id_token = token.Token, nonce = nonce, device_id = DeviceId() });
            transport.Post(SignInPath, request, reply => ProviderAnswered(provider, reply, done));
        }

        void ProviderAnswered(AuthProvider provider, HttpReply reply, Action<AuthResult> done)
        {
            var result = Tokens(reply, false);
            if (result.Outcome == AuthOutcome.SignedIn) ProviderSignedIn?.Invoke(provider);
            End(done, result);
        }

        AuthResult Tokens(HttpReply reply, bool refreshing)
        {
            if (reply.Status != 200) return Failure(reply, refreshing);
            var tokens = AuthJson.Read<TokenReply>(reply.Body);
            var key = RequestKeyBytes(tokens?.request_key);
            if (tokens == null || string.IsNullOrEmpty(tokens.access_token) || string.IsNullOrEmpty(tokens.refresh_token) || key == null)
                return new AuthResult(AuthOutcome.Unavailable);
            store.Write(RefreshTokenKey, tokens.refresh_token);
            AccountId = tokens.account_id;
            AccessToken = tokens.access_token;
            RequestKey = key;
            ExpiresInSeconds = tokens.expires_in;
            return new AuthResult(AuthOutcome.SignedIn, tokens.created);
        }

        AuthResult Failure(HttpReply reply, bool refreshing)
        {
            if (reply.Status == HttpReply.NoResponse) return new AuthResult(AuthOutcome.Offline);
            var error = AuthJson.Read<ErrorReply>(reply.Body);
            var code = error?.error;
            if (reply.Status == 429) return new AuthResult(AuthOutcome.RateLimited, error: code);
            if (reply.Status >= 500) return new AuthResult(AuthOutcome.Unavailable, error: code);
            if (refreshing) Forget();
            switch (code)
            {
                case "banned":
                    ServerClock.TryReadUnixMs(error.banned_until, out var bannedUntilUnixMs);
                    return new AuthResult(AuthOutcome.Banned, error: code, banCategory: error.category, appeal: SanctionAppeal.Of(error.appeal), bannedUntilUnixMs: bannedUntilUnixMs);
                case "deleted": return new AuthResult(AuthOutcome.Deleted, error: code);
                case "deletion_pending": return new AuthResult(AuthOutcome.DeletionPending, error: code, deleteAt: error.delete_at);
            }
            return new AuthResult(refreshing ? AuthOutcome.SignInRequired : AuthOutcome.Rejected, error: code);
        }

        static byte[] RequestKeyBytes(string base64)
        {
            if (string.IsNullOrEmpty(base64)) return null;
            try
            {
                return Convert.FromBase64String(base64);
            }
            catch (FormatException)
            {
                return null;
            }
        }

        string DeviceId()
        {
            var id = store.Read(DeviceIdKey);
            if (!string.IsNullOrEmpty(id)) return id;
            id = Guid.NewGuid().ToString("N");
            store.Write(DeviceIdKey, id);
            return id;
        }

        void Forget()
        {
            store.Delete(RefreshTokenKey);
            AccountId = null;
            AccessToken = null;
            RequestKey = null;
            ExpiresInSeconds = 0;
        }

        bool Begin(Action<AuthResult> done)
        {
            if (busy)
            {
                done(new AuthResult(AuthOutcome.Busy));
                return false;
            }
            busy = true;
            return true;
        }

        void End(Action<AuthResult> done, AuthResult result)
        {
            busy = false;
            done(result);
        }
    }
}
