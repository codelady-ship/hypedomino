using System;
using System.Collections.Generic;
using HypeDomino.Net;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Shop;
using HypeDomino.UI.Tokens;
using TMPro;
using UnityEngine;
using UnityEngine.Events;
using UnityEngine.UI;

namespace HypeDomino.Game.Shop
{
    public sealed class ShopScreenView : MonoBehaviour, IRowBinder
    {
        const long NotShown = -1;

        [SerializeField] Button[] categories;
        [SerializeField] TMP_Text[] categoryTitles;
        [SerializeField] TMP_Text sectionTimeLeft;
        [SerializeField] RecyclingList offers;
        [SerializeField] ShopOfferRow offerRow;
        [SerializeField] TMP_Text coins;
        [SerializeField] TMP_Text crystals;
        [SerializeField] TMP_Text message;
        [SerializeField] GameObject preview;
        [SerializeField] TMP_Text previewTitle;
        [SerializeField] TMP_Text previewPrice;
        [SerializeField] TMP_Text previewCurrency;
        [SerializeField] TMP_Text previewTimeLeft;
        [SerializeField] Button confirm;
        [SerializeField] Button closePreview;

        readonly Dictionary<string, object> durationArgs = new();
        Action<ShopLoadResult> loaded;
        Action<ShopCheckoutResult> settled;
        Action<OfferCard> openOffer;
        UnityAction[] categoryPicks;
        ShopLoader shop;
        ShopCheckout checkout;
        ServerClock clock;
        GameStrings strings;
        ShopShowcase showcase;
        Balances balances;
        string loadMessageKey;
        int selected;
        bool built;
        long shownSectionSecond = NotShown;
        long shownPreviewSecond = NotShown;

        public void Configure(Button[] categoryButtons, TMP_Text[] categoryTitleTexts, TMP_Text sectionTimeLeftText, RecyclingList offerList, ShopOfferRow offerRowTemplate,
            TMP_Text coinsText, TMP_Text crystalsText, TMP_Text messageLine)
        {
            Unbind();
            categories = categoryButtons;
            categoryTitles = categoryTitleTexts;
            sectionTimeLeft = sectionTimeLeftText;
            offers = offerList;
            offerRow = offerRowTemplate;
            coins = coinsText;
            crystals = crystalsText;
            message = messageLine;
            categoryPicks = null;
        }

        public void ConfigurePreview(GameObject panel, TMP_Text title, TMP_Text price, TMP_Text currency, TMP_Text timeLeft, Button confirmButton, Button closeButton)
        {
            Unbind();
            preview = panel;
            previewTitle = title;
            previewPrice = price;
            previewCurrency = currency;
            previewTimeLeft = timeLeft;
            confirm = confirmButton;
            closePreview = closeButton;
        }

        public void Bind(ShopLoader loader, ShopCheckout shopCheckout, ServerClock serverClock, GameStrings texts)
        {
            Unbind();
            shop = loader;
            checkout = shopCheckout;
            clock = serverClock;
            strings = texts;
            loaded ??= OnLoaded;
            settled ??= OnSettled;
            openOffer ??= OnOpenOffer;
            categoryPicks ??= CategoryPicks(categories.Length);
            for (var i = 0; i < categories.Length; i++) categories[i].onClick.AddListener(categoryPicks[i]);
            confirm.onClick.AddListener(OnConfirm);
            closePreview.onClick.AddListener(OnClosePreview);
            strings.Changed += Redraw;
            BuildList();
            Redraw();
            Load();
        }

        public void Tick(double nowSeconds)
        {
            var section = SelectedSection;
            var sectionSeconds = 0d;
            var sectionHas = section != null && section.TrySecondsLeft(clock, nowSeconds, out sectionSeconds) && sectionSeconds > 0;
            ShowTimeLeft(sectionTimeLeft, sectionHas, sectionSeconds, ref shownSectionSecond, strings, durationArgs);
            var offer = checkout.Step == ShopCheckoutStep.Closed ? null : checkout.Offer;
            var offerSeconds = 0d;
            var offerHas = offer != null && offer.TrySecondsLeft(clock, nowSeconds, out offerSeconds) && offerSeconds > 0;
            ShowTimeLeft(previewTimeLeft, offerHas, offerSeconds, ref shownPreviewSecond, strings, durationArgs);
        }

        void IRowBinder.Bind(RectTransform row, int index) => row.GetComponent<ShopOfferRow>().Show(SelectedSection.Offers[index], strings, openOffer);

        void Start()
        {
            if (shop == null && NetBoot.Shop != null) Bind(NetBoot.Shop, NetBoot.Checkout, NetBoot.Connection.Clock, LanguageBoot.Strings);
        }

        void Update()
        {
            if (shop != null) Tick(Time.realtimeSinceStartupAsDouble);
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (shop == null) return;
            for (var i = 0; i < categories.Length; i++) categories[i].onClick.RemoveListener(categoryPicks[i]);
            confirm.onClick.RemoveListener(OnConfirm);
            closePreview.onClick.RemoveListener(OnClosePreview);
            strings.Changed -= Redraw;
            shop = null;
            checkout = null;
            clock = null;
            strings = null;
        }

        UnityAction[] CategoryPicks(int count)
        {
            var picks = new UnityAction[count];
            for (var i = 0; i < count; i++)
            {
                var index = i;
                picks[i] = () => OnCategory(index);
            }
            return picks;
        }

        void BuildList()
        {
            if (built) return;
            var template = (RectTransform)offerRow.transform;
            offers.Build(template, template.rect.height, TokenSpacing.Sm);
            built = true;
        }

        ShopSection SelectedSection => showcase != null && selected < showcase.Sections.Length ? showcase.Sections[selected] : null;

        void Load() => shop.Load(loaded);

        void OnLoaded(ShopLoadResult result)
        {
            if (shop == null) return;
            loadMessageKey = result.MessageKey;
            if (result.Outcome == ShopLoadOutcome.Loaded)
            {
                showcase = result.Showcase;
                balances = result.Balances;
                if (selected >= showcase.Sections.Length) selected = 0;
            }
            Redraw();
        }

        void OnCategory(int index)
        {
            selected = index;
            shownSectionSecond = NotShown;
            ShowCategories();
            ShowOffers();
        }

        void OnOpenOffer(OfferCard offer)
        {
            if (!checkout.Open(offer)) return;
            shownPreviewSecond = NotShown;
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
            if (shop == null) return;
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
            shownSectionSecond = NotShown;
            shownPreviewSecond = NotShown;
            ShowBalances();
            ShowCategories();
            ShowOffers();
            ShowPreview();
            ShowMessage();
        }

        void ShowBalances()
        {
            coins.enabled = showcase != null;
            crystals.enabled = showcase != null;
            if (showcase == null) return;
            coins.text = MessageFormat.Number(balances.Coins.Amount, strings.Language);
            crystals.text = MessageFormat.Number(balances.Crystals.Amount, strings.Language);
        }

        void ShowCategories()
        {
            for (var i = 0; i < categories.Length; i++)
            {
                var shown = showcase != null && i < showcase.Sections.Length;
                categories[i].gameObject.SetActive(shown);
                if (!shown) continue;
                categoryTitles[i].text = strings.Text(showcase.Sections[i].TitleKey);
                categories[i].interactable = i != selected;
            }
        }

        void ShowOffers()
        {
            var section = SelectedSection;
            offers.Show(section == null ? 0 : section.Offers.Length, this);
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
            _ => loadMessageKey,
        };

        public static void ShowTimeLeft(TMP_Text text, bool has, double secondsLeft, ref long shown, GameStrings strings, Dictionary<string, object> durationArgs)
        {
            if (text.enabled != has) text.enabled = has;
            if (!has)
            {
                shown = NotShown;
                return;
            }
            var seconds = (long)Math.Ceiling(secondsLeft);
            if (seconds == shown) return;
            shown = seconds;
            var span = TimeSpan.FromSeconds(seconds);
            durationArgs["days"] = (long)span.Days;
            durationArgs["hours"] = (long)span.Hours;
            durationArgs["minutes"] = (long)span.Minutes;
            durationArgs["seconds"] = (long)span.Seconds;
            text.text = strings.Text(MessageFormat.DurationKey(span), durationArgs);
        }
    }
}
