using System;

namespace HypeDomino.Game.Promo
{
    public sealed class PromoCodeEntry
    {
        readonly PromoRedeemer redeemer;
        readonly Action<PromoRedeemResult> finish;

        public PromoCodeEntry(PromoRedeemer redeemer)
        {
            this.redeemer = redeemer;
            finish = Finish;
        }

        public event Action Changed;

        public event Action<PromoRedeemResult> Redeemed;

        public string Text { get; private set; } = string.Empty;

        public bool WellFormed { get; private set; }

        public string MessageKey { get; private set; }

        public bool Busy => redeemer.Busy;

        public bool CanSubmit => WellFormed && !redeemer.Busy;

        public void Edit(string typed)
        {
            typed ??= string.Empty;
            if (typed == Text)
            {
                return;
            }
            Text = typed;
            WellFormed = PromoCode.IsWellFormed(typed);
            MessageKey = null;
            Changed?.Invoke();
        }

        public void Paste(string clipboard)
        {
            if (string.IsNullOrEmpty(clipboard))
            {
                return;
            }
            Edit(clipboard);
        }

        public bool Submit()
        {
            if (!CanSubmit || !redeemer.Redeem(Text, finish))
            {
                return false;
            }
            Changed?.Invoke();
            return true;
        }

        void Finish(PromoRedeemResult result)
        {
            MessageKey = result.MessageKey;
            Changed?.Invoke();
            Redeemed?.Invoke(result);
        }
    }
}
