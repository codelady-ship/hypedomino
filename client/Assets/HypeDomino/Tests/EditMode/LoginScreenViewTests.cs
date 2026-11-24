using System;
using System.Collections.Generic;
using HypeDomino.Game.Login;
using HypeDomino.Game.Privacy;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LoginScreenViewTests
    {
        const string NonceJson = "{\"nonce\":\"nonce-1\"}";

        sealed class Transport : IAuthTransport
        {
            public readonly List<string> Paths = new();
            readonly List<Action<HttpReply>> pending = new();

            public void Post(string path, string json, Action<HttpReply> done)
            {
                Paths.Add(path);
                pending.Add(done);
            }

            public void Answer(int status, string body) => pending[pending.Count - 1](new HttpReply(status, body));
        }

        sealed class Identity : IIdentityProvider
        {
            Action<IdTokenResult> pending;

            public bool Supports(AuthProvider provider) => true;

            public void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done) => pending = done;

            public void Answer() => pending(new IdTokenResult(IdTokenOutcome.Token, "id-token"));
        }

        sealed class Store : ISecureStore
        {
            readonly Dictionary<string, string> values = new();

            public string Read(string key) => values.TryGetValue(key, out var value) ? value : null;

            public void Write(string key, string value) => values[key] = value;

            public void Delete(string key) => values.Remove(key);
        }

        GameObject root;
        Button google;
        Button apple;
        TMP_Text message;
        GameObject deletionPending;
        Button cancelDeletion;
        Transport transport;
        Identity identity;
        GameStrings strings;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Login", typeof(RectTransform));
            google = NewButton("Google");
            apple = NewButton("Apple");
            message = new GameObject("Message", typeof(RectTransform)).AddComponent<TextMeshProUGUI>();
            message.transform.SetParent(root.transform, false);
            deletionPending = new GameObject("Deletion pending", typeof(RectTransform));
            deletionPending.transform.SetParent(root.transform, false);
            cancelDeletion = NewButton("Cancel deletion");
            cancelDeletion.transform.SetParent(deletionPending.transform, false);
            transport = new Transport();
            identity = new Identity();
            strings = new GameStrings(BuiltinStrings.Load(), LanguageChoice.Load(GameLanguage.English, () => Array.Empty<string>()), false);
        }

        [TearDown]
        public void Destroy() => Object.DestroyImmediate(root);

        Button NewButton(string name)
        {
            var button = new GameObject(name, typeof(RectTransform), typeof(Image)).AddComponent<Button>();
            button.transform.SetParent(root.transform, false);
            return button;
        }

        LoginScreen Show(params AuthProvider[] options)
        {
            var screen = new LoginScreen(new AuthSession(transport, identity, new Store()), new AccountDeletion(transport, identity), options, () => { });
            var view = root.AddComponent<LoginScreenView>();
            view.Configure(google, apple, message, deletionPending, cancelDeletion);
            view.Bind(screen, strings);
            return screen;
        }

        void SignInAnswering(Button option, int status, string body)
        {
            option.onClick.Invoke();
            transport.Answer(200, NonceJson);
            identity.Answer();
            transport.Answer(status, body);
        }

        [Test]
        public void OnlyTheOfferedSignInOptionsAreShownAndNothingElse()
        {
            Show(AuthProvider.Google);

            Assert.IsTrue(google.gameObject.activeSelf);
            Assert.IsFalse(apple.gameObject.activeSelf);
            Assert.IsFalse(message.enabled);
            Assert.IsFalse(deletionPending.activeSelf);
        }

        [Test]
        public void TappingAnOptionSignsInAndLocksTheOptionsUntilTheAnswer()
        {
            var screen = Show(AuthProvider.Google, AuthProvider.Apple);

            apple.onClick.Invoke();

            Assert.IsTrue(screen.Busy);
            Assert.AreEqual(AuthSession.NoncePath, transport.Paths[0]);
            Assert.IsFalse(google.interactable);
            Assert.IsFalse(apple.interactable);

            transport.Answer(503, "");

            Assert.IsTrue(google.interactable);
            Assert.IsTrue(apple.interactable);
        }

        [Test]
        public void ARefusedSignInShowsItsTextInTheChosenLanguage()
        {
            Show(AuthProvider.Google);

            SignInAnswering(google, 503, "");

            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(LoginKeys.Unavailable), message.text);
        }

        [Test]
        public void AnAccountWaitingForDeletionShowsTheDateAndCanBeRestored()
        {
            var screen = Show(AuthProvider.Google);

            SignInAnswering(google, 409, "{\"error\":\"deletion_pending\",\"delete_at\":\"2026-12-21T09:00:00Z\"}");

            Assert.IsTrue(deletionPending.activeSelf);
            Assert.IsTrue(message.enabled);
            StringAssert.DoesNotContain("{", message.text);
            StringAssert.Contains("2026", message.text);

            cancelDeletion.onClick.Invoke();

            Assert.IsTrue(screen.Busy);
            Assert.IsFalse(cancelDeletion.interactable);
            transport.Answer(200, NonceJson);
            identity.Answer();
            Assert.AreEqual(AccountDeletion.CancelPath, transport.Paths[transport.Paths.Count - 1]);
            transport.Answer(204, "");

            Assert.IsFalse(deletionPending.activeSelf);
            Assert.AreEqual(strings.Text(AccountDeletionKeys.Restored), message.text);
        }

        [Test]
        public void TheShownTextFollowsALanguageChange()
        {
            var choice = LanguageChoice.Load(GameLanguage.English, () => Array.Empty<string>());
            strings = new GameStrings(BuiltinStrings.Load(), choice, false);
            Show(AuthProvider.Google);
            SignInAnswering(google, 503, "");

            Assert.IsTrue(choice.Choose("az"));

            Assert.AreEqual(BuiltinStrings.Load().Value(LoginKeys.Unavailable, "az"), message.text);
        }
    }
}
