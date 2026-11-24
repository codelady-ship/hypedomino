using System.Collections.Generic;
using System.Globalization;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Settings;
using HypeDomino.UI.Tokens;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game
{
    public sealed class SettingsScreenView : MonoBehaviour
    {
        [SerializeField] LanguageRow language;
        [SerializeField] VolumeSliderRow[] volumes;
        [SerializeField] SettingToggleRow reduceMotion;
        [SerializeField] SettingToggleRow haptics;
        [SerializeField] SettingToggleRow autoFrameRate;
        [SerializeField] TMP_Dropdown frameRate;
        [SerializeField] SettingToggleRow batterySaver;
        [SerializeField] Button clearCache;
        [SerializeField] TMP_Text message;

        readonly List<int> offeredRates = new();
        SettingsModel settings;
        GameStrings strings;
        string messageKey;

        public void Configure(LanguageRow languageRow, VolumeSliderRow[] volumeRows, SettingToggleRow reduceMotionRow, SettingToggleRow hapticsRow,
            SettingToggleRow autoFrameRateRow, TMP_Dropdown frameRates, SettingToggleRow batterySaverRow, Button clearCacheButton, TMP_Text messageLine)
        {
            Unbind();
            language = languageRow;
            volumes = volumeRows;
            reduceMotion = reduceMotionRow;
            haptics = hapticsRow;
            autoFrameRate = autoFrameRateRow;
            frameRate = frameRates;
            batterySaver = batterySaverRow;
            clearCache = clearCacheButton;
            message = messageLine;
        }

        public void Bind(SettingsModel model, GameStrings texts)
        {
            Unbind();
            settings = model;
            strings = texts;
            messageKey = null;
            language.Bind(settings.Language);
            foreach (var row in volumes) row.Bind(settings.Volumes);
            reduceMotion.Bind(settings.Motion.ReduceMotion, OnReduceMotion);
            haptics.Bind(settings.Haptics.Enabled, OnHaptics);
            autoFrameRate.Bind(settings.FrameRate.Choice == null, OnAutoFrameRate);
            batterySaver.Bind(settings.FrameRate.BatterySaver, OnBatterySaver);
            settings.Changed += ShowToggles;
            settings.FrameRate.Changed += ShowRates;
            strings.Changed += ShowMessage;
            frameRate.onValueChanged.AddListener(OnRatePicked);
            clearCache.onClick.AddListener(OnClearCache);
            ShowRates();
            ShowMessage();
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (settings == null) return;
            settings.Changed -= ShowToggles;
            settings.FrameRate.Changed -= ShowRates;
            strings.Changed -= ShowMessage;
            frameRate.onValueChanged.RemoveListener(OnRatePicked);
            clearCache.onClick.RemoveListener(OnClearCache);
            settings = null;
            strings = null;
        }

        void OnReduceMotion(bool on) => settings.Motion.Choose(on);

        void OnHaptics(bool on) => settings.Haptics.Set(on);

        void OnAutoFrameRate(bool on) => settings.FrameRate.Choose(on ? null : settings.FrameRate.Chosen);

        void OnBatterySaver(bool on) => settings.FrameRate.SetBatterySaver(on);

        void OnRatePicked(int index) => settings.FrameRate.Choose(offeredRates[index]);

        void OnClearCache()
        {
            messageKey = CacheClearKeys.Of(settings.ClearCache());
            ShowMessage();
        }

        void ShowToggles()
        {
            reduceMotion.Show(settings.Motion.ReduceMotion);
            haptics.Show(settings.Haptics.Enabled);
            autoFrameRate.Show(settings.FrameRate.Choice == null);
            batterySaver.Show(settings.FrameRate.BatterySaver);
        }

        void ShowRates()
        {
            offeredRates.Clear();
            frameRate.ClearOptions();
            foreach (var rate in TokenGraphics.FrameRateOptions)
            {
                if (!settings.FrameRate.IsOffered(rate)) continue;
                offeredRates.Add(rate);
                frameRate.options.Add(new TMP_Dropdown.OptionData(rate.ToString(CultureInfo.InvariantCulture)));
            }
            frameRate.SetValueWithoutNotify(Mathf.Max(0, offeredRates.IndexOf(settings.FrameRate.Chosen)));
            frameRate.RefreshShownValue();
        }

        void ShowMessage()
        {
            message.enabled = messageKey != null;
            message.text = messageKey == null ? string.Empty : strings.Text(messageKey);
        }
    }
}
