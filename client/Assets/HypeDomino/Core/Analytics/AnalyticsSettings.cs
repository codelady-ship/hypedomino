using System;

namespace HypeDomino.Core.Analytics
{
    public readonly struct AnalyticsSettings
    {
        public const int MaxBatchSize = 400;

        public static readonly AnalyticsSettings Draft = new AnalyticsSettings(50, 500, 60);

        public readonly int BatchSize;
        public readonly int QueueLimit;
        public readonly double FlushSeconds;

        public AnalyticsSettings(int batchSize, int queueLimit, double flushSeconds)
        {
            if (!Valid(batchSize, queueLimit, flushSeconds)) throw new ArgumentOutOfRangeException(nameof(batchSize), "analytics settings outside their limits");
            BatchSize = batchSize;
            QueueLimit = queueLimit;
            FlushSeconds = flushSeconds;
        }

        public static bool Valid(int batchSize, int queueLimit, double flushSeconds) =>
            batchSize >= 1 && batchSize <= MaxBatchSize && queueLimit >= batchSize && flushSeconds > 0;
    }
}
