using System;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Pass
{
    public enum PassLoadOutcome
    {
        Loaded,
        NoSeason,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class PassLoadKeys
    {
        public const string Offline = "pass.load.offline";
        public const string RateLimited = "pass.load.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(PassLoadOutcome outcome) => outcome switch
        {
            PassLoadOutcome.Loaded => null,
            PassLoadOutcome.NoSeason => PassStanding.NoSeason,
            PassLoadOutcome.Offline => Offline,
            PassLoadOutcome.RateLimited => RateLimited,
            PassLoadOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct PassLoadResult
    {
        PassLoadResult(PassLoadOutcome outcome, PassStanding standing = null, PassTracks tracks = null)
        {
            Outcome = outcome;
            Standing = standing;
            Tracks = tracks;
        }

        public PassLoadOutcome Outcome { get; }
        public PassStanding Standing { get; }
        public PassTracks Tracks { get; }
        public string MessageKey => PassLoadKeys.Of(Outcome);

        public static PassLoadResult Loaded(PassStanding standing, PassTracks tracks) => new(PassLoadOutcome.Loaded, standing, tracks);

        public static PassLoadResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => PassLoadOutcome.Offline,
            ApiOutcome.SignInRequired => PassLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => PassLoadOutcome.RateLimited,
            ApiOutcome.Rejected when reply.Error == PassStanding.NoSeason => PassLoadOutcome.NoSeason,
            _ => PassLoadOutcome.Failed,
        });

        public static PassLoadResult Unreadable() => new(PassLoadOutcome.Failed);
    }

    public sealed class PassLoader
    {
        readonly ApiClient api;

        public PassLoader(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Load(Action<PassLoadResult> done)
        {
            if (Busy)
            {
                return false;
            }
            Busy = true;
            api.Get(PassStanding.Path, reply => Answered(reply, done));
            return true;
        }

        void Answered(ApiReply reply, Action<PassLoadResult> done)
        {
            Busy = false;
            done(Read(reply));
        }

        static PassLoadResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return PassLoadResult.Refused(reply);
            }
            if (!PassStanding.TryRead(ParseReply(reply.Body), out var standing))
            {
                return PassLoadResult.Unreadable();
            }
            return PassTracks.TryRead(reply.Body, standing.Levels, out var tracks) ? PassLoadResult.Loaded(standing, tracks) : PassLoadResult.Unreadable();
        }

        static PassReply ParseReply(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return PassReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
