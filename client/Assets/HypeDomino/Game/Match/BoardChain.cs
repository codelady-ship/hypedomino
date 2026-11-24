using System;
using System.Collections.Generic;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using UnityEngine;

namespace HypeDomino.Game.Match
{
    public sealed class BoardChain
    {
        readonly List<Domino> line = new(MatchSnapshot.MaxTiles);
        readonly List<Domino> top = new(MatchSnapshot.MaxTiles);
        readonly List<Domino> bottom = new(MatchSnapshot.MaxTiles);

        public ChainLayout Layout { get; } = new(MatchSnapshot.MaxTiles);

        public void Arrange(MatchSnapshot view, Vector2 board, float maxScale)
        {
            Fill(line, view.Line);
            Fill(top, view.Top);
            Fill(bottom, view.Bottom);
            Layout.Arrange(line, AnchorIndex(view), top, bottom, board, maxScale);
        }

        public static int AnchorIndex(MatchSnapshot view)
        {
            if (!view.HasSpinner) return view.OpeningIndex;
            var tiles = view.Line;
            for (var i = 0; i < tiles.Length; i++)
            {
                if (tiles[i].First == view.Spinner.Low && tiles[i].Second == view.Spinner.High) return i;
            }
            return view.OpeningIndex;
        }

        static void Fill(List<Domino> target, ReadOnlySpan<OrientedTileFace> tiles)
        {
            target.Clear();
            for (var i = 0; i < tiles.Length; i++) target.Add(new Domino((byte)tiles[i].First, (byte)tiles[i].Second));
        }
    }
}
