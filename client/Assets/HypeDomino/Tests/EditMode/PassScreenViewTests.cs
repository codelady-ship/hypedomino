using System;
using System.Collections.Generic;
using HypeDomino.Game.Pass;
using HypeDomino.Game.Shop;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.Net.Integrity;
using HypeDomino.UI.Cosmetics;
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
    public sealed class PassScreenViewTests
    {
        const ulong SeasonEndsUnixMs = 1803859200000;
        const string PassJson = @"{""season"":""season_1"",""starts_at"":""2026-12-01T00:00:00Z"",""ends_at"":""2027-03-01T00:00:00Z"",""config_version"":4,""levels"":3,""xp"":150,""level"":2,""level_xp"":100,""next_level_xp"":300,""premium"":false,""level_coin_price"":250,""level_crystal_price"":20,""tracks"":{""free"":{""1"":{""id"":""bundle.pass_season_1_free_1"",""revision"":1,""coins"":30,""cosmetics"":null,""medals"":null,""passXp"":0},""3"":{""id"":""bundle.pass_season_1_free_3"",""revision"":1,""coins"":0,""cosmetics"":[""emoji.s1_cheer""],""medals"":null,""passXp"":0}},""premium"":{""1"":{""id"":""bundle.pass_season_1_premium_1"",""revision"":1,""coins"":0,""crystals"":6,""cosmetics"":null,""medals"":null,""passXp"":0}}}}";

        const string ManifestJson = @"{""schemaVersion"":1,""manifestVersion"":7,""items"":[
{""id"":""emoji.s1_cheer"",""type"":""emoji"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/emoji/s1_cheer/static"",""sources"":[{""type"":""pass"",""ref"":""pass.season_1.free.3""}]}]}";

        const string ShopJson = @"{""version"":12,""sections"":[
{""section"":""featured"",""offers"":[{""id"":""offer.starter"",""kind"":""bundle"",""name_key"":""shop.offer.starter"",""product_id"":""offer_starter"",""crystals"":300,""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]},
{""section"":""pass"",""offers"":[{""id"":""offer.pass_premium"",""kind"":""pass"",""name_key"":""shop.offer.pass_premium"",""product_id"":""pass_premium"",""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true},{""id"":""offer.pass_premium_plus"",""kind"":""pass"",""name_key"":""shop.offer.pass_premium_plus"",""product_id"":""pass_premium_plus"",""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]}]}";

        const string WalletJson = "{\"coins\":3200,\"crystals\":40}";
        const string LevelBoughtJson = "{\"season\":\"season_1\",\"count\":1,\"currency\":\"coin\",\"price\":250,\"level\":3,\"xp\":300,\"balance\":2950,\"repeated\":false}";

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

            public void Answer(int index, int status, string body) => Requests[index].Done(new HttpReply(status, body));
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
        TMP_Text level;
        Image levelFill;
        TMP_Text timeLeft;
        ScrollRect scroll;
        TMP_Text message;
        Button buyCoinLevel;
        TMP_Text coinLevelPrice;
        Button buyCrystalLevel;
        TMP_Text crystalLevelPrice;
        ShopOfferRow[] premiumOffers;
        GameObject preview;
        TMP_Text previewTitle;
        TMP_Text previewPrice;
        TMP_Text previewCurrency;
        Button confirm;
        Button closePreview;
        Api api;
        ServerClock clock;
        GameStrings strings;
        PassScreenView view;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Pass", typeof(RectTransform), typeof(Canvas));
            level = NewText("Level", root.transform);
            levelFill = new GameObject("Fill", typeof(RectTransform)).AddComponent<Image>();
            levelFill.transform.SetParent(root.transform, false);
            timeLeft = NewText("Time left", root.transform);
            message = NewText("Message", root.transform);
            buyCoinLevel = NewButton("Buy coin level", root.transform);
            coinLevelPrice = NewText("Price", buyCoinLevel.transform);
            buyCrystalLevel = NewButton("Buy crystal level", root.transform);
            crystalLevelPrice = NewText("Price", buyCrystalLevel.transform);
            premiumOffers = new[] { NewOfferRow(), NewOfferRow(), NewOfferRow() };
            preview = new GameObject("Preview", typeof(RectTransform));
            preview.transform.SetParent(root.transform, false);
            previewTitle = NewText("Title", preview.transform);
            previewPrice = NewText("Price", preview.transform);
            previewCurrency = NewText("Currency", preview.transform);
            confirm = NewButton("Confirm", preview.transform);
            closePreview = NewButton("Close", preview.transform);
            api = new Api();
            var client = SignedInClient(api);
            var checkout = new ShopCheckout(new ShopBuyer(client), new ShopPurchase(client, new NoStoreBilling(), NoProof));
            Assert.IsTrue(CosmeticManifest.TryParse(ManifestJson, out var manifest));
            clock = new ServerClock();
            strings = new GameStrings(BuiltinStrings.Load(), LanguageChoice.Load(GameLanguage.English, () => Array.Empty<string>()), true);
            view = root.AddComponent<PassScreenView>();
            view.Configure(level, levelFill, timeLeft, NewList(), NewRowTemplate(), message);
            view.ConfigureLevelPurchase(buyCoinLevel, coinLevelPrice, buyCrystalLevel, crystalLevelPrice);
            view.ConfigurePremium(premiumOffers, preview, previewTitle, previewPrice, previewCurrency, confirm, closePreview);
            view.Bind(new PassLoader(client), new PassLevelBuyer(client), new ShopLoader(client), checkout, new ActiveCosmeticManifest(manifest), clock, strings);
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

        static GameObject NewMark(string name, Transform parent)
        {
            var mark = new GameObject(name, typeof(RectTransform));
            mark.transform.SetParent(parent, false);
            return mark;
        }

        RecyclingList NewList()
        {
            var holder = new GameObject("Levels", typeof(RectTransform), typeof(ScrollRect));
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

        PassLevelRow NewRowTemplate()
        {
            var row = new GameObject("Level", typeof(RectTransform));
            row.transform.SetParent(root.transform, false);
            ((RectTransform)row.transform).sizeDelta = new Vector2(600f, 120f);
            var template = row.AddComponent<PassLevelRow>();
            template.Configure(NewText("Number", row.transform), NewCell("Free", row.transform), NewCell("Premium", row.transform));
            row.SetActive(false);
            return template;
        }

        static PassCellView NewCell(string name, Transform parent)
        {
            var cell = new GameObject(name, typeof(RectTransform)).AddComponent<PassCellView>();
            cell.transform.SetParent(parent, false);
            cell.Configure(NewText("Coins", cell.transform), NewText("Crystals", cell.transform), NewMark("Cosmetics", cell.transform), NewMark("Earned", cell.transform), NewMark("Locked", cell.transform));
            return cell;
        }

        ShopOfferRow NewOfferRow()
        {
            var row = new GameObject("Offer", typeof(RectTransform)).AddComponent<ShopOfferRow>();
            row.transform.SetParent(root.transform, false);
            row.Configure(NewText("Title", row.transform), NewText("Price", row.transform), NewText("Currency", row.transform), NewButton("Open", row.transform));
            return row;
        }

        void AnswerPass() => api.Answer(0, 200, PassJson);

        void AnswerShop()
        {
            api.Answer(1, 200, ShopJson);
            api.Answer(api.Requests.Count - 1, 200, WalletJson);
        }

        List<PassLevelRow> ShownRows()
        {
            var rows = new List<PassLevelRow>();
            foreach (Transform row in scroll.content)
                if (row.gameObject.activeSelf) rows.Add(row.GetComponent<PassLevelRow>());
            rows.Sort((a, b) => b.transform.localPosition.y.CompareTo(a.transform.localPosition.y));
            return rows;
        }

        static Transform Cell(Component row, string track) => row.transform.Find(track);

        static TMP_Text Amount(Component row, string track, string name) => Cell(row, track).Find(name).GetComponent<TMP_Text>();

        static bool Marked(Component row, string track, string name) => Cell(row, track).Find(name).gameObject.activeSelf;

        [Test]
        public void TheLaneShowsEverySeasonLevelWithTheServerRewardsAndStates()
        {
            Assert.AreEqual(PassStanding.Path, api.Requests[0].PathAndQuery);
            Assert.AreEqual(ShopShowcase.Path, api.Requests[1].PathAndQuery);
            Assert.IsFalse(level.enabled);
            Assert.AreEqual(0, ShownRows().Count);

            AnswerPass();

            Assert.IsTrue(level.enabled);
            Assert.AreEqual(MessageFormat.Number(2, strings.Language), level.text);
            Assert.AreEqual(0.25f, levelFill.fillAmount, 0.0001f);
            var rows = ShownRows();
            Assert.AreEqual(3, rows.Count);
            Assert.AreEqual(MessageFormat.Number(1, strings.Language), rows[0].transform.Find("Number").GetComponent<TMP_Text>().text);
            Assert.IsTrue(Amount(rows[0], "Free", "Coins").enabled);
            Assert.AreEqual(MessageFormat.Number(30, strings.Language), Amount(rows[0], "Free", "Coins").text);
            Assert.IsFalse(Amount(rows[0], "Free", "Crystals").enabled);
            Assert.IsTrue(Marked(rows[0], "Free", "Earned"));
            Assert.IsFalse(Marked(rows[0], "Free", "Locked"));
            Assert.AreEqual(MessageFormat.Number(6, strings.Language), Amount(rows[0], "Premium", "Crystals").text);
            Assert.IsTrue(Marked(rows[0], "Premium", "Locked"));
            Assert.IsFalse(Marked(rows[0], "Premium", "Earned"));
            Assert.IsFalse(Marked(rows[2], "Free", "Earned"));
            Assert.IsTrue(Marked(rows[2], "Free", "Cosmetics"));
            Assert.IsFalse(Amount(rows[2], "Free", "Coins").enabled);
            Assert.IsFalse(message.enabled);
        }

        [Test]
        public void ALevelIsBoughtAtTheServerPriceAndThePassReloads()
        {
            Assert.IsFalse(buyCoinLevel.gameObject.activeSelf);
            AnswerPass();
            Assert.IsTrue(buyCoinLevel.gameObject.activeSelf);
            Assert.AreEqual(MessageFormat.Number(250, strings.Language), coinLevelPrice.text);
            Assert.AreEqual(MessageFormat.Number(20, strings.Language), crystalLevelPrice.text);

            buyCoinLevel.onClick.Invoke();
            var sent = api.Requests.Count - 1;
            Assert.AreEqual(PassLevelBuyer.Path, api.Requests[sent].PathAndQuery);
            Assert.IsFalse(buyCoinLevel.interactable);
            buyCrystalLevel.onClick.Invoke();
            Assert.AreEqual(sent + 1, api.Requests.Count);

            api.Answer(sent, 200, LevelBoughtJson);

            Assert.IsTrue(buyCoinLevel.interactable);
            Assert.IsFalse(message.enabled);
            Assert.AreEqual(PassStanding.Path, api.Requests[api.Requests.Count - 1].PathAndQuery);
        }

        [Test]
        public void ARefusedLevelPurchaseShowsTheServerReason()
        {
            AnswerPass();
            buyCrystalLevel.onClick.Invoke();
            var sent = api.Requests.Count - 1;

            api.Answer(sent, 402, "{\"error\":\"wallet.insufficient\"}");

            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(PassLevelBuyKeys.Insufficient), message.text);
            Assert.AreEqual(sent + 1, api.Requests.Count);
        }

        [Test]
        public void ThePremiumOffersComeFromTheShopPassSectionAndUseTheCheckout()
        {
            AnswerShop();

            Assert.IsTrue(premiumOffers[0].gameObject.activeSelf);
            Assert.IsTrue(premiumOffers[1].gameObject.activeSelf);
            Assert.IsFalse(premiumOffers[2].gameObject.activeSelf);
            Assert.AreEqual(strings.Text("shop.offer.pass_premium"), premiumOffers[0].transform.Find("Title").GetComponent<TMP_Text>().text);
            Assert.AreEqual(strings.Text("shop.offer.pass_premium_plus"), premiumOffers[1].transform.Find("Title").GetComponent<TMP_Text>().text);
            Assert.IsFalse(preview.activeSelf);

            premiumOffers[1].transform.Find("Open").GetComponent<Button>().onClick.Invoke();
            Assert.IsTrue(preview.activeSelf);
            Assert.AreEqual(strings.Text("shop.offer.pass_premium_plus"), previewTitle.text);
            Assert.IsFalse(previewPrice.enabled);

            confirm.onClick.Invoke();

            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(ShopPurchaseResult.FromBilling(StoreBillingOutcome.Unavailable).MessageKey), message.text);

            closePreview.onClick.Invoke();
            Assert.IsFalse(preview.activeSelf);
            Assert.IsFalse(message.enabled);
        }

        [Test]
        public void NoRunningSeasonShowsItsMessageAndNoLevelPurchase()
        {
            api.Answer(0, 404, "{\"error\":\"pass.no_season\"}");

            Assert.IsTrue(message.enabled);
            Assert.AreEqual(strings.Text(PassStanding.NoSeason), message.text);
            Assert.IsFalse(buyCoinLevel.gameObject.activeSelf);
            Assert.IsFalse(buyCrystalLevel.gameObject.activeSelf);
            Assert.AreEqual(0, ShownRows().Count);
        }

        [Test]
        public void TheSeasonCountdownFollowsTheServerClockOnly()
        {
            AnswerPass();

            view.Tick(0);
            Assert.IsFalse(timeLeft.enabled);

            clock.Sync(SeasonEndsUnixMs - 3_725_000, 0, 0);
            view.Tick(0);
            Assert.IsTrue(timeLeft.enabled);
            Assert.AreEqual(Duration(TimeSpan.FromSeconds(3725)), timeLeft.text);

            view.Tick(3725);
            Assert.IsFalse(timeLeft.enabled);
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
