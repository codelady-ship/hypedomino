using System;
using HypeDomino.Net.Api;
using HypeDomino.UI.Medals;

namespace HypeDomino.Game.Medals
{
    public enum MedalShowcaseOutcome
    {
        Saved,
        Invalid,
        NotOwned,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class MedalShowcaseKeys
    {
        public const string Invalid = "medal.showcase.invalid";
        public const string NotOwned = "medal.showcase.not_owned";
        public const string Offline = "medal.showcase.offline";
        public const string RateLimited = "medal.showcase.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(MedalShowcaseOutcome outcome) => outcome switch
        {
            MedalShowcaseOutcome.Saved => null,
            MedalShowcaseOutcome.Invalid => Invalid,
            MedalShowcaseOutcome.NotOwned => NotOwned,
            MedalShowcaseOutcome.Offline => Offline,
            MedalShowcaseOutcome.RateLimited => RateLimited,
            MedalShowcaseOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct MedalShowcaseResult
    {
        MedalShowcaseResult(MedalShowcaseOutcome outcome, MedalShowcaseReply showcase = null)
        {
            Outcome = outcome;
            Showcase = showcase;
        }

        public MedalShowcaseOutcome Outcome { get; }
        public MedalShowcaseReply Showcase { get; }
        public string MessageKey => MedalShowcaseKeys.Of(Outcome);

        public static MedalShowcaseResult Saved(MedalShowcaseReply showcase) => new(MedalShowcaseOutcome.Saved, showcase);

        public static MedalShowcaseResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => MedalShowcaseOutcome.Offline,
            ApiOutcome.SignInRequired => MedalShowcaseOutcome.SignInRequired,
            ApiOutcome.RateLimited => MedalShowcaseOutcome.RateLimited,
            ApiOutcome.Rejected when reply.Error == "medal.showcase_invalid" => MedalShowcaseOutcome.Invalid,
            ApiOutcome.Rejected when reply.Error == "medal.not_owned" => MedalShowcaseOutcome.NotOwned,
            _ => MedalShowcaseOutcome.Failed,
        });

        public static MedalShowcaseResult Unreadable() => new(MedalShowcaseOutcome.Failed);
    }

    public sealed class MedalShowcaseSaver
    {
        readonly ApiClient api;

        public MedalShowcaseSaver(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Save(MedalShowcaseRequest request, Action<MedalShowcaseResult> done)
        {
            if (Busy || request?.medals == null)
            {
                return false;
            }
            Busy = true;
            api.Put(MedalCollection.ShowcasePath, request.ToJson(), reply => Answered(reply, done));
            return true;
        }

        void Answered(ApiReply reply, Action<MedalShowcaseResult> done)
        {
            Busy = false;
            done(Read(reply));
        }

        static MedalShowcaseResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return MedalShowcaseResult.Refused(reply);
            }
            var showcase = ParseReply(reply.Body);
            return showcase == null ? MedalShowcaseResult.Unreadable() : MedalShowcaseResult.Saved(showcase);
        }

        static MedalShowcaseReply ParseReply(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return MedalShowcaseReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
