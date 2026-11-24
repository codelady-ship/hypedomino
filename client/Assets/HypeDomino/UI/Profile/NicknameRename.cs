using System;
using HypeDomino.Net;
using UnityEngine;

namespace HypeDomino.UI.Profile
{
    [Serializable]
    public sealed class NicknameRequest
    {
        public string nickname;

        public string ToJson() => JsonUtility.ToJson(this);
    }

    public enum NicknameRenameStatus
    {
        Unreadable,
        Accepted,
        Refused,
    }

    public readonly struct NicknameRenameOutcome
    {
        NicknameRenameOutcome(NicknameRenameStatus status, PlayerProfile profile, string reason, ulong retryAtUnixMs)
        {
            Status = status;
            Profile = profile;
            Reason = reason;
            RetryAtUnixMs = retryAtUnixMs;
        }

        public NicknameRenameStatus Status { get; }
        public PlayerProfile Profile { get; }
        public string Reason { get; }
        public ulong RetryAtUnixMs { get; }

        public static NicknameRenameOutcome Read(string json)
        {
            if (!JsonMap.TryParse(json, out var reply)) return default;
            if (!reply.IsAbsent("error")) return Refused(reply);
            return PlayerProfile.TryFromOwn(json, out var profile) ? new NicknameRenameOutcome(NicknameRenameStatus.Accepted, profile, null, 0) : default;
        }

        static NicknameRenameOutcome Refused(JsonMap reply)
        {
            if (!reply.TryString("error", out var reason) || reason.Length == 0) return default;
            ServerClock.TryReadUnixMs(reply.TryString("retry_at", out var retryAt) ? retryAt : null, out var retryAtUnixMs);
            return new NicknameRenameOutcome(NicknameRenameStatus.Refused, null, reason, retryAtUnixMs);
        }
    }

    public static class NicknameRename
    {
        public const string Path = "/profile/nickname";

        public static bool TryRequest(string nickname, out NicknameRequest request)
        {
            request = null;
            if (string.IsNullOrWhiteSpace(nickname)) return false;
            request = new NicknameRequest { nickname = nickname };
            return true;
        }
    }
}
