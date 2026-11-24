using System;
using Hypedomino.V1;

namespace HypeDomino.Net
{
    [Serializable]
    public sealed class AppealGrantReply
    {
        public long sanction_id;
        public string token;
        public string expires_at;
    }

    public readonly struct SanctionAppeal
    {
        public SanctionAppeal(long sanctionId, string token, ulong expiresUnixMs)
        {
            SanctionId = sanctionId;
            Token = token;
            ExpiresUnixMs = expiresUnixMs;
        }

        public long SanctionId { get; }
        public string Token { get; }
        public ulong ExpiresUnixMs { get; }
        public bool Offered => SanctionId > 0 && !string.IsNullOrEmpty(Token);

        public static SanctionAppeal Of(AppealGrantReply reply)
        {
            if (reply == null || reply.sanction_id <= 0 || string.IsNullOrEmpty(reply.token)) return default;
            ServerClock.TryReadUnixMs(reply.expires_at, out var expiresUnixMs);
            return new SanctionAppeal(reply.sanction_id, reply.token, expiresUnixMs);
        }

        public static SanctionAppeal FromWire(AppealGrant grant)
        {
            if (grant == null || grant.SanctionId == 0 || grant.SanctionId > (ulong)long.MaxValue || string.IsNullOrEmpty(grant.Token)) return default;
            return new SanctionAppeal((long)grant.SanctionId, grant.Token, grant.ExpiresUnixMs);
        }
    }
}
