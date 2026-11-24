using UnityEngine;
using UnityEngine.Sprites;
using UnityEngine.UI;

namespace HypeDomino.UI.Cosmetics
{
    public sealed class TableImage : Image
    {
        readonly TableQuad[] quads = new TableQuad[TableCloth.MaxQuads];

        public int QuadCount { get; private set; }

        public bool IsTurned { get; private set; }

        public TableQuad Quad(int index) => quads[index];

        protected override void OnPopulateMesh(VertexHelper toFill)
        {
            if (sprite == null)
            {
                QuadCount = 0;
                base.OnPopulateMesh(toFill);
                return;
            }
            toFill.Clear();
            var area = GetPixelAdjustedRect();
            var size = sprite.rect.size;
            QuadCount = TableCloth.Layout(size, sprite.border, area.size, quads, out var turned);
            IsTurned = turned;
            var outer = DataUtility.GetOuterUV(sprite);
            for (var at = 0; at < QuadCount; at++) AddQuad(toFill, quads[at], area, size, outer, turned);
        }

        void AddQuad(VertexHelper toFill, TableQuad quad, Rect area, Vector2 size, Vector4 outer, bool turned)
        {
            var first = toFill.currentVertCount;
            AddCorner(toFill, quad, area, size, outer, turned, false, false);
            AddCorner(toFill, quad, area, size, outer, turned, false, true);
            AddCorner(toFill, quad, area, size, outer, turned, true, true);
            AddCorner(toFill, quad, area, size, outer, turned, true, false);
            toFill.AddTriangle(first, first + 1, first + 2);
            toFill.AddTriangle(first + 2, first + 3, first);
        }

        void AddCorner(VertexHelper toFill, TableQuad quad, Rect area, Vector2 size, Vector4 outer, bool turned, bool right, bool top)
        {
            var target = new Vector2(right ? quad.Target.xMax : quad.Target.xMin, top ? quad.Target.yMax : quad.Target.yMin);
            var source = new Vector2(right ? quad.Source.xMax : quad.Source.xMin, top ? quad.Source.yMax : quad.Source.yMin);
            var position = area.position + TableCloth.ToRect(target, area.size, turned);
            var uv = new Vector2(Mathf.Lerp(outer.x, outer.z, source.x / size.x), Mathf.Lerp(outer.y, outer.w, source.y / size.y));
            Color32 tint = color;
            tint.a = (byte)Mathf.RoundToInt(tint.a * quad.AlphaAt(right, top));
            toFill.AddVert(position, tint, uv);
        }
    }
}
