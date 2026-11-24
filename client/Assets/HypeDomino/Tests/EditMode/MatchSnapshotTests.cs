using System.IO;
using Google.Protobuf;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchSnapshotTests
    {
        static readonly string[] GoldenViews = { "match_view", "match_view_loading", "match_view_waiting" };

        static MatchView GoldenView(string name)
        {
            var path = Path.GetFullPath(Path.Combine(Application.dataPath, "..", "..", "proto", "testdata", "frames", name + ".bin"));
            return Envelope.Parser.ParseFrom(File.ReadAllBytes(path)).MatchView;
        }

        static MatchSnapshot Read(MatchView view, MatchSnapshot snapshot = null)
        {
            snapshot ??= new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        static void AssertTile(Tile expected, TileFace actual, string what)
        {
            Assert.AreEqual(expected?.Low ?? 0, actual.Low, what);
            Assert.AreEqual(expected?.High ?? 0, actual.High, what);
        }

        static void AssertPoints(TeamPoints expected, TeamPointsValue actual, string what)
        {
            Assert.AreEqual(expected?.TeamA ?? 0, actual.TeamA, what);
            Assert.AreEqual(expected?.TeamB ?? 0, actual.TeamB, what);
        }

        static void AssertOriented(Google.Protobuf.Collections.RepeatedField<OrientedTile> expected, System.ReadOnlySpan<OrientedTileFace> actual, string what)
        {
            Assert.AreEqual(expected.Count, actual.Length, what);
            for (var at = 0; at < expected.Count; at++)
            {
                Assert.AreEqual(expected[at].First, actual[at].First, what);
                Assert.AreEqual(expected[at].Second, actual[at].Second, what);
            }
        }

        static void AssertSame(MatchView expected, MatchSnapshot actual, string name)
        {
            Assert.AreEqual(expected.State, actual.State, name);
            Assert.AreEqual(expected.Seat, actual.Seat, name);
            Assert.AreEqual(expected.Mode, actual.Mode, name);
            Assert.AreEqual(expected.HandsPlayed, actual.HandsPlayed, name);
            AssertPoints(expected.Scores, actual.Scores, name + " scores");
            Assert.AreEqual(expected.CarryOver, actual.CarryOver, name);
            Assert.AreEqual(expected.BoneyardSize, actual.BoneyardSize, name);
            Assert.AreEqual(expected.Current, actual.Current, name);
            Assert.AreEqual(expected.OpeningTeam, actual.OpeningTeam, name);
            Assert.AreEqual(expected.TurnPlacements, actual.TurnPlacements, name);
            Assert.AreEqual(expected.TurnDeadlineUnixMs, actual.TurnDeadlineUnixMs, name);
            Assert.AreEqual(expected.Winner, actual.Winner, name);
            Assert.AreEqual(expected.LoadingDeadlineUnixMs, actual.LoadingDeadlineUnixMs, name);
            Assert.AreEqual(expected.AfkConsecutiveTimeouts, actual.AfkConsecutiveTimeouts, name);
            Assert.AreEqual(expected.TurnSeconds, actual.TurnSeconds, name);
            Assert.AreEqual(expected.TurnWarningSeconds, actual.TurnWarningSeconds, name);

            Assert.AreEqual(expected.Hand.Count, actual.Hand.Length, name + " hand");
            for (var at = 0; at < expected.Hand.Count; at++) AssertTile(expected.Hand[at], actual.Hand[at], name + " hand");

            Assert.AreEqual(expected.Seats.Count, actual.Seats.Length, name + " seats");
            for (var at = 0; at < expected.Seats.Count; at++)
            {
                var seat = expected.Seats[at];
                var read = actual.Seats[at];
                Assert.AreEqual(seat.Seat, read.Seat, name);
                Assert.AreEqual(seat.HandSize, read.HandSize, name);
                Assert.AreEqual(seat.Controller, read.Controller, name);
                Assert.AreEqual(seat.LoadingProgress, read.LoadingProgress, name);
                Assert.AreEqual(seat.Ready, read.Ready, name);
                Assert.AreEqual(seat.Disconnected, read.Disconnected, name);
                Assert.AreEqual(seat.ConsecutiveTimeouts, read.ConsecutiveTimeouts, name);
            }

            var board = expected.Board ?? new BoardView();
            AssertOriented(board.Line, actual.Line, name + " line");
            Assert.AreEqual(board.OpeningIndex, actual.OpeningIndex, name + " opening index");
            AssertOriented(board.Top, actual.Top, name + " top");
            AssertOriented(board.Bottom, actual.Bottom, name + " bottom");
            Assert.AreEqual(board.Spinner != null, actual.HasSpinner, name + " spinner");
            AssertTile(board.Spinner, actual.Spinner, name + " spinner");
            Assert.AreEqual(board.OpenEnds.Count, actual.OpenEnds.Length, name + " open ends");
            for (var at = 0; at < board.OpenEnds.Count; at++)
            {
                Assert.AreEqual(board.OpenEnds[at].End, actual.OpenEnds[at].End, name);
                Assert.AreEqual(board.OpenEnds[at].Value, actual.OpenEnds[at].Value, name);
            }

            Assert.AreEqual(expected.ForcedOpening != null, actual.HasForcedOpening, name + " opening");
            Assert.AreEqual(expected.ForcedOpening?.Seat ?? Seat.Unspecified, actual.ForcedOpening.Seat, name);
            AssertTile(expected.ForcedOpening?.Tile, actual.ForcedOpening.Tile, name + " opening");

            Assert.AreEqual(expected.LegalMoves.Count, actual.LegalMoves.Length, name + " legal moves");
            for (var at = 0; at < expected.LegalMoves.Count; at++)
            {
                AssertTile(expected.LegalMoves[at].Tile, actual.LegalMoves[at].Tile, name + " legal move");
                Assert.AreEqual(expected.LegalMoves[at].End, actual.LegalMoves[at].End, name);
            }

            Assert.AreEqual(expected.LastResult != null, actual.HasLastResult, name + " result");
            var result = expected.LastResult ?? new HandResult();
            AssertPoints(result.Pips, actual.LastResult.Pips, name + " pips");
            Assert.AreEqual(result.Winner, actual.LastResult.Winner, name);
            AssertPoints(result.Awarded, actual.LastResult.Awarded, name + " awarded");
            Assert.AreEqual(result.CarryOver, actual.LastResult.CarryOver, name);
            AssertPoints(result.Scores, actual.LastResult.Scores, name + " result scores");
            Assert.AreEqual(result.NextOpener, actual.LastResult.NextOpener, name);
            Assert.AreEqual(result.Ending, actual.LastResult.Ending, name);
            Assert.AreEqual(result.HandPointsLocked, actual.LastResult.HandPointsLocked, name);
            Assert.AreEqual(result.CarriedIn, actual.LastResult.CarriedIn, name);
            Assert.AreEqual(result.Left.Count, actual.LeftSeats.Length, name + " left");
            for (var at = 0; at < result.Left.Count; at++)
            {
                var left = actual.LeftSeats[at];
                Assert.AreEqual(result.Left[at].Seat, left.Seat, name + " left seat");
                var tiles = actual.LeftTiles(left);
                Assert.AreEqual(result.Left[at].Tiles.Count, tiles.Length, name + " left tiles");
                for (var tile = 0; tile < tiles.Length; tile++) AssertTile(result.Left[at].Tiles[tile], tiles[tile], name + " left tile");
            }
            Assert.AreEqual(expected.TargetScore, actual.TargetScore, name);
            Assert.AreEqual(expected.HandPointsLockThreshold, actual.HandPointsLockThreshold, name);
            Assert.AreEqual(expected.LockedTeams.Contains(Team.A), actual.IsHandPointsLocked(Team.A), name + " lock A");
            Assert.AreEqual(expected.LockedTeams.Contains(Team.B), actual.IsHandPointsLocked(Team.B), name + " lock B");

            CollectionAssert.AreEqual(expected.Roster, actual.Roster, name + " roster");
            Assert.AreEqual(expected.Table, actual.Table, name);
        }

        [Test]
        public void EveryGoldenViewReadsLikeTheGeneratedParser()
        {
            foreach (var name in GoldenViews)
            {
                var view = GoldenView(name);
                AssertSame(view, Read(view), name);
            }
        }

        [Test]
        public void ARichViewCoversEveryField()
        {
            var view = GoldenView("match_view");
            view.Board ??= new BoardView();
            view.Board.Spinner = new Tile { Low = 0, High = 0 };
            view.Board.Top.Add(new OrientedTile { First = 0, Second = 4 });
            view.Board.Bottom.Add(new OrientedTile { First = 0, Second = 2 });
            view.ForcedOpening = new Opening { Seat = Seat.East, Tile = new Tile { Low = 6, High = 6 } };
            view.LastResult = new HandResult
            {
                Pips = new TeamPoints { TeamA = 12, TeamB = 3 },
                Winner = Team.B,
                Awarded = new TeamPoints { TeamB = 15 },
                CarryOver = 2,
                Scores = new TeamPoints { TeamA = 40, TeamB = 55 },
                NextOpener = Team.A,
                Ending = HandEnding.Block,
                HandPointsLocked = true,
                CarriedIn = 10,
                Left =
                {
                    new SeatTiles { Seat = Seat.North, Tiles = { new Tile { Low = 6, High = 6 } } },
                    new SeatTiles { Seat = Seat.East },
                    new SeatTiles { Seat = Seat.South, Tiles = { new Tile { Low = 1, High = 2 }, new Tile { Low = 0, High = 0 } } },
                },
            };
            view.TargetScore = 365;
            view.HandPointsLockThreshold = 300;
            view.LockedTeams.Clear();
            view.LockedTeams.Add(Team.A);
            view.Table = "table_neon";
            view.AfkConsecutiveTimeouts = 3;
            view.TurnSeconds = 20;
            view.TurnWarningSeconds = 5;
            view.Seats[1].Disconnected = true;
            view.Seats[1].ConsecutiveTimeouts = 2;

            AssertSame(view, Read(view), "rich");
        }

        [Test]
        public void AViewWithoutARosterKeepsTheRosterAndTable()
        {
            var first = GoldenView("match_view_loading");
            first.Table = "table_neon";
            if (first.Roster.Count == 0) first.Roster.Add(new RosterEntry { Seat = Seat.North, Nickname = "first" });
            var snapshot = Read(first);

            var later = GoldenView("match_view");
            later.Roster.Clear();
            later.Table = string.Empty;
            Read(later, snapshot);

            CollectionAssert.AreEqual(first.Roster, snapshot.Roster);
            Assert.AreEqual("table_neon", snapshot.Table);
        }

        [Test]
        public void ANewRosterReplacesTheOldOne()
        {
            var first = GoldenView("match_view_loading");
            var snapshot = Read(first);
            var replaced = new MatchView();
            replaced.Roster.Add(new RosterEntry { Seat = Seat.South, Nickname = "replacement" });

            Read(replaced, snapshot);

            Assert.AreEqual(1, snapshot.Roster.Count);
            Assert.AreEqual("replacement", snapshot.Roster[0].Nickname);
        }

        [Test]
        public void ListsShrinkWhenTheNextViewHasFewerEntries()
        {
            var full = GoldenView("match_view");
            Assume.That(full.Hand.Count, NUnit.Framework.Is.GreaterThan(1));
            var snapshot = Read(full);
            var smaller = full.Clone();
            smaller.Hand.RemoveAt(0);
            smaller.LegalMoves.Clear();

            Read(smaller, snapshot);

            AssertSame(smaller, snapshot, "smaller");
        }

        [Test]
        public void AnOpeningIndexOutsideTheLineIsRefused()
        {
            var view = GoldenView("match_view");
            view.Board.OpeningIndex = (uint)view.Board.Line.Count;

            Assert.IsFalse(new MatchSnapshot().TryRead(view.ToByteArray()));
        }

        [Test]
        public void MoreTilesThanTheSetIsRefused()
        {
            var view = new MatchView();
            for (var at = 0; at <= MatchSnapshot.MaxTiles; at++) view.Hand.Add(new Tile { Low = 1, High = 2 });

            Assert.IsFalse(new MatchSnapshot().TryRead(view.ToByteArray()));
        }

        [Test]
        public void UnknownFieldsAreSkipped()
        {
            var view = GoldenView("match_view");
            var bytes = view.ToByteArray();
            var withUnknownField = new byte[bytes.Length + 3];
            bytes.CopyTo(withUnknownField, 0);
            withUnknownField[bytes.Length] = 0xB8;
            withUnknownField[bytes.Length + 1] = 0x06;
            withUnknownField[bytes.Length + 2] = 0x2A;
            var snapshot = new MatchSnapshot();

            Assert.IsTrue(snapshot.TryRead(withUnknownField));
            AssertSame(view, snapshot, "unknown field");
        }

        [Test]
        public void ATruncatedViewIsRefused()
        {
            var bytes = GoldenView("match_view").ToByteArray();

            Assert.IsFalse(new MatchSnapshot().TryRead(new System.ReadOnlySpan<byte>(bytes, 0, bytes.Length - 1)));
        }

        [Test]
        public void ReadingAViewWithoutARosterAllocatesNothing()
        {
            var view = GoldenView("match_view");
            view.Roster.Clear();
            view.Table = string.Empty;
            var bytes = view.ToByteArray();
            var snapshot = new MatchSnapshot();
            snapshot.TryRead(bytes);

            TestDelegate step = () =>
            {
                if (!snapshot.TryRead(bytes)) throw new System.InvalidOperationException();
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
