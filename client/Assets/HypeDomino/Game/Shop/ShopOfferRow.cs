using System;
using HypeDomino.UI.Localization;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game.Shop
{
    public sealed class ShopOfferRow : MonoBehaviour
    {
        [SerializeField] TMP_Text title;
        [SerializeField] TMP_Text price;
        [SerializeField] TMP_Text currency;
        [SerializeField] Button open;

        OfferCard card;
        Action<OfferCard> opened;
        bool listening;

        public void Configure(TMP_Text titleText, TMP_Text priceText, TMP_Text currencyText, Button openButton)
        {
            StopListening();
            title = titleText;
            price = priceText;
            currency = currencyText;
            open = openButton;
        }

        public void Show(OfferCard offer, GameStrings strings, Action<OfferCard> onOpen)
        {
            card = offer;
            opened = onOpen;
            if (!listening)
            {
                open.onClick.AddListener(OnOpen);
                listening = true;
            }
            title.text = strings.Text(offer.NameKey);
            ShowPrice(offer.Price, strings, price, currency);
        }

        public static void ShowPrice(OfferPrice offerPrice, GameStrings strings, TMP_Text amount, TMP_Text currencyName)
        {
            amount.enabled = !offerPrice.InStore;
            currencyName.enabled = !offerPrice.InStore;
            if (offerPrice.InStore) return;
            amount.text = MessageFormat.Number(offerPrice.Amount.Amount, strings.Language);
            currencyName.text = strings.Text(offerPrice.Amount.NameKey);
        }

        void OnOpen() => opened?.Invoke(card);

        void StopListening()
        {
            if (!listening) return;
            open.onClick.RemoveListener(OnOpen);
            listening = false;
        }

        void OnDestroy() => StopListening();
    }
}
