using System;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using UnityEngine;

namespace HypeDomino.Game.Lobby
{
    public static class AppealKeys
    {
        public const string Sent = "appeal.sent";
        public const string Expired = "appeal.refused.expired";
        public const string TextRequired = "appeal.refused.text_required";
        public const string TextTooLong = "appeal.refused.text_too_long";
        public const string Open = "appeal.refused.open";
        public const string Rejected = "appeal.refused.rejected";
        public const string Ended = "appeal.refused.ended";
        public const string NotAppealable = "appeal.refused.not_appealable";
        public const string Offline = "appeal.offline";
        public const string Unavailable = "appeal.unavailable";
        public const string Failed = "notice.generic";

        public static string Of(ApiOutcome outcome, string error) => outcome switch
        {
            ApiOutcome.Ok => Sent,
            ApiOutcome.Offline => Offline,
            ApiOutcome.RateLimited or ApiOutcome.Unavailable => Unavailable,
            _ => OfRefusal(error),
        };

        static string OfRefusal(string error) => error switch
        {
            "appeal_token_expired" or "appeal_token_rejected" or "sanction_mismatch" => Expired,
            "text_required" => TextRequired,
            "text_too_long" => TextTooLong,
            "appeal_open" => Open,
            "appeal_rejected" => Rejected,
            "sanction_ended" => Ended,
            "not_appealable" => NotAppealable,
            _ => Failed,
        };
    }

    public readonly struct AppealResult
    {
        const int TooManyRequests = 429;
        const int ServerError = 500;

        [Serializable]
        sealed class ErrorBody
        {
            public string error;
        }

        AppealResult(ApiOutcome outcome, int status, string error)
        {
            Outcome = outcome;
            Status = status;
            Error = error;
        }

        public ApiOutcome Outcome { get; }
        public int Status { get; }
        public string Error { get; }
        public bool Filed => Outcome == ApiOutcome.Ok;
        public string MessageKey => AppealKeys.Of(Outcome, Error);

        public static AppealResult Of(HttpReply reply)
        {
            if (reply.Status == HttpReply.NoResponse) return new(ApiOutcome.Offline, reply.Status, null);
            if (reply.Status >= 200 && reply.Status < 300) return new(ApiOutcome.Ok, reply.Status, null);
            var error = ErrorOf(reply.Body);
            if (reply.Status == TooManyRequests) return new(ApiOutcome.RateLimited, reply.Status, error);
            if (reply.Status >= ServerError) return new(ApiOutcome.Unavailable, reply.Status, error);
            return new(ApiOutcome.Rejected, reply.Status, error);
        }

        static string ErrorOf(string body)
        {
            if (string.IsNullOrEmpty(body)) return null;
            try
            {
                return JsonUtility.FromJson<ErrorBody>(body)?.error;
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }
}
