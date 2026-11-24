using System;
using System.Collections.Generic;

namespace HypeDomino.Core.DeviceRisk
{
    public static class DeviceRiskSignals
    {
        public const string Root = "root";
        public const string TestKeys = "test_keys";
        public const string Emulator = "emulator";
        public const string Jailbreak = "jailbreak";

        static readonly string[] RootPaths =
        {
            "/system/bin/su", "/system/xbin/su", "/sbin/su", "/su/bin/su", "/system/sd/xbin/su",
            "/system/app/Superuser.apk", "/system/bin/magisk", "/sbin/magisk",
        };

        static readonly string[] JailbreakPaths =
        {
            "/Applications/Cydia.app", "/Applications/Sileo.app", "/Library/MobileSubstrate/MobileSubstrate.dylib",
            "/usr/sbin/sshd", "/etc/apt", "/private/var/lib/apt/", "/var/jb", "/usr/lib/libjailbreak.dylib",
        };

        static readonly string[] EmulatorFingerprintMarkers = { "generic/", "/vbox86p/" };
        static readonly string[] EmulatorModelMarkers = { "google_sdk", "Emulator", "Android SDK built for", "sdk_gphone" };
        static readonly string[] EmulatorHardware = { "goldfish", "ranchu", "vbox86" };
        static readonly string[] EmulatorProductMarkers = { "sdk_gphone", "google_sdk", "vbox86p", "emulator", "simulator" };

        public static void CollectAndroid(DeviceFacts facts, Func<string, bool> pathExists, List<string> signals)
        {
            if (AnyExists(RootPaths, pathExists)) signals.Add(Root);
            if (facts.Tags.IndexOf("test-keys", StringComparison.Ordinal) >= 0) signals.Add(TestKeys);
            if (LooksEmulated(facts)) signals.Add(Emulator);
        }

        public static void CollectIos(Func<string, bool> pathExists, List<string> signals)
        {
            if (AnyExists(JailbreakPaths, pathExists)) signals.Add(Jailbreak);
        }

        static bool LooksEmulated(DeviceFacts facts) =>
            ContainsAny(facts.Fingerprint, EmulatorFingerprintMarkers)
            || ContainsAny(facts.Model, EmulatorModelMarkers)
            || ContainsAny(facts.Product, EmulatorProductMarkers)
            || Array.IndexOf(EmulatorHardware, facts.Hardware) >= 0
            || facts.Manufacturer.IndexOf("Genymotion", StringComparison.OrdinalIgnoreCase) >= 0;

        static bool AnyExists(string[] paths, Func<string, bool> pathExists)
        {
            for (var i = 0; i < paths.Length; i++)
                if (pathExists(paths[i])) return true;
            return false;
        }

        static bool ContainsAny(string value, string[] markers)
        {
            for (var i = 0; i < markers.Length; i++)
                if (value.IndexOf(markers[i], StringComparison.OrdinalIgnoreCase) >= 0) return true;
            return false;
        }
    }
}
