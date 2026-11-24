using HypeDomino.Game.Match;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LoadingReportTests
    {
        LoadingReport report;

        [SetUp]
        public void CreateReport() => report = new LoadingReport();

        [Test]
        public void TheFirstReportIsSentAtOnce()
        {
            Assert.IsTrue(report.TryReport(0f, false, out var intent));
            Assert.AreEqual(0, intent.Progress);
            Assert.IsFalse(intent.Ready);
        }

        [Test]
        public void ProgressIsSentOnlyWhenItReachesTheNextStep()
        {
            report.TryReport(0.12f, false, out _);

            Assert.IsFalse(report.TryReport(0.19f, false, out _));
            Assert.IsTrue(report.TryReport(0.21f, false, out var intent));
            Assert.AreEqual(20, intent.Progress);
        }

        [Test]
        public void EverythingLoadedIsNotReadyUntilTheSceneSaysSo()
        {
            Assert.IsTrue(report.TryReport(1f, false, out var loaded));
            Assert.AreEqual(90, loaded.Progress);
            Assert.IsFalse(loaded.Ready);

            Assert.IsTrue(report.TryReport(1f, true, out var ready));
            Assert.AreEqual(100, ready.Progress);
            Assert.IsTrue(ready.Ready);
        }

        [Test]
        public void ReadyIsSentOnceAndEndsTheReports()
        {
            Assert.IsTrue(report.TryReport(0.5f, true, out _));

            Assert.IsFalse(report.TryReport(0.5f, true, out _));
            Assert.IsFalse(report.TryReport(0.9f, false, out _));
        }

        [Test]
        public void ResetStartsANewMatch()
        {
            report.TryReport(1f, true, out _);

            report.Reset();

            Assert.IsTrue(report.TryReport(0f, false, out var intent));
            Assert.AreEqual(0, intent.Progress);
        }

        [Test]
        public void ReportingAllocatesNothing()
        {
            var loaded = 0f;
            TestDelegate step = () =>
            {
                report.TryReport(loaded, false, out _);
                loaded += 0.1f;
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
