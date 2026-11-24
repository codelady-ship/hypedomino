using System;
using HypeDomino.Net.Api;
using HypeDomino.UI.Shop;
using UnityEngine;

namespace HypeDomino.Game.Pass
{
    [Serializable]
    public sealed class PassLevelsRequest
    {
        public int count;
        public string currency;
        public string request_id;

        public static string Json(int count, Currency currency, string requestId) =>
            JsonUtility.ToJson(new PassLevelsRequest { count = count, currency = currency == Currency.Crystal ? "crystal" : "coin", request_id = requestId });
    }

    [Serializable]
    public sealed class PassLevelsReply
    {
        public string season;
        public int count;
        public string currency;
        public long price;
        public int level;
        public long xp;
        public long balance;
        public bool repeated;

        public static PassLevelsReply FromJson(string json) => JsonUtility.FromJson<PassLevelsReply>(json);
    }

    public enum PassLevelBuyOutcome
    {
        Bought,
        Insufficient,
        TooManyLevels,
        SeasonOver,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class PassLevelBuyKeys
    {
        public const string Insufficient = "pass.levels.insufficient";
        public const string TooManyLevels = "pass.levels.too_many";
        public const string SeasonOver = "pass.levels.season_over";
        public const string Offline = "pass.levels.offline";
        public const string RateLimited = "pass.levels.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(PassLevelBuyOutcome outcome) => outcome switch
        {
            PassLevelBuyOutcome.Bought => null,
            PassLevelBuyOutcome.Insufficient => Insufficient,
            PassLevelBuyOutcome.TooManyLevels => TooManyLevels,
            PassLevelBuyOutcome.SeasonOver => SeasonOver,
            PassLevelBuyOutcome.Offline => Offline,
            PassLevelBuyOutcome.RateLimited => RateLimited,
            PassLevelBuyOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct PassLevelBuyResult
    {
        PassLevelBuyResult(PassLevelBuyOutcome outcome, PassLevelsReply purchase = null, Balance balance = default)
        {
            Outcome = outcome;
            Purchase = purchase;
            Balance = balance;
        }

        public PassLevelBuyOutcome Outcome { get; }
        public PassLevelsReply Purchase { get; }
        public Balance Balance { get; }
        public string MessageKey => PassLevelBuyKeys.Of(Outcome);

        public static PassLevelBuyResult From(ApiReply reply) => reply.Outcome switch
        {
            ApiOutcome.Ok => Bought(reply.Body),
            ApiOutcome.Offline => new(PassLevelBuyOutcome.Offline),
            ApiOutcome.SignInRequired => new(PassLevelBuyOutcome.SignInRequired),
            ApiOutcome.RateLimited => new(PassLevelBuyOutcome.RateLimited),
            ApiOutcome.Rejected => new(Refusal(reply.Error)),
            _ => new(PassLevelBuyOutcome.Failed),
        };

        public static PassLevelBuyOutcome Refusal(string error) => error switch
        {
            "wallet.insufficient" => PassLevelBuyOutcome.Insufficient,
            "pass.level_count" => PassLevelBuyOutcome.TooManyLevels,
            PassStanding.NoSeason => PassLevelBuyOutcome.SeasonOver,
            _ => PassLevelBuyOutcome.Failed,
        };

        static PassLevelBuyResult Bought(string body)
        {
            var purchase = Parse(body);
            if (!IsReadable(purchase, out var currency)) return new(PassLevelBuyOutcome.Failed);
            return new(PassLevelBuyOutcome.Bought, purchase, new Balance(currency, purchase.balance));
        }

        static bool IsReadable(PassLevelsReply purchase, out Currency currency)
        {
            currency = Currency.Coin;
            if (purchase == null || string.IsNullOrEmpty(purchase.season)) return false;
            if (purchase.count < 1 || purchase.level < 1 || purchase.price < 0 || purchase.xp < 0) return false;
            return CurrencyKeys.TryParse(purchase.currency, out currency);
        }

        static PassLevelsReply Parse(string body)
        {
            if (string.IsNullOrEmpty(body))
            {
                return null;
            }
            try
            {
                return PassLevelsReply.FromJson(body);
            }
            catch (ArgumentException)
            {
                return null;
            }
        }
    }

    public sealed class PassLevelBuyer
    {
        public const string Path = "/pass/levels";

        readonly ApiClient api;
        int pendingCount;
        Currency pendingCurrency;
        string pendingRequestId;

        public PassLevelBuyer(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public static int MostLevelsFor(PassStanding standing) => standing == null ? 0 : standing.Levels - standing.Level;

        public static bool TryPriceOf(PassStanding standing, int count, Currency currency, out long price)
        {
            price = 0;
            if (count < 1 || count > MostLevelsFor(standing)) return false;
            var levelPrice = currency == Currency.Crystal ? standing.LevelCrystalPrice : standing.LevelCoinPrice;
            if (levelPrice < 1) return false;
            price = count * levelPrice;
            return true;
        }

        public bool IsUnsettled(int count, Currency currency) => pendingRequestId != null && count == pendingCount && currency == pendingCurrency;

        public bool Buy(PassStanding standing, int count, Currency currency, Action<PassLevelBuyResult> done)
        {
            if (Busy || count < 1 || count > MostLevelsFor(standing))
            {
                return false;
            }
            Busy = true;
            api.Post(Path, PassLevelsRequest.Json(count, currency, RequestIdFor(count, currency)), reply =>
            {
                Busy = false;
                var result = PassLevelBuyResult.From(reply);
                if (reply.Outcome == ApiOutcome.Rejected || result.Outcome == PassLevelBuyOutcome.Bought)
                {
                    ForgetPending();
                }
                done(result);
            });
            return true;
        }

        string RequestIdFor(int count, Currency currency)
        {
            if (!IsUnsettled(count, currency))
            {
                pendingCount = count;
                pendingCurrency = currency;
                pendingRequestId = Guid.NewGuid().ToString("N");
            }
            return pendingRequestId;
        }

        void ForgetPending()
        {
            pendingCount = 0;
            pendingRequestId = null;
        }
    }
}
