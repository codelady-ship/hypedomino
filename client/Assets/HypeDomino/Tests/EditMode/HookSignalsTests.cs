using System.Collections.Generic;
using HypeDomino.Core.DeviceRisk;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class HookSignalsTests
    {
        const string CleanMaps =
            "7a1b200000-7a1b300000 r-xp 00000000 fd:05 1234 /apex/com.android.runtime/lib64/bionic/libc.so\n" +
            "7a2c000000-7a2c100000 r-xp 00000000 fd:05 5678 /data/app/com.hypedomino/lib/arm64/libil2cpp.so\n" +
            "7a3d000000-7a3d100000 r-xp 00000000 fd:05 9012 /data/app/com.hypedomino/lib/arm64/libunity.so\n";

        static List<string> Collect(string maps, bool debugger)
        {
            var signals = new List<string>();
            HookSignals.Collect(maps, debugger, signals);
            return signals;
        }

        [Test]
        public void CleanProcessRaisesNothing()
        {
            Assert.IsEmpty(Collect(CleanMaps, false));
        }

        [TestCase("7b00000000-7b00100000 r-xp 00000000 00:00 0 /data/local/tmp/re.frida.server/frida-agent-64.so")]
        [TestCase("7b00000000-7b00100000 r-xp 00000000 fd:05 77 /data/app/com.hypedomino/lib/arm64/libfrida-gadget.so")]
        public void FridaAgentOrGadgetRaisesInstrumentation(string line)
        {
            Assert.AreEqual(new[] { HookSignals.Instrumentation }, Collect(CleanMaps + line, false));
        }

        [TestCase("7c00000000-7c00100000 r--p 00000000 fd:05 88 /system/framework/XposedBridge.jar")]
        [TestCase("7c00000000-7c00100000 r-xp 00000000 fd:05 99 /data/adb/modules/zygisk_lsposed/lib/arm64/liblspd.so")]
        [TestCase("7c00000000-7c00100000 r-xp 00000000 fd:05 99 /data/app/com.saurik.substrate/lib/arm/libsubstrate.so")]
        public void HookFrameworkLibrariesRaiseHookFramework(string line)
        {
            Assert.AreEqual(new[] { HookSignals.HookFramework }, Collect(CleanMaps + line, false));
        }

        [Test]
        public void AttachedDebuggerRaisesDebugger()
        {
            Assert.AreEqual(new[] { HookSignals.Debugger }, Collect(CleanMaps, true));
        }

        [Test]
        public void UnreadableMapsRaiseNothing()
        {
            Assert.IsEmpty(Collect(null, false));
        }
    }
}
