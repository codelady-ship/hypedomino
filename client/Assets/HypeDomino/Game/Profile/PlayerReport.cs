using System;
using HypeDomino.Net.Api;
using UnityEngine;

namespace HypeDomino.Game.Profile
{
    public enum PlayerReportReason
    {
        OffensiveNickname,
        OffensiveRoomName,
        Impersonation,
        Other,
    }

    public enum PlayerReportOutcome
    {
        Received,
        UnknownPlayer,
        Self,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    [Serializable]
    public sealed class PlayerReportRequest
    {
        public string target;
        public string reason;
        public string match_id;

        public string ToJson() => JsonUtility.ToJson(this);
    }

    public static class PlayerReportKeys
    {
        public const string Received = "report.send.received";
        public const string UnknownPlayer = "report.send.unknown_player";
        public const string Self = "report.send.self";
        public const string Offline = "report.send.offline";
        public const string RateLimited = "report.send.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(PlayerReportOutcome outcome) => outcome switch
        {
            PlayerReportOutcome.Received => Received,
            PlayerReportOutcome.UnknownPlayer => UnknownPlayer,
            PlayerReportOutcome.Self => Self,
            PlayerReportOutcome.Offline => Offline,
            PlayerReportOutcome.RateLimited => RateLimited,
            PlayerReportOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct PlayerReportResult
    {
        PlayerReportResult(PlayerReportOutcome outcome)
        {
            Outcome = outcome;
        }

        public PlayerReportOutcome Outcome { get; }
        public string MessageKey => PlayerReportKeys.Of(Outcome);

        public static PlayerReportResult From(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Ok => PlayerReportOutcome.Received,
            ApiOutcome.Offline => PlayerReportOutcome.Offline,
            ApiOutcome.SignInRequired => PlayerReportOutcome.SignInRequired,
            ApiOutcome.RateLimited => PlayerReportOutcome.RateLimited,
            ApiOutcome.Rejected when reply.Error == "report.unknown_player" => PlayerReportOutcome.UnknownPlayer,
            ApiOutcome.Rejected when reply.Error == "report.self" => PlayerReportOutcome.Self,
            _ => PlayerReportOutcome.Failed,
        });
    }

    public sealed class PlayerReport
    {
        public const string Path = "/reports";

        readonly ApiClient api;

        public PlayerReport(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public static string Wire(PlayerReportReason reason) => reason switch
        {
            PlayerReportReason.OffensiveNickname => "offensive_nickname",
            PlayerReportReason.OffensiveRoomName => "offensive_room_name",
            PlayerReportReason.Impersonation => "impersonation",
            PlayerReportReason.Other => "other",
            _ => string.Empty,
        };

        public bool Send(string targetAccountId, PlayerReportReason reason, string matchId, Action<PlayerReportResult> done)
        {
            if (Busy || string.IsNullOrEmpty(targetAccountId))
            {
                return false;
            }
            Busy = true;
            var request = new PlayerReportRequest { target = targetAccountId, reason = Wire(reason), match_id = matchId ?? string.Empty };
            api.Post(Path, request.ToJson(), reply =>
            {
                Busy = false;
                done(PlayerReportResult.From(reply));
            });
            return true;
        }
    }
}
