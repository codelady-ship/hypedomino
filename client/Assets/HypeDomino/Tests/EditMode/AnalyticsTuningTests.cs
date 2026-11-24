using System;
using System.Collections.Generic;
using HypeDomino.Core.Analytics;
using HypeDomino.Game.Analytics;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AnalyticsTuningTests
    {
        sealed class FakeSender : IAnalyticsSender
        {
            public readonly List<List<string>> Batches = new List<List<string>>();

            public void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done)
            {
                var labels = new List<string>();
                for (var i = 0; i < batch.Count; i++) labels.Add(batch[i].Label);
                Batches.Add(labels);
                done(true);
            }
        }

        FakeSender sender;
        AnalyticsQueue queue;
        AnalyticsFlushTimer timer;
        AnalyticsTuning tuning;

        [SetUp]
        public void Create()
        {
            var draft = AnalyticsSettings.Draft;
            sender = new FakeSender();
            queue = new AnalyticsQueue(sender, () => 1000, draft.BatchSize, draft.QueueLimit);
            queue.Consent("a1");
            timer = new AnalyticsFlushTimer(queue, draft.FlushSeconds);
            tuning = new AnalyticsTuning(queue, timer);
        }

        static ServerHello Hello(HelloStatus status, uint batchSize, uint queueLimit, uint flushSeconds) => new ServerHello
        {
            Status = status,
            AnalyticsConfig = new AnalyticsConfig { BatchSize = batchSize, QueueLimit = queueLimit, FlushSeconds = flushSeconds },
        };

        void AssertDraft()
        {
            var draft = AnalyticsSettings.Draft;
            Assert.AreEqual(draft.BatchSize, queue.BatchSize);
            Assert.AreEqual(draft.QueueLimit, queue.QueueLimit);
            Assert.AreEqual(draft.FlushSeconds, timer.IntervalSeconds);
        }

        void TrackScreens(int count)
        {
            for (var i = 1; i <= count; i++) queue.Track(AnalyticsEvents.ScreenView, "s" + i);
        }

        [Test]
        public void TheDraftHoldsUntilTheFirstHello()
        {
            AssertDraft();
        }

        [Test]
        public void AnAcceptedHelloAppliesTheAnnouncedValues()
        {
            tuning.Read(Hello(HelloStatus.Accepted, 100, 1000, 30));
            Assert.AreEqual(100, queue.BatchSize);
            Assert.AreEqual(1000, queue.QueueLimit);
            Assert.AreEqual(30, timer.IntervalSeconds);
        }

        [Test]
        public void AHelloWithoutAnalyticsConfigKeepsTheCurrentSettings()
        {
            tuning.Read(new ServerHello { Status = HelloStatus.Accepted });
            AssertDraft();
        }

        [TestCase(HelloStatus.UpdateRequired)]
        [TestCase(HelloStatus.Maintenance)]
        public void OnlyAnAcceptedHelloTunes(HelloStatus status)
        {
            tuning.Read(Hello(status, 100, 1000, 30));
            AssertDraft();
        }

        [TestCase(0u, 10u, 60u)]
        [TestCase((uint)AnalyticsSettings.MaxBatchSize + 1, 1000u, 60u)]
        [TestCase(10u, 9u, 60u)]
        [TestCase(10u, 10u, 0u)]
        [TestCase(uint.MaxValue, uint.MaxValue, 60u)]
        [TestCase(10u, uint.MaxValue, 60u)]
        public void ValuesOutsideTheClientLimitsAreIgnored(uint batchSize, uint queueLimit, uint flushSeconds)
        {
            tuning.Read(Hello(HelloStatus.Accepted, batchSize, queueLimit, flushSeconds));
            AssertDraft();
        }

        [Test]
        public void ALowerQueueLimitDropsTheOldestEvents()
        {
            TrackScreens(5);
            tuning.Read(Hello(HelloStatus.Accepted, 2, 3, 60));
            Assert.AreEqual(3, queue.Count);
            queue.Flush();
            CollectionAssert.AreEqual(new[] { "s3", "s4" }, sender.Batches[0]);
        }

        [Test]
        public void TheNewBatchSizeIsUsedByTheNextFlush()
        {
            TrackScreens(5);
            tuning.Read(Hello(HelloStatus.Accepted, 4, 10, 60));
            queue.Flush();
            CollectionAssert.AreEqual(new[] { "s1", "s2", "s3", "s4" }, sender.Batches[0]);
        }

        [Test]
        public void ANewIntervalStartsAtTheNextTick()
        {
            TrackScreens(1);
            timer.Tick(0);
            tuning.Read(Hello(HelloStatus.Accepted, 50, 500, 30));
            timer.Tick(10);
            timer.Tick(39.9);
            Assert.IsEmpty(sender.Batches);
            timer.Tick(40);
            Assert.AreEqual(1, sender.Batches.Count);
        }

        [Test]
        public void TheSameIntervalKeepsTheRunningOne()
        {
            TrackScreens(1);
            timer.Tick(0);
            tuning.Read(Hello(HelloStatus.Accepted, 50, 500, 60));
            timer.Tick(60);
            Assert.AreEqual(1, sender.Batches.Count);
        }

        [TestCase(0, 10)]
        [TestCase(10, 9)]
        public void TheQueueRefusesAnInvalidSize(int batchSize, int queueLimit)
        {
            Assert.Throws<ArgumentOutOfRangeException>(() => queue.Resize(batchSize, queueLimit));
        }

        [TestCase(0.0)]
        [TestCase(-1.0)]
        [TestCase(double.NaN)]
        public void TheTimerRefusesANonPositiveInterval(double seconds)
        {
            Assert.Throws<ArgumentOutOfRangeException>(() => timer.Retime(seconds));
        }
    }
}
