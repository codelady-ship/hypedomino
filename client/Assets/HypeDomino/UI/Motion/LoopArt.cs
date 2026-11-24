using System;
using UnityEngine;

namespace HypeDomino.UI.Motion
{
    public sealed class LoopArt
    {
        readonly Sprite[] loop;
        readonly float framesPerSecond;

        public LoopArt(Sprite still, Sprite[] loop, float framesPerSecond)
        {
            Still = still;
            this.loop = loop ?? Array.Empty<Sprite>();
            this.framesPerSecond = framesPerSecond;
        }

        public Sprite Still { get; }

        public bool IsAnimated => loop.Length > 0 && framesPerSecond > 0f;

        public Sprite Frame(EffectLevel level, float elapsedSeconds)
        {
            if (level == EffectLevel.Low || !IsAnimated) return Still;
            return loop[(int)(Mathf.Max(0f, elapsedSeconds) * framesPerSecond) % loop.Length];
        }
    }
}
