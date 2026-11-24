using HypeDomino.UI.Effects;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using Unity.Profiling;
using UnityEngine;

namespace HypeDomino.UI.Board
{
    public sealed class TilePlaceLayer : MonoBehaviour, IEffectSettingsTarget
    {
        public const string ProfilerMarkerName = "Effect.tilePlace";

        static readonly ProfilerMarker TickMarker = new(ProfilerMarkerName);

        TweenPool pool;
        TransformTweenSink sink;
        TilePlaceQueue queue;
        Transform[] placed;
        EffectPlayer effects;
        EffectLevel level = EffectLevel.Medium;
        bool reduceMotion;

        public int MovingCount => pool.Count;
        public int Capacity => pool.Capacity;
        public float SettledAtMs => queue.SettledAtMs;
        public Transform LastPlaced { get; private set; }

        public void Build(Transform[] tiles)
        {
            placed = tiles;
            sink = new TransformTweenSink(tiles);
            pool = TweenPool.For(level);
        }

        public void Configure(EffectLevel effectLevel, bool reduce)
        {
            reduceMotion = reduce;
            if (effectLevel == level) return;
            level = effectLevel;
            pool.FinishAll(ref sink);
            pool = TweenPool.For(level);
        }

        public void Attach(EffectPlayer player) => effects = player;

        public void Place(int tile, Vector3 fromPosition, float fromAngle, Vector3 toPosition, float toAngle, float nowMs)
        {
            queue.Place(pool, ref sink, tile, fromPosition, fromAngle, toPosition, toAngle, nowMs, reduceMotion);
            LastPlaced = placed[tile];
            if (effects == null || queue.SettledAtMs <= nowMs) return;
            var board = placed[tile].parent;
            var startMs = queue.SettledAtMs - TileTravel.PlaceMs(reduceMotion);
            var reach = placed[tile] is RectTransform rect ? EffectRequest.ReachOf(rect) : Vector2.zero;
            var at = board != null ? board.TransformPoint(toPosition) : toPosition;
            var from = board != null ? board.TransformPoint(fromPosition) : fromPosition;
            effects.Play(EffectRequest.On(TokenEffectKind.TilePlace, at, reach, startMs).TrailFrom(from, TileTravel.DurationMs(reduceMotion)));
        }

        public void Tick(float nowMs)
        {
            using (TickMarker.Auto()) pool.Step(nowMs, ref sink);
        }

        void Update()
        {
            if (pool != null) Tick(Time.unscaledTime * 1000f);
        }
    }
}
