using HypeDomino.Core.Diagnostics;
using HypeDomino.UI.Diagnostics;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class FrameStatsOverlayTests
    {
        GameObject host;
        FrameStatsOverlay overlay;

        [SetUp]
        public void CreateOverlay()
        {
            host = new GameObject("Diagnostics");
            overlay = host.AddComponent<FrameStatsOverlay>();
        }

        [TearDown]
        public void DestroyOverlay() => Object.Destroy(host);

        static FrameWindowReport Report(double workP95Ms, double missedVsyncPercent, int allocatingFrames, long maxBatches) =>
            new(60, workP95Ms / 2, workP95Ms, missedVsyncPercent, allocatingFrames, allocatingFrames * 64, maxBatches);

        [Test]
        public void ItShowsTheLiveWindow()
        {
            overlay.Show(Report(12.34, 1.5, 2, 47));

            Assert.AreEqual("12.3", overlay.ValueOf(FrameStatsOverlay.WorkP95Row).text);
            Assert.AreEqual("1.5", overlay.ValueOf(FrameStatsOverlay.MissedVsyncRow).text);
            Assert.AreEqual("2", overlay.ValueOf(FrameStatsOverlay.GcFramesRow).text);
            Assert.AreEqual("47", overlay.ValueOf(FrameStatsOverlay.BatchesRow).text);
        }

        [Test]
        public void AllocatingFramesShowInTheDangerColour()
        {
            overlay.Show(Report(8, 0, 1, 10));
            Assert.AreEqual((Color)TokenColor.RoleDangerText, overlay.ValueOf(FrameStatsOverlay.GcFramesRow).color);

            overlay.Show(Report(8, 0, 0, 10));
            Assert.AreEqual((Color)TokenColor.TextPrimary, overlay.ValueOf(FrameStatsOverlay.GcFramesRow).color);
        }

        [Test]
        public void ValuesPastTheTableShowItsLastEntry()
        {
            overlay.Show(Report(250, 100, 0, 5000));

            Assert.AreEqual("99.9", overlay.ValueOf(FrameStatsOverlay.WorkP95Row).text);
            Assert.AreEqual("999", overlay.ValueOf(FrameStatsOverlay.BatchesRow).text);
        }

        [Test]
        public void ShowingAllocatesNothing()
        {
            var steady = Report(6.2, 0, 0, 12);
            var busy = Report(13.8, 2.5, 3, 40);
            void ShowBoth()
            {
                overlay.Show(steady);
                Canvas.ForceUpdateCanvases();
                overlay.Show(busy);
                Canvas.ForceUpdateCanvases();
            }
            ShowBoth();

            Assert.That(ShowBoth, Allocates.Nothing);
        }
    }
}
