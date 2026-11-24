using HypeDomino.UI.Graphics;
using HypeDomino.UI.Motion;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class GraphicsQualityTests
    {
        [TestCase(3650, "Mali-G52 MC2", 60, DeviceTier.Low)]
        [TestCase(1850, "Mali-G57 MC2", 60, DeviceTier.Low)]
        [TestCase(2800, "Apple A12 GPU", 60, DeviceTier.Low)]
        [TestCase(5600, "Adreno (TM) 506", 60, DeviceTier.Low)]
        [TestCase(5600, "Mali-G72 MP3", 60, DeviceTier.Low)]
        [TestCase(5600, "PowerVR Rogue GE8320", 60, DeviceTier.Low)]
        [TestCase(5500, "Mali-G68 MC4", 120, DeviceTier.Mid)]
        [TestCase(3700, "Apple A14 GPU", 60, DeviceTier.Mid)]
        [TestCase(7400, "Adreno (TM) 710", 120, DeviceTier.Mid)]
        [TestCase(5600, "Adreno (TM) 610", 60, DeviceTier.Mid)]
        [TestCase(5600, "Adreno (TM) 750", 120, DeviceTier.Mid)]
        [TestCase(7400, "Adreno (TM) 750", 60, DeviceTier.Mid)]
        [TestCase(7400, "", 120, DeviceTier.Mid)]
        [TestCase(7400, "Adreno (TM) 750", 120, DeviceTier.High)]
        [TestCase(7400, "Samsung Xclipse 940", 120, DeviceTier.High)]
        [TestCase(7400, "Mali-G710 MC10", 120, DeviceTier.High)]
        [TestCase(7400, "Immortalis-G720 MC12", 120, DeviceTier.High)]
        [TestCase(7500, "Apple A17 Pro GPU", 120, DeviceTier.High)]
        public void TheClassRulePutsEachDeviceInItsTier(int memoryMb, string gpuName, int maxRefreshHz, DeviceTier tier)
        {
            Assert.AreEqual(tier, DeviceClass.TierOf(new DeviceFacts(memoryMb, gpuName, maxRefreshHz)));
        }

        [TestCase(DeviceTier.Low, GraphicsLevel.Low, GraphicsLevel.Low)]
        [TestCase(DeviceTier.Mid, GraphicsLevel.Medium, GraphicsLevel.Medium)]
        [TestCase(DeviceTier.High, GraphicsLevel.High, GraphicsLevel.Ultra)]
        public void AutoPicksTheTierDefaultBelowTheHighest(DeviceTier tier, GraphicsLevel auto, GraphicsLevel highest)
        {
            var quality = new GraphicsQuality(tier, null, 0, false);
            Assert.IsNull(quality.Choice);
            Assert.AreEqual(auto, quality.Level);
            Assert.AreEqual(highest, quality.Highest);
        }

        [Test]
        public void ALevelAboveTheHighestCannotBeChosen()
        {
            var quality = new GraphicsQuality(DeviceTier.Mid, null, 0, false);
            var changes = 0;
            quality.Changed += () => changes++;

            Assert.IsFalse(quality.Choose(GraphicsLevel.High));
            Assert.IsFalse(quality.Choose(GraphicsLevel.Ultra));
            Assert.AreEqual(GraphicsLevel.Medium, quality.Level);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void ALowerLevelAppliesOnceAndAutoComesBack()
        {
            var quality = new GraphicsQuality(DeviceTier.High, null, 0, false);
            var changes = 0;
            quality.Changed += () => changes++;

            Assert.IsTrue(quality.Choose(GraphicsLevel.Low));
            Assert.IsTrue(quality.Choose(GraphicsLevel.Low));
            Assert.AreEqual(GraphicsLevel.Low, quality.Level);
            Assert.AreEqual(1, changes);

            Assert.IsTrue(quality.Choose(null));
            Assert.AreEqual(GraphicsLevel.High, quality.Level);
            Assert.AreEqual(2, changes);
        }

        [Test]
        public void ASavedChoiceAboveTheHighestCountsAsTheHighest()
        {
            var quality = new GraphicsQuality(DeviceTier.Mid, GraphicsLevel.Ultra, 0, false);
            Assert.AreEqual(GraphicsLevel.Medium, quality.Choice);
            Assert.AreEqual(GraphicsLevel.Medium, quality.Level);
        }

        [TestCase(GraphicsLevel.Low, EffectLevel.Low)]
        [TestCase(GraphicsLevel.Medium, EffectLevel.Medium)]
        [TestCase(GraphicsLevel.High, EffectLevel.High)]
        [TestCase(GraphicsLevel.Ultra, EffectLevel.High)]
        public void EachLevelRunsItsEffectLevel(GraphicsLevel level, EffectLevel effects)
        {
            Assert.AreEqual(effects, new GraphicsQuality(DeviceTier.High, level, 0, false).Effects);
        }

        [TestCase(GraphicsLevel.Low, 1080, 0.6667f)]
        [TestCase(GraphicsLevel.Low, 720, 1f)]
        [TestCase(GraphicsLevel.Medium, 1440, 0.75f)]
        [TestCase(GraphicsLevel.High, 1440, 0.75f)]
        [TestCase(GraphicsLevel.Ultra, 1440, 1f)]
        [TestCase(GraphicsLevel.Ultra, 0, 1f)]
        public void TheCameraRendersNoLargerThanTheLevelCap(GraphicsLevel level, int nativeShortSide, float renderScale)
        {
            Assert.AreEqual(renderScale, new GraphicsQuality(DeviceTier.High, level, 0, false).RenderScale(nativeShortSide), 0.0001f);
        }

        [Test]
        public void ASlowProbeDropsAutoOneLevelAndLocksTheLevelsAbove()
        {
            var quality = new GraphicsQuality(DeviceTier.High, null, 0, false);
            var changes = 0;
            quality.Changed += () => changes++;

            Assert.AreEqual(ProbeResult.Dropped, quality.SettleProbe(7.5));
            Assert.AreEqual(GraphicsLevel.Medium, quality.Level);
            Assert.AreEqual(GraphicsLevel.Medium, quality.Highest);
            Assert.IsFalse(quality.CanChoose(GraphicsLevel.High));
            Assert.IsTrue(quality.ProbeWanted);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void APassingProbeStopsWithoutAChange()
        {
            var quality = new GraphicsQuality(DeviceTier.Mid, null, 0, false);
            var changes = 0;
            quality.Changed += () => changes++;

            Assert.AreEqual(ProbeResult.Passed, quality.SettleProbe(14.0));
            Assert.AreEqual(GraphicsLevel.Medium, quality.Level);
            Assert.IsFalse(quality.ProbeWanted);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void AProbeWithoutTimingDataChangesNothing()
        {
            var quality = new GraphicsQuality(DeviceTier.Mid, null, 0, false);
            Assert.AreEqual(ProbeResult.NoData, quality.SettleProbe(0));
            Assert.AreEqual(GraphicsLevel.Medium, quality.Level);
            Assert.IsTrue(quality.ProbeWanted);
        }

        [Test]
        public void ASlowProbeOnLowStopsThere()
        {
            var quality = new GraphicsQuality(DeviceTier.Low, null, 0, false);
            Assert.AreEqual(ProbeResult.SlowAtLowest, quality.SettleProbe(40.0));
            Assert.AreEqual(GraphicsLevel.Low, quality.Level);
            Assert.IsFalse(quality.ProbeWanted);
        }

        [Test]
        public void AChosenLevelIsNotProbed()
        {
            Assert.IsFalse(new GraphicsQuality(DeviceTier.High, GraphicsLevel.Medium, 0, false).ProbeWanted);
        }

        [TestCase(DeviceTier.High, 1, GraphicsLevel.Medium)]
        [TestCase(DeviceTier.High, 2, GraphicsLevel.Low)]
        [TestCase(DeviceTier.High, 5, GraphicsLevel.Low)]
        [TestCase(DeviceTier.Mid, -3, GraphicsLevel.Medium)]
        public void SavedDropsLowerAutoButNeverBelowLow(DeviceTier tier, int drops, GraphicsLevel level)
        {
            Assert.AreEqual(level, new GraphicsQuality(tier, null, drops, false).Level);
        }

        [Test]
        public void ASavedChoiceAboveTheDroppedLevelCountsAsIt()
        {
            var quality = new GraphicsQuality(DeviceTier.High, GraphicsLevel.Ultra, 1, true);
            Assert.AreEqual(GraphicsLevel.Medium, quality.Level);
        }

        [TestCase(GraphicsLevel.Ultra, ThermalStatus.Moderate, GraphicsLevel.High)]
        [TestCase(GraphicsLevel.Medium, ThermalStatus.Moderate, GraphicsLevel.Low)]
        [TestCase(GraphicsLevel.Low, ThermalStatus.Moderate, GraphicsLevel.Low)]
        [TestCase(GraphicsLevel.Ultra, ThermalStatus.Severe, GraphicsLevel.Low)]
        public void HeatLowersTheLevelInUseAtOnceButKeepsTheChoice(GraphicsLevel chosen, ThermalStatus heat, GraphicsLevel inUse)
        {
            var quality = new GraphicsQuality(DeviceTier.High, chosen, 0, false);
            var changes = 0;
            quality.Changed += () => changes++;

            quality.UpdateHeat(heat, 10f);

            Assert.AreEqual(inUse, quality.Level);
            Assert.AreEqual(chosen, quality.Choice);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void TheChosenLevelComesBackOnlyAfterTheDeviceStaysCool()
        {
            var quality = new GraphicsQuality(DeviceTier.High, GraphicsLevel.High, 0, false);
            quality.UpdateHeat(ThermalStatus.Severe, 0f);

            quality.UpdateHeat(ThermalStatus.Normal, 100f);
            quality.UpdateHeat(ThermalStatus.Normal, 159f);
            Assert.AreEqual(GraphicsLevel.Low, quality.Level);

            quality.UpdateHeat(ThermalStatus.Normal, 160f);
            Assert.AreEqual(GraphicsLevel.High, quality.Level);
        }

        [Test]
        public void AFlareDuringCoolingRestartsTheWait()
        {
            var quality = new GraphicsQuality(DeviceTier.High, GraphicsLevel.High, 0, false);
            quality.UpdateHeat(ThermalStatus.Moderate, 0f);
            quality.UpdateHeat(ThermalStatus.Normal, 10f);
            quality.UpdateHeat(ThermalStatus.Moderate, 50f);
            quality.UpdateHeat(ThermalStatus.Normal, 60f);
            quality.UpdateHeat(ThermalStatus.Normal, 100f);
            Assert.AreEqual(GraphicsLevel.Medium, quality.Level);

            quality.UpdateHeat(ThermalStatus.Normal, 120f);
            Assert.AreEqual(GraphicsLevel.High, quality.Level);
        }

        [Test]
        public void AHotDeviceIsNotProbed()
        {
            var quality = new GraphicsQuality(DeviceTier.High, null, 0, false);
            quality.UpdateHeat(ThermalStatus.Moderate, 0f);

            Assert.IsFalse(quality.ProbeWanted);
            Assert.AreEqual(ProbeResult.NoData, quality.SettleProbe(20.0));
            Assert.AreEqual(0, quality.ProbeDrops);
        }

        [TestCase(GraphicsLevel.Low, 28f)]
        [TestCase(GraphicsLevel.Medium, 14f)]
        [TestCase(GraphicsLevel.Ultra, 7f)]
        public void TheProbeGateIsTheWorkTimeOfTheEffectLevel(GraphicsLevel level, float gateMs)
        {
            Assert.AreEqual(gateMs, new GraphicsQuality(DeviceTier.High, level, 0, false).WorkTimeGateMs);
        }
    }
}
