using System;
using System.Collections.Generic;
using HypeDomino.Core.Analytics;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AnalyticsFlushTimerTests
    {
        sealed class FakeSender : IAnalyticsSender
        {
            public int Sends;

            public void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done)
            {
                Sends++;
                done(false);
            }
        }

        FakeSender sender;
        AnalyticsQueue queue;
        AnalyticsFlushTimer timer;

        [SetUp]
        public void Create()
        {
            sender = new FakeSender();
            queue = new AnalyticsQueue(sender, Clock, 2, 4);
            queue.Consent("a1");
            queue.Track(AnalyticsEvents.AppOpen);
            timer = new AnalyticsFlushTimer(queue, 60);
        }

        static long Clock() => 1000;

        [Test]
        public void TheFirstTickStartsTheIntervalWithoutFlushing()
        {
            timer.Tick(100);
            Assert.AreEqual(0, sender.Sends);
        }

        [Test]
        public void ItFlushesOnceTheIntervalHasPassed()
        {
            timer.Tick(100);
            timer.Tick(159.9);
            Assert.AreEqual(0, sender.Sends);
            timer.Tick(160);
            Assert.AreEqual(1, sender.Sends);
        }

        [Test]
        public void ItFlushesOncePerInterval()
        {
            timer.Tick(0);
            for (var second = 1; second <= 180; second++) timer.Tick(second);
            Assert.AreEqual(3, sender.Sends);
        }

        [Test]
        public void ALongGapFlushesOnceAndRestartsTheInterval()
        {
            timer.Tick(0);
            timer.Tick(1000);
            timer.Tick(1001);
            Assert.AreEqual(1, sender.Sends);
            timer.Tick(1060);
            Assert.AreEqual(2, sender.Sends);
        }

        [Test]
        public void ATickBeforeTheIntervalAllocatesNothing()
        {
            timer.Tick(0);
            TickBeforeDue();
            Assert.That(TickBeforeDue, Allocates.Nothing);
        }

        void TickBeforeDue() => timer.Tick(1);

        [Test]
        public void PausingTheAppFlushesAtOnce()
        {
            timer.AppPaused();
            Assert.AreEqual(1, sender.Sends);
        }

        [Test]
        public void NothingIsSentWithoutConsent()
        {
            queue.Withdraw();
            timer.Tick(0);
            timer.Tick(60);
            timer.AppPaused();
            Assert.AreEqual(0, sender.Sends);
        }

        [TestCase(0.0)]
        [TestCase(-1.0)]
        [TestCase(double.NaN)]
        public void TheIntervalMustBePositive(double seconds)
        {
            Assert.Throws<ArgumentOutOfRangeException>(() => new AnalyticsFlushTimer(queue, seconds));
        }

        [Test]
        public void TheDraftSettingsAreValid()
        {
            var draft = AnalyticsSettings.Draft;
            Assert.That(draft.BatchSize, Is.InRange(1, AnalyticsSettings.MaxBatchSize));
            Assert.GreaterOrEqual(draft.QueueLimit, draft.BatchSize);
            Assert.Greater(draft.FlushSeconds, 0);
        }

        [TestCase(0, 10, 60.0)]
        [TestCase(AnalyticsSettings.MaxBatchSize + 1, 1000, 60.0)]
        [TestCase(10, 9, 60.0)]
        [TestCase(10, 10, 0.0)]
        public void SettingsOutsideTheLimitsAreRefused(int batchSize, int queueLimit, double flushSeconds)
        {
            Assert.Throws<ArgumentOutOfRangeException>(() => new AnalyticsSettings(batchSize, queueLimit, flushSeconds));
        }
    }
}
