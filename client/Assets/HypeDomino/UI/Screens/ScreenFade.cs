using HypeDomino.UI.Motion;
using UnityEngine;

namespace HypeDomino.UI.Screens
{
    public struct ScreenFade
    {
        float from;
        float to;
        float durationMs;
        float elapsedMs;
        Vector4 curve;

        public float Value { get; private set; }

        public bool AtRest => elapsedMs >= durationMs;

        public void Retarget(float target, float fullDurationMs, Vector4 easing)
        {
            from = Value;
            to = target;
            curve = easing;
            durationMs = fullDurationMs * Mathf.Abs(target - from);
            elapsedMs = 0f;
            if (durationMs <= 0f) Value = target;
        }

        public bool Advance(float deltaMs)
        {
            if (AtRest) return true;
            elapsedMs += deltaMs;
            Value = Mathf.LerpUnclamped(from, to, Easing.Evaluate(curve, elapsedMs / durationMs));
            return AtRest;
        }
    }
}
