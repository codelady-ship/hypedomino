using System;
using UnityEngine;

namespace HypeDomino.Core.Diagnostics
{
    public readonly struct FrameWindowReport
    {
        public readonly int Frames;
        public readonly double WorkP50Ms;
        public readonly double WorkP95Ms;
        public readonly double MissedVsyncPercent;
        public readonly int AllocatingFrames;
        public readonly long MaxGcBytes;
        public readonly long MaxBatches;

        public FrameWindowReport(int frames, double workP50Ms, double workP95Ms, double missedVsyncPercent, int allocatingFrames, long maxGcBytes, long maxBatches)
        {
            Frames = frames;
            WorkP50Ms = workP50Ms;
            WorkP95Ms = workP95Ms;
            MissedVsyncPercent = missedVsyncPercent;
            AllocatingFrames = allocatingFrames;
            MaxGcBytes = maxGcBytes;
            MaxBatches = maxBatches;
        }
    }

    public sealed class FrameWindow
    {
        readonly double[] workTimesMs;
        readonly double missedVsyncThresholdSeconds;
        int sampledFrames;
        int missedVsyncs;
        int allocatingFrames;
        long maxGcBytes;
        long maxBatches;

        public FrameWindow(int frames, double missedVsyncThresholdSeconds)
        {
            workTimesMs = new double[frames];
            this.missedVsyncThresholdSeconds = missedVsyncThresholdSeconds;
        }

        public bool IsFull => sampledFrames == workTimesMs.Length;

        public static double WorkMs(in FrameTiming timing) =>
            Math.Max(Math.Max(timing.cpuMainThreadFrameTime, timing.cpuRenderThreadFrameTime), timing.gpuFrameTime);

        public void Add(double workMs, double intervalSeconds, long gcBytes, long batches)
        {
            workTimesMs[sampledFrames++] = workMs;
            if (intervalSeconds > missedVsyncThresholdSeconds) missedVsyncs++;
            if (gcBytes > 0) allocatingFrames++;
            maxGcBytes = Math.Max(maxGcBytes, gcBytes);
            maxBatches = Math.Max(maxBatches, batches);
        }

        public FrameWindowReport Close()
        {
            var frames = workTimesMs.Length;
            Array.Sort(workTimesMs);
            var report = new FrameWindowReport(
                frames,
                workTimesMs[frames / 2],
                workTimesMs[frames * 95 / 100],
                100.0 * missedVsyncs / frames,
                allocatingFrames,
                maxGcBytes,
                maxBatches);
            sampledFrames = 0;
            missedVsyncs = 0;
            allocatingFrames = 0;
            maxGcBytes = 0;
            maxBatches = 0;
            return report;
        }
    }
}
