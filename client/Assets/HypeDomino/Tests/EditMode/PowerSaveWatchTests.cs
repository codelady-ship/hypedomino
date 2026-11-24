using HypeDomino.Game;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PowerSaveWatchTests
    {
        [TestCase(RuntimePlatform.Android)]
        [TestCase(RuntimePlatform.IPhonePlayer)]
        public void TheSystemSaverIsWatchedOnPhones(RuntimePlatform platform)
        {
            Assert.IsTrue(PowerSaveWatch.Watches(platform));
        }

        [TestCase(RuntimePlatform.WindowsEditor)]
        [TestCase(RuntimePlatform.OSXEditor)]
        [TestCase(RuntimePlatform.LinuxEditor)]
        [TestCase(RuntimePlatform.WindowsPlayer)]
        public void TheEditorAndDesktopHaveNoSystemSaver(RuntimePlatform platform)
        {
            Assert.IsFalse(PowerSaveWatch.Watches(platform));
        }
    }
}
