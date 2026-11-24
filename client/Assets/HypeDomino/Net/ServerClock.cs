using System;
using System.Globalization;

namespace HypeDomino.Net
{
    public sealed class ServerClock
    {
        double offsetMs;

        public bool Synced { get; private set; }

        public void Sync(ulong serverUnixMs, double sentSeconds, double receivedSeconds)
        {
            offsetMs = serverUnixMs - (sentSeconds + receivedSeconds) * 500;
            Synced = true;
        }

        public double UnixMs(double nowSeconds) => nowSeconds * 1000 + offsetMs;

        public double SecondsUntil(ulong serverUnixMs, double nowSeconds) => (serverUnixMs - UnixMs(nowSeconds)) / 1000;

        public bool TrySecondsLeft(ulong serverUnixMs, double nowSeconds, out double seconds)
        {
            seconds = 0;
            if (!Synced || serverUnixMs == 0) return false;
            seconds = Math.Max(0d, SecondsUntil(serverUnixMs, nowSeconds));
            return true;
        }

        public static bool TryReadUnixMs(string text, out ulong unixMs)
        {
            unixMs = 0;
            if (string.IsNullOrEmpty(text)) return false;
            if (!DateTimeOffset.TryParse(text, CultureInfo.InvariantCulture, DateTimeStyles.AssumeUniversal, out var at)) return false;
            var ms = at.ToUnixTimeMilliseconds();
            if (ms <= 0) return false;
            unixMs = (ulong)ms;
            return true;
        }
    }
}
