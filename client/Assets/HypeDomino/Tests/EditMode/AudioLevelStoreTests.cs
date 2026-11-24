using HypeDomino.Game;
using HypeDomino.UI.Audio;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AudioLevelStoreTests
    {
        const float Unset = -1f;
        const float Low = 0.25f;
        const float High = 0.75f;
        const int DragFrames = 120;

        static readonly string[] Keys = { AudioLevelStore.MusicKey, AudioLevelStore.SfxKey, AudioLevelStore.UiKey };

        readonly float[] kept = new float[Keys.Length];
        AudioVolumes volumes;
        float dragLevel;

        [SetUp]
        public void KeepDeviceLevels()
        {
            for (var i = 0; i < Keys.Length; i++)
            {
                kept[i] = PlayerPrefs.GetFloat(Keys[i], Unset);
                PlayerPrefs.DeleteKey(Keys[i]);
            }
        }

        [TearDown]
        public void RestoreDeviceLevels()
        {
            for (var i = 0; i < Keys.Length; i++)
            {
                if (kept[i] == Unset) PlayerPrefs.DeleteKey(Keys[i]);
                else PlayerPrefs.SetFloat(Keys[i], kept[i]);
            }
            PlayerPrefs.Save();
        }

        [Test]
        public void EveryGroupStartsFullWhenTheDeviceHoldsNoLevel()
        {
            var loaded = AudioLevelStore.Load();
            Assert.AreEqual(AudioVolumes.Full, loaded.Level(AudioGroup.Music));
            Assert.AreEqual(AudioVolumes.Full, loaded.Level(AudioGroup.Sfx));
            Assert.AreEqual(AudioVolumes.Full, loaded.Level(AudioGroup.Ui));
        }

        [Test]
        public void ALevelIsWrittenToTheDeviceOnlyWhenSaved()
        {
            var store = new AudioLevelStore(volumes = AudioLevelStore.Load());
            volumes.Set(AudioGroup.Sfx, Low);
            Assert.IsTrue(store.Unsaved);
            Assert.IsFalse(PlayerPrefs.HasKey(AudioLevelStore.SfxKey));

            Assert.IsTrue(store.Save());

            Assert.IsFalse(store.Unsaved);
            Assert.AreEqual(Low, PlayerPrefs.GetFloat(AudioLevelStore.SfxKey));
            Assert.AreEqual(AudioVolumes.Full, PlayerPrefs.GetFloat(AudioLevelStore.MusicKey));
            Assert.AreEqual(AudioVolumes.Full, PlayerPrefs.GetFloat(AudioLevelStore.UiKey));
        }

        [Test]
        public void TheNextStartLoadsTheSavedLevels()
        {
            var store = new AudioLevelStore(volumes = AudioLevelStore.Load());
            volumes.Set(AudioGroup.Music, 0f);
            volumes.Set(AudioGroup.Ui, High);
            store.Save();

            var loaded = AudioLevelStore.Load();

            Assert.AreEqual(0f, loaded.Level(AudioGroup.Music));
            Assert.AreEqual(AudioVolumes.Full, loaded.Level(AudioGroup.Sfx));
            Assert.AreEqual(High, loaded.Level(AudioGroup.Ui));
        }

        [Test]
        public void ADragOfManyChangesIsSavedOnceAtItsLastLevel()
        {
            var store = new AudioLevelStore(volumes = AudioLevelStore.Load());
            for (var frame = 0; frame < DragFrames; frame++) Drag();
            volumes.Set(AudioGroup.Music, High);

            Assert.IsTrue(store.Save());
            Assert.IsFalse(store.Save());
            Assert.AreEqual(High, PlayerPrefs.GetFloat(AudioLevelStore.MusicKey));
        }

        [Test]
        public void NothingIsWrittenWithoutAChange()
        {
            var store = new AudioLevelStore(volumes = AudioLevelStore.Load());
            volumes.Set(AudioGroup.Sfx, AudioVolumes.Full);

            Assert.IsFalse(store.Unsaved);
            Assert.IsFalse(store.Save());
            Assert.IsFalse(PlayerPrefs.HasKey(AudioLevelStore.SfxKey));
        }

        [Test]
        public void DraggingALevelAllocatesNothing()
        {
            new AudioLevelStore(volumes = AudioLevelStore.Load());
            Drag();

            Assert.That(Drag, Allocates.Nothing);
        }

        void Drag()
        {
            dragLevel = dragLevel == Low ? High : Low;
            volumes.Set(AudioGroup.Music, dragLevel);
        }
    }
}
