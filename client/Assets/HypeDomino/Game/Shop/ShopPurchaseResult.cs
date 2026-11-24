using System;
using HypeDomino.Net.Api;
using UnityEngine;

namespace HypeDomino.Game.Shop
{
    [Serializable]
    public sealed class ShopBeginRequest
    {
        public string offer;
    }

    [Serializable]
    public sealed class ShopCheckoutReply
    {
        public string offer;
        public string product_id;
        public string account_token;

        public static ShopCheckoutReply TryRead(string body)
        {
            var reply = ShopJson.Read<ShopCheckoutReply>(body);
            return reply == null || string.IsNullOrEmpty(reply.product_id) || string.IsNullOrEmpty(reply.account_token) ? null : reply;
        }
    }

    [Serializable]
    public sealed class ShopCompleteRequest
    {
        public string offer;
        public string platform;
        public string token;
    }

    [Serializable]
    public sealed class ShopCompleteReply
    {
        public const string Granted = "granted";
        public const string Pending = "pending";

        public string platform;
        public string transaction_id;
        public string offer;
        public string state;
        public bool repeated;
        public long crystals;
        public bool finished;

        public static ShopCompleteReply TryRead(string body)
        {
            var reply = ShopJson.Read<ShopCompleteReply>(body);
            return reply == null || string.IsNullOrEmpty(reply.transaction_id) || string.IsNullOrEmpty(reply.state) ? null : reply;
        }
    }

    static class ShopJson
    {
        public static T Read<T>(string body) where T : class
        {
            if (string.IsNullOrEmpty(body)) return null;
            try
            {
                return JsonUtility.FromJson<T>(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }

    public enum ShopPurchaseStep
    {
        Begin,
        Billing,
        Complete,
    }

    public enum ShopPurchaseOutcome
    {
        Granted,
        Pending,
        Cancelled,
        StoreUnavailable,
        Owned,
        LimitReached,
        NotEligible,
        Unavailable,
        Unknown,
        Unsupported,
        InGameOnly,
        PurchaseBlocked,
        PurchasesDisabled,
        NoPassSeason,
        ReceiptRefused,
        IntegrityFailed,
        Unfinished,
        RateLimited,
        Offline,
        SignInRequired,
        Failed,
    }

    public static class ShopPurchaseKeys
    {
        public const string Pending = "shop.purchase.pending";
        public const string StoreUnavailable = "shop.purchase.store_unavailable";
        public const string NotGranted = "shop.purchase.not_granted";
        public const string Unfinished = "shop.purchase.unfinished";

        public static string Of(ShopPurchaseOutcome outcome, ShopPurchaseStep step)
        {
            switch (outcome)
            {
                case ShopPurchaseOutcome.Granted:
                    return ShopBuyKeys.Bought;
                case ShopPurchaseOutcome.Pending:
                    return Pending;
                case ShopPurchaseOutcome.Cancelled:
                case ShopPurchaseOutcome.SignInRequired:
                    return null;
                case ShopPurchaseOutcome.Unfinished:
                    return Unfinished;
                case ShopPurchaseOutcome.RateLimited:
                    return ShopBuyKeys.RateLimited;
                case ShopPurchaseOutcome.Offline:
                    return ShopBuyKeys.Offline;
            }
            return step == ShopPurchaseStep.Complete ? NotGranted : BeforePayment(outcome);
        }

        static string BeforePayment(ShopPurchaseOutcome outcome) => outcome switch
        {
            ShopPurchaseOutcome.StoreUnavailable => StoreUnavailable,
            ShopPurchaseOutcome.Owned => ShopBuyKeys.Owned,
            ShopPurchaseOutcome.LimitReached => ShopBuyKeys.LimitReached,
            ShopPurchaseOutcome.NotEligible => ShopBuyKeys.NotEligible,
            ShopPurchaseOutcome.Unavailable => ShopBuyKeys.Unavailable,
            ShopPurchaseOutcome.Unknown => ShopBuyKeys.Unknown,
            ShopPurchaseOutcome.Unsupported => ShopBuyKeys.Unsupported,
            ShopPurchaseOutcome.InGameOnly => ShopBuyKeys.InGameOnly,
            ShopPurchaseOutcome.PurchaseBlocked => ShopBuyKeys.PurchaseBlocked,
            ShopPurchaseOutcome.PurchasesDisabled => ShopBuyKeys.PurchasesDisabled,
            _ => ShopBuyKeys.Failed,
        };
    }

    public readonly struct ShopPurchaseResult
    {
        public ShopPurchaseResult(ShopPurchaseOutcome outcome, ShopPurchaseStep step, ShopCompleteReply reply = null)
        {
            Outcome = outcome;
            Step = step;
            Reply = reply;
        }

        public ShopPurchaseOutcome Outcome { get; }
        public ShopPurchaseStep Step { get; }
        public ShopCompleteReply Reply { get; }
        public string MessageKey => ShopPurchaseKeys.Of(Outcome, Step);

        public bool KeepsReceipt => Step == ShopPurchaseStep.Complete && Outcome is ShopPurchaseOutcome.Unfinished or ShopPurchaseOutcome.Offline or ShopPurchaseOutcome.RateLimited or ShopPurchaseOutcome.SignInRequired;

        public static ShopPurchaseResult FromBegin(ApiReply reply) => new(Failure(reply, ShopPurchaseOutcome.Failed), ShopPurchaseStep.Begin);

        public static ShopPurchaseResult FromBilling(StoreBillingOutcome outcome) => new(outcome switch
        {
            StoreBillingOutcome.Deferred => ShopPurchaseOutcome.Pending,
            StoreBillingOutcome.Cancelled => ShopPurchaseOutcome.Cancelled,
            StoreBillingOutcome.Unavailable => ShopPurchaseOutcome.StoreUnavailable,
            _ => ShopPurchaseOutcome.Failed,
        }, ShopPurchaseStep.Billing);

        public static ShopPurchaseResult FromComplete(ApiReply reply)
        {
            if (reply.Outcome != ApiOutcome.Ok) return new(Failure(reply, ShopPurchaseOutcome.Unfinished), ShopPurchaseStep.Complete);
            var completed = ShopCompleteReply.TryRead(reply.Body);
            if (completed == null) return new(ShopPurchaseOutcome.Unfinished, ShopPurchaseStep.Complete);
            return new(StateOutcome(completed.state), ShopPurchaseStep.Complete, completed);
        }

        static ShopPurchaseOutcome StateOutcome(string state) => state switch
        {
            ShopCompleteReply.Granted => ShopPurchaseOutcome.Granted,
            ShopCompleteReply.Pending => ShopPurchaseOutcome.Pending,
            _ => ShopPurchaseOutcome.Failed,
        };

        static ShopPurchaseOutcome Failure(ApiReply reply, ShopPurchaseOutcome serverError) => reply.Outcome switch
        {
            ApiOutcome.Offline => ShopPurchaseOutcome.Offline,
            ApiOutcome.SignInRequired => ShopPurchaseOutcome.SignInRequired,
            ApiOutcome.RateLimited => ShopPurchaseOutcome.RateLimited,
            ApiOutcome.Rejected => Refusal(reply.Error),
            ApiOutcome.Unavailable when reply.Error == "purchases.disabled" => ShopPurchaseOutcome.PurchasesDisabled,
            ApiOutcome.Unavailable => serverError,
            _ => ShopPurchaseOutcome.Failed,
        };

        public static ShopPurchaseOutcome Refusal(string error) => error switch
        {
            "offer.owned" or "pass.premium_owned" => ShopPurchaseOutcome.Owned,
            "offer.limit_reached" => ShopPurchaseOutcome.LimitReached,
            "offer.not_eligible" => ShopPurchaseOutcome.NotEligible,
            "offer.unavailable" => ShopPurchaseOutcome.Unavailable,
            "offer.unknown" => ShopPurchaseOutcome.Unknown,
            "offer.unsupported" => ShopPurchaseOutcome.Unsupported,
            "offer.in_game_only" => ShopPurchaseOutcome.InGameOnly,
            "purchase.blocked" => ShopPurchaseOutcome.PurchaseBlocked,
            "purchases.disabled" => ShopPurchaseOutcome.PurchasesDisabled,
            "purchase.platform_unavailable" => ShopPurchaseOutcome.StoreUnavailable,
            "pass.no_season" => ShopPurchaseOutcome.NoPassSeason,
            "receipt.malformed" or "receipt.signature" or "receipt.unknown" or "receipt.account" or "receipt.reused"
                or "receipt.product" or "receipt.cancelled" or "receipt.environment" or "receipt.app" => ShopPurchaseOutcome.ReceiptRefused,
            "integrity_failed" or "bad_integrity_proof" => ShopPurchaseOutcome.IntegrityFailed,
            _ => ShopPurchaseOutcome.Failed,
        };
    }
}
