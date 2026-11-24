using System;
using System.Collections.Generic;
using HypeDomino.Core.DeviceRisk;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class DeviceRiskSignalsTests
    {
        static readonly DeviceFacts Pixel = new DeviceFacts(
            "google/oriole/oriole:14/AP2A.240805.005/12025142:user/release-keys", "Pixel 6", "Google", "oriole", "oriole", "release-keys");

        static readonly DeviceFacts Galaxy = new DeviceFacts(
            "samsung/a54xnsxx/a54x:14/UP1A.231005.007/A546BXXS8BXF1:user/release-keys", "SM-A546B", "samsung", "s5e8835", "a54xnsxx", "release-keys");

        static readonly Func<string, bool> Nothing = _ => false;

        static List<string> Android(DeviceFacts facts, Func<string, bool> exists)
        {
            var signals = new List<string>();
            DeviceRiskSignals.CollectAndroid(facts, exists, signals);
            return signals;
        }

        [Test]
        public void RetailDevicesRaiseNoSignal()
        {
            Assert.IsEmpty(Android(Pixel, Nothing));
            Assert.IsEmpty(Android(Galaxy, Nothing));
        }

        [Test]
        public void SuBinaryRaisesRoot()
        {
            Assert.AreEqual(new[] { DeviceRiskSignals.Root }, Android(Pixel, path => path == "/system/xbin/su"));
        }

        [Test]
        public void TestKeysBuildRaisesTestKeys()
        {
            var custom = new DeviceFacts("lineage/x/x:14/A/1:userdebug/test-keys", "Pixel 6", "Google", "oriole", "lineage_oriole", "test-keys");
            Assert.AreEqual(new[] { DeviceRiskSignals.TestKeys }, Android(custom, Nothing));
        }

        [TestCase("google/sdk_gphone64_x86_64/emu64xa:14/UE1A.230829.036/10808000:user/release-keys", "sdk_gphone64_x86_64", "Google", "ranchu", "sdk_gphone64_x86_64")]
        [TestCase("generic/sdk/generic:9/PSR1.180720.075/5124027:userdebug/dev-keys", "Android SDK built for x86", "unknown", "goldfish", "sdk")]
        [TestCase("Android/vbox86p/vbox86p:11/RQ1A/1:userdebug/release-keys", "Custom Phone", "Genymotion", "vbox86", "vbox86p")]
        public void EmulatorsRaiseEmulator(string fingerprint, string model, string manufacturer, string hardware, string product)
        {
            var facts = new DeviceFacts(fingerprint, model, manufacturer, hardware, product, "release-keys");
            CollectionAssert.Contains(Android(facts, Nothing), DeviceRiskSignals.Emulator);
        }

        [Test]
        public void MissingBuildFieldsRaiseNothing()
        {
            Assert.IsEmpty(Android(new DeviceFacts(null, null, null, null, null, null), Nothing));
        }

        [Test]
        public void JailbreakPathRaisesJailbreak()
        {
            var signals = new List<string>();
            DeviceRiskSignals.CollectIos(path => path == "/var/jb", signals);
            Assert.AreEqual(new[] { DeviceRiskSignals.Jailbreak }, signals);
        }

        [Test]
        public void StockIosRaisesNothing()
        {
            var signals = new List<string>();
            DeviceRiskSignals.CollectIos(Nothing, signals);
            Assert.IsEmpty(signals);
        }
    }
}
