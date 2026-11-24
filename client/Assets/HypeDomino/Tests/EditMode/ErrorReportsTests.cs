using System.Collections.Generic;
using HypeDomino.Core.Diagnostics;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ErrorReportsTests
    {
        sealed class FakeSender : IErrorSender
        {
            public readonly List<ErrorReport> Sent = new List<ErrorReport>();

            public void Send(ErrorReport report) => Sent.Add(report);
        }

        FakeSender sender;
        ErrorReports reports;

        [SetUp]
        public void Create()
        {
            sender = new FakeSender();
            reports = new ErrorReports(sender);
        }

        [Test]
        public void NothingLeavesTheDeviceWithoutDiagnosticsConsent()
        {
            reports.OnLog("boom", "at Match.Play", LogType.Exception);
            Assert.IsFalse(reports.Consented);
            Assert.IsEmpty(sender.Sent);
        }

        [TestCase(LogType.Exception)]
        [TestCase(LogType.Error)]
        [TestCase(LogType.Assert)]
        public void ErrorsAreSentAfterConsent(LogType type)
        {
            reports.Consent();
            reports.OnLog("boom", "at Match.Play", type);
            Assert.AreEqual(1, sender.Sent.Count);
            Assert.AreEqual(type, sender.Sent[0].Type);
            Assert.AreEqual("boom", sender.Sent[0].Message);
            Assert.AreEqual("at Match.Play", sender.Sent[0].StackTrace);
        }

        [TestCase(LogType.Log)]
        [TestCase(LogType.Warning)]
        public void PlainLogsAreNotSent(LogType type)
        {
            reports.Consent();
            reports.OnLog("FrameTiming target=60", "", type);
            Assert.IsEmpty(sender.Sent);
        }

        [Test]
        public void WithdrawingStopsSendingAtOnce()
        {
            reports.Consent();
            reports.Withdraw();
            reports.OnLog("boom", "at Match.Play", LogType.Exception);
            Assert.IsEmpty(sender.Sent);
        }
    }
}
