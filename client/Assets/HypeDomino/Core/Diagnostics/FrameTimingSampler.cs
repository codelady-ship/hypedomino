using System;
using Unity.Profiling;
using UnityEngine;

namespace HypeDomino.Core.Diagnostics
{
    public sealed class FrameTimingSampler : MonoBehaviour
    {
        const int WindowFrames = 1200;
        const int LiveFrames = 60;
        const double MissedVsyncFactor = 1.5;

        readonly FrameTiming[] latestTiming = new FrameTiming[1];
        FrameWindow window;
        FrameWindow live;
        ProfilerRecorder gcAllocatedInFrame;
        ProfilerRecorder batchesCount;
        int collectionsAtWindowStart;
        bool skipReportFrame;

        public FrameWindowReport Live { get; private set; }

        public int LiveVersion { get; private set; }

        void Awake()
        {
            var frameRate = Application.targetFrameRate > 0 ? Application.targetFrameRate : HighestRefreshRate();
            window = new FrameWindow(WindowFrames, MissedVsyncFactor / frameRate);
            live = new FrameWindow(LiveFrames, MissedVsyncFactor / frameRate);
            collectionsAtWindowStart = GC.CollectionCount(0);
        }

        void OnEnable()
        {
            gcAllocatedInFrame = ProfilerRecorder.StartNew(ProfilerCategory.Memory, "GC Allocated In Frame");
            batchesCount = ProfilerRecorder.StartNew(ProfilerCategory.Render, "Batches Count");
        }

        void OnDisable()
        {
            gcAllocatedInFrame.Dispose();
            batchesCount.Dispose();
        }

        void Update()
        {
            FrameTimingManager.CaptureFrameTimings();
            if (FrameTimingManager.GetLatestTimings(1, latestTiming) == 0) return;

            var gcBytes = skipReportFrame ? 0 : LastValue(gcAllocatedInFrame);
            skipReportFrame = false;
            var workMs = FrameWindow.WorkMs(latestTiming[0]);
            var batches = LastValue(batchesCount);
            window.Add(workMs, Time.unscaledDeltaTime, gcBytes, batches);
            live.Add(workMs, Time.unscaledDeltaTime, gcBytes, batches);
            if (live.IsFull)
            {
                Live = live.Close();
                LiveVersion++;
            }
            if (!window.IsFull) return;

            Report(window.Close());
            skipReportFrame = true;
            collectionsAtWindowStart = GC.CollectionCount(0);
        }

        void Report(FrameWindowReport report)
        {
            var collections = GC.CollectionCount(0) - collectionsAtWindowStart;
            Debug.Log($"FrameTiming target={Application.targetFrameRate} frames={report.Frames} workP50={report.WorkP50Ms:F2}ms workP95={report.WorkP95Ms:F2}ms missedVsync={report.MissedVsyncPercent:F2}% " +
                      $"gcFrames={Counter(gcAllocatedInFrame, report.AllocatingFrames)} gcMax={Counter(gcAllocatedInFrame, report.MaxGcBytes)}B gcCollections={collections} batchesMax={Counter(batchesCount, report.MaxBatches)}");
        }

        static long LastValue(ProfilerRecorder recorder) => recorder.Valid ? recorder.LastValue : 0;

        static string Counter(ProfilerRecorder recorder, long value) => recorder.Valid ? value.ToString() : "n/a";

        static int HighestRefreshRate()
        {
            var highest = Screen.currentResolution.refreshRateRatio.value;
            foreach (var resolution in Screen.resolutions)
                highest = Math.Max(highest, resolution.refreshRateRatio.value);
            return (int)Math.Round(highest);
        }
    }
}
