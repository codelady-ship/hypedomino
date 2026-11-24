using System;
using System.Collections.Generic;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Privacy
{
    public enum ConsentPurpose
    {
        Analytics,
        Diagnostics,
    }

    public sealed class FirstLaunchConsents
    {
        public const string Path = "/consents/first-launch";

        readonly List<FirstLaunchChoice> choices = new List<FirstLaunchChoice>();
        readonly string policyVersion;
        readonly string appVersion;

        public FirstLaunchConsents(string policyVersion, string appVersion)
        {
            if (string.IsNullOrEmpty(policyVersion)) throw new ArgumentException("policy version required", nameof(policyVersion));
            if (string.IsNullOrEmpty(appVersion)) throw new ArgumentException("app version required", nameof(appVersion));
            this.policyVersion = policyVersion;
            this.appVersion = appVersion;
        }

        public bool Busy { get; private set; }

        public int Held => choices.Count;

        public void Choose(ConsentPurpose purpose, bool granted) =>
            choices.Add(new FirstLaunchChoice { purpose = WireName(purpose), granted = granted });

        public bool Write(ApiClient api, Action<FirstLaunchConsentsResult> done)
        {
            if (Busy || choices.Count == 0) return false;
            Busy = true;
            var sent = choices.Count;
            var request = new FirstLaunchConsentsRequest { choices = new List<FirstLaunchChoice>(choices), policyVersion = policyVersion, appVersion = appVersion };
            api.Post(Path, request.ToJson(), reply => Answer(reply, sent, done));
            return true;
        }

        public static string WireName(ConsentPurpose purpose) => purpose switch
        {
            ConsentPurpose.Analytics => "analytics",
            ConsentPurpose.Diagnostics => "diagnostics",
            _ => throw new ArgumentOutOfRangeException(nameof(purpose)),
        };

        void Answer(ApiReply reply, int sent, Action<FirstLaunchConsentsResult> done)
        {
            Busy = false;
            var result = FirstLaunchConsentsResult.From(reply);
            if (result.Outcome == FirstLaunchConsentsOutcome.Written) choices.RemoveRange(0, sent);
            done(result);
        }
    }
}
