using UnityEngine;

namespace HypeDomino.UI.Board
{
    public readonly struct PlacedDomino
    {
        public PlacedDomino(Vector2 center, Vector2 facing, byte near, byte far, bool across)
        {
            Center = center;
            Facing = facing;
            Near = near;
            Far = far;
            Across = across;
        }

        public Vector2 Center { get; }

        public Vector2 Facing { get; }

        public byte Near { get; }

        public byte Far { get; }

        public bool Across { get; }

        public Vector2 Size => (Facing.x != 0f) != Across ? new Vector2(2f, 1f) : new Vector2(1f, 2f);

        public Rect Area => new Rect(Center - Size * 0.5f, Size);
    }
}
