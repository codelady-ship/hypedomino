using System;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Ranking
{
    public enum RankStandingLoadOutcome
    {
        Loaded,
        NoSeason,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public readonly struct RankStandingLoadResult
    {
        RankStandingLoadResult(RankStandingLoadOutcome outcome, RankStanding standing = null)
        {
            Outcome = outcome;
            Standing = standing;
        }

        public RankStandingLoadOutcome Outcome { get; }
        public RankStanding Standing { get; }

        public static RankStandingLoadResult Loaded(RankStanding standing) => new(RankStandingLoadOutcome.Loaded, standing);

        public static RankStandingLoadResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => RankStandingLoadOutcome.Offline,
            ApiOutcome.SignInRequired => RankStandingLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => RankStandingLoadOutcome.RateLimited,
            ApiOutcome.Rejected when reply.Error == RankStanding.NoSeason => RankStandingLoadOutcome.NoSeason,
            _ => RankStandingLoadOutcome.Failed,
        });

        public static RankStandingLoadResult Unreadable() => new(RankStandingLoadOutcome.Failed);
    }

    public sealed class RankStandingLoader
    {
        readonly ApiClient api;
        readonly Action<ApiReply> answered;
        Action<RankStandingLoadResult> pending;

        public RankStandingLoader(ApiClient api)
        {
            this.api = api;
            answered = Answered;
        }

        public bool Busy => pending != null;

        public bool Load(Action<RankStandingLoadResult> done)
        {
            if (Busy || done == null)
            {
                return false;
            }
            pending = done;
            api.Get(RankStanding.Path, answered);
            return true;
        }

        void Answered(ApiReply reply)
        {
            var done = pending;
            pending = null;
            done(Read(reply));
        }

        static RankStandingLoadResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return RankStandingLoadResult.Refused(reply);
            }
            return RankStanding.TryRead(reply.Body, out var standing) ? RankStandingLoadResult.Loaded(standing) : RankStandingLoadResult.Unreadable();
        }
    }
}
