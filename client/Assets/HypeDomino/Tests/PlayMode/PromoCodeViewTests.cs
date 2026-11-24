using System;
using System.Collections.Generic;
using HypeDomino.Game.Promo;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class PromoCodeViewTests
    {
        sealed class Api : IApiTransport
        {
            public readonly List<Action<HttpReply>> Replies = new();
            public readonly List<string> Bodies = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done)
            {
                Bodies.Add(body);
                Replies.Add(done);
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

        static readonly string[] NoLocales = { };

        GameObject root;
        TMP_InputField field;
        Button paste;
        Button redeem;
        TMP_Text message;
        Api api;
        LanguageChoice choice;
        PromoCodeEntry entry;

        [SetUp]
        public void BuildView()
        {
            root = new GameObject("Promo screen", typeof(RectTransform), typeof(Canvas));
            field = new GameObject("Code", typeof(RectTransform)).AddComponent<TMP_InputField>();
            field.transform.SetParent(root.transform, false);
            paste = NewButton("Paste");
            redeem = NewButton("Redeem");
            message = new GameObject("Message", typeof(RectTransform)).AddComponent<TextMeshProUGUI>();
            message.transform.SetParent(root.transform, false);
            var view = new GameObject("View", typeof(RectTransform)).AddComponent<PromoCodeView>();
            view.transform.SetParent(root.transform, false);
            view.Configure(field, paste, redeem, message);

            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            entry = new PromoCodeEntry(new PromoRedeemer(new ApiClient(api, session)));
            choice = LanguageChoice.Load("az", () => NoLocales);
            view.Bind(entry, new GameStrings(BuiltinStrings.Load(), choice, false));
        }

        [TearDown]
        public void DestroyView() => UnityEngine.Object.Destroy(root);

        Button NewButton(string name)
        {
            var button = new GameObject(name, typeof(RectTransform), typeof(Image)).AddComponent<Button>();
            button.transform.SetParent(root.transform, false);
            return button;
        }

        [Test]
        public void AnEmptyScreenOffersNoRedeemAndNoMessage()
        {
            Assert.AreEqual(string.Empty, field.text);
            Assert.IsFalse(redeem.interactable);
            Assert.IsTrue(paste.interactable);
            Assert.IsFalse(message.enabled);
        }

        [Test]
        public void TypingAWellFormedCodeOffersRedeem()
        {
            field.text = "abc";
            Assert.AreEqual("abc", entry.Text);
            Assert.IsFalse(redeem.interactable);

            field.text = "abcd-2345";
            Assert.AreEqual("abcd-2345", entry.Text);
            Assert.IsTrue(redeem.interactable);

            field.text = "abcd-1234";
            Assert.IsFalse(redeem.interactable);
        }

        [Test]
        public void PasteFillsTheFieldFromTheClipboard()
        {
            GUIUtility.systemCopyBuffer = "wxyz-2345";
            if (GUIUtility.systemCopyBuffer != "wxyz-2345") Assert.Ignore("No clipboard on this machine");

            paste.onClick.Invoke();

            Assert.AreEqual("wxyz-2345", entry.Text);
            Assert.AreEqual("wxyz-2345", field.text);
            Assert.IsTrue(redeem.interactable);
        }

        [Test]
        public void RedeemSendsOnceLocksTheFieldAndShowsTheReply()
        {
            field.text = "abcd-2345";
            redeem.onClick.Invoke();
            redeem.onClick.Invoke();

            Assert.AreEqual(1, api.Bodies.Count);
            Assert.AreEqual("{\"code\":\"ABCD2345\"}", api.Bodies[0]);
            Assert.IsTrue(field.readOnly);
            Assert.IsFalse(paste.interactable);
            Assert.IsFalse(redeem.interactable);

            api.Replies[0](new HttpReply(409, "{\"error\":\"promo.expired\"}"));

            Assert.IsFalse(field.readOnly);
            Assert.IsTrue(paste.interactable);
            Assert.IsTrue(message.enabled);
            Assert.AreEqual("Bu kodun vaxtı keçib", message.text);

            field.text = "abcd-2346";
            Assert.IsFalse(message.enabled);
        }

        [Test]
        public void ACodeThatUnlocksThePremiumPassSaysSo()
        {
            field.text = "abcd-2345";
            redeem.onClick.Invoke();
            api.Replies[0](new HttpReply(200, "{\"promo\":\"promo-1\",\"grant\":{\"coins\":0},\"pass_premium\":true}"));

            Assert.AreEqual("Kod qəbul olundu. Premium Hype Pass açıldı", message.text);
        }

        [Test]
        public void AGrantWithoutThePassShowsThePlainGrant()
        {
            field.text = "abcd-2345";
            redeem.onClick.Invoke();
            api.Replies[0](new HttpReply(200, "{\"promo\":\"promo-1\",\"grant\":{\"coins\":500}}"));

            Assert.AreEqual("Kod qəbul olundu", message.text);
        }

        [Test]
        public void TheMessageFollowsTheChosenLanguage()
        {
            field.text = "abcd-2345";
            redeem.onClick.Invoke();
            api.Replies[0](new HttpReply(409, "{\"error\":\"promo.expired\"}"));

            choice.Choose(GameLanguage.English);

            Assert.AreEqual("This code has expired", message.text);
        }
    }
}
