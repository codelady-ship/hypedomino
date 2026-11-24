using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.UI.Board
{
    public struct TilePlaceQueue
    {
        public float SettledAtMs { get; private set; }

        public void Place<TSink>(TweenPool pool, ref TSink sink, int tile, Vector3 fromPosition, float fromAngle, Vector3 toPosition, float toAngle, float nowMs, bool reduceMotion)
            where TSink : ITweenSink
        {
            var startMs = Mathf.Max(nowMs, SettledAtMs);
            var settledAtMs = startMs + TileTravel.PlaceMs(reduceMotion);
            if (settledAtMs - nowMs > TokenMotion.MaxPresentationLagMs)
            {
                JumpToFinalState(pool, ref sink, tile, toPosition, toAngle, nowMs);
                return;
            }
            sink.Apply(tile, TweenChannel.Position, fromPosition);
            sink.Apply(tile, TweenChannel.Rotation, new Vector3(0f, 0f, fromAngle));
            TileTravel.Start(pool, ref sink, tile, fromPosition, fromAngle, toPosition, toAngle, startMs, reduceMotion);
            SettledAtMs = settledAtMs;
        }

        void JumpToFinalState<TSink>(TweenPool pool, ref TSink sink, int tile, Vector3 toPosition, float toAngle, float nowMs)
            where TSink : ITweenSink
        {
            pool.FinishAll(ref sink);
            sink.Apply(tile, TweenChannel.Position, toPosition);
            sink.Apply(tile, TweenChannel.Rotation, new Vector3(0f, 0f, toAngle));
            sink.Apply(tile, TweenChannel.Scale, Vector3.one);
            SettledAtMs = nowMs;
        }
    }
}
