using System;
using System.Globalization;

namespace HypeDomino.Core.Diagnostics
{
    public static class SentryRateLimits
    {
        public const int TooManyRequests = 429;
        public const string ErrorCategory = "error";
        public static readonly TimeSpan DefaultRetryAfter = TimeSpan.FromSeconds(60);

        public static TimeSpan Pause(EnvelopeReply reply)
        {
            if (!string.IsNullOrEmpty(reply.RateLimits)) return PauseFromLimits(reply.RateLimits);
            if (reply.Status != TooManyRequests) return TimeSpan.Zero;
            return TrySeconds(reply.RetryAfter, out var seconds) ? TimeSpan.FromSeconds(seconds) : DefaultRetryAfter;
        }

        static TimeSpan PauseFromLimits(string header)
        {
            var longest = TimeSpan.Zero;
            foreach (var limit in header.Split(','))
            {
                var fields = limit.Trim().Split(':');
                if (!TrySeconds(fields[0], out var seconds)) continue;
                if (fields.Length > 1 && !CoversErrors(fields[1])) continue;
                var pause = TimeSpan.FromSeconds(seconds);
                if (pause > longest) longest = pause;
            }
            return longest;
        }

        static bool CoversErrors(string categories)
        {
            if (categories.Length == 0) return true;
            foreach (var category in categories.Split(';'))
                if (category == ErrorCategory) return true;
            return false;
        }

        static bool TrySeconds(string text, out double seconds) =>
            double.TryParse(text, NumberStyles.AllowDecimalPoint, CultureInfo.InvariantCulture, out seconds) && seconds > 0;
    }
}
