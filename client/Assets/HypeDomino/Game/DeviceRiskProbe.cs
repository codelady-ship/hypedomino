using System.Collections.Generic;
using System.IO;
using HypeDomino.Core.DeviceRisk;
using UnityEngine;

namespace HypeDomino.Game
{
    public static class DeviceRiskProbe
    {
        const string ProcessMaps = "/proc/self/maps";

        public static void Collect(List<string> signals)
        {
            if (Application.platform == RuntimePlatform.Android) DeviceRiskSignals.CollectAndroid(ReadAndroidFacts(), PathExists, signals);
            else if (Application.platform == RuntimePlatform.IPhonePlayer) DeviceRiskSignals.CollectIos(PathExists, signals);
        }

        public static void CollectHooks(List<string> signals)
        {
            if (Application.platform != RuntimePlatform.Android) return;
            HookSignals.Collect(ReadLoadedLibraries(), System.Diagnostics.Debugger.IsAttached || AndroidDebuggerConnected(), signals);
        }

        static string ReadLoadedLibraries()
        {
            try
            {
                return File.ReadAllText(ProcessMaps);
            }
            catch (IOException)
            {
                return null;
            }
            catch (System.UnauthorizedAccessException)
            {
                return null;
            }
        }

        static bool AndroidDebuggerConnected()
        {
            using (var debug = new AndroidJavaClass("android.os.Debug"))
                return debug.CallStatic<bool>("isDebuggerConnected");
        }

        static bool PathExists(string path) => File.Exists(path) || Directory.Exists(path);

        static DeviceFacts ReadAndroidFacts()
        {
            using (var build = new AndroidJavaClass("android.os.Build"))
            {
                return new DeviceFacts(
                    build.GetStatic<string>("FINGERPRINT"),
                    build.GetStatic<string>("MODEL"),
                    build.GetStatic<string>("MANUFACTURER"),
                    build.GetStatic<string>("HARDWARE"),
                    build.GetStatic<string>("PRODUCT"),
                    build.GetStatic<string>("TAGS"));
            }
        }
    }
}
