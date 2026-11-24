using System;
using HypeDomino.UI.Match;
using HypeDomino.UI.Tokens;

namespace HypeDomino.UI.Emoji
{
    public sealed class SeatMessages
    {
        readonly string[] texts = new string[TableSides.Count];
        readonly long[] untilMs = new long[TableSides.Count];

        public void Show(TableSide side, string text, long nowMs)
        {
            texts[(int)side] = text;
            untilMs[(int)side] = nowMs + TokenEffect.EmojiDurationMs;
        }

        public bool TryGet(TableSide side, long nowMs, out string text)
        {
            text = texts[(int)side];
            if (text != null && nowMs < untilMs[(int)side]) return true;
            text = null;
            return false;
        }

        public void Clear() => Array.Clear(texts, 0, texts.Length);
    }
}
