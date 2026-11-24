using System;
using System.Collections.Generic;
using HypeDomino.Core.Analytics;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AnalyticsQueueTests
    {
        sealed class FakeSender : IAnalyticsSender
        {
            public readonly List<string> Ids = new List<string>();
            public readonly List<List<AnalyticsEvent>> Batches = new List<List<AnalyticsEvent>>();
            public Action<bool> Pending;

            public void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done)
            {
                Ids.Add(analyticsId);
                Batches.Add(new List<AnalyticsEvent>(batch));
                Pending = done;
            }
        }

        FakeSender sender;
        AnalyticsQueue queue;
        long now;

        [SetUp]
        public void Create()
        {
            sender = new FakeSender();
            now = 1000;
            queue = new AnalyticsQueue(sender, () => now, 2, 3);
        }

        [Test]
        public void EventsBeforeConsentAreDroppedNotHeld()
        {
            Assert.IsFalse(queue.Track(AnalyticsEvents.AppOpen));
            queue.Consent("a1");
            queue.Flush();
            Assert.AreEqual(0, queue.Count);
            Assert.IsEmpty(sender.Batches);
        }

        [Test]
        public void UnknownEventNamesAreIgnored()
        {
            queue.Consent("a1");
            Assert.IsFalse(queue.Track("email_entered", "someone@example.com"));
            Assert.AreEqual(0, queue.Count);
        }

        [Test]
        public void FlushSendsOneBatchWithTheAnalyticsId()
        {
            queue.Consent("a1");
            queue.Track(AnalyticsEvents.AppOpen);
            now = 2000;
            queue.Track(AnalyticsEvents.ScreenView, "shop");
            queue.Track(AnalyticsEvents.QueueJoin, "ranked");
            queue.Flush();
            Assert.AreEqual(new[] { "a1" }, sender.Ids);
            var sent = sender.Batches[0];
            Assert.AreEqual(2, sent.Count);
            Assert.AreEqual(AnalyticsEvents.AppOpen, sent[0].Name);
            Assert.AreEqual(1000, sent[0].ClientTimeMs);
            Assert.AreEqual("shop", sent[1].Label);
            Assert.AreEqual(2000, sent[1].ClientTimeMs);
        }

        [Test]
        public void SuccessRemovesTheBatchAndTheNextFlushSendsTheRest()
        {
            queue.Consent("a1");
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Track(AnalyticsEvents.SignIn, "google");
            queue.Track(AnalyticsEvents.NicknameSet);
            queue.Flush();
            sender.Pending(true);
            Assert.AreEqual(1, queue.Count);
            queue.Flush();
            Assert.AreEqual(AnalyticsEvents.NicknameSet, sender.Batches[1][0].Name);
        }

        [Test]
        public void FailureKeepsEventsForTheNextFlush()
        {
            queue.Consent("a1");
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Flush();
            sender.Pending(false);
            Assert.AreEqual(1, queue.Count);
            queue.Flush();
            Assert.AreEqual(2, sender.Batches.Count);
            Assert.AreEqual(AnalyticsEvents.AppOpen, sender.Batches[1][0].Name);
        }

        [Test]
        public void OnlyOneSendIsInFlight()
        {
            queue.Consent("a1");
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Flush();
            queue.Flush();
            Assert.AreEqual(1, sender.Batches.Count);
        }

        [Test]
        public void FullQueueDropsTheOldestEvent()
        {
            queue.Consent("a1");
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Track(AnalyticsEvents.SignIn);
            queue.Track(AnalyticsEvents.NicknameSet);
            queue.Track(AnalyticsEvents.ScreenView);
            Assert.AreEqual(3, queue.Count);
            queue.Flush();
            Assert.AreEqual(AnalyticsEvents.SignIn, sender.Batches[0][0].Name);
        }

        [Test]
        public void DroppingAnEventInFlightDoesNotRemoveANewerOneOnSuccess()
        {
            queue.Consent("a1");
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Track(AnalyticsEvents.SignIn);
            queue.Track(AnalyticsEvents.NicknameSet);
            queue.Flush();
            queue.Track(AnalyticsEvents.ScreenView);
            sender.Pending(true);
            Assert.AreEqual(2, queue.Count);
            queue.Flush();
            Assert.AreEqual(AnalyticsEvents.NicknameSet, sender.Batches[1][0].Name);
            Assert.AreEqual(AnalyticsEvents.ScreenView, sender.Batches[1][1].Name);
        }

        [Test]
        public void WithdrawDropsQueuedEventsAndStopsSending()
        {
            queue.Consent("a1");
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Withdraw();
            Assert.IsFalse(queue.Consented);
            Assert.AreEqual(0, queue.Count);
            Assert.IsFalse(queue.Track(AnalyticsEvents.AppOpen));
            queue.Flush();
            Assert.IsEmpty(sender.Batches);
        }

        [Test]
        public void ASendCompletingAfterWithdrawAndNewConsentKeepsNewEvents()
        {
            queue.Consent("a1");
            queue.Track(AnalyticsEvents.AppOpen);
            queue.Flush();
            queue.Withdraw();
            queue.Consent("a2");
            queue.Track(AnalyticsEvents.SignIn);
            sender.Pending(true);
            Assert.AreEqual(1, queue.Count);
            queue.Flush();
            Assert.AreEqual("a2", sender.Ids[1]);
            Assert.AreEqual(AnalyticsEvents.SignIn, sender.Batches[1][0].Name);
        }

        [Test]
        public void ConsentNeedsAnAnalyticsId()
        {
            Assert.Throws<ArgumentException>(() => queue.Consent(""));
        }
    }
}
