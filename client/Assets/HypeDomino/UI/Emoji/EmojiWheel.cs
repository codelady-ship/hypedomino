using System;
using System.Collections.Generic;
using HypeDomino.UI.Tokens;

namespace HypeDomino.UI.Emoji
{
    public sealed class EmojiWheel
    {
        readonly string[] slots;
        long calmUntilMs = long.MinValue;

        public EmojiWheel(int capacity) => slots = new string[capacity];

        public int Count { get; private set; }
        public bool Visible => Count > 0;
        public bool IsOpen { get; private set; }

        public string Slot(int index) => slots[index];

        public bool Calm(long nowMs) => nowMs < calmUntilMs;

        public void Equip(IReadOnlyList<string> equipped) => Equip(equipped, slots.Length);

        public void Equip(IReadOnlyList<string> equipped, int slotCount)
        {
            Count = Math.Max(0, Math.Min(Math.Min(equipped.Count, slotCount), slots.Length));
            for (var i = 0; i < slots.Length; i++) slots[i] = i < Count ? equipped[i] : null;
            if (Count == 0) IsOpen = false;
        }

        public void Toggle() => IsOpen = !IsOpen && Visible;

        public void Close() => IsOpen = false;

        public bool TrySend(int index, long nowMs, out string item)
        {
            item = null;
            if (!IsOpen || index < 0 || index >= Count || Calm(nowMs)) return false;
            item = slots[index];
            IsOpen = false;
            calmUntilMs = nowMs + TokenEffect.EmojiDurationMs;
            return true;
        }

        public void OwnEmojiShown(long nowMs) => calmUntilMs = nowMs + TokenEffect.EmojiDurationMs;
    }
}
