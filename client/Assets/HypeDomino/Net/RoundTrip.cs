namespace HypeDomino.Net
{
    public sealed class RoundTrip
    {
        public const int MaxPending = 8;
        public const double SmoothingGain = 1.0 / 8;

        readonly ulong[] pendingSequences = new ulong[MaxPending];
        readonly double[] pendingSentSeconds = new double[MaxPending];
        int pendingCount;

        public bool HasSample { get; private set; }
        public double LastMs { get; private set; }
        public double SmoothedMs { get; private set; }

        public bool HasPending => pendingCount > 0;
        public double OldestPendingSentSeconds => pendingSentSeconds[0];

        public void Sent(ulong sequence, double nowSeconds)
        {
            if (pendingCount == MaxPending) RemoveAt(0);
            pendingSequences[pendingCount] = sequence;
            pendingSentSeconds[pendingCount] = nowSeconds;
            pendingCount++;
        }

        public void Acked(ulong ackedSequence, double nowSeconds)
        {
            while (pendingCount > 0 && pendingSequences[0] <= ackedSequence)
            {
                Sample((nowSeconds - pendingSentSeconds[0]) * 1000);
                RemoveAt(0);
            }
        }

        public void Rejected(ulong sequence)
        {
            for (var i = 0; i < pendingCount; i++)
            {
                if (pendingSequences[i] != sequence) continue;
                RemoveAt(i);
                return;
            }
        }

        public void ForgetPending() => pendingCount = 0;

        void Sample(double roundTripMs)
        {
            LastMs = roundTripMs;
            SmoothedMs = HasSample ? SmoothedMs + SmoothingGain * (roundTripMs - SmoothedMs) : roundTripMs;
            HasSample = true;
        }

        void RemoveAt(int index)
        {
            pendingCount--;
            for (var i = index; i < pendingCount; i++)
            {
                pendingSequences[i] = pendingSequences[i + 1];
                pendingSentSeconds[i] = pendingSentSeconds[i + 1];
            }
        }
    }
}
