using System;
using HypeDomino.Content;
using HypeDomino.Game;
using HypeDomino.UI.Audio;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Settings;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SettingsScreenViewTests
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

        GameObject root;
        FakeStore store;
        bool inMatch;
        GraphicsQuality graphics;
        SettingsModel settings;
        GameStrings strings;
        TMP_Dropdown languages;
        Slider music;
        Toggle reduceMotion;
        Toggle haptics;
        Toggle autoFrameRate;
        TMP_Dropdown frameRate;
        Toggle batterySaver;
        Button clearCache;
        TMP_Text message;

        [SetUp]
        public void Build()
        {
            root = new GameObject(nameof(SettingsScreenViewTests), typeof(RectTransform));
            store = new FakeStore { OccupiedBytes = 4096 };
            inMatch = false;
            graphics = new GraphicsQuality(DeviceTier.High, GraphicsLevel.High, 0, false);
            var language = LanguageChoice.Load(GameLanguage.English, Array.Empty<string>);
            settings = new SettingsModel(
                language,
                new AudioVolumes(AudioVolumes.Full, AudioVolumes.Full, AudioVolumes.Full),
                new MotionPreference(null, false),
                graphics,
                new FrameRatePolicy(graphics, Panel120, null, false),
                new HapticsPreference(true),
                new ContentCache(store, () => inMatch));
            strings = new GameStrings(BuiltinStrings.Load(), language, false);
            languages = Add<TMP_Dropdown>();
            music = Add<Slider>();
            reduceMotion = Add<Toggle>();
            haptics = Add<Toggle>();
            autoFrameRate = Add<Toggle>();
            frameRate = Add<TMP_Dropdown>();
            batterySaver = Add<Toggle>();
            clearCache = Add<Button>();
            message = Add<TextMeshProUGUI>();
        }

        [TearDown]
        public void Destroy()
        {
            settings.Dispose();
            Object.DestroyImmediate(root);
        }

        T Add<T>() where T : Component
        {
            var child = new GameObject(typeof(T).Name, typeof(RectTransform));
            child.transform.SetParent(root.transform, false);
            return child.AddComponent<T>();
        }

        SettingToggleRow Row(Toggle toggle)
        {
            var row = Add<SettingToggleRow>();
            row.Configure(toggle);
            return row;
        }

        void Show()
        {
            var languageRow = Add<LanguageRow>();
            languageRow.Configure(languages);
            var musicRow = Add<VolumeSliderRow>();
            musicRow.Configure(music, AudioGroup.Music);
            var view = root.AddComponent<SettingsScreenView>();
            view.Configure(languageRow, new[] { musicRow }, Row(reduceMotion), Row(haptics), Row(autoFrameRate), frameRate, Row(batterySaver), clearCache, message);
            view.Bind(settings, strings);
        }

        static int[] Rates(TMP_Dropdown dropdown)
        {
            var rates = new int[dropdown.options.Count];
            for (var i = 0; i < rates.Length; i++) rates[i] = int.Parse(dropdown.options[i].text);
            return rates;
        }

        [Test]
        public void EveryRowStartsAtItsValueWithoutChangingIt()
        {
            var changes = 0;
            settings.Changed += () => changes++;

            Show();

            Assert.AreEqual(GameLanguage.LaunchCount, languages.options.Count);
            Assert.AreEqual(AudioVolumes.Full, music.value);
            Assert.IsFalse(reduceMotion.isOn);
            Assert.IsTrue(haptics.isOn);
            Assert.IsTrue(autoFrameRate.isOn);
            Assert.IsFalse(batterySaver.isOn);
            CollectionAssert.AreEqual(new[] { 30, 60, 120 }, Rates(frameRate));
            Assert.AreEqual(60, Rates(frameRate)[frameRate.value]);
            Assert.IsFalse(message.enabled);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void EachRowChangesItsOwnSetting()
        {
            Show();

            music.value = 0.25f;
            reduceMotion.isOn = true;
            haptics.isOn = false;
            batterySaver.isOn = true;

            Assert.AreEqual(0.25f, settings.Volumes.Level(AudioGroup.Music), 1e-5f);
            Assert.IsTrue(settings.Motion.ReduceMotion);
            Assert.IsFalse(settings.Haptics.Enabled);
            Assert.IsTrue(settings.FrameRate.BatterySaver);
        }

        [Test]
        public void PickingARateLeavesAutoAndTurningAutoOnForgetsTheRate()
        {
            Show();

            frameRate.value = 2;
            Assert.AreEqual(120, settings.FrameRate.Choice);
            Assert.IsFalse(autoFrameRate.isOn);

            autoFrameRate.isOn = true;
            Assert.IsNull(settings.FrameRate.Choice);
            Assert.AreEqual(60, Rates(frameRate)[frameRate.value]);
        }

        [Test]
        public void TurningAutoOffKeepsTheRateInUse()
        {
            Show();

            autoFrameRate.isOn = false;

            Assert.AreEqual(60, settings.FrameRate.Choice);
        }

        [Test]
        public void ALowerGraphicsLevelOffersOnlyTheRatesItAllows()
        {
            Show();

            graphics.Choose(GraphicsLevel.Low);

            CollectionAssert.AreEqual(new[] { 30 }, Rates(frameRate));
            Assert.AreEqual(0, frameRate.value);
        }

        [Test]
        public void ChangesMadeElsewhereShowOnTheToggles()
        {
            Show();

            settings.Haptics.Set(false);
            settings.Motion.Choose(true);
            settings.FrameRate.Choose(30);

            Assert.IsFalse(haptics.isOn);
            Assert.IsTrue(reduceMotion.isOn);
            Assert.IsFalse(autoFrameRate.isOn);
            Assert.AreEqual(30, Rates(frameRate)[frameRate.value]);
        }

        [TestCase(false, false, CacheClearKeys.Cleared)]
        [TestCase(true, false, CacheClearKeys.AtNextStart)]
        [TestCase(false, true, CacheClearKeys.NotInMatch)]
        public void ClearingTheCacheShowsItsOutcome(bool inUse, bool playing, string key)
        {
            store.InUse = inUse;
            inMatch = playing;
            Show();

            clearCache.onClick.Invoke();

            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(key), message.text);
        }
    }
}
