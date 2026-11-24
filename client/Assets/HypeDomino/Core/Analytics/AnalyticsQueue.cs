using System;
using System.Collections.Generic;

namespace HypeDomino.Core.Analytics
{
    public sealed class AnalyticsQueue
    {
        readonly IAnalyticsSender sender;
        readonly Func<long> clockMs;
        int batchSize;
        int queueLimit;
        readonly List<AnalyticsEvent> queued = new List<AnalyticsEvent>();
        readonly List<AnalyticsEvent> batch = new List<AnalyticsEvent>();
        string analyticsId;
        int generation;
        int inFlight;
        bool sending;

        public AnalyticsQueue(IAnalyticsSender sender, Func<long> clockMs, int batchSize, int queueLimit)
        {
            if (batchSize < 1) throw new ArgumentOutOfRangeException(nameof(batchSize));
            if (queueLimit < batchSize) throw new ArgumentOutOfRangeException(nameof(queueLimit));
            this.sender = sender;
            this.clockMs = clockMs;
            this.batchSize = batchSize;
            this.queueLimit = queueLimit;
        }

        public bool Consented => analyticsId != null;

        public int BatchSize => batchSize;

        public int QueueLimit => queueLimit;

        public int Count => queued.Count;

        public void Consent(string id)
        {
            if (string.IsNullOrEmpty(id)) throw new ArgumentException("analytics id required", nameof(id));
            analyticsId = id;
        }

        public void Withdraw()
        {
            analyticsId = null;
            queued.Clear();
            inFlight = 0;
            generation++;
        }

        public void Resize(int newBatchSize, int newQueueLimit)
        {
            if (newBatchSize < 1) throw new ArgumentOutOfRangeException(nameof(newBatchSize));
            if (newQueueLimit < newBatchSize) throw new ArgumentOutOfRangeException(nameof(newQueueLimit));
            batchSize = newBatchSize;
            queueLimit = newQueueLimit;
            while (queued.Count > queueLimit) DropOldest();
        }

        public bool Track(string name, string label = null, long value = 0)
        {
            if (!Consented || !AnalyticsEvents.IsClientEvent(name)) return false;
            if (queued.Count == queueLimit) DropOldest();
            queued.Add(new AnalyticsEvent(name, label, value, clockMs()));
            return true;
        }

        public void Flush()
        {
            if (!Consented || sending || queued.Count == 0) return;
            batch.Clear();
            inFlight = Math.Min(batchSize, queued.Count);
            for (var i = 0; i < inFlight; i++) batch.Add(queued[i]);
            sending = true;
            var sentGeneration = generation;
            sender.Send(analyticsId, batch, ok => Sent(ok, sentGeneration));
        }

        void DropOldest()
        {
            queued.RemoveAt(0);
            if (inFlight > 0) inFlight--;
        }

        void Sent(bool ok, int sentGeneration)
        {
            sending = false;
            if (ok && sentGeneration == generation) queued.RemoveRange(0, inFlight);
            inFlight = 0;
        }
    }
}
