using UnityEngine;

namespace HypeDomino.UI.Motion
{
    public enum TweenChannel
    {
        Position,
        Rotation,
        Scale,
        Alpha,
    }

    public interface ITweenSink
    {
        void Apply(int target, TweenChannel channel, Vector3 value);
    }

    public readonly struct Tween
    {
        public readonly int Target;
        public readonly TweenChannel Channel;
        public readonly Vector3 From;
        public readonly Vector3 To;
        public readonly float StartMs;
        public readonly float DurationMs;
        public readonly Vector4 Curve;
        public readonly bool Cosmetic;

        public Tween(int target, TweenChannel channel, Vector3 from, Vector3 to, float startMs, float durationMs, Vector4 curve, bool cosmetic)
        {
            Target = target;
            Channel = channel;
            From = from;
            To = to;
            StartMs = startMs;
            DurationMs = durationMs;
            Curve = curve;
            Cosmetic = cosmetic;
        }

        public float EndMs => StartMs + DurationMs;

        public bool Moves(int target, TweenChannel channel) => Target == target && Channel == channel;

        public Vector3 ValueAt(float nowMs)
        {
            if (nowMs >= EndMs) return To;
            if (nowMs <= StartMs) return From;
            return Vector3.LerpUnclamped(From, To, Easing.Evaluate(Curve, (nowMs - StartMs) / DurationMs));
        }
    }
}
