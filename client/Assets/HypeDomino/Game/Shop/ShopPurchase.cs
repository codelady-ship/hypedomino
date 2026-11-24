using System;
using HypeDomino.Core.Analytics;
using HypeDomino.Net.Api;
using HypeDomino.Net.Integrity;
using Hypedomino.V1;
using UnityEngine;

namespace HypeDomino.Game.Shop
{
    public sealed class ShopPurchase
    {
        public const string BeginPath = "/shop/begin";
        public const string CompletePath = "/shop/complete";

        readonly ApiClient api;
        readonly IStoreBilling billing;
        readonly Action<IntegrityPurpose, Action<IntegrityProof>> requestProof;
        readonly AnalyticsQueue analytics;
        readonly Action<ApiReply> begun;
        readonly Action<StoreBillingResult> billed;
        readonly Action<IntegrityProof> proved;
        readonly Action<ApiReply> completed;
        Action<ShopPurchaseResult> done;
        string offer;
        string product;
        StoreReceipt receipt;

        public ShopPurchase(ApiClient api, IStoreBilling billing, Action<IntegrityPurpose, Action<IntegrityProof>> requestProof, AnalyticsQueue analytics = null)
        {
            this.api = api;
            this.billing = billing;
            this.requestProof = requestProof;
            this.analytics = analytics;
            begun = Begun;
            billed = Billed;
            proved = Proved;
            completed = Completed;
        }

        public bool Busy { get; private set; }
        public string UnsettledOffer => receipt == null ? null : offer;
        public bool CanRetry => !Busy && receipt != null;

        public bool Buy(string offer, Action<ShopPurchaseResult> done)
        {
            if (Busy || receipt != null || string.IsNullOrEmpty(offer)) return false;
            if (!billing.CanPurchase)
            {
                done(ShopPurchaseResult.FromBilling(StoreBillingOutcome.Unavailable));
                return true;
            }
            Busy = true;
            this.offer = offer;
            this.done = done;
            api.Post(BeginPath, JsonUtility.ToJson(new ShopBeginRequest { offer = offer }), begun);
            return true;
        }

        public bool Retry(Action<ShopPurchaseResult> done)
        {
            if (!CanRetry) return false;
            Busy = true;
            this.done = done;
            requestProof(IntegrityPurpose.Purchase, proved);
            return true;
        }

        void Begun(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                Settle(ShopPurchaseResult.FromBegin(reply));
                return;
            }
            var checkout = ShopCheckoutReply.TryRead(reply.Body);
            if (checkout == null)
            {
                Settle(new ShopPurchaseResult(ShopPurchaseOutcome.Failed, ShopPurchaseStep.Begin));
                return;
            }
            product = checkout.product_id;
            analytics?.Track(AnalyticsEvents.PurchaseStart, product);
            billing.Purchase(product, checkout.account_token, billed);
        }

        void Billed(StoreBillingResult result)
        {
            if (result.Outcome != StoreBillingOutcome.Purchased)
            {
                if (result.Outcome == StoreBillingOutcome.Cancelled) analytics?.Track(AnalyticsEvents.PurchaseCancel, product);
                Settle(ShopPurchaseResult.FromBilling(result.Outcome));
                return;
            }
            if (result.Receipt == null || string.IsNullOrEmpty(result.Receipt.Token))
            {
                Settle(ShopPurchaseResult.FromBilling(StoreBillingOutcome.Failed));
                return;
            }
            receipt = result.Receipt;
            requestProof(IntegrityPurpose.Purchase, proved);
        }

        void Proved(IntegrityProof proof)
        {
            var body = JsonUtility.ToJson(new ShopCompleteRequest { offer = offer, platform = receipt.PlatformWireName, token = receipt.Token });
            api.Post(CompletePath, body, IntegrityHeaders.Of(proof), completed);
        }

        void Completed(ApiReply reply)
        {
            var result = ShopPurchaseResult.FromComplete(reply);
            if (!result.KeepsReceipt)
            {
                var settled = receipt;
                receipt = null;
                if (result.Reply != null && result.Outcome != ShopPurchaseOutcome.Pending) billing.Finish(settled, result.Reply.finished);
            }
            Settle(result);
        }

        void Settle(ShopPurchaseResult result)
        {
            Busy = false;
            var notify = done;
            done = null;
            notify?.Invoke(result);
        }
    }
}
