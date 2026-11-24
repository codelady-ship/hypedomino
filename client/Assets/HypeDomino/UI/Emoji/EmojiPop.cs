using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.UI.Emoji
{
    public readonly struct EmojiPop
    {
        EmojiPop(float scale, float alpha)
        {
            Scale = scale;
            Alpha = alpha;
        }

        public float Scale { get; }
        public float Alpha { get; }

        public static bool At(float elapsedMs, bool reduceMotion, out EmojiPop pop)
        {
            if (elapsedMs >= TokenEffect.EmojiDurationMs)
            {
                pop = new EmojiPop(1f, 0f);
                return false;
            }
            var alpha = FadeOut(elapsedMs);
            if (reduceMotion)
            {
                pop = new EmojiPop(1f, Mathf.Min(alpha, Easing.Evaluate(TokenEasing.Standard, elapsedMs / TokenDurationMs.Fast)));
                return true;
            }
            pop = new EmojiPop(Easing.Evaluate(TokenEasing.Overshoot, elapsedMs / TokenDurationMs.Base), alpha);
            return true;
        }

        static float FadeOut(float elapsedMs)
        {
            const float exitStartMs = TokenEffect.EmojiDurationMs - TokenDurationMs.Base;
            if (elapsedMs <= exitStartMs) return 1f;
            return 1f - Easing.Evaluate(TokenEasing.Exit, (elapsedMs - exitStartMs) / TokenDurationMs.Base);
        }
    }
}
