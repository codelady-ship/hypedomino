using System;
using System.Collections.Generic;
using System.Text;
using HypeDomino.Core.Diagnostics;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SentryErrorSenderTests
    {
        const string Dsn = "https://publickey@o42.ingest.de.sentry.io/4507";

        sealed class FakeTransport : IEnvelopeTransport
        {
            public readonly List<string> Urls = new List<string>();
            public readonly List<string> Auths = new List<string>();
            public readonly List<string> Bodies = new List<string>();
            public readonly List<Action<EnvelopeReply>> Pending = new List<Action<EnvelopeReply>>();

            public void Post(string url, string authHeader, byte[] envelope, Action<EnvelopeReply> done)
            {
                Urls.Add(url);
                Auths.Add(authHeader);
                Bodies.Add(Encoding.UTF8.GetString(envelope));
                Pending.Add(done);
            }

            public void Reply(int status, string retryAfter = null, string rateLimits = null)
            {
                var done = Pending[0];
                Pending.RemoveAt(0);
                done(new EnvelopeReply(status, retryAfter, rateLimits));
            }
        }

        FakeTransport transport;
        DateTime now;
        SentryErrorSender sender;

        [SetUp]
        public void Create()
        {
            transport = new FakeTransport();
            now = new DateTime(2026, 11, 21, 9, 0, 0, DateTimeKind.Utc);
            sender = new SentryErrorSender(SentryDsn.Parse(Dsn), new SentryEnvelope("1.4.2", "staging", "os", "model"), transport, Now);
        }

        DateTime Now() => now;

        static ErrorReport Report(string message) => new ErrorReport(LogType.Exception, message, "");

        [Test]
        public void NothingIsPostedUntilThePumpRuns()
        {
            sender.Send(Report("boom"));

            Assert.IsEmpty(transport.Bodies);
            Assert.AreEqual(1, sender.Queued);
        }

        [Test]
        public void AReportIsPostedAsAnEnvelopeToTheDsnProject()
        {
            sender.Send(Report("InvalidOperationException: boom"));
            sender.Pump();

            CollectionAssert.AreEqual(new[] { "https://o42.ingest.de.sentry.io/api/4507/envelope/" }, transport.Urls);
            CollectionAssert.AreEqual(new[] { "Sentry sentry_version=7, sentry_key=publickey" }, transport.Auths);
            StringAssert.Contains("\"value\":\"boom\"", transport.Bodies[0]);
            StringAssert.Contains("\"timestamp\":\"2026-11-21T09:00:00.000Z\"", transport.Bodies[0]);
        }

        [Test]
        public void EachReportGetsItsOwnEventId()
        {
            sender.Send(Report("a"));
            sender.Send(Report("b"));
            sender.Pump();
            transport.Reply(200);
            sender.Pump();

            Assert.AreEqual(2, transport.Bodies.Count);
            Assert.AreNotEqual(transport.Bodies[0].Split('\n')[0], transport.Bodies[1].Split('\n')[0]);
        }

        [Test]
        public void OneEnvelopeIsInFlightAtATime()
        {
            sender.Send(Report("a"));
            sender.Send(Report("b"));
            sender.Pump();
            sender.Pump();

            Assert.AreEqual(1, transport.Bodies.Count);
            Assert.IsTrue(sender.Sending);

            transport.Reply(200);
            sender.Pump();

            Assert.AreEqual(2, transport.Bodies.Count);
        }

        [Test]
        public void TheQueueDropsReportsBeyondItsLimit()
        {
            for (var i = 0; i < SentryErrorSender.MaxQueuedReports + 5; i++) sender.Send(Report("r" + i));

            Assert.AreEqual(SentryErrorSender.MaxQueuedReports, sender.Queued);
        }

        [Test]
        public void ANetworkFailureDropsTheReportAndTheNextOneIsSent()
        {
            sender.Send(Report("a"));
            sender.Send(Report("b"));
            sender.Pump();
            transport.Reply(EnvelopeReply.NoResponse);
            sender.Pump();

            Assert.AreEqual(2, transport.Bodies.Count);
            Assert.AreEqual(0, sender.Queued);
        }

        [Test]
        public void TooManyRequestsPausesSendingForRetryAfterAndDropsWhatArrivesMeanwhile()
        {
            sender.Send(Report("a"));
            sender.Pump();
            transport.Reply(429, "30");
            sender.Send(Report("b"));
            now = now.AddSeconds(29);
            sender.Pump();

            Assert.AreEqual(1, transport.Bodies.Count);
            Assert.AreEqual(0, sender.Queued);

            now = now.AddSeconds(1);
            sender.Send(Report("c"));
            sender.Pump();

            Assert.AreEqual(2, transport.Bodies.Count);
            StringAssert.Contains("\"value\":\"c\"", transport.Bodies[1]);
        }

        [Test]
        public void TooManyRequestsWithoutRetryAfterPausesForTheDefault()
        {
            sender.Send(Report("a"));
            sender.Pump();
            transport.Reply(429);
            now = now + SentryRateLimits.DefaultRetryAfter - TimeSpan.FromSeconds(1);
            sender.Send(Report("b"));
            sender.Pump();

            Assert.AreEqual(1, transport.Bodies.Count);
        }

        [TestCase(200, null, "60:error:organization", 60)]
        [TestCase(200, null, "60::organization", 60)]
        [TestCase(429, "5", "120:error;default:key, 30::organization", 120)]
        [TestCase(200, null, "60:transaction;session:key", 0)]
        [TestCase(429, "12", null, 12)]
        [TestCase(429, "soon", null, 60)]
        [TestCase(200, "30", null, 0)]
        [TestCase(500, null, null, 0)]
        public void ThePauseFollowsTheRateLimitHeaders(int status, string retryAfter, string rateLimits, int seconds)
        {
            Assert.AreEqual(TimeSpan.FromSeconds(seconds), SentryRateLimits.Pause(new EnvelopeReply(status, retryAfter, rateLimits)));
        }

        [Test]
        public void ReportsFromTheConsentGateReachTheSender()
        {
            var reports = new ErrorReports(sender);
            reports.OnLog("boom", "", LogType.Exception);
            reports.Consent();
            reports.OnLog("boom", "", LogType.Exception);

            Assert.AreEqual(1, sender.Queued);
        }
    }
}
