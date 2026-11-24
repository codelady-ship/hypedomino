using System;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using UnityEngine;

namespace HypeDomino.Game.Privacy
{
    [Serializable]
    public sealed class DataExportReply
    {
        public string url;
        public string expires_at;
        public string retry_at;
    }

    public enum DataExportOutcome
    {
        Ready,
        Cooldown,
        Unavailable,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class DataExportKeys
    {
        public const string Cooldown = "account.export.cooldown";
        public const string Unavailable = "account.export.unavailable";
        public const string Offline = "account.export.offline";
        public const string RateLimited = "account.export.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(DataExportOutcome outcome) => outcome switch
        {
            DataExportOutcome.Ready => null,
            DataExportOutcome.Cooldown => Cooldown,
            DataExportOutcome.Unavailable => Unavailable,
            DataExportOutcome.Offline => Offline,
            DataExportOutcome.RateLimited => RateLimited,
            DataExportOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct DataExportResult
    {
        const string CooldownError = "export_cooldown";

        DataExportResult(DataExportOutcome outcome, string url = null, ulong expiresAtUnixMs = 0, ulong retryAtUnixMs = 0)
        {
            Outcome = outcome;
            Url = url;
            ExpiresAtUnixMs = expiresAtUnixMs;
            RetryAtUnixMs = retryAtUnixMs;
        }

        public DataExportOutcome Outcome { get; }
        public string Url { get; }
        public ulong ExpiresAtUnixMs { get; }
        public ulong RetryAtUnixMs { get; }
        public string MessageKey => DataExportKeys.Of(Outcome);

        public static DataExportResult From(ApiReply reply) => reply.Outcome switch
        {
            ApiOutcome.Ok => Ready(reply.Body),
            ApiOutcome.RateLimited when reply.Error == CooldownError => Cooldown(reply.Body),
            ApiOutcome.RateLimited => new(DataExportOutcome.RateLimited),
            ApiOutcome.Offline => new(DataExportOutcome.Offline),
            ApiOutcome.SignInRequired => new(DataExportOutcome.SignInRequired),
            ApiOutcome.Unavailable => new(DataExportOutcome.Unavailable),
            _ => new(DataExportOutcome.Failed),
        };

        static DataExportResult Ready(string body)
        {
            var reply = Parse(body);
            if (reply == null || !IsWebLink(reply.url)) return new(DataExportOutcome.Failed);
            if (!ServerClock.TryReadUnixMs(reply.expires_at, out var expiresAtUnixMs)) return new(DataExportOutcome.Failed);
            return new(DataExportOutcome.Ready, reply.url, expiresAtUnixMs);
        }

        static DataExportResult Cooldown(string body)
        {
            var reply = Parse(body);
            ServerClock.TryReadUnixMs(reply?.retry_at, out var retryAtUnixMs);
            return new(DataExportOutcome.Cooldown, retryAtUnixMs: retryAtUnixMs);
        }

        static bool IsWebLink(string url) =>
            Uri.TryCreate(url, UriKind.Absolute, out var link) && (link.Scheme == Uri.UriSchemeHttps || link.Scheme == Uri.UriSchemeHttp) && !string.IsNullOrEmpty(link.Host);

        static DataExportReply Parse(string body)
        {
            if (string.IsNullOrEmpty(body)) return null;
            try
            {
                return JsonUtility.FromJson<DataExportReply>(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
