using System;
using System.Collections.Generic;

namespace HypeDomino.Core.DeviceRisk
{
    public static class HookSignals
    {
        public const string Instrumentation = "instrumentation";
        public const string HookFramework = "hook_framework";
        public const string Debugger = "debugger";

        static readonly string[] InstrumentationMarkers = { "frida", "gum-js-loop", "linjector" };
        static readonly string[] HookFrameworkMarkers = { "XposedBridge", "libxposed", "liblspd","libriru", "edxp", "substrate", "libsandhook", "libpine" };

        public static void Collect(string loadedLibraries, bool debuggerAttached, List<string> signals)
        {
            var libraries = loadedLibraries ?? "";
            if (ContainsAny(libraries, InstrumentationMarkers)) signals.Add(Instrumentation);
            if (ContainsAny(libraries, HookFrameworkMarkers)) signals.Add(HookFramework);
            if (debuggerAttached) signals.Add(Debugger);
        }

        static bool ContainsAny(string value, string[] markers)
        {
            for (var i = 0; i < markers.Length; i++)
                if (value.IndexOf(markers[i], StringComparison.OrdinalIgnoreCase) >= 0) return true;
            return false;
        }
    }
}
