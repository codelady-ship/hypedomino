using System;

namespace HypeDomino.UI.Shop
{
    public readonly struct ChestOdds
    {
        public ChestOdds(string rarity, int percent)
        {
            Rarity = rarity;
            Percent = percent;
        }

        public string Rarity { get; }
        public int Percent { get; }
        public string RarityNameKey => RarityKeys.Name(Rarity);
    }

    public static class RarityKeys
    {
        public static string Name(string rarity) => "rarity." + rarity + ".name";
    }

    public sealed class ChestCard
    {
        public const string Kind = "chest";

        ChestCard(OfferReply offer)
        {
            OfferId = offer.id;
            NameKey = offer.name_key;
            Price = new Balance(Currency.Coin, offer.price_coins);
            Pool = offer.cosmetics ?? Array.Empty<string>();
            Owned = offer.owned ?? Array.Empty<string>();
            Odds = ToOdds(offer.odds);
            OpensToPity = offer.opens_to_pity;
            Purchasable = offer.purchasable;
            Refusal = offer.reason ?? string.Empty;
        }

        public string OfferId { get; }
        public string NameKey { get; }
        public Balance Price { get; }
        public string[] Pool { get; }
        public string[] Owned { get; }
        public ChestOdds[] Odds { get; }
        public int OpensToPity { get; }
        public bool Purchasable { get; }
        public string Refusal { get; }

        public bool IsOwned(string item) => Array.IndexOf(Owned, item) >= 0;

        public static bool TryFrom(OfferReply offer, out ChestCard card)
        {
            card = offer != null && offer.kind == Kind ? new ChestCard(offer) : null;
            return card != null;
        }

        static ChestOdds[] ToOdds(ChestOddsReply[] replies)
        {
            if (replies == null)
            {
                return Array.Empty<ChestOdds>();
            }
            var odds = new ChestOdds[replies.Length];
            for (var i = 0; i < replies.Length; i++)
            {
                odds[i] = new ChestOdds(replies[i].rarity, replies[i].percent);
            }
            return odds;
        }
    }

    public readonly struct ChestReveal
    {
        ChestReveal(ChestOpenedReply chest)
        {
            Item = chest.item;
            RarityNameKey = RarityKeys.Name(chest.rarity);
            OpensToPity = chest.opens_to_pity;
        }

        public string Item { get; }
        public string RarityNameKey { get; }
        public int OpensToPity { get; }

        public static bool TryFrom(PurchaseReply purchase, out ChestReveal reveal)
        {
            var chest = purchase?.chest;
            var opened = chest != null && !string.IsNullOrEmpty(chest.item);
            reveal = opened ? new ChestReveal(chest) : default;
            return opened;
        }
    }
}
