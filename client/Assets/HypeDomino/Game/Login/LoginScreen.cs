using System;
using System.Collections.Generic;
using HypeDomino.Game.Lobby;
using HypeDomino.Game.Privacy;
using HypeDomino.Net;
using HypeDomino.Net.Auth;

namespace HypeDomino.Game.Login
{
    public static class LoginKeys
    {
        public const string Unsupported = "login.unsupported";
        public const string Unavailable = "login.unavailable";
        public const string Rejected = "login.rejected";
        public const string Banned = "login.banned";
        public const string BanEndsIn = "login.banned.ends_in";
        public const string BanEndsInArgument = "time";

        public static string Of(AuthOutcome outcome) => outcome switch
        {
            AuthOutcome.SignedIn => null,
            AuthOutcome.Cancelled => null,
            AuthOutcome.Busy => null,
            AuthOutcome.Unsupported => Unsupported,
            AuthOutcome.ProviderFailed => AccountDeletionKeys.SignInFailed,
            AuthOutcome.Offline => AccountDeletionKeys.Offline,
            AuthOutcome.Unavailable => Unavailable,
            AuthOutcome.RateLimited => AccountDeletionKeys.RateLimited,
            AuthOutcome.Rejected => Rejected,
            AuthOutcome.Banned => Banned,
            AuthOutcome.Deleted => AccountDeletionKeys.Deleted,
            _ => AccountDeletionKeys.Failed,
        };
    }

    public sealed class LoginScreen
    {
        readonly AuthSession auth;
        readonly AccountDeletion deletion;
        readonly Action signedIn;
        readonly Action<AuthResult> signInAnswered;
        readonly Action<AccountDeletionResult> cancelAnswered;
        AuthProvider provider;

        public LoginScreen(AuthSession auth, AccountDeletion deletion, IReadOnlyList<AuthProvider> options, Action signedIn)
        {
            this.auth = auth ?? throw new ArgumentNullException(nameof(auth));
            this.deletion = deletion ?? throw new ArgumentNullException(nameof(deletion));
            Options = options ?? throw new ArgumentNullException(nameof(options));
            this.signedIn = signedIn ?? throw new ArgumentNullException(nameof(signedIn));
            signInAnswered = SignInAnswered;
            cancelAnswered = CancelAnswered;
        }

        public IReadOnlyList<AuthProvider> Options { get; }
        public bool Busy { get; private set; }
        public bool DeletionPending { get; private set; }
        public DateTimeOffset DeleteAt { get; private set; }
        public string MessageKey { get; private set; }
        public string BanReasonKey { get; private set; }
        public SanctionAppeal Appeal { get; private set; }
        public ulong BannedUntilUnixMs { get; private set; }

        public event Action Changed;

        public bool TrySignIn(AuthProvider chosen)
        {
            if (Busy || !Offers(chosen)) return false;
            provider = chosen;
            Busy = true;
            DeletionPending = false;
            MessageKey = null;
            BanReasonKey = null;
            Appeal = default;
            BannedUntilUnixMs = 0;
            Changed?.Invoke();
            auth.SignIn(chosen, signInAnswered);
            return true;
        }

        public bool TryCancelDeletion()
        {
            if (Busy || !DeletionPending) return false;
            Busy = true;
            if (!deletion.Cancel(provider, cancelAnswered))
            {
                Busy = false;
                return false;
            }
            Changed?.Invoke();
            return true;
        }

        public double SecondsUntilBanEnds(ServerClock clock, double nowSeconds) =>
            BannedUntilUnixMs != 0 && clock.Synced ? Math.Max(0d, clock.SecondsUntil(BannedUntilUnixMs, nowSeconds)) : 0d;

        public bool Offers(AuthProvider chosen)
        {
            for (var i = 0; i < Options.Count; i++)
            {
                if (Options[i] == chosen) return true;
            }
            return false;
        }

        void SignInAnswered(AuthResult result)
        {
            Busy = false;
            if (AccountDeletionResult.TryPending(result, out var pending))
            {
                DeletionPending = true;
                DeleteAt = pending.DeleteAt;
                MessageKey = pending.MessageKey;
            }
            else
            {
                MessageKey = LoginKeys.Of(result.Outcome);
                BanReasonKey = result.Outcome == AuthOutcome.Banned ? BanReasonKeys.Of(result.BanCategory) : null;
                Appeal = result.Outcome == AuthOutcome.Banned ? result.Appeal : default;
                BannedUntilUnixMs = result.Outcome == AuthOutcome.Banned ? result.BannedUntilUnixMs : 0;
            }
            Changed?.Invoke();
            if (result.Outcome == AuthOutcome.SignedIn) signedIn();
        }

        void CancelAnswered(AccountDeletionResult result)
        {
            Busy = false;
            if (result.Outcome == AccountDeletionOutcome.Restored || result.Outcome == AccountDeletionOutcome.NotPending) DeletionPending = false;
            MessageKey = result.MessageKey;
            Changed?.Invoke();
        }
    }
}
