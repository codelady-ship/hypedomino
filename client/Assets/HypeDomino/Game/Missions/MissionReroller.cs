using System;
using HypeDomino.Net.Api;
using HypeDomino.UI.Missions;

namespace HypeDomino.Game.Missions
{
    public enum MissionRerollOutcome
    {
        Rerolled,
        RerollUsed,
        NotAssigned,
        NoReplacement,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class MissionRerollKeys
    {
        public const string RerollUsed = "mission.reroll.used";
        public const string NotAssigned = "mission.reroll.not_assigned";
        public const string NoReplacement = "mission.reroll.no_replacement";
        public const string Offline = "mission.reroll.offline";
        public const string RateLimited = "mission.reroll.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(MissionRerollOutcome outcome) => outcome switch
        {
            MissionRerollOutcome.Rerolled => null,
            MissionRerollOutcome.RerollUsed => RerollUsed,
            MissionRerollOutcome.NotAssigned => NotAssigned,
            MissionRerollOutcome.NoReplacement => NoReplacement,
            MissionRerollOutcome.Offline => Offline,
            MissionRerollOutcome.RateLimited => RateLimited,
            MissionRerollOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct MissionRerollResult
    {
        MissionRerollResult(MissionRerollOutcome outcome, MissionBoard board = null)
        {
            Outcome = outcome;
            Board = board;
        }

        public MissionRerollOutcome Outcome { get; }
        public MissionBoard Board { get; }
        public string MessageKey => MissionRerollKeys.Of(Outcome);

        public static MissionRerollResult Rerolled(MissionBoard board) => new(MissionRerollOutcome.Rerolled, board);

        public static MissionRerollResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => MissionRerollOutcome.Offline,
            ApiOutcome.SignInRequired => MissionRerollOutcome.SignInRequired,
            ApiOutcome.RateLimited => MissionRerollOutcome.RateLimited,
            ApiOutcome.Rejected when reply.Error == "mission.reroll_used" => MissionRerollOutcome.RerollUsed,
            ApiOutcome.Rejected when reply.Error == "mission.not_assigned" => MissionRerollOutcome.NotAssigned,
            ApiOutcome.Rejected when reply.Error == "mission.no_replacement" => MissionRerollOutcome.NoReplacement,
            _ => MissionRerollOutcome.Failed,
        });

        public static MissionRerollResult Unreadable() => new(MissionRerollOutcome.Failed);
    }

    public sealed class MissionReroller
    {
        readonly ApiClient api;

        public MissionReroller(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Reroll(MissionBoard board, in MissionCard card, Action<MissionRerollResult> done)
        {
            if (Busy || board == null || string.IsNullOrEmpty(card.Id) || !board.CanReroll(card))
            {
                return false;
            }
            Busy = true;
            api.Post(MissionBoard.RerollPath, RerollRequest.Json(card.Id), reply => Answered(reply, done));
            return true;
        }

        void Answered(ApiReply reply, Action<MissionRerollResult> done)
        {
            Busy = false;
            done(Read(reply));
        }

        static MissionRerollResult Read(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                return MissionRerollResult.Refused(reply);
            }
            return MissionBoard.TryFrom(MissionLoader.ParseReply(reply.Body), out var board) ? MissionRerollResult.Rerolled(board) : MissionRerollResult.Unreadable();
        }
    }
}
