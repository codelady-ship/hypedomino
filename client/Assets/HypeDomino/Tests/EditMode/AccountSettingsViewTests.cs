using System;
using System.Collections.Generic;
using HypeDomino.Game.Privacy;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AccountSettingsViewTests
    {
        const string NonceJson = "{\"nonce\":\"nonce-1\"}";
        const string DeleteAt = "2026-12-21T09:00:00.5Z";
        const long DeleteAtUnixMs = 1797843600500;
        const string Link = "https://exports.example.test/exports/abc.json?sig=1";

        sealed class Sent
        {
            public string Path;
            public Action<HttpReply> Done;
        }

        sealed class Transport : IAuthTransport
        {
            public readonly List<Sent> Requests = new();

            public void Post(string path, string json, Action<HttpReply> done) => Requests.Add(new Sent { Path = path, Done = done });

            public void Answer(int status, string body) => Requests[Requests.Count - 1].Done(new HttpReply(status, body));
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Action<HttpReply>> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) => Requests.Add(done);
        }

        sealed class Identity : IIdentityProvider
        {
            Action<IdTokenResult> pending;

            public bool Supports(AuthProvider provider) => true;

            public void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done) => pending = done;

            public void Answer(IdTokenOutcome outcome) => pending(new IdTokenResult(outcome, outcome == IdTokenOutcome.Token ? "id-token" : null));
        }

        sealed class Store : ISecureStore
        {
            readonly Dictionary<string, string> values = new();

            public string Read(string key) => values.TryGetValue(key, out var value) ? value : null;

            public void Write(string key, string value) => values[key] = value;

            public void Delete(string key) => values.Remove(key);
        }

        GameObject root;
        Transport transport;
        Identity identity;
        Api api;
        AuthSession session;
        DataExport export;
        AccountDeletion deletion;
        GameStrings strings;
        Button exportData;
        Button deleteAccount;
        GameObject warning;
        Button confirmWithGoogle;
        Button confirmWithApple;
        Button keepAccount;
        GameObject pending;
        TMP_Text message;
        List<string> opened;
        int signedOut;

        [SetUp]
        public void Build()
        {
            root = new GameObject(nameof(AccountSettingsViewTests), typeof(RectTransform));
            transport = new Transport();
            identity = new Identity();
            api = new Api();
            var store = new Store();
            session = new AuthSession(transport, identity, store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            transport.Answer(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}");
            export = new DataExport(new ApiClient(api, session));
            deletion = new AccountDeletion(transport, identity);
            strings = new GameStrings(BuiltinStrings.Load(), LanguageChoice.Load(GameLanguage.English, Array.Empty<string>), false);
            exportData = NewButton("Export", root);
            deleteAccount = NewButton("Delete", root);
            warning = new GameObject("Warning", typeof(RectTransform));
            warning.transform.SetParent(root.transform, false);
            confirmWithGoogle = NewButton("Google", warning);
            confirmWithApple = NewButton("Apple", warning);
            keepAccount = NewButton("Keep", warning);
            pending = new GameObject("Pending", typeof(RectTransform));
            pending.transform.SetParent(root.transform, false);
            message = new GameObject("Message", typeof(RectTransform)).AddComponent<TextMeshProUGUI>();
            message.transform.SetParent(root.transform, false);
            opened = new List<string>();
            signedOut = 0;
        }

        [TearDown]
        public void Destroy() => Object.DestroyImmediate(root);

        static Button NewButton(string name, GameObject parent)
        {
            var button = new GameObject(name, typeof(RectTransform), typeof(Image)).AddComponent<Button>();
            button.transform.SetParent(parent.transform, false);
            return button;
        }

        void Show(params AuthProvider[] options)
        {
            var view = root.AddComponent<AccountSettingsView>();
            view.Configure(exportData, deleteAccount, warning, confirmWithGoogle, confirmWithApple, keepAccount, pending, message);
            view.Bind(export, deletion, session, options, strings, opened.Add, () => signedOut++);
        }

        void ConfirmAnswering(Button confirm, IdTokenOutcome token, int status, string body)
        {
            confirm.onClick.Invoke();
            transport.Answer(200, NonceJson);
            identity.Answer(token);
            if (token == IdTokenOutcome.Token) transport.Answer(status, body);
        }

        [Test]
        public void TheScreenStartsWithNoWarningNoPendingStateAndNoMessage()
        {
            Show(AuthProvider.Google);

            Assert.IsFalse(warning.activeSelf);
            Assert.IsFalse(pending.activeSelf);
            Assert.IsFalse(message.enabled);
            Assert.IsTrue(exportData.interactable);
            Assert.IsTrue(deleteAccount.interactable);
        }

        [Test]
        public void AReadyExportOpensItsLinkOnce()
        {
            Show(AuthProvider.Google);

            exportData.onClick.Invoke();
            Assert.IsFalse(exportData.interactable);
            exportData.onClick.Invoke();
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0](new HttpReply(201, "{\"url\":\"" + Link + "\",\"expires_at\":\"" + DeleteAt + "\"}"));

            CollectionAssert.AreEqual(new[] { Link }, opened);
            Assert.IsTrue(exportData.interactable);
            Assert.IsFalse(message.enabled);
        }

        [Test]
        public void ARefusedExportOpensNothingAndSaysWhy()
        {
            Show(AuthProvider.Google);

            exportData.onClick.Invoke();
            api.Requests[0](new HttpReply(429, "{\"error\":\"export_cooldown\"}"));

            Assert.IsEmpty(opened);
            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(DataExportKeys.Cooldown), message.text);
        }

        [Test]
        public void DeleteOpensTheWarningWithTheOfferedSignInsAndKeepClosesIt()
        {
            Show(AuthProvider.Google);

            deleteAccount.onClick.Invoke();
            Assert.IsTrue(warning.activeSelf);
            Assert.IsTrue(confirmWithGoogle.gameObject.activeSelf);
            Assert.IsFalse(confirmWithApple.gameObject.activeSelf);
            Assert.AreEqual(1, transport.Requests.Count);

            keepAccount.onClick.Invoke();
            Assert.IsFalse(warning.activeSelf);
            Assert.AreEqual(1, transport.Requests.Count);
        }

        [Test]
        public void AnAcceptedDeletionShowsThePendingStateAndSignsOutLocally()
        {
            Show(AuthProvider.Google, AuthProvider.Apple);
            deleteAccount.onClick.Invoke();

            ConfirmAnswering(confirmWithApple, IdTokenOutcome.Token, 202, "{\"delete_at\":\"" + DeleteAt + "\"}");

            Assert.AreEqual(AccountDeletion.RequestPath, transport.Requests[2].Path);
            Assert.IsFalse(warning.activeSelf);
            Assert.IsTrue(pending.activeSelf);
            Assert.IsFalse(deleteAccount.interactable);
            Assert.IsFalse(session.SignedIn);
            Assert.AreEqual(AuthSession.LogoutPath, transport.Requests[3].Path);
            Assert.AreEqual(0, signedOut);
            transport.Answer(204, null);
            Assert.AreEqual(1, signedOut);
            var args = new Dictionary<string, object> { [AccountDeletionKeys.DeleteAtArgument] = DateTimeOffset.FromUnixTimeMilliseconds(DeleteAtUnixMs) };
            Assert.AreEqual(strings.Text(AccountDeletionKeys.Pending, args), message.text);
        }

        [Test]
        public void ARefusedDeletionKeepsTheWarningOpenAndSaysWhy()
        {
            Show(AuthProvider.Google);
            deleteAccount.onClick.Invoke();

            ConfirmAnswering(confirmWithGoogle, IdTokenOutcome.Token, 409, "{\"error\":\"in_match\"}");

            Assert.IsTrue(warning.activeSelf);
            Assert.IsFalse(pending.activeSelf);
            Assert.IsTrue(session.SignedIn);
            Assert.AreEqual(strings.Text(AccountDeletionKeys.InMatch), message.text);
        }

        [Test]
        public void AClosedSignInSheetSaysNothing()
        {
            Show(AuthProvider.Google);
            deleteAccount.onClick.Invoke();

            ConfirmAnswering(confirmWithGoogle, IdTokenOutcome.Cancelled, 0, null);

            Assert.IsTrue(warning.activeSelf);
            Assert.IsFalse(message.enabled);
            Assert.IsTrue(confirmWithGoogle.interactable);
        }
    }
}
