using System;
using HypeDomino.Net.Api;
using UnityEngine;

namespace HypeDomino.Game.Push
{
    [Serializable]
    public sealed class PushSettingsRequest
    {
        public bool gameplay;
        public bool promotional;
        public string policyVersion;
        public string appVersion;

        public string ToJson() => JsonUtility.ToJson(this);
    }

    [Serializable]
    public sealed class PushSettingsReply
    {
        public bool gameplay;
        public bool promotional;

        public static PushSettingsReply FromJson(string json) => JsonUtility.FromJson<PushSettingsReply>(json);
    }

    public enum PushSettingsOutcome
    {
        Done,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class PushSettingsKeys
    {
        public const string Offline = "push.settings.offline";
        public const string RateLimited = "push.settings.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(PushSettingsOutcome outcome) => outcome switch
        {
            PushSettingsOutcome.Done => null,
            PushSettingsOutcome.Offline => Offline,
            PushSettingsOutcome.RateLimited => RateLimited,
            PushSettingsOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct PushSettingsResult
    {
        PushSettingsResult(PushSettingsOutcome outcome, PushSettingsReply settings = null)
        {
            Outcome = outcome;
            Settings = settings;
        }

        public PushSettingsOutcome Outcome { get; }
        public PushSettingsReply Settings { get; }
        public string MessageKey => PushSettingsKeys.Of(Outcome);

        public static PushSettingsResult From(ApiReply reply) => reply.Outcome switch
        {
            ApiOutcome.Ok => Answered(reply.Body),
            ApiOutcome.Offline => new(PushSettingsOutcome.Offline),
            ApiOutcome.SignInRequired => new(PushSettingsOutcome.SignInRequired),
            ApiOutcome.RateLimited => new(PushSettingsOutcome.RateLimited),
            _ => new(PushSettingsOutcome.Failed),
        };

        static PushSettingsResult Answered(string body)
        {
            var settings = Parse(body);
            return settings == null ? new(PushSettingsOutcome.Failed) : new(PushSettingsOutcome.Done, settings);
        }

        static PushSettingsReply Parse(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return PushSettingsReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
