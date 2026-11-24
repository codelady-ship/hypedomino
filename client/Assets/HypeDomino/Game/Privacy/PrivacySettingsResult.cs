using System;
using HypeDomino.Net.Api;
using UnityEngine;

namespace HypeDomino.Game.Privacy
{
    [Serializable]
    public sealed class ConsentChangeRequest
    {
        public bool analytics;
        public bool diagnostics;
        public string policyVersion;
        public string appVersion;

        public string ToJson() => JsonUtility.ToJson(this);
    }

    public enum PrivacySettingsOutcome
    {
        Done,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class PrivacySettingsKeys
    {
        public const string Offline = "privacy.settings.offline";
        public const string RateLimited = "privacy.settings.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(PrivacySettingsOutcome outcome) => outcome switch
        {
            PrivacySettingsOutcome.Done => null,
            PrivacySettingsOutcome.Offline => Offline,
            PrivacySettingsOutcome.RateLimited => RateLimited,
            PrivacySettingsOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct PrivacySettingsResult
    {
        PrivacySettingsResult(PrivacySettingsOutcome outcome, ConsentChoicesReply choices = null)
        {
            Outcome = outcome;
            Choices = choices;
        }

        public PrivacySettingsOutcome Outcome { get; }
        public ConsentChoicesReply Choices { get; }
        public string MessageKey => PrivacySettingsKeys.Of(Outcome);

        public static PrivacySettingsResult From(ApiReply reply) => reply.Outcome switch
        {
            ApiOutcome.Ok => Answered(reply.Body),
            ApiOutcome.Offline => new(PrivacySettingsOutcome.Offline),
            ApiOutcome.SignInRequired => new(PrivacySettingsOutcome.SignInRequired),
            ApiOutcome.RateLimited => new(PrivacySettingsOutcome.RateLimited),
            _ => new(PrivacySettingsOutcome.Failed),
        };

        static PrivacySettingsResult Answered(string body)
        {
            var choices = Parse(body);
            return choices == null ? new(PrivacySettingsOutcome.Failed) : new(PrivacySettingsOutcome.Done, choices);
        }

        static ConsentChoicesReply Parse(string body)
        {
            if (string.IsNullOrEmpty(body)) return null;
            try
            {
                return ConsentChoicesReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
