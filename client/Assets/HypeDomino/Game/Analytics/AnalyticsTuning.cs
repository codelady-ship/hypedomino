using HypeDomino.Core.Analytics;
using Hypedomino.V1;

namespace HypeDomino.Game.Analytics
{
    public sealed class AnalyticsTuning
    {
        readonly AnalyticsQueue queue;
        readonly AnalyticsFlushTimer timer;

        public AnalyticsTuning(AnalyticsQueue queue, AnalyticsFlushTimer timer)
        {
            this.queue = queue;
            this.timer = timer;
        }

        public void Read(ServerHello hello)
        {
            var config = hello.AnalyticsConfig;
            if (hello.Status != HelloStatus.Accepted || config == null) return;
            var batchSize = (int)config.BatchSize;
            var queueLimit = (int)config.QueueLimit;
            double flushSeconds = config.FlushSeconds;
            if (!AnalyticsSettings.Valid(batchSize, queueLimit, flushSeconds)) return;
            queue.Resize(batchSize, queueLimit);
            timer.Retime(flushSeconds);
        }
    }
}
