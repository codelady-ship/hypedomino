using System;
using System.Collections.Generic;
using System.Text;

namespace HypeDomino.Core.Diagnostics
{
    public sealed class SentryErrorSender : IErrorSender
    {
        public const int MaxQueuedReports = 30;

        readonly SentryDsn dsn;
        readonly SentryEnvelope envelope;
        readonly IEnvelopeTransport transport;
        readonly Func<DateTime> utcNow;
        readonly Queue<ErrorReport> queue = new Queue<ErrorReport>(MaxQueuedReports);
        readonly object gate = new object();
        readonly Action<EnvelopeReply> replied;
        bool sending;
        DateTime pausedUntil;

        public SentryErrorSender(SentryDsn dsn, SentryEnvelope envelope, IEnvelopeTransport transport, Func<DateTime> utcNow)
        {
            this.dsn = dsn ?? throw new ArgumentNullException(nameof(dsn));
            this.envelope = envelope ?? throw new ArgumentNullException(nameof(envelope));
            this.transport = transport ?? throw new ArgumentNullException(nameof(transport));
            this.utcNow = utcNow ?? throw new ArgumentNullException(nameof(utcNow));
            replied = Replied;
        }

        public bool Sending => sending;

        public int Queued
        {
            get { lock (gate) return queue.Count; }
        }

        public void Send(ErrorReport report)
        {
            lock (gate)
            {
                if (queue.Count >= MaxQueuedReports) return;
                queue.Enqueue(report);
            }
        }

        public void Pump()
        {
            if (sending) return;
            ErrorReport next;
            DateTime now;
            lock (gate)
            {
                if (queue.Count == 0) return;
                now = utcNow();
                if (now < pausedUntil)
                {
                    queue.Clear();
                    return;
                }
                next = queue.Dequeue();
            }
            sending = true;
            var body = Encoding.UTF8.GetBytes(envelope.Build(next, Guid.NewGuid().ToString("N"), now));
            transport.Post(dsn.EnvelopeUrl, dsn.AuthHeader, body, replied);
        }

        void Replied(EnvelopeReply reply)
        {
            var pause = SentryRateLimits.Pause(reply);
            if (pause > TimeSpan.Zero)
                lock (gate) pausedUntil = utcNow() + pause;
            sending = false;
        }
    }
}
