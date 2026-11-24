using System;
using NUnit.Framework;
using NUnit.Framework.Constraints;
using Unity.Profiling;

namespace HypeDomino.Tests
{
    public static class Allocates
    {
        public static AllocatesNothingConstraint Nothing => new();
    }

    public sealed class AllocatesNothingConstraint : Constraint
    {
        const string AllocationMarker = "GC.Alloc";
        const int RecordedAllocationsCap = 1024;

        public override string Description => "no GC allocation on the calling thread";

        public override ConstraintResult ApplyTo(object actual)
        {
            if (actual is not TestDelegate step) throw new ArgumentException($"expected a TestDelegate but was {actual?.GetType().Name ?? "null"}", nameof(actual));
            var allocations = CountOnCallingThread(step);
            return new ConstraintResult(this, $"{allocations} GC allocation(s)", allocations == 0);
        }

        public static int CountOnCallingThread(TestDelegate step)
        {
            var recorder = new ProfilerRecorder(AllocationMarker, RecordedAllocationsCap, ProfilerRecorderOptions.CollectOnlyOnCurrentThread);
            try
            {
                if (!recorder.Valid) throw new InvalidOperationException($"the {AllocationMarker} profiler marker is not available");
                recorder.Start();
                step();
                recorder.Stop();
                return recorder.Count;
            }
            finally
            {
                recorder.Dispose();
            }
        }
    }
}
