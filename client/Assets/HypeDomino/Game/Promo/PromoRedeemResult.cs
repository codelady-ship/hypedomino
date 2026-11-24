using System;
using System.Collections.Generic;
using HypeDomino.Net.Api;
using UnityEngine;

namespace HypeDomino.Game.Promo
{
    [Serializable]
    public sealed class PromoRedeemRequest
    {
        public string code;

        public string ToJson() => JsonUtility.ToJson(this);
    }

    [Serializable]
    public sealed class PromoGrantReply
    {
        public long coins;
        public string[] cosmetics;
        public int pass_xp;
        public int pass_xp_dropped;
    }

    [Serializable]
    public sealed class PromoRedemptionReply
    {
        public string promo;
        public string campaign;
        public PromoGrantReply grant;
        public bool pass_premium;

        public static PromoRedemptionReply FromJson(string json) => JsonUtility.FromJson<PromoRedemptionReply>(json);
    }

    public enum PromoRedeemOutcome
    {
        Granted,
        CodeInvalid,
        Unknown,
        NotActive,
        Disabled,
        NotStarted,
        Expired,
        Exhausted,
        AlreadyRedeemed,
        NotEligible,
        NoPassSeason,
        PassPremiumOwned,
        RateLimited,
        Locked,
        Offline,
        SignInRequired,
        Failed,
    }

    public static class PromoRedeemKeys
    {
        public const string Granted = "promo.redeem.granted";
        public const string GrantedPassPremium = "promo.redeem.granted_pass_premium";
        public const string CodeInvalid = "promo.redeem.code_invalid";
        public const string Unknown = "promo.redeem.unknown";
        public const string NotActive = "promo.redeem.not_active";
        public const string Disabled = "promo.redeem.disabled";
        public const string NotStarted = "promo.redeem.not_started";
        public const string Expired = "promo.redeem.expired";
        public const string Exhausted = "promo.redeem.exhausted";
        public const string AlreadyRedeemed = "promo.redeem.already_redeemed";
        public const string NotEligible = "promo.redeem.not_eligible";
        public const string NoPassSeason = "promo.redeem.no_pass_season";
        public const string PassPremiumOwned = "promo.redeem.pass_premium_owned";
        public const string RateLimited = "promo.redeem.rate_limited";
        public const string Locked = "promo.redeem.locked";
        public const string Offline = "promo.redeem.offline";
        public const string Failed = "notice.generic";

        public static string Of(PromoRedeemOutcome outcome) => outcome switch
        {
            PromoRedeemOutcome.Granted => Granted,
            PromoRedeemOutcome.CodeInvalid => CodeInvalid,
            PromoRedeemOutcome.Unknown => Unknown,
            PromoRedeemOutcome.NotActive => NotActive,
            PromoRedeemOutcome.Disabled => Disabled,
            PromoRedeemOutcome.NotStarted => NotStarted,
            PromoRedeemOutcome.Expired => Expired,
            PromoRedeemOutcome.Exhausted => Exhausted,
            PromoRedeemOutcome.AlreadyRedeemed => AlreadyRedeemed,
            PromoRedeemOutcome.NotEligible => NotEligible,
            PromoRedeemOutcome.NoPassSeason => NoPassSeason,
            PromoRedeemOutcome.PassPremiumOwned => PassPremiumOwned,
            PromoRedeemOutcome.RateLimited => RateLimited,
            PromoRedeemOutcome.Locked => Locked,
            PromoRedeemOutcome.Offline => Offline,
            PromoRedeemOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct PromoRedeemResult
    {
        PromoRedeemResult(PromoRedeemOutcome outcome, PromoGrantReply grant = null, IReadOnlyDictionary<string, long> duplicates = null, bool passPremium = false)
        {
            Outcome = outcome;
            Coins = grant?.coins ?? 0;
            Cosmetics = grant?.cosmetics ?? Array.Empty<string>();
            Duplicates = duplicates ?? GrantDuplicates.None;
            PassXp = grant?.pass_xp ?? 0;
            PassXpDropped = grant?.pass_xp_dropped ?? 0;
            PassPremium = passPremium;
        }

        public PromoRedeemOutcome Outcome { get; }
        public long Coins { get; }
        public string[] Cosmetics { get; }
        public IReadOnlyDictionary<string, long> Duplicates { get; }
        public int PassXp { get; }
        public int PassXpDropped { get; }
        public bool PassPremium { get; }
        public string MessageKey => PassPremium ? PromoRedeemKeys.GrantedPassPremium : PromoRedeemKeys.Of(Outcome);

        public static PromoRedeemResult Of(PromoRedeemOutcome outcome) => new(outcome);

        public static PromoRedeemResult From(ApiReply reply) => reply.Outcome switch
        {
            ApiOutcome.Ok => Granted(reply.Body),
            ApiOutcome.Offline => new(PromoRedeemOutcome.Offline),
            ApiOutcome.SignInRequired => new(PromoRedeemOutcome.SignInRequired),
            ApiOutcome.RateLimited => new(reply.Error == "promo.locked" ? PromoRedeemOutcome.Locked : PromoRedeemOutcome.RateLimited),
            _ => new(Refusal(reply.Error)),
        };

        static PromoRedeemResult Granted(string body)
        {
            var redemption = Parse(body);
            if (redemption == null || !GrantDuplicates.TryRead(body, out var duplicates)) return new(PromoRedeemOutcome.Failed);
            return new(PromoRedeemOutcome.Granted, redemption.grant, duplicates, redemption.pass_premium);
        }

        static PromoRedemptionReply Parse(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return PromoRedemptionReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }

        static PromoRedeemOutcome Refusal(string error) => error switch
        {
            "promo.code_invalid" => PromoRedeemOutcome.CodeInvalid,
            "promo.unknown" => PromoRedeemOutcome.Unknown,
            "promo.not_active" => PromoRedeemOutcome.NotActive,
            "promo.disabled" => PromoRedeemOutcome.Disabled,
            "promo.not_started" => PromoRedeemOutcome.NotStarted,
            "promo.expired" => PromoRedeemOutcome.Expired,
            "promo.exhausted" => PromoRedeemOutcome.Exhausted,
            "promo.already_redeemed" => PromoRedeemOutcome.AlreadyRedeemed,
            "promo.not_eligible" => PromoRedeemOutcome.NotEligible,
            "promo.no_pass_season" => PromoRedeemOutcome.NoPassSeason,
            "promo.pass_premium_owned" => PromoRedeemOutcome.PassPremiumOwned,
            _ => PromoRedeemOutcome.Failed,
        };
    }
}
