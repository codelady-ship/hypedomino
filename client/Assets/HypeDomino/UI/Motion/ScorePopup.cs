using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.UI.Motion
{
    public readonly struct ScorePopup
    {
        ScorePopup(float scale, float alpha)
        {
            Scale = scale;
            Alpha = alpha;
        }

        public float Scale { get; }

        public float Alpha { get; }

        public static float LifetimeMs(int holdMs) => TokenDurationMs.Fast + holdMs + TokenDurationMs.Base;

        public static bool At(float elapsedMs, int holdMs, bool reduceMotion, out ScorePopup popup)
        {
            var lifetimeMs = LifetimeMs(holdMs);
            if (elapsedMs >= lifetimeMs)
            {
                popup = new ScorePopup(reduceMotion ? 1f : 0f, 0f);
                return false;
            }
            var enter = elapsedMs / TokenDurationMs.Fast;
            var exitStartMs = lifetimeMs - TokenDurationMs.Base;
            var leaving = elapsedMs <= exitStartMs ? 0f : Easing.Evaluate(TokenEasing.Exit, (elapsedMs - exitStartMs) / TokenDurationMs.Base);
            if (reduceMotion)
            {
                popup = new ScorePopup(1f, Mathf.Min(Easing.Evaluate(TokenEasing.Standard, enter), 1f - leaving));
                return true;
            }
            popup = new ScorePopup(elapsedMs <= exitStartMs ? Easing.Evaluate(TokenEasing.Overshoot, enter) : 1f - leaving, 1f);
            return true;
        }
    }
}
