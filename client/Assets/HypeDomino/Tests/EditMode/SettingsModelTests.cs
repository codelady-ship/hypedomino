using System;
using HypeDomino.Content;
using HypeDomino.Game;
using HypeDomino.UI.Audio;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Motion;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SettingsModelTests
    {
        static readonly int[] Panel120 = { 60, 120 };

        sealed class FakeStore : ICacheStore
        {
            public bool InUse;

            public long OccupiedBytes { get; set; }

            public bool ClearAtStart { get; set; }

            public void ApplyLimits(long limitBytes, int unusedForSeconds)
            {
            }

            public bool TryClear()
            {
                if (InUse) return false;
                OccupiedBytes = 0;
                return true;
            }
        }

        FakeStore store;
        bool inMatch;
        SettingsModel settings;
        int changes;

        [SetUp]
        public void Create()
        {
            store = new FakeStore { OccupiedBytes = 4096 };
            inMatch = false;
            var graphics = new GraphicsQuality(DeviceTier.High, null, 0, false);
            settings = new SettingsModel(
                LanguageChoice.Load("az", Array.Empty<string>),
                new AudioVolumes(AudioVolumes.Full, AudioVolumes.Full, AudioVolumes.Full),
                new MotionPreference(null, false),
                graphics,
                new FrameRatePolicy(graphics, Panel120, null, false),
                new HapticsPreference(true),
                new ContentCache(store, () => inMatch));
            changes = 0;
            settings.Changed += () => changes++;
        }

        [Test]
        public void EveryRowRaisesOneChangePerNewValue()
        {
            settings.Language.Choose("ru");
            settings.Volumes.Set(AudioGroup.Music, 0f);
            settings.Motion.Choose(true);
            settings.Graphics.Choose(GraphicsLevel.Low);
            settings.FrameRate.Choose(30);
            settings.FrameRate.SetBatterySaver(true);
            settings.Haptics.Set(false);
            Assert.AreEqual(7, changes);
            Assert.AreEqual("ru", settings.Language.Language);
            Assert.IsFalse(settings.Haptics.Enabled);
        }

        [Test]
        public void ChoosingTheValueInUseRaisesNothing()
        {
            settings.Language.Choose("az");
            settings.Volumes.Set(AudioGroup.Sfx, AudioVolumes.Full);
            settings.Motion.Choose(null);
            settings.Graphics.Choose(null);
            settings.FrameRate.Choose(null);
            settings.FrameRate.SetBatterySaver(false);
            settings.Haptics.Set(true);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void ASystemChangeShowsWhileTheRowFollowsTheSystem()
        {
            settings.Motion.SetSystemReduce(true);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void ACacheClearedAtOnceShowsTheNewSize()
        {
            Assert.AreEqual(4096L, settings.CacheBytes);
            Assert.AreEqual(CacheClearResult.Cleared, settings.ClearCache());
            Assert.AreEqual(0L, settings.CacheBytes);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void ACacheClearThatWaitsChangesNothingOnScreen()
        {
            store.InUse = true;
            Assert.AreEqual(CacheClearResult.AtNextStart, settings.ClearCache());
            inMatch = true;
            Assert.AreEqual(CacheClearResult.NotInMatch, settings.ClearCache());
            Assert.AreEqual(4096L, settings.CacheBytes);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void AClosedScreenNoLongerHearsChanges()
        {
            settings.Dispose();
            settings.Language.Choose("en");
            settings.Volumes.Set(AudioGroup.Ui, 0f);
            settings.Motion.Choose(true);
            settings.Graphics.Choose(GraphicsLevel.Low);
            settings.Haptics.Set(false);
            Assert.AreEqual(0, changes);
        }
    }
}
