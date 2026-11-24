using System;
using UnityEngine;

namespace HypeDomino.UI.Cosmetics
{
    [Serializable]
    public sealed class HeldItemReply
    {
        public string id;
    }

    [Serializable]
    public sealed class LoadoutReply
    {
        public string avatar;
        public string frame;
        public string table;
        public string tile;
        public string[] emojis;

        public bool Wears(CosmeticItem item)
        {
            switch (item.Type)
            {
                case CosmeticAddresses.Avatar: return avatar == item.Id;
                case CosmeticAddresses.Frame: return frame == item.Id;
                case CosmeticAddresses.Table: return table == item.Id;
                case CosmeticAddresses.Tile: return tile == item.Id;
                case CosmeticAddresses.Emoji: return emojis != null && Array.IndexOf(emojis, item.Id) >= 0;
                default: return false;
            }
        }
    }

    [Serializable]
    public sealed class InventoryReply
    {
        public HeldItemReply[] items;
        public LoadoutReply loadout;
        public int emoji_slots;

        public static InventoryReply FromJson(string json) => JsonUtility.FromJson<InventoryReply>(json);
    }

    [Serializable]
    public sealed class EquipReply
    {
        public LoadoutReply loadout;

        public static EquipReply FromJson(string json) => JsonUtility.FromJson<EquipReply>(json);
    }

    [Serializable]
    public sealed class EquipRequest
    {
        public string slot;
        public string[] items;

        public string ToJson() => JsonUtility.ToJson(this);
    }
}
