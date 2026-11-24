using System;

namespace HypeDomino.Game.Shop
{
    public enum ShopCheckoutStep
    {
        Closed,
        Preview,
        Buying,
        Done,
    }

    public readonly struct ShopCheckoutResult
    {
        public ShopCheckoutResult(ShopBuyResult bought)
        {
            InStore = false;
            Bought = bought;
            Paid = default;
        }

        public ShopCheckoutResult(ShopPurchaseResult paid)
        {
            InStore = true;
            Bought = default;
            Paid = paid;
        }

        public bool InStore { get; }
        public ShopBuyResult Bought { get; }
        public ShopPurchaseResult Paid { get; }
        public bool Granted => InStore ? Paid.Outcome == ShopPurchaseOutcome.Granted : Bought.Outcome == ShopBuyOutcome.Bought;
        public string MessageKey => InStore ? Paid.MessageKey : Bought.MessageKey;
    }

    public sealed class ShopCheckout
    {
        readonly ShopBuyer buyer;
        readonly ShopPurchase purchase;
        readonly Action<ShopBuyResult> bought;
        readonly Action<ShopPurchaseResult> paid;
        Action<ShopCheckoutResult> done;

        public ShopCheckout(ShopBuyer buyer, ShopPurchase purchase)
        {
            this.buyer = buyer;
            this.purchase = purchase;
            bought = Bought;
            paid = Paid;
        }

        public ShopCheckoutStep Step { get; private set; }
        public OfferCard Offer { get; private set; }
        public ShopCheckoutResult Result { get; private set; }

        public bool CanConfirm => Step switch
        {
            ShopCheckoutStep.Preview => Offer.Purchasable && (Offer.Price.InStore ? CanPay : !buyer.Busy),
            ShopCheckoutStep.Done => Offer.Price.InStore ? CanFinishPayment : !buyer.Busy && buyer.IsUnsettled(Offer.OfferId),
            _ => false,
        };

        bool CanPay => !purchase.Busy && (purchase.UnsettledOffer == null || CanFinishPayment);

        bool CanFinishPayment => purchase.CanRetry && purchase.UnsettledOffer == Offer.OfferId;

        public bool Open(OfferCard offer)
        {
            if (offer == null || Step == ShopCheckoutStep.Buying) return false;
            Offer = offer;
            Result = default;
            Step = ShopCheckoutStep.Preview;
            return true;
        }

        public bool Confirm(Action<ShopCheckoutResult> done)
        {
            if (!CanConfirm) return false;
            var previous = Step;
            Step = ShopCheckoutStep.Buying;
            this.done = done;
            if (Send()) return true;
            Step = previous;
            this.done = null;
            return false;
        }

        bool Send()
        {
            if (!Offer.Price.InStore) return buyer.Buy(Offer.OfferId, bought);
            return CanFinishPayment ? purchase.Retry(paid) : purchase.Buy(Offer.OfferId, paid);
        }

        public bool Close()
        {
            if (Step == ShopCheckoutStep.Buying) return false;
            Step = ShopCheckoutStep.Closed;
            Offer = null;
            Result = default;
            return true;
        }

        void Bought(ShopBuyResult result) => Settle(new ShopCheckoutResult(result));

        void Paid(ShopPurchaseResult result) => Settle(new ShopCheckoutResult(result));

        void Settle(ShopCheckoutResult result)
        {
            Result = result;
            Step = ShopCheckoutStep.Done;
            var notify = done;
            done = null;
            notify?.Invoke(result);
        }
    }
}
