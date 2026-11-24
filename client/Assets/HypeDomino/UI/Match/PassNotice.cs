using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.UI.Match
{
    public readonly struct PassNotice
    {
        public const float LifetimeMs = TokenDurationMs.Fast + TokenDurationMs.Slow + TokenDurationMs.Base;
        const float ExitStartMs = LifetimeMs - TokenDurationMs.Base;

        PassNotice(float scale, float alpha)
        {
            Scale = scale;
            Alpha = alpha;
        }

        public float Scale { get; }

        public float Alpha { get; }

        public static bool At(float elapsedMs, bool reduceMotion, out PassNotice notice)
        {
            if (elapsedMs >= LifetimeMs)
            {
                notice = new PassNotice(1f, 0f);
                return false;
            }
            var alpha = FadeOut(elapsedMs);
            var enter = elapsedMs / TokenDurationMs.Fast;
            notice = reduceMotion
                ? new PassNotice(1f, Mathf.Min(alpha, Easing.Evaluate(TokenEasing.Standard, enter)))
                : new PassNotice(Easing.Evaluate(TokenEasing.Overshoot, enter), alpha);
            return true;
        }

        static float FadeOut(float elapsedMs)
        {
            if (elapsedMs <= ExitStartMs) return 1f;
            return 1f - Easing.Evaluate(TokenEasing.Exit, (elapsedMs - ExitStartMs) / TokenDurationMs.Base);
        }
    }
}
