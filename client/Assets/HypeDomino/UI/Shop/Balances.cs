using System;

namespace HypeDomino.UI.Shop
{
    public enum Currency
    {
        Coin,
        Crystal,
    }

    public readonly struct Balance
    {
        public Balance(Currency currency, long amount)
        {
            Currency = currency;
            Amount = amount;
        }

        public Currency Currency { get; }
        public long Amount { get; }
        public string NameKey => CurrencyKeys.Name(Currency);
    }

    public static class CurrencyKeys
    {
        public const string Coin = "currency.coin.name";
        public const string Crystal = "currency.crystal.name";

        public static string Name(Currency currency) => currency == Currency.Crystal ? Crystal : Coin;

        public static bool TryParse(string wire, out Currency currency)
        {
            currency = wire == "crystal" ? Currency.Crystal : Currency.Coin;
            return wire == "coin" || wire == "crystal";
        }
    }

    public readonly struct Balances
    {
        public Balances(WalletReply reply)
        {
            Coins = new Balance(Currency.Coin, reply.coins);
            Crystals = new Balance(Currency.Crystal, reply.crystals);
        }

        public Balance Coins { get; }
        public Balance Crystals { get; }

        public static bool TryRead(string walletBody, out Balances balances)
        {
            balances = default;
            if (string.IsNullOrEmpty(walletBody))
            {
                return false;
            }
            try
            {
                var reply = WalletReply.FromJson(walletBody);
                if (reply == null)
                {
                    return false;
                }
                balances = new Balances(reply);
                return true;
            }
            catch (ArgumentException)
            {
                return false;
            }
        }

        public Balances After(PurchaseReply purchase)
        {
            if (!CurrencyKeys.TryParse(purchase.currency, out var currency))
            {
                return this;
            }
            return currency == Currency.Crystal
                ? new Balances(Coins, new Balance(Currency.Crystal, purchase.balance))
                : new Balances(new Balance(Currency.Coin, purchase.balance), Crystals);
        }

        Balances(Balance coins, Balance crystals)
        {
            Coins = coins;
            Crystals = crystals;
        }
    }
}
