using System;
using System.Collections.Generic;
using UnityEngine;

namespace HypeDomino.UI.Cosmetics
{
    public sealed class CosmeticItem
    {
        public CosmeticItem(string id, string type, string rarityId, string nameKey, string availability, string[] sources, CosmeticArt art, TableLook table)
        {
            Id = id;
            Type = type;
            RarityId = rarityId;
            NameKey = nameKey;
            Availability = availability;
            Sources = sources;
            IsDefault = Array.IndexOf(sources, CosmeticManifest.DefaultSource) >= 0;
            Art = art;
            Table = table;
        }

        public string Id { get; }

        public string Type { get; }

        public string RarityId { get; }

        public string NameKey { get; }

        public string Availability { get; }

        public IReadOnlyList<string> Sources { get; }

        public bool IsDefault { get; }

        public bool IsLive => Availability == CosmeticManifest.Live;

        public bool IsRendered => IsLive || Availability == CosmeticManifest.Retired;

        public CosmeticArt Art { get; }

        public TableLook Table { get; }
    }

    public sealed class CosmeticManifest
    {
        public const int KnownSchemaVersion = 1;
        public const string Live = "live";
        public const string Retired = "retired";
        public const string DefaultSource = "default";
        const string BuiltinResource = "cosmetics-builtin";

        static readonly string[] KnownSources = { DefaultSource, "shop", "pass", "mission", "promo", "rank_reward", "first_purchase", "event", "chest" };

        static readonly string[] KnownTypes = { CosmeticAddresses.Avatar, CosmeticAddresses.Frame, CosmeticAddresses.Table, CosmeticAddresses.Tile, CosmeticAddresses.Emoji };

        [Serializable]
        sealed class SourceEntry
        {
            public string type;
        }

        [Serializable]
        sealed class LayerEntry
        {
            public string name;
            public string address;
            public bool animated;
        }

        [Serializable]
        sealed class ItemEntry
        {
            public string id;
            public string type;
            public string rarityId;
            public string nameKey;
            public string availability;
            public bool animated;
            public string staticAddress;
            public string animationAddress;
            public LayerEntry[] layers;
            public SourceEntry[] sources;
        }

        [Serializable]
        sealed class ManifestFile
        {
            public int schemaVersion;
            public int manifestVersion;
            public ItemEntry[] items;
        }

        readonly List<CosmeticItem> items = new();
        readonly Dictionary<string, CosmeticItem> byId = new();

        CosmeticManifest(int manifestVersion) => ManifestVersion = manifestVersion;

        public int ManifestVersion { get; }

        public int Count => items.Count;

        public CosmeticItem this[int index] => items[index];

        public bool TryGet(string id, out CosmeticItem item)
        {
            item = null;
            return !string.IsNullOrEmpty(id) && byId.TryGetValue(id, out item);
        }

        public CosmeticItem DefaultOf(string type)
        {
            foreach (var item in items)
                if (item.Type == type && item.IsDefault && item.IsLive) return item;
            return null;
        }

        public CosmeticItem ShownOf(string type, string id)
        {
            if (TryGet(id, out var item) && item.Type == type && item.IsRendered) return item;
            return DefaultOf(type);
        }

        public static CosmeticManifest LoadBuiltin()
        {
            var asset = Resources.Load<TextAsset>(BuiltinResource);
            if (asset == null)
            {
                Debug.LogError($"Cosmetics builtin missing resource={BuiltinResource}");
                return null;
            }
            if (!TryParse(asset.text, out var manifest)) Debug.LogError($"Cosmetics builtin refused resource={BuiltinResource}");
            Resources.UnloadAsset(asset);
            return manifest;
        }

        public static bool TryParse(string json, out CosmeticManifest manifest)
        {
            manifest = null;
            ManifestFile file;
            try
            {
                file = JsonUtility.FromJson<ManifestFile>(json);
            }
            catch (ArgumentException)
            {
                return false;
            }
            if (file == null || file.schemaVersion != KnownSchemaVersion || file.items == null || file.items.Length == 0) return false;
            manifest = new CosmeticManifest(file.manifestVersion);
            foreach (var entry in file.items) manifest.Add(entry);
            return true;
        }

        void Add(ItemEntry entry)
        {
            if (string.IsNullOrEmpty(entry.id) || string.IsNullOrEmpty(entry.staticAddress)) return;
            if (Array.IndexOf(KnownTypes, entry.type) < 0 || byId.ContainsKey(entry.id)) return;
            var item = new CosmeticItem(entry.id, entry.type, entry.rarityId, entry.nameKey, entry.availability, Sources(entry), Art(entry), Table(entry));
            items.Add(item);
            byId.Add(item.Id, item);
        }

        static CosmeticArt Art(ItemEntry entry) =>
            new(entry.staticAddress, entry.animated && entry.type != CosmeticAddresses.Table ? entry.animationAddress : null);

        static TableLook Table(ItemEntry entry)
        {
            if (entry.type != CosmeticAddresses.Table || !entry.animated || entry.layers == null) return new TableLook(entry.staticAddress, null);
            var layers = new TableLayer[entry.layers.Length];
            for (var at = 0; at < layers.Length; at++)
            {
                var layer = entry.layers[at];
                layers[at] = new TableLayer(layer.name, layer.address, layer.animated);
            }
            return new TableLook(entry.staticAddress, layers);
        }

        static string[] Sources(ItemEntry entry)
        {
            var known = new List<string>();
            if (entry.sources != null)
                foreach (var source in entry.sources)
                    if (source != null && Array.IndexOf(KnownSources, source.type) >= 0 && !known.Contains(source.type)) known.Add(source.type);
            return known.ToArray();
        }
    }
}
