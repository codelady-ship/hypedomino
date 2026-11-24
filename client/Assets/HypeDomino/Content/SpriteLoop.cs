using System;
using UnityEngine;

namespace HypeDomino.Content
{
    public sealed class SpriteLoop : ScriptableObject
    {
        [SerializeField] Sprite[] frames = Array.Empty<Sprite>();

        public Sprite[] Frames => frames;

        public void SetFrames(Sprite[] value) => frames = value ?? Array.Empty<Sprite>();
    }
}
