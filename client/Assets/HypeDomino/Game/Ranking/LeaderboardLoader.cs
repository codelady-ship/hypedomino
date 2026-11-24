using System;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Ranking
{
    public enum LeaderboardLoadOutcome
    {
        Loaded,
        NoSeason,
        UnknownBoard,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public readonly struct LeaderboardLoadResult
    {
        const string SeasonUnknownError = "rating.season_unknown";
        const string UnknownBoardError = "leaderboard.unknown_board";

        LeaderboardLoadResult(LeaderboardLoadOutcome outcome, LeaderboardStanding standing = null)
        {
            Outcome = outcome;
            Standing = standing;
        }

        public LeaderboardLoadOutcome Outcome { get; }
        public LeaderboardStanding Standing { get; }

        public static LeaderboardLoadResult Loaded(LeaderboardStanding standing) => new(LeaderboardLoadOutcome.Loaded, standing);

        public static LeaderboardLoadResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => LeaderboardLoadOutcome.Offline,
            ApiOutcome.SignInRequired => LeaderboardLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => LeaderboardLoadOutcome.RateLimited,
            ApiOutcome.Rejected when reply.Error == SeasonUnknownError => LeaderboardLoadOutcome.NoSeason,
            ApiOutcome.Rejected when reply.Error == UnknownBoardError => LeaderboardLoadOutcome.UnknownBoard,
            _ => LeaderboardLoadOutcome.Failed,
        });

        public static LeaderboardLoadResult Unreadable() => new(LeaderboardLoadOutcome.Failed);
    }

    public sealed class LeaderboardLoader
    {
        readonly ApiClient api;

        public LeaderboardLoader(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Load(string mode, string board, Action<LeaderboardLoadResult> done)
        {
            if (Busy || string.IsNullOrEmpty(mode) || string.IsNullOrEmpty(board))
            {
                return false;
            }
            Busy = true;
            api.Get(LeaderboardStanding.PathOf(mode, board), reply => Answered(reply, done));
            return true;
        }

        void Answered(ApiReply reply, Action<LeaderboardLoadResult> done)
        {
            Busy = false;
            done(Read(reply));
        }

        static LeaderboardLoadResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return LeaderboardLoadResult.Refused(reply);
            }
            return LeaderboardStanding.TryRead(reply.Body, out var standing) ? LeaderboardLoadResult.Loaded(standing) : LeaderboardLoadResult.Unreadable();
        }
    }
}
