using System;
using System.Collections.Generic;
using HypeDomino.Core.Analytics;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Game.Privacy;
using HypeDomino.Game.Push;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Settings;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ConsentSettingsViewTests
    {
        const string PolicyVersion = "2026-11-01";
        const string AppVersion = "0.4.2";

        sealed class Sent
        {
            public string Method;
            public string Path;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, Path = pathAndQuery, Done = done });

            public void Answer(string path, int status, string body)
            {
                for (var i = Requests.Count - 1; i >= 0; i--)
                {
                    if (Requests[i].Path != path) continue;
                    Requests[i].Done(new HttpReply(status, body));
                    return;
                }
                Assert.Fail("no request to " + path);
            }
        }

        sealed class Auth : IAuthTransport
        {
            public readonly List<Action<HttpReply>> Refreshes = new();

            public void Post(string path, string json, Action<HttpReply> done) => Refreshes.Add(done);
        }

        sealed class Identity : IIdentityProvider
        {
            public bool Supports(AuthProvider provider) => false;

            public void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done) => throw new NotSupportedException();
        }

        sealed class Store : ISecureStore
        {
            readonly Dictionary<string, string> values = new();

            public string Read(string key) => values.TryGetValue(key, out var value) ? value : null;

            public void Write(string key, string value) => values[key] = value;

            public void Delete(string key) => values.Remove(key);
        }

        sealed class AnalyticsSender : IAnalyticsSender
        {
            public void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done) => done(true);
        }

        sealed class ErrorSender : IErrorSender
        {
            public void Send(ErrorReport report)
            {
            }
        }

        GameObject root;
        Api api;
        PrivacySettings privacy;
        PushSettings push;
        GameStrings strings;
        Toggle analytics;
        Toggle diagnostics;
        Toggle gameplay;
        Toggle promotional;
        TMP_Text message;

        [SetUp]
        public void Build()
        {
            root = new GameObject(nameof(ConsentSettingsViewTests), typeof(RectTransform));
            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            var client = new ApiClient(api, session);
            privacy = new PrivacySettings(client, new AnalyticsQueue(new AnalyticsSender(), () => 1000, 2, 4), new ErrorReports(new ErrorSender()), PrivacySettings.RandomAnalyticsId, PolicyVersion, AppVersion);
            push = new PushSettings(client, PolicyVersion, AppVersion);
            strings = new GameStrings(BuiltinStrings.Load(), LanguageChoice.Load(GameLanguage.English, Array.Empty<string>), false);
            analytics = Add<Toggle>();
            diagnostics = Add<Toggle>();
            gameplay = Add<Toggle>();
            promotional = Add<Toggle>();
            message = Add<TextMeshProUGUI>();
        }

        [TearDown]
        public void Destroy() => Object.DestroyImmediate(root);

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
            var view = root.AddComponent<ConsentSettingsView>();
            view.Configure(Row(analytics), Row(diagnostics), Row(gameplay), Row(promotional), message);
            view.Bind(privacy, push, strings);
        }

        static string Consents(bool analyticsOn, bool diagnosticsOn) =>
            "{\"analytics\":" + Json(analyticsOn) + ",\"diagnostics\":" + Json(diagnosticsOn) + "}";

        static string Pushes(bool gameplayOn, bool promotionalOn) =>
            "{\"gameplay\":" + Json(gameplayOn) + ",\"promotional\":" + Json(promotionalOn) + "}";

        static string Json(bool on) => on ? "true" : "false";

        void ShowLoaded()
        {
            Show();
            api.Answer(PrivacySettings.Path, 200, Consents(true, false));
            api.Answer(PushSettings.Path, 200, Pushes(true, false));
        }

        [Test]
        public void OpeningTheScreenLoadsBothAndShowsWhatTheServerKeeps()
        {
            Show();

            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("GET", api.Requests[0].Method);
            Assert.AreEqual("GET", api.Requests[1].Method);

            api.Answer(PrivacySettings.Path, 200, Consents(true, false));
            api.Answer(PushSettings.Path, 200, Pushes(false, true));

            Assert.IsTrue(analytics.isOn);
            Assert.IsFalse(diagnostics.isOn);
            Assert.IsFalse(gameplay.isOn);
            Assert.IsTrue(promotional.isOn);
            Assert.IsFalse(message.enabled);
        }

        [Test]
        public void TogglingSendsTheChangeAndShowsTheServerAnswer()
        {
            ShowLoaded();

            diagnostics.isOn = true;
            promotional.isOn = true;

            Assert.AreEqual("PUT", api.Requests[2].Method);
            Assert.AreEqual(PrivacySettings.Path, api.Requests[2].Path);
            Assert.AreEqual("PUT", api.Requests[3].Method);
            Assert.AreEqual(PushSettings.Path, api.Requests[3].Path);

            api.Answer(PrivacySettings.Path, 200, Consents(true, true));
            api.Answer(PushSettings.Path, 200, Pushes(true, true));

            Assert.IsTrue(privacy.Diagnostics);
            Assert.IsTrue(push.Promotional);
            Assert.IsTrue(diagnostics.isOn);
            Assert.IsTrue(promotional.isOn);
        }

        [Test]
        public void AFailedChangeGoesBackToTheKeptValueAndSaysWhy()
        {
            ShowLoaded();

            analytics.isOn = false;
            api.Answer(PrivacySettings.Path, HttpReply.NoResponse, null);

            Assert.IsTrue(analytics.isOn);
            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(PrivacySettingsKeys.Offline), message.text);
        }

        [Test]
        public void AToggleTappedBeforeTheLoadSendsNothingAndStaysAtTheKeptValue()
        {
            Show();

            gameplay.isOn = true;

            Assert.AreEqual(2, api.Requests.Count);
            Assert.IsFalse(gameplay.isOn);
        }
    }
}
