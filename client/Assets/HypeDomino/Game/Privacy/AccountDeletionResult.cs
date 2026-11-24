using System;
using HypeDomino.Net;
using HypeDomino.Net.Auth;
using UnityEngine;

namespace HypeDomino.Game.Privacy
{
    [Serializable]
    public sealed class AccountDeletionRequest
    {
        public string provider;
        public string id_token;
        public string nonce;
    }

    [Serializable]
    public sealed class AccountDeletionReply
    {
        public string error;
        public string delete_at;
    }

    public enum AccountDeletionOutcome
    {
        Pending,
        Restored,
        SignInCancelled,
        SignInFailed,
        InMatch,
        NotPending,
        UnknownAccount,
        Banned,
        Deleted,
        Offline,
        RateLimited,
        Unavailable,
        Failed,
    }

    public static class AccountDeletionKeys
    {
        public const string Pending = "account.deletion.pending";
        public const string PendingUndated = "account.deletion.pending_undated";
        public const string Restored = "account.deletion.restored";
        public const string SignInFailed = "account.deletion.sign_in_failed";
        public const string InMatch = "account.deletion.in_match";
        public const string NotPending = "account.deletion.not_pending";
        public const string UnknownAccount = "account.deletion.unknown_account";
        public const string Banned = "account.deletion.banned";
        public const string Deleted = "account.deletion.deleted";
        public const string Offline = "account.deletion.offline";
        public const string RateLimited = "account.deletion.rate_limited";
        public const string Unavailable = "account.deletion.unavailable";
        public const string Failed = "notice.generic";
        public const string DeleteAtArgument = "deleteAt";

        public static string Of(AccountDeletionOutcome outcome) => outcome switch
        {
            AccountDeletionOutcome.Pending => Pending,
            AccountDeletionOutcome.Restored => Restored,
            AccountDeletionOutcome.SignInCancelled => null,
            AccountDeletionOutcome.SignInFailed => SignInFailed,
            AccountDeletionOutcome.InMatch => InMatch,
            AccountDeletionOutcome.NotPending => NotPending,
            AccountDeletionOutcome.UnknownAccount => UnknownAccount,
            AccountDeletionOutcome.Banned => Banned,
            AccountDeletionOutcome.Deleted => Deleted,
            AccountDeletionOutcome.Offline => Offline,
            AccountDeletionOutcome.RateLimited => RateLimited,
            AccountDeletionOutcome.Unavailable => Unavailable,
            _ => Failed,
        };
    }

    public readonly struct AccountDeletionResult
    {
        const int Accepted = 202;
        const int NoContent = 204;
        const int TooManyRequests = 429;
        const int ServerError = 500;

        public static readonly AccountDeletionResult SignInCancelled = new(AccountDeletionOutcome.SignInCancelled);
        public static readonly AccountDeletionResult SignInFailed = new(AccountDeletionOutcome.SignInFailed);
        public static readonly AccountDeletionResult Unavailable = new(AccountDeletionOutcome.Unavailable);

        AccountDeletionResult(AccountDeletionOutcome outcome, ulong deleteAtUnixMs = 0)
        {
            Outcome = outcome;
            DeleteAtUnixMs = deleteAtUnixMs;
        }

        public AccountDeletionOutcome Outcome { get; }
        public ulong DeleteAtUnixMs { get; }

        public string MessageKey => Outcome == AccountDeletionOutcome.Pending && DeleteAtUnixMs == 0
            ? AccountDeletionKeys.PendingUndated
            : AccountDeletionKeys.Of(Outcome);

        public DateTimeOffset DeleteAt => DateTimeOffset.FromUnixTimeMilliseconds((long)DeleteAtUnixMs);

        public static AccountDeletionResult FromRequest(HttpReply reply) =>
            reply.Status == Accepted ? Pending(Parse(reply.Body)?.delete_at) : Refused(reply);

        public static AccountDeletionResult FromCancel(HttpReply reply) =>
            reply.Status == NoContent ? new(AccountDeletionOutcome.Restored) : Refused(reply);

        public static bool TryPending(AuthResult signIn, out AccountDeletionResult pending)
        {
            pending = default;
            if (signIn.Outcome != AuthOutcome.DeletionPending) return false;
            pending = Pending(signIn.DeleteAt);
            return true;
        }

        public static AccountDeletionResult Refused(HttpReply reply)
        {
            if (reply.Status == HttpReply.NoResponse) return new(AccountDeletionOutcome.Offline);
            if (reply.Status == TooManyRequests) return new(AccountDeletionOutcome.RateLimited);
            if (reply.Status >= ServerError) return Unavailable;
            return new(OutcomeOf(Parse(reply.Body)?.error));
        }

        static AccountDeletionOutcome OutcomeOf(string error) => error switch
        {
            "token_expired" or "nonce_rejected" or "token_rejected" => AccountDeletionOutcome.SignInFailed,
            "in_match" => AccountDeletionOutcome.InMatch,
            "no_deletion_pending" => AccountDeletionOutcome.NotPending,
            "account_unknown" => AccountDeletionOutcome.UnknownAccount,
            "banned" => AccountDeletionOutcome.Banned,
            "deleted" => AccountDeletionOutcome.Deleted,
            _ => AccountDeletionOutcome.Failed,
        };

        static AccountDeletionResult Pending(string deleteAt)
        {
            ServerClock.TryReadUnixMs(deleteAt, out var deleteAtUnixMs);
            return new(AccountDeletionOutcome.Pending, deleteAtUnixMs);
        }

        static AccountDeletionReply Parse(string body)
        {
            if (string.IsNullOrEmpty(body)) return null;
            try
            {
                return JsonUtility.FromJson<AccountDeletionReply>(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
