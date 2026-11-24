using System;
using System.Collections.Generic;

namespace HypeDomino.UI.Cosmetics
{
    public readonly struct CollectionEntry
    {
        public CollectionEntry(CosmeticItem item, bool held, bool equipped)
        {
            Item = item;
            Held = held;
            Equipped = equipped;
        }

        public CosmeticItem Item { get; }

        public bool Held { get; }

        public bool Equipped { get; }

        public bool Obtainable => Item.IsLive;

        public IReadOnlyList<string> EarnPaths => Held || !Obtainable ? Array.Empty<string>() : Item.Sources;
    }

    public sealed class CosmeticCollection
    {
        public const string InventoryPath = "/inventory";
        public const string EquipPath = "/inventory/equip";

        static readonly LoadoutReply NothingWorn = new();

        readonly CosmeticManifest manifest;
        readonly HashSet<string> held = new();
        readonly List<CollectionEntry> shown = new();
        LoadoutReply loadout = NothingWorn;
        int emojiSlots;
        string type;
        string rarityId;

        public CosmeticCollection(CosmeticManifest manifest) => this.manifest = manifest;

        public int Count => shown.Count;

        public CollectionEntry this[int index] => shown[index];

        public void Show(InventoryReply reply)
        {
            held.Clear();
            if (reply?.items != null)
                foreach (var item in reply.items)
                    if (item != null && !string.IsNullOrEmpty(item.id)) held.Add(item.id);
            loadout = reply?.loadout ?? NothingWorn;
            emojiSlots = reply?.emoji_slots ?? 0;
            Refresh();
        }

        public void Apply(EquipReply reply)
        {
            if (reply?.loadout == null) return;
            loadout = reply.loadout;
            Refresh();
        }

        public void Filter(string itemType, string tierId)
        {
            type = itemType;
            rarityId = string.IsNullOrEmpty(tierId) ? null : tierId;
            Refresh();
        }

        public bool TryEquip(int index, out EquipRequest request)
        {
            request = null;
            if (index < 0 || index >= shown.Count) return false;
            var entry = shown[index];
            if (!entry.Held) return false;
            if (entry.Item.Type == CosmeticAddresses.Emoji) return TryToggleEmoji(entry, out request);
            if (entry.Equipped) return false;
            request = new EquipRequest { slot = entry.Item.Type, items = new[] { entry.Item.Id } };
            return true;
        }

        bool TryToggleEmoji(CollectionEntry entry, out EquipRequest request)
        {
            request = null;
            var worn = new List<string>(loadout.emojis ?? Array.Empty<string>());
            if (entry.Equipped) worn.Remove(entry.Item.Id);
            else if (worn.Count >= emojiSlots) return false;
            else worn.Add(entry.Item.Id);
            request = new EquipRequest { slot = CosmeticAddresses.Emoji, items = worn.ToArray() };
            return true;
        }

        void Refresh()
        {
            shown.Clear();
            if (manifest == null || type == null) return;
            for (var at = 0; at < manifest.Count; at++)
            {
                var item = manifest[at];
                if (item.Type != type || !IsShown(item)) continue;
                if (rarityId != null && item.RarityId != rarityId) continue;
                shown.Add(new CollectionEntry(item, held.Contains(item.Id), loadout.Wears(item)));
            }
        }

        static bool IsShown(CosmeticItem item) => item.IsLive || item.Availability == CosmeticManifest.Retired;
    }
}
