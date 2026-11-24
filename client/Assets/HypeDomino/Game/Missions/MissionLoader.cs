using System;
using HypeDomino.Net.Api;
using HypeDomino.UI.Missions;

namespace HypeDomino.Game.Missions
{
    public enum MissionLoadOutcome
    {
        Loaded,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class MissionLoadKeys
    {
        public const string Offline = "mission.load.offline";
        public const string RateLimited = "mission.load.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(MissionLoadOutcome outcome) => outcome switch
        {
            MissionLoadOutcome.Loaded => null,
            MissionLoadOutcome.Offline => Offline,
            MissionLoadOutcome.RateLimited => RateLimited,
            MissionLoadOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct MissionLoadResult
    {
        MissionLoadResult(MissionLoadOutcome outcome, MissionBoard board = null)
        {
            Outcome = outcome;
            Board = board;
        }

        public MissionLoadOutcome Outcome { get; }
        public MissionBoard Board { get; }
        public string MessageKey => MissionLoadKeys.Of(Outcome);

        public static MissionLoadResult Loaded(MissionBoard board) => new(MissionLoadOutcome.Loaded, board);

        public static MissionLoadResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => MissionLoadOutcome.Offline,
            ApiOutcome.SignInRequired => MissionLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => MissionLoadOutcome.RateLimited,
            _ => MissionLoadOutcome.Failed,
        });

        public static MissionLoadResult Unreadable() => new(MissionLoadOutcome.Failed);
    }

    public sealed class MissionLoader
    {
        readonly ApiClient api;

        public MissionLoader(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Load(Action<MissionLoadResult> done)
        {
            if (Busy)
            {
                return false;
            }
            Busy = true;
            api.Get(MissionBoard.BoardPath, reply => Answered(reply, done));
            return true;
        }

        void Answered(ApiReply reply, Action<MissionLoadResult> done)
        {
            Busy = false;
            done(Read(reply));
        }

        static MissionLoadResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return MissionLoadResult.Refused(reply);
            }
            return MissionBoard.TryFrom(ParseReply(reply.Body), out var board) ? MissionLoadResult.Loaded(board) : MissionLoadResult.Unreadable();
        }

        internal static MissionBoardReply ParseReply(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return MissionBoardReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
