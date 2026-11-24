using System;
using System.Collections.Generic;
using HypeDomino.Net.Api;
using UnityEngine;

namespace HypeDomino.Game.Privacy
{
    [Serializable]
    public sealed class FirstLaunchChoice
    {
        public string purpose;
        public bool granted;
    }

    [Serializable]
    public sealed class FirstLaunchConsentsRequest
    {
        public List<FirstLaunchChoice> choices;
        public string policyVersion;
        public string appVersion;

        public string ToJson() => JsonUtility.ToJson(this);
    }

    [Serializable]
    public sealed class ConsentChoicesReply
    {
        public bool analytics;
        public bool diagnostics;

        public static ConsentChoicesReply FromJson(string json) => JsonUtility.FromJson<ConsentChoicesReply>(json);
    }

    public enum FirstLaunchConsentsOutcome
    {
        Written,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public readonly struct FirstLaunchConsentsResult
    {
        FirstLaunchConsentsResult(FirstLaunchConsentsOutcome outcome, ConsentChoicesReply choices = null)
        {
            Outcome = outcome;
            Choices = choices;
        }

        public FirstLaunchConsentsOutcome Outcome { get; }
        public ConsentChoicesReply Choices { get; }

        public static FirstLaunchConsentsResult From(ApiReply reply) => reply.Outcome switch
        {
            ApiOutcome.Ok => Answered(reply.Body),
            ApiOutcome.Offline => new(FirstLaunchConsentsOutcome.Offline),
            ApiOutcome.SignInRequired => new(FirstLaunchConsentsOutcome.SignInRequired),
            ApiOutcome.RateLimited => new(FirstLaunchConsentsOutcome.RateLimited),
            _ => new(FirstLaunchConsentsOutcome.Failed),
        };

        static FirstLaunchConsentsResult Answered(string body)
        {
            var choices = Parse(body);
            return choices == null ? new(FirstLaunchConsentsOutcome.Failed) : new(FirstLaunchConsentsOutcome.Written, choices);
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
