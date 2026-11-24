using UnityEngine;

namespace HypeDomino.UI.Motion
{
    public static class Easing
    {
        const int Steps = 20;

        public static float Evaluate(Vector4 curve, float t)
        {
            if (t <= 0f) return 0f;
            if (t >= 1f) return 1f;
            var low = 0f;
            var high = 1f;
            var s = t;
            for (var i = 0; i < Steps; i++)
            {
                s = (low + high) * 0.5f;
                if (Bezier(curve.x, curve.z, s) < t) low = s;
                else high = s;
            }
            return Bezier(curve.y, curve.w, s);
        }

        static float Bezier(float first, float second, float s)
        {
            var rest = 1f - s;
            return 3f * rest * rest * s * first + 3f * rest * s * s * second + s * s * s;
        }
    }
}
