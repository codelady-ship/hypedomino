using System;
using System.Collections.Generic;
using HypeDomino.Net.Api;
using HypeDomino.UI.Shop;
using UnityEngine;

namespace HypeDomino.Game.Shop
{
    [Serializable]
    public sealed class ShopBuyRequest
    {
        public string offer;
        public string request_id;

        public string ToJson() => JsonUtility.ToJson(this);
    }

    public enum ShopBuyOutcome
    {
        Bought,
        Insufficient,
        Owned,
        LimitReached,
        NotEligible,
        Unavailable,
        Unknown,
        Unsupported,
        StoreOnly,
        InGameOnly,
        PurchaseBlocked,
        PurchasesDisabled,
        RateLimited,
        Offline,
        SignInRequired,
        Failed,
    }

    public static class ShopBuyKeys
    {
        public const string Bought = "shop.buy.bought";
        public const string Insufficient = "shop.buy.insufficient";
        public const string Owned = "shop.buy.owned";
        public const string LimitReached = "shop.buy.limit_reached";
        public const string NotEligible = "shop.buy.not_eligible";
        public const string Unavailable = "shop.buy.unavailable";
        public const string Unknown = "shop.buy.unknown";
        public const string Unsupported = "shop.buy.unsupported";
        public const string StoreOnly = "shop.buy.store_only";
        public const string InGameOnly = "shop.buy.in_game_only";
        public const string PurchaseBlocked = "shop.buy.purchase_blocked";
        public const string PurchasesDisabled = "shop.buy.purchases_disabled";
        public const string RateLimited = "shop.buy.rate_limited";
        public const string Offline = "shop.buy.offline";
        public const string Failed = "notice.generic";

        public static string Of(ShopBuyOutcome outcome) => outcome switch
        {
            ShopBuyOutcome.Bought => Bought,
            ShopBuyOutcome.Insufficient => Insufficient,
            ShopBuyOutcome.Owned => Owned,
            ShopBuyOutcome.LimitReached => LimitReached,
            ShopBuyOutcome.NotEligible => NotEligible,
            ShopBuyOutcome.Unavailable => Unavailable,
            ShopBuyOutcome.Unknown => Unknown,
            ShopBuyOutcome.Unsupported => Unsupported,
            ShopBuyOutcome.StoreOnly => StoreOnly,
            ShopBuyOutcome.InGameOnly => InGameOnly,
            ShopBuyOutcome.PurchaseBlocked => PurchaseBlocked,
            ShopBuyOutcome.PurchasesDisabled => PurchasesDisabled,
            ShopBuyOutcome.RateLimited => RateLimited,
            ShopBuyOutcome.Offline => Offline,
            ShopBuyOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct ShopBuyResult
    {
        ShopBuyResult(ShopBuyOutcome outcome, PurchaseReply purchase = null, IReadOnlyDictionary<string, long> duplicates = null)
        {
            Outcome = outcome;
            Purchase = purchase;
            Duplicates = duplicates ?? GrantDuplicates.None;
        }

        public ShopBuyOutcome Outcome { get; }
        public PurchaseReply Purchase { get; }
        public IReadOnlyDictionary<string, long> Duplicates { get; }
        public string MessageKey => ShopBuyKeys.Of(Outcome);

        public static ShopBuyResult From(ApiReply reply) => reply.Outcome switch
        {
            ApiOutcome.Ok => Bought(reply.Body),
            ApiOutcome.Offline => new(ShopBuyOutcome.Offline),
            ApiOutcome.SignInRequired => new(ShopBuyOutcome.SignInRequired),
            ApiOutcome.RateLimited => new(ShopBuyOutcome.RateLimited),
            ApiOutcome.Rejected or ApiOutcome.Unavailable => new(Refusal(reply.Error)),
            _ => new(ShopBuyOutcome.Failed),
        };

        static ShopBuyResult Bought(string body)
        {
            var purchase = Parse(body);
            if (purchase == null || string.IsNullOrEmpty(purchase.offer) || !GrantDuplicates.TryRead(body, out var duplicates)) return new(ShopBuyOutcome.Failed);
            return new(ShopBuyOutcome.Bought, purchase, duplicates);
        }

        static PurchaseReply Parse(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return PurchaseReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }

        public static ShopBuyOutcome Refusal(string error) => error switch
        {
            "wallet.insufficient" => ShopBuyOutcome.Insufficient,
            "offer.owned" => ShopBuyOutcome.Owned,
            "offer.limit_reached" => ShopBuyOutcome.LimitReached,
            "offer.not_eligible" => ShopBuyOutcome.NotEligible,
            "offer.unavailable" => ShopBuyOutcome.Unavailable,
            "offer.unknown" => ShopBuyOutcome.Unknown,
            "offer.unsupported" => ShopBuyOutcome.Unsupported,
            "offer.store_only" => ShopBuyOutcome.StoreOnly,
            "offer.in_game_only" => ShopBuyOutcome.InGameOnly,
            "purchase.blocked" => ShopBuyOutcome.PurchaseBlocked,
            "purchases.disabled" => ShopBuyOutcome.PurchasesDisabled,
            _ => ShopBuyOutcome.Failed,
        };
    }
}
