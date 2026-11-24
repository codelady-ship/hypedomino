using System;
using HypeDomino.Net.Api;
using HypeDomino.UI.Shop;

namespace HypeDomino.Game.Shop
{
    public enum ShopLoadOutcome
    {
        Loaded,
        Offline,
        SignInRequired,
        RateLimited,
        Failed,
    }

    public static class ShopLoadKeys
    {
        public const string Offline = "shop.load.offline";
        public const string RateLimited = "shop.load.rate_limited";
        public const string Failed = "notice.generic";

        public static string Of(ShopLoadOutcome outcome) => outcome switch
        {
            ShopLoadOutcome.Loaded => null,
            ShopLoadOutcome.Offline => Offline,
            ShopLoadOutcome.RateLimited => RateLimited,
            ShopLoadOutcome.SignInRequired => null,
            _ => Failed,
        };
    }

    public readonly struct ShopLoadResult
    {
        ShopLoadResult(ShopLoadOutcome outcome, ShopShowcase showcase = null, Balances balances = default)
        {
            Outcome = outcome;
            Showcase = showcase;
            Balances = balances;
        }

        public ShopLoadOutcome Outcome { get; }
        public ShopShowcase Showcase { get; }
        public Balances Balances { get; }
        public string MessageKey => ShopLoadKeys.Of(Outcome);

        public static ShopLoadResult Loaded(ShopShowcase showcase, Balances balances) => new(ShopLoadOutcome.Loaded, showcase, balances);

        public static ShopLoadResult Refused(ApiReply reply) => new(reply.Outcome switch
        {
            ApiOutcome.Offline => ShopLoadOutcome.Offline,
            ApiOutcome.SignInRequired => ShopLoadOutcome.SignInRequired,
            ApiOutcome.RateLimited => ShopLoadOutcome.RateLimited,
            _ => ShopLoadOutcome.Failed,
        });

        public static ShopLoadResult Unreadable() => new(ShopLoadOutcome.Failed);
    }

    public sealed class ShopLoader
    {
        public const string WalletPath = "/wallet";

        readonly ApiClient api;

        public ShopLoader(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Load(Action<ShopLoadResult> done)
        {
            if (Busy)
            {
                return false;
            }
            Busy = true;
            api.Get(ShopShowcase.Path, reply => ShopAnswered(reply, done));
            return true;
        }

        void ShopAnswered(ApiReply reply, Action<ShopLoadResult> done)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                Finish(ShopLoadResult.Refused(reply), done);
                return;
            }
            if (!ShopShowcase.TryRead(reply.Body, out var showcase))
            {
                Finish(ShopLoadResult.Unreadable(), done);
                return;
            }
            api.Get(WalletPath, wallet => WalletAnswered(wallet, showcase, done));
        }

        void WalletAnswered(ApiReply reply, ShopShowcase showcase, Action<ShopLoadResult> done)
        {
            if (reply.Outcome != ApiOutcome.Ok)
            {
                Finish(ShopLoadResult.Refused(reply), done);
                return;
            }
            Finish(Balances.TryRead(reply.Body, out var balances) ? ShopLoadResult.Loaded(showcase, balances) : ShopLoadResult.Unreadable(), done);
        }

        void Finish(ShopLoadResult result, Action<ShopLoadResult> done)
        {
            Busy = false;
            done(result);
        }
    }
}
