using System.Collections.Generic;
using HypeDomino.UI.Board;
using HypeDomino.UI.Match;
using HypeDomino.UI.Screens;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ChainReadabilityTests
    {
        const int Seed = 169;
        const int BoardsPerShape = 400;
        const int SetSize = 28;
        const int TwoVersusTwoFullestBoard = 25;
        const int OneVersusOneFullestBoard = 27;
        const float ReadableShortSidePixels = 32f;
        const float SmallestPhoneShortPixels = 828f;
        const float SmallestPhoneLongPixels = 1792f;
        const float SmallestPhonePixelsPerPoint = 2f;
        const float NotchInsetPoints = 44f;
        const float LandscapeHomeIndicatorPoints = 21f;
        const float PortraitHomeIndicatorPoints = 34f;
        const int NoSpinner = int.MinValue;

        enum End
        {
            West,
            East,
            North,
            South,
        }

        static float Units(float pixels) => pixels * ScreenView.ReferenceShortSide / SmallestPhoneShortPixels;

        static float PointUnits(float points) => Units(points * SmallestPhonePixelsPerPoint);

        static Rect SmallestPhoneSafeArea(bool landscape)
        {
            var longSide = Units(SmallestPhoneLongPixels);
            var notch = PointUnits(NotchInsetPoints);
            return landscape
                ? Rect.MinMaxRect(notch, PointUnits(LandscapeHomeIndicatorPoints), longSide - notch, ScreenView.ReferenceShortSide)
                : Rect.MinMaxRect(0f, PointUnits(PortraitHomeIndicatorPoints), ScreenView.ReferenceShortSide, longSide - notch);
        }

        [TestCase(true, 4, TwoVersusTwoFullestBoard)]
        [TestCase(false, 4, TwoVersusTwoFullestBoard)]
        [TestCase(true, 2, OneVersusOneFullestBoard)]
        [TestCase(false, 2, OneVersusOneFullestBoard)]
        public void TheFullestBoardsOnTheSmallestPhoneKeepEveryTileReadable(bool landscape, int seats, int tiles)
        {
            var match = new MatchLayout();
            match.Arrange(SmallestPhoneSafeArea(landscape), seats);
            var board = match.Board.size;
            var layout = new ChainLayout(SetSize);
            var random = new System.Random(Seed);
            var line = new List<Domino>(SetSize);
            var top = new List<Domino>(SetSize);
            var bottom = new List<Domino>(SetSize);
            var smallest = float.MaxValue;

            for (var played = 0; played < BoardsPerShape; played++)
            {
                int anchor;
                while (!TryPlayBoard(random, tiles, line, top, bottom, out anchor)) { }
                layout.Arrange(line, anchor, top, bottom, board, float.MaxValue);
                Assert.AreEqual(tiles, layout.Count);
                Assert.LessOrEqual(layout.Bounds.width * layout.Scale, board.x + 0.01f);
                Assert.LessOrEqual(layout.Bounds.height * layout.Scale, board.y + 0.01f);
                smallest = Mathf.Min(smallest, layout.Scale);
            }

            Assert.GreaterOrEqual(smallest, Units(ReadableShortSidePixels), $"smallest tile short side on a {board.x} x {board.y} board");
        }

        static bool TryPlayBoard(System.Random random, int count, List<Domino> line, List<Domino> top, List<Domino> bottom, out int anchor)
        {
            var pool = new List<Domino>(SetSize);
            for (byte low = 0; low <= 6; low++)
            for (var high = low; high <= 6; high++)
                pool.Add(new Domino(low, high));

            var opening = Take(pool, random.Next(pool.Count));
            var west = new List<Domino>(SetSize);
            var east = new List<Domino>(SetSize);
            top.Clear();
            bottom.Clear();
            var open = new[] { opening.First, opening.Second, (byte)0, (byte)0 };
            var spinner = NoSpinner;
            var moves = new List<(int tile, End end)>();
            anchor = 0;

            for (var placed = 1; placed < count; placed++)
            {
                moves.Clear();
                var lastEnd = spinner == NoSpinner ? End.East : End.South;
                for (var i = 0; i < pool.Count; i++)
                for (var end = End.West; end <= lastEnd; end++)
                    if (pool[i].First == open[(int)end] || pool[i].Second == open[(int)end]) moves.Add((i, end));
                if (moves.Count == 0) return false;

                var (index, chosen) = moves[random.Next(moves.Count)];
                var tile = Take(pool, index);
                var near = open[(int)chosen];
                var far = tile.First == near ? tile.Second : tile.First;
                open[(int)chosen] = far;
                if (chosen == End.West) west.Add(new Domino(far, near));
                else if (chosen == End.East) east.Add(new Domino(near, far));
                else if (chosen == End.North) top.Add(new Domino(near, far));
                else bottom.Add(new Domino(near, far));

                if (spinner != NoSpinner) continue;
                spinner = SpinnerOffset(opening, west, east, chosen);
                if (spinner == NoSpinner) continue;
                open[(int)End.North] = SpinnerValue(opening, west, east, spinner);
                open[(int)End.South] = open[(int)End.North];
            }

            line.Clear();
            for (var i = west.Count - 1; i >= 0; i--) line.Add(west[i]);
            line.Add(opening);
            line.AddRange(east);
            anchor = west.Count + (spinner == NoSpinner ? 0 : spinner);
            return true;
        }

        static Domino Take(List<Domino> pool, int index)
        {
            var tile = pool[index];
            pool.RemoveAt(index);
            return tile;
        }

        static int SpinnerOffset(Domino opening, List<Domino> west, List<Domino> east, End chosen)
        {
            if (opening.IsDouble && west.Count > 0 && east.Count > 0) return 0;
            if (chosen == End.East && east.Count >= 2 && east[east.Count - 2].IsDouble) return east.Count - 1;
            if (chosen == End.West && west.Count >= 2 && west[west.Count - 2].IsDouble) return 1 - west.Count;
            return NoSpinner;
        }

        static byte SpinnerValue(Domino opening, List<Domino> west, List<Domino> east, int offset)
        {
            if (offset == 0) return opening.First;
            return offset > 0 ? east[offset - 1].First : west[-offset - 1].First;
        }
    }
}
