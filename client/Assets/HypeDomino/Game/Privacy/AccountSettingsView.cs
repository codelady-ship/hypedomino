using System;
using System.Collections.Generic;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game.Privacy
{
    public sealed class AccountSettingsView : MonoBehaviour
    {
        [SerializeField] Button exportData;
        [SerializeField] Button deleteAccount;
        [SerializeField] GameObject deletionWarning;
        [SerializeField] Button confirmWithGoogle;
        [SerializeField] Button confirmWithApple;
        [SerializeField] Button keepAccount;
        [SerializeField] GameObject deletionPending;
        [SerializeField] TMP_Text message;

        readonly Dictionary<string, object> messageArgs = new();
        DataExport export;
        AccountDeletion deletion;
        AuthSession auth;
        IReadOnlyList<AuthProvider> options;
        GameStrings strings;
        Action<string> openLink;
        Action signedOut;
        bool warningOpen;
        bool pending;
        DateTimeOffset deleteAt;
        string messageKey;

        public void Configure(Button exportButton, Button deleteButton, GameObject warningPanel, Button googleConfirmButton, Button appleConfirmButton,
            Button keepButton, GameObject pendingPanel, TMP_Text messageLine)
        {
            Unbind();
            exportData = exportButton;
            deleteAccount = deleteButton;
            deletionWarning = warningPanel;
            confirmWithGoogle = googleConfirmButton;
            confirmWithApple = appleConfirmButton;
            keepAccount = keepButton;
            deletionPending = pendingPanel;
            message = messageLine;
        }

        public void Bind(DataExport dataExport, AccountDeletion accountDeletion, AuthSession session, IReadOnlyList<AuthProvider> signInOptions,
            GameStrings texts, Action<string> openExportLink, Action afterSignOut)
        {
            Unbind();
            export = dataExport;
            deletion = accountDeletion;
            auth = session;
            options = signInOptions;
            strings = texts;
            openLink = openExportLink;
            signedOut = afterSignOut;
            warningOpen = false;
            pending = false;
            messageKey = null;
            strings.Changed += Refresh;
            exportData.onClick.AddListener(OnExport);
            deleteAccount.onClick.AddListener(OnDelete);
            confirmWithGoogle.onClick.AddListener(ConfirmWithGoogle);
            confirmWithApple.onClick.AddListener(ConfirmWithApple);
            keepAccount.onClick.AddListener(OnKeep);
            Refresh();
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (export == null) return;
            strings.Changed -= Refresh;
            exportData.onClick.RemoveListener(OnExport);
            deleteAccount.onClick.RemoveListener(OnDelete);
            confirmWithGoogle.onClick.RemoveListener(ConfirmWithGoogle);
            confirmWithApple.onClick.RemoveListener(ConfirmWithApple);
            keepAccount.onClick.RemoveListener(OnKeep);
            export = null;
            deletion = null;
            auth = null;
            strings = null;
        }

        void OnExport()
        {
            if (export.Request(OnExported)) Refresh();
        }

        void OnExported(DataExportResult result)
        {
            if (export == null) return;
            if (result.Outcome == DataExportOutcome.Ready) openLink(result.Url);
            messageKey = result.MessageKey;
            Refresh();
        }

        void OnDelete()
        {
            warningOpen = true;
            messageKey = null;
            Refresh();
        }

        void OnKeep()
        {
            warningOpen = false;
            Refresh();
        }

        void ConfirmWithGoogle() => Confirm(AuthProvider.Google);

        void ConfirmWithApple() => Confirm(AuthProvider.Apple);

        void Confirm(AuthProvider provider)
        {
            if (deletion.Request(provider, OnDeletionAnswered)) Refresh();
        }

        void OnDeletionAnswered(AccountDeletionResult result)
        {
            if (deletion == null) return;
            messageKey = result.MessageKey;
            if (result.Outcome == AccountDeletionOutcome.Pending)
            {
                pending = true;
                warningOpen = false;
                deleteAt = result.DeleteAt;
                auth.SignOut(signedOut);
            }
            Refresh();
        }

        void Refresh()
        {
            exportData.interactable = !export.Busy;
            deleteAccount.interactable = !pending && !deletion.Busy;
            deletionWarning.SetActive(warningOpen);
            ShowConfirm(confirmWithGoogle, AuthProvider.Google);
            ShowConfirm(confirmWithApple, AuthProvider.Apple);
            keepAccount.interactable = !deletion.Busy;
            deletionPending.SetActive(pending);
            ShowMessage();
        }

        void ShowConfirm(Button button, AuthProvider provider)
        {
            button.gameObject.SetActive(Offers(provider));
            button.interactable = !deletion.Busy;
        }

        bool Offers(AuthProvider provider)
        {
            for (var i = 0; i < options.Count; i++)
                if (options[i] == provider) return true;
            return false;
        }

        void ShowMessage()
        {
            message.enabled = messageKey != null;
            if (messageKey == null) return;
            messageArgs[AccountDeletionKeys.DeleteAtArgument] = deleteAt;
            message.text = strings.Text(messageKey, messageArgs);
        }
    }
}
