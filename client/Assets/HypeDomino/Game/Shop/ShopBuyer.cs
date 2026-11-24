using System;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.Shop
{
    public sealed class ShopBuyer
    {
        public const string Path = "/shop/buy";

        readonly ApiClient api;
        string pendingOffer;
        string pendingRequestId;

        public ShopBuyer(ApiClient api)
        {
            this.api = api;
        }

        public bool Busy { get; private set; }

        public bool IsUnsettled(string offer) => offer != null && offer == pendingOffer;

        public bool Buy(string offer, Action<ShopBuyResult> done)
        {
            if (Busy || string.IsNullOrEmpty(offer))
            {
                return false;
            }
            Busy = true;
            var request = new ShopBuyRequest { offer = offer, request_id = RequestIdFor(offer) };
            api.Post(Path, request.ToJson(), reply =>
            {
                Busy = false;
                var result = ShopBuyResult.From(reply);
                if (reply.Outcome == ApiOutcome.Rejected || result.Outcome == ShopBuyOutcome.Bought)
                {
                    ForgetPending();
                }
                done(result);
            });
            return true;
        }

        string RequestIdFor(string offer)
        {
            if (offer != pendingOffer)
            {
                pendingOffer = offer;
                pendingRequestId = Guid.NewGuid().ToString("N");
            }
            return pendingRequestId;
        }

        void ForgetPending()
        {
            pendingOffer = null;
            pendingRequestId = null;
        }
    }
}
