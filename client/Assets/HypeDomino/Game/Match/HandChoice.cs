using System;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using HypeDomino.UI.Tokens;
using Hypedomino.V1;
using UnityEngine;

namespace HypeDomino.Game.Match
{
    public enum HandTap
    {
        Ignored,
        Selected,
        Deselected,
        Placed,
    }

    public readonly struct TilePlacement
    {
        public readonly TileFace Tile;
        public readonly BoardEnd End;

        public TilePlacement(TileFace tile, BoardEnd end)
        {
            Tile = tile;
            End = end;
        }
    }

    public sealed class HandChoice
    {
        public const int MaxEnds = MatchSnapshot.MaxOpenEnds;
        public const float SnapDistance = TokenSize.TouchTargetMin;

        readonly TileFace[] hand = new TileFace[MatchSnapshot.MaxTiles];
        readonly LegalMoveValue[] moves = new LegalMoveValue[MatchSnapshot.MaxLegalMoves];
        int handCount;
        int moveCount;
        bool opening;
        bool hasForcedTile;
        TileFace forcedTile;
        bool placing;
        bool mayEndTurn;
        bool mayDraw;
        bool mayPass;

        public int HandCount => handCount;
        public bool AwaitingView { get; private set; }
        public int Selected { get; private set; } = -1;
        public bool CanEndTurn => mayEndTurn && !AwaitingView;
        public bool CanDraw => mayDraw && !AwaitingView;
        public bool CanPass => mayPass && !AwaitingView;

        public TileFace TileAt(int index) => hand[index];

        public void Read(MatchSnapshot view)
        {
            var hadSelection = Selected >= 0;
            var selectedTile = hadSelection ? hand[Selected] : default;
            Copy(view);
            Decide(view);
            AwaitingView = false;
            Selected = -1;
            if (hadSelection) Reselect(selectedTile);
        }

        public bool IsSelectable(int index)
        {
            if (AwaitingView || !placing || (uint)index >= (uint)handCount) return false;
            if (opening) return !hasForcedTile || Same(hand[index], forcedTile);
            for (var i = 0; i < moveCount; i++)
                if (Same(moves[i].Tile, hand[index])) return true;
            return false;
        }

        public int EndsOf(int index, Span<BoardEnd> ends)
        {
            if (!IsSelectable(index)) return 0;
            if (opening)
            {
                ends[0] = BoardEnd.Unspecified;
                return 1;
            }
            var count = 0;
            for (var i = 0; i < moveCount && count < ends.Length; i++)
            {
                if (!Same(moves[i].Tile, hand[index]) || Contains(ends[..count], moves[i].End)) continue;
                ends[count++] = moves[i].End;
            }
            return count;
        }

        public int EndPlacesOf(int index, ChainLayout board, Span<Vector2> places)
        {
            Span<BoardEnd> ends = stackalloc BoardEnd[MaxEnds];
            var count = Math.Min(EndsOf(index, ends), places.Length);
            for (var i = 0; i < count; i++) places[i] = PlaceOf(ends[i], board);
            return count;
        }

        public HandTap Tap(int index, out TilePlacement placement)
        {
            placement = default;
            if (!IsSelectable(index)) return HandTap.Ignored;
            if (index == Selected)
            {
                Selected = -1;
                return HandTap.Deselected;
            }
            Span<BoardEnd> ends = stackalloc BoardEnd[MaxEnds];
            if (EndsOf(index, ends) == 1)
            {
                placement = Place(index, ends[0]);
                return HandTap.Placed;
            }
            Selected = index;
            return HandTap.Selected;
        }

        public bool TryChooseEnd(BoardEnd end, out TilePlacement placement)
        {
            placement = default;
            if (Selected < 0) return false;
            Span<BoardEnd> ends = stackalloc BoardEnd[MaxEnds];
            var count = EndsOf(Selected, ends);
            if (!Contains(ends[..count], end)) return false;
            placement = Place(Selected, end);
            return true;
        }

        public bool TryDrop(int index, ReadOnlySpan<Vector2> endPositions, Vector2 drop, out TilePlacement placement)
        {
            placement = default;
            Span<BoardEnd> ends = stackalloc BoardEnd[MaxEnds];
            var count = Math.Min(EndsOf(index, ends), endPositions.Length);
            var nearest = -1;
            var nearestDistance = SnapDistance * SnapDistance;
            for (var i = 0; i < count; i++)
            {
                var distance = (endPositions[i] - drop).sqrMagnitude;
                if (distance > nearestDistance) continue;
                nearest = i;
                nearestDistance = distance;
            }
            if (nearest < 0) return false;
            placement = Place(index, ends[nearest]);
            return true;
        }

        public bool TryEndTurn() => Commit(CanEndTurn);

        public bool TryDraw() => Commit(CanDraw);

        public bool TryPass() => Commit(CanPass);

        void Copy(MatchSnapshot view)
        {
            var tiles = view.Hand;
            handCount = Math.Min(tiles.Length, hand.Length);
            for (var i = 0; i < handCount; i++) hand[i] = tiles[i];
            var legal = view.LegalMoves;
            moveCount = Math.Min(legal.Length, moves.Length);
            for (var i = 0; i < moveCount; i++) moves[i] = legal[i];
        }

        void Decide(MatchSnapshot view)
        {
            opening = false;
            hasForcedTile = false;
            placing = mayEndTurn = mayDraw = mayPass = false;
            if (view.Seat == Seat.Unspecified) return;
            if (view.State == MatchState.Opening)
            {
                DecideOpening(view);
                return;
            }
            if (view.State != MatchState.Playing || view.Current != view.Seat) return;
            var placed = view.TurnPlacements > 0;
            placing = moveCount > 0;
            mayEndTurn = placed;
            var stuck = !placed && moveCount == 0;
            mayDraw = stuck && view.Mode == Mode._1V1 && view.BoneyardSize > 1;
            mayPass = stuck && !mayDraw;
        }

        void DecideOpening(MatchSnapshot view)
        {
            opening = true;
            if (view.HasForcedOpening)
            {
                hasForcedTile = true;
                forcedTile = view.ForcedOpening.Tile;
                placing = view.ForcedOpening.Seat == view.Seat;
                return;
            }
            placing = view.OpeningTeam != Team.Unspecified && SeatOrder.TeamOf(view.Seat) == view.OpeningTeam;
        }

        void Reselect(TileFace tile)
        {
            Span<BoardEnd> ends = stackalloc BoardEnd[MaxEnds];
            for (var i = 0; i < handCount; i++)
            {
                if (!Same(hand[i], tile)) continue;
                if (EndsOf(i, ends) > 1) Selected = i;
                return;
            }
        }

        TilePlacement Place(int index, BoardEnd end)
        {
            var placement = new TilePlacement(hand[index], end);
            Selected = -1;
            AwaitingView = true;
            return placement;
        }

        bool Commit(bool allowed)
        {
            if (!allowed) return false;
            Selected = -1;
            AwaitingView = true;
            return true;
        }

        static Vector2 PlaceOf(BoardEnd end, ChainLayout board)
        {
            if (end == BoardEnd.Unspecified || !board.TryGetEnd(ChainEndOf(end), out var area)) return Vector2.zero;
            return board.OnBoard(area.center);
        }

        static ChainEnd ChainEndOf(BoardEnd end) => end switch
        {
            BoardEnd.Left => ChainEnd.West,
            BoardEnd.Right => ChainEnd.East,
            BoardEnd.Top => ChainEnd.North,
            _ => ChainEnd.South,
        };

        static bool Same(TileFace a, TileFace b) =>
            (a.Low == b.Low && a.High == b.High) || (a.Low == b.High && a.High == b.Low);

        static bool Contains(Span<BoardEnd> ends, BoardEnd end)
        {
            for (var i = 0; i < ends.Length; i++)
                if (ends[i] == end) return true;
            return false;
        }
    }
}
