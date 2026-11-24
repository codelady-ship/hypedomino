using System;
using HypeDomino.UI.Tokens;

namespace HypeDomino.UI.Motion
{
    public sealed class TweenPool
    {
        readonly Tween[] tweens;
        int count;

        public TweenPool(int capacity) => tweens = new Tween[capacity];

        public int Count => count;

        public int Capacity => tweens.Length;

        public static TweenPool For(EffectLevel level) => new TweenPool(MaxTweensOf(level));

        public bool IsMoving(int target, TweenChannel channel) => IndexOf(target, channel) >= 0;

        public void Start<TSink>(in Tween tween, ref TSink sink) where TSink : ITweenSink
        {
            var moving = IndexOf(tween.Target, tween.Channel);
            if (moving >= 0) RemoveAt(moving);
            if (count == tweens.Length) Finish(OldestToFinish(), ref sink);
            if (count == tweens.Length)
            {
                sink.Apply(tween.Target, tween.Channel, tween.To);
                return;
            }
            tweens[count++] = tween;
        }

        public void Step<TSink>(float nowMs, ref TSink sink) where TSink : ITweenSink
        {
            var at = 0;
            while (at < count)
            {
                var tween = tweens[at];
                if (nowMs < tween.StartMs)
                {
                    at++;
                    continue;
                }
                sink.Apply(tween.Target, tween.Channel, tween.ValueAt(nowMs));
                if (nowMs >= tween.EndMs) RemoveAt(at);
                else at++;
            }
        }

        public void FinishAll<TSink>(ref TSink sink) where TSink : ITweenSink
        {
            for (var at = 0; at < count; at++) sink.Apply(tweens[at].Target, tweens[at].Channel, tweens[at].To);
            count = 0;
        }

        int OldestToFinish()
        {
            if (count == 0) return -1;
            for (var at = 0; at < count; at++)
                if (tweens[at].Cosmetic) return at;
            return 0;
        }

        void Finish<TSink>(int at, ref TSink sink) where TSink : ITweenSink
        {
            if (at < 0) return;
            sink.Apply(tweens[at].Target, tweens[at].Channel, tweens[at].To);
            RemoveAt(at);
        }

        int IndexOf(int target, TweenChannel channel)
        {
            for (var at = 0; at < count; at++)
                if (tweens[at].Moves(target, channel)) return at;
            return -1;
        }

        void RemoveAt(int at)
        {
            count--;
            if (at < count) Array.Copy(tweens, at + 1, tweens, at, count - at);
        }

        static int MaxTweensOf(EffectLevel level)
        {
            switch (level)
            {
                case EffectLevel.High: return TokenEffectLevel.HighMaxTweens;
                case EffectLevel.Medium: return TokenEffectLevel.MediumMaxTweens;
                default: return TokenEffectLevel.LowMaxTweens;
            }
        }
    }
}
