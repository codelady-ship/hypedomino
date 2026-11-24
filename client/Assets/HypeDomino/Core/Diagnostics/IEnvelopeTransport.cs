using System;

namespace HypeDomino.Core.Diagnostics
{
    public interface IEnvelopeTransport
    {
        void Post(string url, string authHeader, byte[] envelope, Action<EnvelopeReply> done);
    }

    public readonly struct EnvelopeReply
    {
        public const int NoResponse = 0;

        public readonly int Status;
        public readonly string RetryAfter;
        public readonly string RateLimits;

        public EnvelopeReply(int status, string retryAfter, string rateLimits)
        {
            Status = status;
            RetryAfter = retryAfter;
            RateLimits = rateLimits;
        }
    }
}
