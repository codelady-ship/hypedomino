using System;
using System.Collections.Generic;
using HypeDomino.Game.Shop;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.Net.Integrity;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Localization;
using Hypedomino.V1;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ShopScreenViewTests
    {
        const ulong DailyEndsUnixMs = 1763164800000;
        const string ShopJson = @"{""version"":12,""sections"":[
{""section"":""featured"",""offers"":[{""id"":""offer.starter"",""kind"":""bundle"",""name_key"":""shop.offer.starter"",""product_id"":""offer_starter"",""crystals"":300,""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]},
{""section"":""daily_rotation"",""ends_at"":""2025-11-15T00:00:00Z"",""offers"":[{""id"":""offer.avatar_comet_fox"",""kind"":""cosmetic"",""name_key"":""shop.offer.avatar_comet_fox"",""price_coins"":400,""cosmetics"":[""avatar.comet_fox""],""limit"":1,""bought"":1,""owned"":[""avatar.comet_fox""],""compensation"":{},""purchasable"":false,""reason"":""offer.limit_reached""}]},
{""section"":""weekly_rotation"",""offers"":[{""id"":""offer.table_night_market"",""kind"":""cosmetic"",""name_key"":""shop.offer.table_night_market"",""price_coins"":3000,""cosmetics"":[""table.night_market""],""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true},{""id"":""offer.table_sunrise"",""kind"":""cosmetic"",""name_key"":""shop.offer.table_sunrise"",""price_crystals"":50,""cosmetics"":[""table.sunrise""],""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]}]}";

        const string WalletJson = "{\"coins\":3200,\"crystals\":40}";
        const string TableBoughtJson = "{\"offer\":\"offer.table_night_market\",\"currency\":\"coin\",\"balance\":200,\"grant\":{\"cosmetics\":[\"table.night_market\"]},\"repeated\":false}";

        sealed class Sent
        {
            public string PathAndQuery;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { PathAndQuery = pathAndQuery, Done = done });

            public void Answer(int status, string body) => Requests[Requests.Count - 1].Done(new HttpReply(status, body));
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

        static void NoProof(IntegrityPurpose purpose, Action<IntegrityProof> done) => done(null);

        GameObject root;
        Button[] categories;
        TMP_Text[] categoryTitles;
        TMP_Text sectionTimeLeft;
        ScrollRect scroll;
        TMP_Text coins;
        TMP_Text crystals;
        TMP_Text message;
        GameObject preview;
        TMP_Text previewTitle;
        TMP_Text previewPrice;
        TMP_Text previewCurrency;
        TMP_Text previewTimeLeft;
        Button confirm;
        Button closePreview;
        Api api;
        ShopCheckout checkout;
        ServerClock clock;
        GameStrings strings;
        ShopScreenView view;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Shop", typeof(RectTransform), typeof(Canvas));
            categories = new Button[4];
            categoryTitles = new TMP_Text[4];
            for (var i = 0; i < categories.Length; i++)
            {
                categories[i] = NewButton("Category", root.transform);
                categoryTitles[i] = NewText("Title", categories[i].transform);
            }
            sectionTimeLeft = NewText("Section time left", root.transform);
            coins = NewText("Coins", root.transform);
            crystals = NewText("Crystals", root.transform);
            message = NewText("Message", root.transform);
            preview = new GameObject("Preview", typeof(RectTransform));
            preview.transform.SetParent(root.transform, false);
            previewTitle = NewText("Title", preview.transform);
            previewPrice = NewText("Price", preview.transform);
            previewCurrency = NewText("Currency", preview.transform);
            previewTimeLeft = NewText("Time left", preview.transform);
            confirm = NewButton("Confirm", preview.transform);
            closePreview = NewButton("Close", preview.transform);
            api = new Api();
            var client = SignedInClient(api);
            checkout = new ShopCheckout(new ShopBuyer(client), new ShopPurchase(client, new NoStoreBilling(), NoProof));
            clock = new ServerClock();
            strings = new GameStrings(BuiltinStrings.Load(), LanguageChoice.Load(GameLanguage.English, () => Array.Empty<string>()), true);
            view = root.AddComponent<ShopScreenView>();
            view.Configure(categories, categoryTitles, sectionTimeLeft, NewList(), NewRowTemplate(), coins, crystals, message);
            view.ConfigurePreview(preview, previewTitle, previewPrice, previewCurrency, previewTimeLeft, confirm, closePreview);
            view.Bind(new ShopLoader(client), checkout, clock, strings);
        }

        [TearDown]
        public void Destroy() => Object.DestroyImmediate(root);

        static ApiClient SignedInClient(Api transport)
        {
            var auth = new Auth();
            var store = new Store();
            var session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            return new ApiClient(transport, session);
        }

        static Button NewButton(string name, Transform parent)
        {
            var button = new GameObject(name, typeof(RectTransform), typeof(Image)).AddComponent<Button>();
            button.transform.SetParent(parent, false);
            return button;
        }

        static TMP_Text NewText(string name, Transform parent)
        {
            var text = new GameObject(name, typeof(RectTransform)).AddComponent<TextMeshProUGUI>();
            text.transform.SetParent(parent, false);
            return text;
        }

        RecyclingList NewList()
        {
            var holder = new GameObject("Offers", typeof(RectTransform), typeof(ScrollRect));
            holder.transform.SetParent(root.transform, false);
            ((RectTransform)holder.transform).sizeDelta = new Vector2(600f, 800f);
            var viewport = Stretched("Viewport", holder.transform);
            var content = Stretched("Content", viewport);
            scroll = holder.GetComponent<ScrollRect>();
            scroll.viewport = viewport;
            scroll.content = content;
            scroll.horizontal = false;
            return holder.AddComponent<RecyclingList>();
        }

        static RectTransform Stretched(string name, Transform parent)
        {
            var rect = new GameObject(name, typeof(RectTransform)).GetComponent<RectTransform>();
            rect.SetParent(parent, false);
            rect.anchorMin = Vector2.zero;
            rect.anchorMax = Vector2.one;
            rect.sizeDelta = Vector2.zero;
            return rect;
        }

        ShopOfferRow NewRowTemplate()
        {
            var row = new GameObject("Offer", typeof(RectTransform));
            row.transform.SetParent(root.transform, false);
            ((RectTransform)row.transform).sizeDelta = new Vector2(600f, 120f);
            var template = row.AddComponent<ShopOfferRow>();
            template.Configure(NewText("Title", row.transform), NewText("Price", row.transform), NewText("Currency", row.transform), NewButton("Open", row.transform));
            row.SetActive(false);
            return template;
        }

        void AnswerShop()
        {
            api.Answer(200, ShopJson);
            api.Answer(200, WalletJson);
        }

        List<ShopOfferRow> ShownRows()
        {
            var rows = new List<ShopOfferRow>();
            foreach (Transform row in scroll.content)
                if (row.gameObject.activeSelf) rows.Add(row.GetComponent<ShopOfferRow>());
            return rows;
        }

        static TMP_Text Child(Component row, string name) => row.transform.Find(name).GetComponent<TMP_Text>();

        void OpenRow(int index) => ShownRows()[index].transform.Find("Open").GetComponent<Button>().onClick.Invoke();

        [Test]
        public void TheShowcaseShowsTheServerSectionsTheirOffersAndTheWallet()
        {
            Assert.AreEqual(ShopShowcase.Path, api.Requests[0].PathAndQuery);
            Assert.IsFalse(coins.enabled);

            AnswerShop();

            Assert.AreEqual(ShopLoader.WalletPath, api.Requests[1].PathAndQuery);
            Assert.IsTrue(categories[0].gameObject.activeSelf);
            Assert.IsTrue(categories[2].gameObject.activeSelf);
            Assert.IsFalse(categories[3].gameObject.activeSelf);
            Assert.AreEqual(strings.Text(ShopSectionKeys.Featured), categoryTitles[0].text);
            Assert.AreEqual(strings.Text(ShopSectionKeys.WeeklyRotation), categoryTitles[2].text);
            Assert.IsFalse(categories[0].interactable);
            Assert.AreEqual(MessageFormat.Number(3200, strings.Language), coins.text);
            Assert.AreEqual(MessageFormat.Number(40, strings.Language), crystals.text);
            Assert.AreEqual(1, ShownRows().Count);
            Assert.AreEqual(strings.Text("shop.offer.starter"), Child(ShownRows()[0], "Title").text);
            Assert.IsFalse(Child(ShownRows()[0], "Price").enabled);
            Assert.IsFalse(preview.activeSelf);
            Assert.IsFalse(message.enabled);
        }

        [Test]
        public void ACategoryShowsItsOwnOffersWithTheServerPrices()
        {
            AnswerShop();

            categories[2].onClick.Invoke();

            var rows = ShownRows();
            Assert.AreEqual(2, rows.Count);
            Assert.IsTrue(categories[0].interactable);
            Assert.IsFalse(categories[2].interactable);
            Assert.AreEqual(MessageFormat.Number(3000, strings.Language), Child(rows[0], "Price").text);
            Assert.AreEqual(strings.Text("currency.coin.name"), Child(rows[0], "Currency").text);
            Assert.AreEqual(MessageFormat.Number(50, strings.Language), Child(rows[1], "Price").text);
            Assert.AreEqual(strings.Text("currency.crystal.name"), Child(rows[1], "Currency").text);
        }

        [Test]
        public void ConfirmingAPreviewBuysOnceShowsTheResultAndReloadsTheShop()
        {
            AnswerShop();
            categories[2].onClick.Invoke();

            OpenRow(0);
            Assert.IsTrue(preview.activeSelf);
            Assert.AreEqual(strings.Text("shop.offer.table_night_market"), previewTitle.text);
            Assert.AreEqual(MessageFormat.Number(3000, strings.Language), previewPrice.text);
            Assert.IsTrue(confirm.interactable);

            confirm.onClick.Invoke();
            Assert.AreEqual(3, api.Requests.Count);
            Assert.AreEqual(ShopBuyer.Path, api.Requests[2].PathAndQuery);
            Assert.IsFalse(confirm.interactable);
            Assert.IsFalse(closePreview.interactable);

            confirm.onClick.Invoke();
            Assert.AreEqual(3, api.Requests.Count);

            api.Answer(200, TableBoughtJson);
            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(ShopBuyKeys.Bought), message.text);
            Assert.AreEqual(4, api.Requests.Count);
            Assert.AreEqual(ShopShowcase.Path, api.Requests[3].PathAndQuery);

            closePreview.onClick.Invoke();
            Assert.IsFalse(preview.activeSelf);
            Assert.IsFalse(message.enabled);
        }

        [Test]
        public void AnInsufficientBalanceShowsTheServerRefusalAndDoesNotReload()
        {
            AnswerShop();
            categories[2].onClick.Invoke();
            OpenRow(0);
            confirm.onClick.Invoke();

            api.Answer(402, "{\"error\":\"wallet.insufficient\"}");

            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(ShopBuyKeys.Insufficient), message.text);
            Assert.AreEqual(3, api.Requests.Count);
            Assert.IsTrue(preview.activeSelf);
        }

        [Test]
        public void AnOfferTheServerRefusesShowsWhyAndCannotBeConfirmed()
        {
            AnswerShop();
            categories[1].onClick.Invoke();

            OpenRow(0);

            Assert.IsFalse(confirm.interactable);
            Assert.AreEqual(strings.Text(ShopBuyKeys.LimitReached), message.text);
            confirm.onClick.Invoke();
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void AFailedLoadShowsItsMessage()
        {
            api.Answer(HttpReply.NoResponse, null);

            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(ShopLoadKeys.Offline), message.text);
            Assert.IsFalse(categories[0].gameObject.activeSelf);
            Assert.AreEqual(0, ShownRows().Count);
        }

        [Test]
        public void TheCountdownFollowsTheServerClockOnly()
        {
            AnswerShop();
            categories[1].onClick.Invoke();

            view.Tick(0);
            Assert.IsFalse(sectionTimeLeft.enabled);

            clock.Sync(DailyEndsUnixMs - 3_725_000, 0, 0);
            view.Tick(0);
            Assert.IsTrue(sectionTimeLeft.enabled);
            Assert.AreEqual(Duration(TimeSpan.FromSeconds(3725)), sectionTimeLeft.text);

            OpenRow(0);
            view.Tick(5);
            Assert.IsTrue(previewTimeLeft.enabled);
            Assert.AreEqual(Duration(TimeSpan.FromSeconds(3720)), previewTimeLeft.text);

            view.Tick(3725);
            Assert.IsFalse(sectionTimeLeft.enabled);

            categories[2].onClick.Invoke();
            view.Tick(0);
            Assert.IsFalse(sectionTimeLeft.enabled);
        }

        string Duration(TimeSpan span) => strings.Text(MessageFormat.DurationKey(span), new Dictionary<string, object>
        {
            ["days"] = (long)span.Days,
            ["hours"] = (long)span.Hours,
            ["minutes"] = (long)span.Minutes,
            ["seconds"] = (long)span.Seconds,
        });
    }
}
