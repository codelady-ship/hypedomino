using System;
using System.Collections.Generic;
using UnityEngine;

namespace HypeDomino.UI.Shop
{
    [Serializable]
    public sealed class WalletReply
    {
        public long coins;
        public long crystals;

        public static WalletReply FromJson(string json) => JsonUtility.FromJson<WalletReply>(json);
    }

    [Serializable]
    public sealed class ChestOddsReply
    {
        public string rarity;
        public int percent;
    }

    [Serializable]
    public sealed class OfferReply
    {
        public string id;
        public string kind;
        public string name_key;
        public long price_coins;
        public long price_crystals;
        public string product_id;
        public long reference_price_coins;
        public long reference_price_crystals;
        public long coins;
        public long crystals;
        public string[] cosmetics;
        public int pass_xp;
        public string ends_at;
        public int limit;
        public int bought;
        public string[] owned;
        [NonSerialized] public Dictionary<string, long> compensation;
        public bool purchasable;
        public string reason;
        public ChestOddsReply[] odds;
        public int opens_to_pity;
    }

    [Serializable]
    public sealed class SectionReply
    {
        public string section;
        public string ends_at;
        public OfferReply[] offers;
    }

    [Serializable]
    public sealed class ShopReply
    {
        public long version;
        public SectionReply[] sections;

        public static ShopReply FromJson(string json) => JsonUtility.FromJson<ShopReply>(json);
    }

    [Serializable]
    public sealed class ChestOpenedReply
    {
        public string item;
        public string rarity;
        public int opens_to_pity;
    }

    [Serializable]
    public sealed class PurchaseReply
    {
        public string offer;
        public string currency;
        public long balance;
        public ChestOpenedReply chest;
        public bool repeated;

        public static PurchaseReply FromJson(string json) => JsonUtility.FromJson<PurchaseReply>(json);
    }
}
