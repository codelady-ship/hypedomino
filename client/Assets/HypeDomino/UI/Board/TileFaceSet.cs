using System;
using UnityEngine;

namespace HypeDomino.UI.Board
{
    public sealed class TileFaceSet : ScriptableObject
    {
        [SerializeField] Sprite[] faces = Array.Empty<Sprite>();

        public bool IsComplete
        {
            get
            {
                if (faces.Length != TileFaces.SetSize) return false;
                for (var i = 0; i < faces.Length; i++)
                    if (faces[i] == null) return false;
                return true;
            }
        }

        public Sprite Face(Domino tile) => faces[TileFaces.Index(tile)];

        public void SetFaces(Sprite[] value) => faces = value ?? Array.Empty<Sprite>();
    }
}
