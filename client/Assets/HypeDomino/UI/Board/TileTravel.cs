using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.UI.Board
{
    public static class TileTravel
    {
        static readonly Vector3 Squashed = new Vector3(TokenMotion.PressScale, TokenMotion.PressScale, 1f);

        public static float DurationMs(bool reduceMotion) => reduceMotion ? TokenDurationMs.Fast : TokenDurationMs.Base;

        public static float PlaceMs(bool reduceMotion) => reduceMotion ? DurationMs(true) : DurationMs(false) + TokenDurationMs.Micro;

        public static void Start<TSink>(TweenPool pool, ref TSink sink, int tile, Vector3 fromPosition, float fromAngle, Vector3 toPosition, float toAngle, float nowMs, bool reduceMotion)
            where TSink : ITweenSink
        {
            var duration = DurationMs(reduceMotion);
            var turnedTo = fromAngle + Mathf.DeltaAngle(fromAngle, toAngle);
            pool.Start(new Tween(tile, TweenChannel.Position, fromPosition, toPosition, nowMs, duration, TokenEasing.Standard, false), ref sink);
            pool.Start(new Tween(tile, TweenChannel.Rotation, new Vector3(0f, 0f, fromAngle), new Vector3(0f, 0f, turnedTo), nowMs, duration, TokenEasing.Standard, false), ref sink);
            if (reduceMotion) return;
            pool.Start(new Tween(tile, TweenChannel.Scale, Squashed, Vector3.one, nowMs + duration, TokenDurationMs.Micro, TokenEasing.Overshoot, true), ref sink);
        }

        public static void Arrive<TSink>(TweenPool pool, ref TSink sink, int tile, int order, Vector3 fromPosition, Vector3 toPosition, float nowMs, bool reduceMotion)
            where TSink : ITweenSink
        {
            var startMs = nowMs + order * TokenDurationMs.Micro;
            var curve = reduceMotion ? TokenEasing.Standard : TokenEasing.Enter;
            pool.Start(new Tween(tile, TweenChannel.Position, fromPosition, toPosition, startMs, DurationMs(reduceMotion), curve, false), ref sink);
        }
    }
}
