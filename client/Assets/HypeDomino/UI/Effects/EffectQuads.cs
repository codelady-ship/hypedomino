using System;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.UI.Effects
{
    public enum EffectMotion
    {
        Pop,
        Burst,
        Travel
    }

    public struct EffectQuad
    {
        public int Owner;
        public int Generation;
        public int Sprite;
        public bool Particle;
        public EffectMotion Motion;
        public Vector2 From;
        public Vector2 To;
        public Vector2 HalfSize;
        public float StartMs;
        public float DurationMs;
        public float HoldMs;
        public float Strength;
        public float Grow;
        public Color32 Tint;
        public Vector2 Center;
        public float Scale;
        public float Cos;
        public float Sin;
        public byte Alpha;
    }

    [RequireComponent(typeof(CanvasRenderer))]
    public sealed class EffectQuads : MaskableGraphic
    {
        EffectQuad[] quads = Array.Empty<EffectQuad>();
        Vector4[] uvs = Array.Empty<Vector4>();
        Texture atlas;

        public int Count { get; private set; }

        public int Capacity => quads.Length;

        public override Texture mainTexture => atlas != null ? atlas : s_WhiteTexture;

        public ref EffectQuad this[int at] => ref quads[at];

        public void Build(int capacity, Vector4[] spriteUvs, Texture texture, Material shared)
        {
            quads = new EffectQuad[capacity];
            uvs = spriteUvs;
            atlas = texture;
            material = shared;
            raycastTarget = false;
            Count = 0;
            SetAllDirty();
        }

        public bool TryAdd(in EffectQuad quad)
        {
            if (Count == quads.Length) return false;
            quads[Count++] = quad;
            return true;
        }

        public void RemoveAt(int at) => quads[at] = quads[--Count];

        public void Clear()
        {
            Count = 0;
            SetVerticesDirty();
        }

        protected override void OnPopulateMesh(VertexHelper vh)
        {
            vh.Clear();
            for (var at = 0; at < Count; at++)
                if (quads[at].Alpha > 0) AddQuad(vh, ref quads[at]);
        }

        void AddQuad(VertexHelper vh, ref EffectQuad quad)
        {
            var uv = uvs[quad.Sprite];
            var half = quad.HalfSize * quad.Scale;
            var right = new Vector2(quad.Cos * half.x, quad.Sin * half.x);
            var up = new Vector2(-quad.Sin * half.y, quad.Cos * half.y);
            var tint = quad.Tint;
            tint.a = quad.Alpha;
            var first = vh.currentVertCount;
            vh.AddVert(quad.Center - right - up, tint, new Vector4(uv.x, uv.y, 0f, 0f));
            vh.AddVert(quad.Center - right + up, tint, new Vector4(uv.x, uv.w, 0f, 0f));
            vh.AddVert(quad.Center + right + up, tint, new Vector4(uv.z, uv.w, 0f, 0f));
            vh.AddVert(quad.Center + right - up, tint, new Vector4(uv.z, uv.y, 0f, 0f));
            vh.AddTriangle(first, first + 1, first + 2);
            vh.AddTriangle(first + 2, first + 3, first);
        }
    }
}
