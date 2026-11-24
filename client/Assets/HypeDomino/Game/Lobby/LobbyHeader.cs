using System;
using HypeDomino.UI.Shop;

namespace HypeDomino.Game.Lobby
{
    public sealed class LobbyHeader
    {
        public bool HasBalances { get; private set; }
        public Balances Balances { get; private set; }
        public bool HasLevel { get; private set; }
        public int Level { get; private set; }

        public event Action Changed;

        public void Show(Balances balances)
        {
            if (HasBalances && Balances.Coins.Amount == balances.Coins.Amount && Balances.Crystals.Amount == balances.Crystals.Amount) return;
            HasBalances = true;
            Balances = balances;
            Changed?.Invoke();
        }

        public void ShowLevel(int level)
        {
            if (HasLevel && Level == level) return;
            HasLevel = true;
            Level = level;
            Changed?.Invoke();
        }

        public void Clear()
        {
            if (!HasBalances && !HasLevel) return;
            HasBalances = false;
            Balances = default;
            HasLevel = false;
            Level = 0;
            Changed?.Invoke();
        }
    }
}
