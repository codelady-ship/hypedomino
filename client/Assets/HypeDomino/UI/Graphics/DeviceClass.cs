using System.Text.RegularExpressions;
using HypeDomino.UI.Tokens;

namespace HypeDomino.UI.Graphics
{
    public readonly struct DeviceFacts
    {
        public DeviceFacts(int memoryMb, string gpuName, int maxRefreshHz)
        {
            MemoryMb = memoryMb;
            GpuName = gpuName;
            MaxRefreshHz = maxRefreshHz;
        }

        public int MemoryMb { get; }
        public string GpuName { get; }
        public int MaxRefreshHz { get; }
    }

    public static class DeviceClass
    {
        static readonly Regex[] LowGpus = Compile(TokenGraphics.LowGpuPatterns);
        static readonly Regex[] HighGpus = Compile(TokenGraphics.HighGpuPatterns);

        public static DeviceTier TierOf(DeviceFacts device)
        {
            if (device.MemoryMb < TokenGraphics.LowBelowMemoryMb || AnyMatches(LowGpus, device.GpuName)) return DeviceTier.Low;
            if (device.MemoryMb < TokenGraphics.HighFromMemoryMb || device.MaxRefreshHz < TokenGraphics.HighFromRefreshHz) return DeviceTier.Mid;
            return AnyMatches(HighGpus, device.GpuName) ? DeviceTier.High : DeviceTier.Mid;
        }

        static Regex[] Compile(string[] patterns)
        {
            var compiled = new Regex[patterns.Length];
            for (var at = 0; at < patterns.Length; at++) compiled[at] = new Regex(patterns[at], RegexOptions.CultureInvariant);
            return compiled;
        }

        static bool AnyMatches(Regex[] patterns, string gpuName)
        {
            if (string.IsNullOrEmpty(gpuName)) return false;
            foreach (var pattern in patterns)
                if (pattern.IsMatch(gpuName)) return true;
            return false;
        }
    }
}
