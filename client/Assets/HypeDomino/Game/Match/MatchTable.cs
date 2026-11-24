using System;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Match;
using Hypedomino.V1;
using UnityEngine;

namespace HypeDomino.Game.Match
{
    public sealed class MatchTable : IMatchTable
    {
        readonly SeatAvatars avatars;
        readonly TableBackground background;
        readonly ActiveCosmeticManifest manifest;
        readonly ScorePanelView score;
        readonly BoardTilesView tiles;
        readonly MatchLayoutView layout;

        public MatchTable(SeatAvatars avatars, TableBackground background, ActiveCosmeticManifest manifest, ScorePanelView score, BoardTilesView tiles, MatchLayoutView layout, float boardMaxScale)
        {
            this.avatars = avatars ?? throw new ArgumentNullException(nameof(avatars));
            this.background = background ?? throw new ArgumentNullException(nameof(background));
            this.manifest = manifest ?? throw new ArgumentNullException(nameof(manifest));
            this.score = score ? score : throw new ArgumentNullException(nameof(score));
            this.tiles = tiles ? tiles : throw new ArgumentNullException(nameof(tiles));
            this.layout = layout ? layout : throw new ArgumentNullException(nameof(layout));
            BoardMaxScale = boardMaxScale;
        }

        public Vector2 BoardSize => layout.Layout.Board.size;
        public float BoardMaxScale { get; }

        public void ShowBackground(MatchSnapshot view) => background.Show(view, manifest.Current);

        public void ShowSeats(MatchSnapshot view)
        {
            if (view.Mode == Mode._1V1 || view.Mode == Mode._2V2) FitSeats(SeatOrder.Size(view.Mode));
            avatars.Show(view, manifest.Current);
        }

        public void ShowScore(ScoreBoard board) => ScorePanel.Show(score, board);

        public void ShowBoard(ChainLayout board) => tiles.Show(board);

        public void RefreshEffects()
        {
            background.RefreshEffects();
            avatars.RefreshEffects();
        }

        void FitSeats(int seatCount)
        {
            if (layout.SeatCount != seatCount) layout.SetSeatCount(seatCount);
        }
    }
}
