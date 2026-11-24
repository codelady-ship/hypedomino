using HypeDomino.Game.Push;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Settings;
using TMPro;
using UnityEngine;

namespace HypeDomino.Game.Privacy
{
    public sealed class ConsentSettingsView : MonoBehaviour
    {
        [SerializeField] SettingToggleRow analytics;
        [SerializeField] SettingToggleRow diagnostics;
        [SerializeField] SettingToggleRow gameplayPush;
        [SerializeField] SettingToggleRow promotionalPush;
        [SerializeField] TMP_Text message;

        PrivacySettings privacy;
        PushSettings push;
        GameStrings strings;
        string messageKey;

        public void Configure(SettingToggleRow analyticsRow, SettingToggleRow diagnosticsRow, SettingToggleRow gameplayPushRow,
            SettingToggleRow promotionalPushRow, TMP_Text messageLine)
        {
            Unbind();
            analytics = analyticsRow;
            diagnostics = diagnosticsRow;
            gameplayPush = gameplayPushRow;
            promotionalPush = promotionalPushRow;
            message = messageLine;
        }

        public void Bind(PrivacySettings privacySettings, PushSettings pushSettings, GameStrings texts)
        {
            Unbind();
            privacy = privacySettings;
            push = pushSettings;
            strings = texts;
            messageKey = null;
            analytics.Bind(privacy.Analytics, OnAnalytics);
            diagnostics.Bind(privacy.Diagnostics, OnDiagnostics);
            gameplayPush.Bind(push.Gameplay, OnGameplayPush);
            promotionalPush.Bind(push.Promotional, OnPromotionalPush);
            privacy.Changed += ShowChoices;
            push.Changed += ShowChoices;
            strings.Changed += ShowMessage;
            ShowMessage();
            if (!privacy.Loaded) privacy.Load(OnPrivacyAnswered);
            if (!push.Loaded) push.Load(OnPushAnswered);
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (privacy == null) return;
            privacy.Changed -= ShowChoices;
            push.Changed -= ShowChoices;
            strings.Changed -= ShowMessage;
            privacy = null;
            push = null;
            strings = null;
        }

        void OnAnalytics(bool on)
        {
            if (!privacy.SetAnalytics(on, OnPrivacyAnswered)) ShowChoices();
        }

        void OnDiagnostics(bool on)
        {
            if (!privacy.SetDiagnostics(on, OnPrivacyAnswered)) ShowChoices();
        }

        void OnGameplayPush(bool on)
        {
            if (!push.SetGameplay(on, OnPushAnswered)) ShowChoices();
        }

        void OnPromotionalPush(bool on)
        {
            if (!push.SetPromotional(on, OnPushAnswered)) ShowChoices();
        }

        void OnPrivacyAnswered(PrivacySettingsResult result) => Answered(result.MessageKey);

        void OnPushAnswered(PushSettingsResult result) => Answered(result.MessageKey);

        void Answered(string key)
        {
            if (privacy == null) return;
            messageKey = key;
            ShowChoices();
            ShowMessage();
        }

        void ShowChoices()
        {
            analytics.Show(privacy.Analytics);
            diagnostics.Show(privacy.Diagnostics);
            gameplayPush.Show(push.Gameplay);
            promotionalPush.Show(push.Promotional);
        }

        void ShowMessage()
        {
            message.enabled = messageKey != null;
            message.text = messageKey == null ? string.Empty : strings.Text(messageKey);
        }
    }
}
