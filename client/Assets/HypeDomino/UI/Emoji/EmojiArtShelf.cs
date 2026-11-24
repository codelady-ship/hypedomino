using System;
using System.Collections.Generic;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Match;
using HypeDomino.UI.Motion;
using UnityEngine;

namespace HypeDomino.UI.Emoji
{
    public sealed class EmojiArtShelf
    {
        sealed class Shelved
        {
            public CosmeticArt Art;
            public bool StillSettled;
            public LoopArt Still;
            public LoopArt Loop;
            public bool HoldsLoop;
        }

        readonly CosmeticManifest manifest;
        readonly CosmeticStage stage;
        readonly Action<TableSide, LoopArt> show;
        readonly Dictionary<string, Shelved> shelved = new();
        readonly string[] waiting = new string[TableSides.Count];

        public EmojiArtShelf(CosmeticManifest manifest, CosmeticStage stage, Action<TableSide, LoopArt> show)
        {
            this.manifest = manifest;
            this.stage = stage;
            this.show = show;
        }

        public int Count => shelved.Count;

        public bool IsWaiting(TableSide side) => waiting[(int)side] != null;

        public void Prefetch(IReadOnlyList<string> itemIds)
        {
            for (var at = 0; at < itemIds.Count; at++) Prefetch(itemIds[at]);
        }

        public void Prefetch(string itemId) => TryShelve(itemId, out _);

        public bool Show(TableSide side, string itemId)
        {
            waiting[(int)side] = null;
            if (!TryShelve(itemId, out var entry)) return false;
            if (!entry.StillSettled)
            {
                waiting[(int)side] = itemId;
                return true;
            }
            if (entry.Still == null) return false;
            show(side, Pick(entry));
            return true;
        }

        public void Clear()
        {
            Array.Clear(waiting, 0, waiting.Length);
            foreach (var pair in shelved) Release(pair.Value);
            shelved.Clear();
        }

        bool TryShelve(string itemId, out Shelved entry)
        {
            if (!shelved.TryGetValue(itemId ?? string.Empty, out entry))
            {
                if (!IsShownEmoji(itemId, out var item)) return false;
                entry = Shelve(itemId, item);
            }
            if (WantsLoop(entry)) AcquireLoop(itemId, entry);
            return true;
        }

        Shelved Shelve(string itemId, CosmeticItem item)
        {
            var entry = new Shelved { Art = item.Art };
            shelved.Add(itemId, entry);
            stage.Stills.Acquire(entry.Art.StaticAddress, sprite => OnStill(itemId, entry, sprite));
            return entry;
        }

        bool IsShownEmoji(string itemId, out CosmeticItem item) =>
            manifest.TryGet(itemId, out item) && item.Type == CosmeticAddresses.Emoji && item.IsRendered;

        bool WantsLoop(Shelved entry) => entry.Art.IsAnimated && !entry.HoldsLoop && stage.Level != EffectLevel.Low;

        void AcquireLoop(string itemId, Shelved entry)
        {
            entry.HoldsLoop = true;
            stage.Loops.Acquire(entry.Art.AnimationAddress, loop => OnLoop(itemId, entry, loop));
        }

        LoopArt Pick(Shelved entry) => entry.Loop != null && stage.Level != EffectLevel.Low ? entry.Loop : entry.Still;

        void OnStill(string itemId, Shelved entry, Sprite sprite)
        {
            entry.StillSettled = true;
            entry.Still = sprite == null ? null : new LoopArt(sprite, null, 0f);
            if (!IsCurrent(itemId, entry)) return;
            for (var side = 0; side < waiting.Length; side++)
            {
                if (waiting[side] != itemId) continue;
                waiting[side] = null;
                if (entry.Still != null) show((TableSide)side, Pick(entry));
            }
        }

        void OnLoop(string itemId, Shelved entry, LoopArt loop)
        {
            if (IsCurrent(itemId, entry)) entry.Loop = loop;
        }

        bool IsCurrent(string itemId, Shelved entry) => shelved.TryGetValue(itemId, out var current) && ReferenceEquals(current, entry);

        void Release(Shelved entry)
        {
            stage.Stills.Release(entry.Art.StaticAddress);
            if (entry.HoldsLoop) stage.Loops.Release(entry.Art.AnimationAddress);
        }
    }
}
