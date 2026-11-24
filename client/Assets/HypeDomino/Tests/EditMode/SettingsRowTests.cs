using System;
using HypeDomino.Game;
using HypeDomino.UI.Audio;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Settings;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SettingsRowTests
    {
        GameObject root;

        [SetUp]
        public void Build() => root = new GameObject(nameof(SettingsRowTests));

        [TearDown]
        public void Destroy() => Object.DestroyImmediate(root);

        T Add<T>() where T : Component
        {
            var child = new GameObject(typeof(T).Name);
            child.transform.SetParent(root.transform);
            return child.AddComponent<T>();
        }

        VolumeSliderRow VolumeRow(AudioGroup group, out Slider slider)
        {
            slider = Add<Slider>();
            var row = Add<VolumeSliderRow>();
            row.Configure(slider, group);
            return row;
        }

        [Test]
        public void TheVolumeSliderStartsAtTheGroupLevelWithoutChangingIt()
        {
            var volumes = new AudioVolumes(0.25f, 0.5f, 0.75f);
            var changes = 0;
            volumes.Changed += () => changes++;

            VolumeRow(AudioGroup.Sfx, out var slider).Bind(volumes);

            Assert.AreEqual(0.5f, slider.value);
            Assert.AreEqual(0f, slider.minValue);
            Assert.AreEqual(AudioVolumes.Full, slider.maxValue);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void MovingTheVolumeSliderSetsOnlyItsGroupAndMarksTheLevelsUnsaved()
        {
            var volumes = new AudioVolumes(AudioVolumes.Full, AudioVolumes.Full, AudioVolumes.Full);
            var store = new AudioLevelStore(volumes);
            VolumeRow(AudioGroup.Music, out var slider).Bind(volumes);

            slider.value = 0.3f;

            Assert.AreEqual(0.3f, volumes.Level(AudioGroup.Music), 1e-5f);
            Assert.AreEqual(AudioVolumes.Full, volumes.Level(AudioGroup.Sfx));
            Assert.AreEqual(AudioVolumes.Full, volumes.Level(AudioGroup.Ui));
            Assert.IsTrue(store.Unsaved);
        }

        [Test]
        public void ARestartedVolumeRowDrivesOnlyTheNewVolumes()
        {
            var first = new AudioVolumes(AudioVolumes.Full, AudioVolumes.Full, AudioVolumes.Full);
            var second = new AudioVolumes(AudioVolumes.Full, AudioVolumes.Full, AudioVolumes.Full);
            var row = VolumeRow(AudioGroup.Ui, out var slider);
            row.Bind(first);
            row.Bind(second);

            slider.value = 0f;

            Assert.AreEqual(AudioVolumes.Full, first.Level(AudioGroup.Ui));
            Assert.AreEqual(0f, second.Level(AudioGroup.Ui));
        }

        [Test]
        public void TheHapticsToggleShowsAndChangesThePreference()
        {
            var haptics = new HapticsPreference(false);
            var toggle = Add<Toggle>();
            toggle.isOn = true;
            var row = Add<SettingToggleRow>();
            row.Configure(toggle);

            row.Bind(haptics.Enabled, haptics.Set);
            Assert.IsFalse(toggle.isOn);
            Assert.IsFalse(haptics.Enabled);

            toggle.isOn = true;
            Assert.IsTrue(haptics.Enabled);
            toggle.isOn = false;
            Assert.IsFalse(haptics.Enabled);
        }

        LanguageRow LanguageRow(out TMP_Dropdown dropdown)
        {
            dropdown = Add<TMP_Dropdown>();
            var row = Add<LanguageRow>();
            row.Configure(dropdown);
            return row;
        }

        static LanguageChoice Chosen(string language) => LanguageChoice.Load(language, Array.Empty<string>);

        [Test]
        public void TheLanguageRowListsEveryLaunchLanguageInItsOwnNameAndShowsTheChoice()
        {
            var choice = Chosen("tr");

            LanguageRow(out var dropdown).Bind(choice);

            Assert.AreEqual(GameLanguage.LaunchCount, dropdown.options.Count);
            for (var i = 0; i < GameLanguage.LaunchCount; i++)
                Assert.AreEqual(GameLanguage.OwnName(GameLanguage.LaunchAt(i)), dropdown.options[i].text);
            Assert.AreEqual("tr", GameLanguage.LaunchAt(dropdown.value));
            Assert.AreEqual("tr", choice.Language);
        }

        [Test]
        public void PickingALanguageChoosesIt()
        {
            var choice = Chosen(GameLanguage.English);
            var changes = 0;
            choice.Changed += () => changes++;
            var row = LanguageRow(out var dropdown);
            row.Bind(choice);
            row.Bind(choice);

            dropdown.value = IndexOf("ru");

            Assert.AreEqual("ru", choice.Language);
            Assert.AreEqual(1, changes);
            Assert.AreEqual(GameLanguage.LaunchCount, dropdown.options.Count);
        }

        static int IndexOf(string language)
        {
            for (var i = 0; i < GameLanguage.LaunchCount; i++)
                if (GameLanguage.LaunchAt(i) == language) return i;
            return -1;
        }
    }
}
