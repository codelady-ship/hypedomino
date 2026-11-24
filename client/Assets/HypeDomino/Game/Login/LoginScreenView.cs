using System.Collections.Generic;
using HypeDomino.Game.Privacy;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game.Login
{
    public sealed class LoginScreenView : MonoBehaviour
    {
        [SerializeField] Button google;
        [SerializeField] Button apple;
        [SerializeField] TMP_Text message;
        [SerializeField] GameObject deletionPending;
        [SerializeField] Button cancelDeletion;

        readonly Dictionary<string, object> messageArgs = new();
        LoginScreen screen;
        GameStrings strings;

        public void Configure(Button googleButton, Button appleButton, TMP_Text messageText, GameObject deletionPendingPanel, Button cancelDeletionButton)
        {
            google = googleButton;
            apple = appleButton;
            message = messageText;
            deletionPending = deletionPendingPanel;
            cancelDeletion = cancelDeletionButton;
        }

        public void Bind(LoginScreen login, GameStrings texts)
        {
            Unbind();
            screen = login;
            strings = texts;
            screen.Changed += Refresh;
            strings.Changed += Refresh;
            google.onClick.AddListener(SignInWithGoogle);
            apple.onClick.AddListener(SignInWithApple);
            cancelDeletion.onClick.AddListener(CancelDeletion);
            Refresh();
        }

        void Start()
        {
            if (screen == null && NetBoot.Login != null) Bind(NetBoot.Login, LanguageBoot.Strings);
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (screen == null) return;
            screen.Changed -= Refresh;
            strings.Changed -= Refresh;
            google.onClick.RemoveListener(SignInWithGoogle);
            apple.onClick.RemoveListener(SignInWithApple);
            cancelDeletion.onClick.RemoveListener(CancelDeletion);
            screen = null;
            strings = null;
        }

        void SignInWithGoogle() => screen.TrySignIn(AuthProvider.Google);

        void SignInWithApple() => screen.TrySignIn(AuthProvider.Apple);

        void CancelDeletion() => screen.TryCancelDeletion();

        void Refresh()
        {
            ShowOption(google, AuthProvider.Google);
            ShowOption(apple, AuthProvider.Apple);
            deletionPending.SetActive(screen.DeletionPending);
            cancelDeletion.interactable = !screen.Busy;
            ShowMessage();
        }

        void ShowOption(Button button, AuthProvider provider)
        {
            button.gameObject.SetActive(screen.Offers(provider));
            button.interactable = !screen.Busy;
        }

        void ShowMessage()
        {
            var key = screen.MessageKey;
            message.enabled = key != null;
            if (key == null) return;
            messageArgs[AccountDeletionKeys.DeleteAtArgument] = screen.DeleteAt;
            message.text = strings.Text(key, messageArgs);
        }
    }
}
