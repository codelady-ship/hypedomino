using System;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Promo
{
    public sealed class PromoRedeemer
    {
        public const string Path = "/promo/redeem";

        readonly ApiClient api;

        public PromoRedeemer(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool Redeem(string typed, Action<PromoRedeemResult> done)
        {
            if (Busy)
            {
                return false;
            }
            if (!PromoCode.TryNormalize(typed, out var code))
            {
                done(PromoRedeemResult.Of(PromoRedeemOutcome.CodeInvalid));
                return true;
            }
            Busy = true;
            api.Post(Path, new PromoRedeemRequest { code = code }.ToJson(), reply =>
            {
                Busy = false;
                done(PromoRedeemResult.From(reply));
            });
            return true;
        }
    }
}
