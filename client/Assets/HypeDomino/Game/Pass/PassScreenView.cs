using System;
using System.Collections.Generic;
using HypeDomino.Content;
using HypeDomino.Game.Shop;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Shop;
using HypeDomino.UI.Tokens;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game.Pass
{
    public sealed class PassScreenView : MonoBehaviour, IRowBinder
    {
        public const string ShopSectionId = "pass";

        const long NotShown = -1;

        [SerializeField] TMP_Text level;
        [SerializeField] Image levelFill;
        [SerializeField] TMP_Text timeLeft;
        [SerializeField] RecyclingList levels;
        [SerializeField] PassLevelRow levelRow;
        [SerializeField] TMP_Text message;
        [SerializeField] Button buyCoinLevel;
        [SerializeField] TMP_Text coinLevelPrice;
        [SerializeField] Button buyCrystalLevel;
        [SerializeField] TMP_Text crystalLevelPrice;
        [SerializeField] ShopOfferRow[] premiumOffers;
        [SerializeField] GameObject preview;
        [SerializeField] TMP_Text previewTitle;
        [SerializeField] TMP_Text previewPrice;
        [SerializeField] TMP_Text previewCurrency;
        [SerializeField] Button confirm;
        [SerializeField] Button closePreview;

        readonly Dictionary<string, object> durationArgs = new();
        Action<PassLoadResult> passLoaded;
        Action<ShopLoadResult> shopLoaded;
        Action<PassLevelBuyResult> levelBought;
        Action<ShopCheckoutResult> settled;
        Action<OfferCard> openOffer;
        PassLoader pass;
        PassLevelBuyer levelBuyer;
        ShopLoader shop;
        ShopCheckout checkout;
        ActiveCosmeticManifest cosmetics;
        ServerClock clock;
        GameStrings strings;
        PassStanding standing;
        PassLane lane;
        OfferCard[] offers = Array.Empty<OfferCard>();
        string messageKey;
        bool built;
        long shownSecond = NotShown;

        public void Configure(TMP_Text levelText, Image levelFillImage, TMP_Text timeLeftText, RecyclingList levelList, PassLevelRow levelRowTemplate, TMP_Text messageLine)
        {
            Unbind();
            level = levelText;
            levelFill = levelFillImage;
            timeLeft = timeLeftText;
            levels = levelList;
            levelRow = levelRowTemplate;
            message = messageLine;
        }

        public void ConfigureLevelPurchase(Button coinButton, TMP_Text coinPrice, Button crystalButton, TMP_Text crystalPrice)
        {
            Unbind();
            buyCoinLevel = coinButton;
            coinLevelPrice = coinPrice;
            buyCrystalLevel = crystalButton;
            crystalLevelPrice = crystalPrice;
        }

        public void ConfigurePremium(ShopOfferRow[] offerRows, GameObject panel, TMP_Text title, TMP_Text price, TMP_Text currency, Button confirmButton, Button closeButton)
        {
            Unbind();
            premiumOffers = offerRows;
            preview = panel;
            previewTitle = title;
            previewPrice = price;
            previewCurrency = currency;
            confirm = confirmButton;
            closePreview = closeButton;
        }

        public void Bind(PassLoader passLoader, PassLevelBuyer buyer, ShopLoader shopLoader, ShopCheckout shopCheckout, ActiveCosmeticManifest manifest, ServerClock serverClock, GameStrings texts)
        {
            Unbind();
            pass = passLoader;
            levelBuyer = buyer;
            shop = shopLoader;
            checkout = shopCheckout;
            cosmetics = manifest;
            clock = serverClock;
            strings = texts;
            passLoaded ??= OnPassLoaded;
            shopLoaded ??= OnShopLoaded;
            levelBought ??= OnLevelBought;
            settled ??= OnSettled;
            openOffer ??= OnOpenOffer;
            buyCoinLevel.onClick.AddListener(OnBuyCoinLevel);
            buyCrystalLevel.onClick.AddListener(OnBuyCrystalLevel);
            confirm.onClick.AddListener(OnConfirm);
            closePreview.onClick.AddListener(OnClosePreview);
            strings.Changed += Redraw;
            BuildList();
            Redraw();
            Load();
        }

        public void Tick(double nowSeconds)
        {
            var seconds = 0d;
            var has = standing != null && standing.TrySecondsLeft(clock, nowSeconds, out seconds) && seconds > 0;
            ShopScreenView.ShowTimeLeft(timeLeft, has, seconds, ref shownSecond, strings, durationArgs);
        }

        void IRowBinder.Bind(RectTransform row, int index) => row.GetComponent<PassLevelRow>().Show(lane, index, strings.Language);

        void Start()
        {
            if (pass == null && NetBoot.Pass != null)
                Bind(NetBoot.Pass, NetBoot.PassLevels, NetBoot.Shop, NetBoot.Checkout, ContentBoot.Cosmetics, NetBoot.Connection.Clock, LanguageBoot.Strings);
        }

        void Update()
        {
            if (pass != null) Tick(Time.realtimeSinceStartupAsDouble);
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (pass == null) return;
            buyCoinLevel.onClick.RemoveListener(OnBuyCoinLevel);
            buyCrystalLevel.onClick.RemoveListener(OnBuyCrystalLevel);
            confirm.onClick.RemoveListener(OnConfirm);
            closePreview.onClick.RemoveListener(OnClosePreview);
            strings.Changed -= Redraw;
            pass = null;
            levelBuyer = null;
            shop = null;
            checkout = null;
            cosmetics = null;
            clock = null;
            strings = null;
        }

        void BuildList()
        {
            if (built) return;
            var template = (RectTransform)levelRow.transform;
            levels.Build(template, template.rect.height, TokenSpacing.Sm);
            built = true;
        }

        void Load()
        {
            pass.Load(passLoaded);
            shop.Load(shopLoaded);
        }

        void OnPassLoaded(PassLoadResult result)
        {
            if (pass == null) return;
            messageKey = result.MessageKey;
            if (result.Outcome == PassLoadOutcome.Loaded)
            {
                standing = result.Standing;
                lane = PassLane.Of(standing, result.Tracks, cosmetics.Current);
            }
            Redraw();
        }

        void OnShopLoaded(ShopLoadResult result)
        {
            if (pass == null || result.Outcome != ShopLoadOutcome.Loaded) return;
            offers = Array.Empty<OfferCard>();
            foreach (var section in result.Showcase.Sections)
                if (section.Id == ShopSectionId) offers = section.Offers;
            ShowOffers();
        }

        void OnBuyCoinLevel() => BuyLevel(Currency.Coin);

        void OnBuyCrystalLevel() => BuyLevel(Currency.Crystal);

        void BuyLevel(Currency currency)
        {
            if (!levelBuyer.Buy(standing, 1, currency, levelBought)) return;
            ShowLevelPrices();
        }

        void OnLevelBought(PassLevelBuyResult result)
        {
            if (pass == null) return;
            messageKey = result.MessageKey;
            ShowLevelPrices();
            ShowMessage();
            if (result.Outcome == PassLevelBuyOutcome.Bought) pass.Load(passLoaded);
        }

        void OnOpenOffer(OfferCard offer)
        {
            if (!checkout.Open(offer)) return;
            ShowPreview();
            ShowMessage();
        }

        void OnConfirm()
        {
            if (!checkout.Confirm(settled)) return;
            ShowPreview();
            ShowMessage();
        }

        void OnSettled(ShopCheckoutResult result)
        {
            if (pass == null) return;
            ShowPreview();
            ShowMessage();
            if (result.Granted) Load();
        }

        void OnClosePreview()
        {
            if (!checkout.Close()) return;
            ShowPreview();
            ShowMessage();
        }

        void Redraw()
        {
            shownSecond = NotShown;
            ShowStanding();
            levels.Show(lane?.Count ?? 0, this);
            ShowLevelPrices();
            ShowOffers();
            ShowPreview();
            ShowMessage();
        }

        void ShowStanding()
        {
            level.enabled = standing != null;
            levelFill.enabled = standing != null;
            if (standing == null) return;
            level.text = MessageFormat.Number(standing.Level, strings.Language);
            levelFill.fillAmount = standing.LevelFill;
        }

        void ShowLevelPrices()
        {
            ShowLevelPrice(buyCoinLevel, coinLevelPrice, Currency.Coin);
            ShowLevelPrice(buyCrystalLevel, crystalLevelPrice, Currency.Crystal);
        }

        void ShowLevelPrice(Button button, TMP_Text price, Currency currency)
        {
            var priced = PassLevelBuyer.TryPriceOf(standing, 1, currency, out var amount);
            button.gameObject.SetActive(priced);
            if (!priced) return;
            price.text = MessageFormat.Number(amount, strings.Language);
            button.interactable = !levelBuyer.Busy;
        }

        void ShowOffers()
        {
            for (var i = 0; i < premiumOffers.Length; i++)
            {
                var shown = i < offers.Length;
                premiumOffers[i].gameObject.SetActive(shown);
                if (shown) premiumOffers[i].Show(offers[i], strings, openOffer);
            }
        }

        void ShowPreview()
        {
            var step = checkout.Step;
            preview.SetActive(step != ShopCheckoutStep.Closed);
            if (step == ShopCheckoutStep.Closed) return;
            previewTitle.text = strings.Text(checkout.Offer.NameKey);
            ShopOfferRow.ShowPrice(checkout.Offer.Price, strings, previewPrice, previewCurrency);
            confirm.interactable = checkout.CanConfirm;
            closePreview.interactable = step != ShopCheckoutStep.Buying;
        }

        void ShowMessage()
        {
            var key = MessageKey;
            message.enabled = key != null;
            message.text = key == null ? string.Empty : strings.Text(key);
        }

        string MessageKey => checkout.Step switch
        {
            ShopCheckoutStep.Done => checkout.Result.MessageKey,
            ShopCheckoutStep.Preview when !checkout.Offer.Purchasable => checkout.Offer.RefusalKey,
            _ => messageKey,
        };
    }
}
