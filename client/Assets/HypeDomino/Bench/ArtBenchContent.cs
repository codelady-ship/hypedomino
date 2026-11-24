using System;
using HypeDomino.UI.Board;
using UnityEngine;

namespace HypeDomino.Bench
{
    [Serializable]
    public sealed class ArtBenchLoop
    {
        public string name;
        public Sprite still;
        public Sprite[] frames = Array.Empty<Sprite>();

        public bool IsAnimated => frames != null && frames.Length > 0;
    }

    [Serializable]
    public sealed class ArtBenchTable
    {
        public string name;
        public Sprite still;
        public ArtBenchLoop[] layers = Array.Empty<ArtBenchLoop>();
    }

    [Serializable]
    public sealed class ArtBenchContent
    {
        public ArtBenchLoop[] avatars = Array.Empty<ArtBenchLoop>();
        public ArtBenchLoop[] frames = Array.Empty<ArtBenchLoop>();
        public ArtBenchTable[] tables = Array.Empty<ArtBenchTable>();
        public ArtBenchLoop[] emoji = Array.Empty<ArtBenchLoop>();
        public ArtBenchLoop[] tileBacks = Array.Empty<ArtBenchLoop>();
        public Sprite[] tileFaces = Array.Empty<Sprite>();
        public TileFaceSet tileFaceSet;
        public ArtBenchLoop[] medals = Array.Empty<ArtBenchLoop>();
        public Sprite[] effectSprites = Array.Empty<Sprite>();
        public Sprite[] kit = Array.Empty<Sprite>();
        public Material additive;
        public Material alpha;
        public TextAsset effectTable;
        public AudioClip[] sfx = Array.Empty<AudioClip>();
        public AudioClip[] music = Array.Empty<AudioClip>();
    }
}
