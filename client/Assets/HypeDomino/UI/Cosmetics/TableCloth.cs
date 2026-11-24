using UnityEngine;

namespace HypeDomino.UI.Cosmetics
{
    public struct TableQuad
    {
        public Rect Target;
        public Rect Source;
        public float AlphaLeft;
        public float AlphaRight;
        public float AlphaBottom;
        public float AlphaTop;

        public float AlphaAt(bool right, bool top) => (right ? AlphaRight : AlphaLeft) * (top ? AlphaTop : AlphaBottom);
    }

    public static class TableCloth
    {
        public const float SeamFadeShare = 16f / 1080f;
        public const int MaxQuads = 15;

        public static int Layout(Vector2 spriteSize, Vector4 border, Vector2 rectSize, TableQuad[] into, out bool rotated)
        {
            rotated = rectSize.y > rectSize.x;
            var frame = rotated ? new Vector2(rectSize.y, rectSize.x) : rectSize;
            if (spriteSize.x <= 0f || spriteSize.y <= 0f || frame.x <= 0f || frame.y <= 0f) return 0;
            var unit = frame.y / spriteSize.y;
            var width = frame.x / unit;
            var depth = border.w;
            var step = spriteSize.x - border.x - border.z;
            if (depth <= 0f || step <= 0f || depth * 2f >= spriteSize.y) return Cover(spriteSize, frame, into);
            var count = Ring(spriteSize, depth, step, width, unit, into, out var across);
            return Field(spriteSize, depth, width, across, unit, into, count);
        }

        public static Vector2 ToRect(Vector2 framePoint, Vector2 rectSize, bool rotated) =>
            rotated ? new Vector2(framePoint.y, rectSize.y - framePoint.x) : framePoint;

        static int Ring(Vector2 size, float depth, float step, float width, float unit, TableQuad[] into, out float across)
        {
            var repeats = Mathf.Round((width - size.x) / step);
            var lowest = Mathf.Ceil((2f * depth - size.x) / step);
            var highest = Mathf.Floor(size.x / step);
            repeats = Mathf.Clamp(repeats, lowest, highest);
            var half = (size.x + repeats * step) / 2f;
            across = width / (half * 2f);
            Band(size, 0f, depth, half, across, unit, into, 0);
            Band(size, size.y - depth, depth, half, across, unit, into, 2);
            var middle = size.y - 2f * depth;
            into[4] = Solid(new Rect(0f, depth, depth, middle), new Rect(0f, depth, depth * across, middle), unit);
            into[5] = Solid(new Rect(size.x - depth, depth, depth, middle), new Rect(width - depth * across, depth, depth * across, middle), unit);
            return 6;
        }

        static void Band(Vector2 size, float bottom, float depth, float half, float across, float unit, TableQuad[] into, int at)
        {
            into[at] = Solid(new Rect(0f, bottom, half, depth), new Rect(0f, bottom, half * across, depth), unit);
            into[at + 1] = Solid(new Rect(size.x - half, bottom, half, depth), new Rect(half * across, bottom, half * across, depth), unit);
        }

        static int Field(Vector2 size, float depth, float width, float across, float unit, TableQuad[] into, int count)
        {
            var fieldWidth = size.x - 2f * depth;
            var fieldHeight = size.y - 2f * depth;
            var shownWidth = width - 2f * depth * across;
            var scale = Mathf.Max(shownWidth / fieldWidth, 1f);
            var sourceWidth = shownWidth / scale;
            var sourceHeight = fieldHeight / scale;
            var fade = size.y * SeamFadeShare;
            var targetX = Edges(depth * across - fade * across, depth * across, width - depth * across, width - depth * across + fade * across);
            var targetY = Edges(depth - fade, depth, size.y - depth, size.y - depth + fade);
            var sourceX = Centred(size.x / 2f, sourceWidth, fade * across / scale);
            var sourceY = Centred(size.y / 2f, sourceHeight, fade / scale);
            for (var row = 0; row < 3; row++)
            for (var column = 0; column < 3; column++)
            {
                var source = Rect.MinMaxRect(sourceX[column], sourceY[row], sourceX[column + 1], sourceY[row + 1]);
                var target = Rect.MinMaxRect(targetX[column], targetY[row], targetX[column + 1], targetY[row + 1]);
                into[count++] = new TableQuad
                {
                    Source = source,
                    Target = Scaled(target, unit),
                    AlphaLeft = column == 0 ? 0f : 1f,
                    AlphaRight = column == 2 ? 0f : 1f,
                    AlphaBottom = row == 0 ? 0f : 1f,
                    AlphaTop = row == 2 ? 0f : 1f,
                };
            }
            return count;
        }

        static int Cover(Vector2 size, Vector2 frame, TableQuad[] into)
        {
            var scale = Mathf.Max(frame.x / size.x, frame.y / size.y);
            var shown = frame / scale;
            var source = new Rect((size - shown) / 2f, shown);
            into[0] = Solid(source, new Rect(Vector2.zero, frame), 1f);
            return 1;
        }

        static TableQuad Solid(Rect source, Rect target, float unit) => new()
        {
            Source = source,
            Target = Scaled(target, unit),
            AlphaLeft = 1f,
            AlphaRight = 1f,
            AlphaBottom = 1f,
            AlphaTop = 1f,
        };

        static Rect Scaled(Rect rect, float unit) => new(rect.position * unit, rect.size * unit);

        static Vector4 Edges(float outerLow, float innerLow, float innerHigh, float outerHigh) => new(outerLow, innerLow, innerHigh, outerHigh);

        static Vector4 Centred(float centre, float size, float fade) =>
            new(centre - size / 2f - fade, centre - size / 2f, centre + size / 2f, centre + size / 2f + fade);
    }
}
