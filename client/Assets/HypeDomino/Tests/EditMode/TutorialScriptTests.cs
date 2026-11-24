using System.Collections.Generic;
using HypeDomino.Game.Tutorial;
using HypeDomino.Net;
using HypeDomino.UI.Localization;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TutorialScriptTests
    {
        const uint ScoringMultiple = 5;
        const int HandSize = 7;

        static readonly string[] ControlKeys =
        {
            TutorialScript.Skip,
            TutorialScript.Replay,
            TutorialScript.Continue,
            TutorialScript.WrongTile,
            TutorialScript.WrongEnd,
        };

        sealed class RulesBoard
        {
            readonly Dictionary<BoardEnd, TutorialEnd> ends = new();
            bool opened;
            bool hasSpinner;
            uint spinnerPips;

            public uint Place(TutorialStep step)
            {
                if (!opened) return Open(step.Tile);
                var pips = AttachedPips(step.End);
                Assert.IsTrue(step.Tile.Low == pips || step.Tile.High == pips, $"{step.PromptKey}: the tile does not match the end");
                if (!hasSpinner && IsLineEnd(step.End) && ends[step.End].Double)
                {
                    hasSpinner = true;
                    spinnerPips = pips;
                }
                var other = step.Tile.Low == pips ? step.Tile.High : step.Tile.Low;
                ends[step.End] = new TutorialEnd(step.End, other, step.Tile.Low == step.Tile.High);
                return Scored(Total());
            }

            public void AssertEnds(TutorialStep step)
            {
                Assert.AreEqual(ends.Count, step.Ends.Length, step.PromptKey);
                foreach (var end in step.Ends)
                {
                    Assert.IsTrue(ends.TryGetValue(end.End, out var expected), $"{step.PromptKey}: {end.End} is not an open end");
                    Assert.AreEqual(expected.Pips, end.Pips, $"{step.PromptKey}: {end.End}");
                    Assert.AreEqual(expected.Double, end.Double, $"{step.PromptKey}: {end.End}");
                }
            }

            uint Open(TileFace tile)
            {
                opened = true;
                var isDouble = tile.Low == tile.High;
                ends[BoardEnd.Left] = new TutorialEnd(BoardEnd.Left, tile.Low, isDouble);
                ends[BoardEnd.Right] = new TutorialEnd(BoardEnd.Right, tile.High, isDouble);
                return 0;
            }

            uint AttachedPips(BoardEnd end)
            {
                if (ends.TryGetValue(end, out var open)) return open.Pips;
                Assert.IsTrue(hasSpinner && !IsLineEnd(end), $"{end} is not open");
                return spinnerPips;
            }

            uint Total()
            {
                var total = 0u;
                foreach (var end in ends.Values) total += end.Value;
                return total;
            }

            static uint Scored(uint total) => total > 0 && total % ScoringMultiple == 0 ? total : 0;

            static bool IsLineEnd(BoardEnd end) => end == BoardEnd.Left || end == BoardEnd.Right;
        }

        [Test]
        public void EveryPlacementFollowsTheRulesAndScoresWhatTheRulesScore()
        {
            var board = new RulesBoard();
            foreach (var step in TutorialScript.Steps)
            {
                if (!step.Places) continue;
                Assert.AreEqual(board.Place(step), step.Points, step.PromptKey);
                board.AssertEnds(step);
                if (step.Points > 0) Assert.AreEqual(step.Points, step.EndsTotal, step.PromptKey);
            }
        }

        [Test]
        public void ThePlayerOpensTheFirstHandWithTwoThree()
        {
            var first = FirstPlacement();
            Assert.AreEqual(TutorialActor.Player, first.Actor);
            Assert.AreEqual(BoardEnd.Unspecified, first.End);
            Assert.AreEqual(2u, first.Tile.Low);
            Assert.AreEqual(3u, first.Tile.High);
            Assert.AreEqual(0u, first.Points);
        }

        [Test]
        public void ThePlayerPlacesOnlyTilesFromTheScriptedHand()
        {
            var hand = new List<TileFace>(TutorialScript.PlayerHand.ToArray());
            foreach (var step in TutorialScript.Steps)
            {
                if (step.Actor != TutorialActor.Player) continue;
                Assert.IsTrue(hand.Exists(tile => Same(tile, step.Tile)), step.PromptKey);
            }
        }

        [Test]
        public void NoTileAppearsTwiceAndBothHandsAreFull()
        {
            var seen = new HashSet<uint>();
            foreach (var tile in TutorialScript.PlayerHand) Assert.IsTrue(seen.Add(Id(tile)));
            var botTiles = 0;
            foreach (var step in TutorialScript.Steps)
            {
                if (step.Actor != TutorialActor.Bot) continue;
                botTiles++;
                Assert.IsTrue(seen.Add(Id(step.Tile)), step.PromptKey);
            }
            Assert.AreEqual(HandSize, TutorialScript.PlayerHand.Length);
            Assert.AreEqual(HandSize, TutorialScript.BotHandSize);
            Assert.LessOrEqual(botTiles, TutorialScript.BotHandSize);
        }

        [Test]
        public void TheTopicsComeInTeachingOrderAndEachIsCovered()
        {
            var last = TutorialTopic.Ends;
            var covered = new HashSet<TutorialTopic>();
            foreach (var step in TutorialScript.Steps)
            {
                Assert.GreaterOrEqual((int)step.Topic, (int)last, step.PromptKey);
                last = step.Topic;
                covered.Add(step.Topic);
            }
            foreach (TutorialTopic topic in System.Enum.GetValues(typeof(TutorialTopic))) Assert.IsTrue(covered.Contains(topic), topic.ToString());
        }

        [Test]
        public void TheScriptTeachesTheSpinnerArmByPlacingOnIt()
        {
            var armPlacements = 0;
            foreach (var step in TutorialScript.Steps)
                if (step.Actor == TutorialActor.Player && (step.End == BoardEnd.Top || step.End == BoardEnd.Bottom)) armPlacements++;
            Assert.AreEqual(1, armPlacements);
        }

        [Test]
        public void ThePromptKeysAreDistinct()
        {
            var keys = new HashSet<string>(ControlKeys);
            foreach (var step in TutorialScript.Steps) Assert.IsTrue(keys.Add(step.PromptKey), step.PromptKey);
        }

        [Test]
        public void TheShippedBundleHasEveryKeyInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            var keys = new List<string>(ControlKeys);
            foreach (var step in TutorialScript.Steps) keys.Add(step.PromptKey);
            foreach (var key in keys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
            Assert.AreEqual("Keç", strings.Value(TutorialScript.Skip, "az"));
            Assert.AreEqual("Skip", strings.Value(TutorialScript.Skip, GameLanguage.English));
        }

        static TutorialStep FirstPlacement()
        {
            foreach (var step in TutorialScript.Steps)
                if (step.Places) return step;
            Assert.Fail("the script places no tile");
            return null;
        }

        static uint Id(TileFace tile) => tile.Low < tile.High ? tile.Low * 7 + tile.High : tile.High * 7 + tile.Low;

        static bool Same(TileFace a, TileFace b) => Id(a) == Id(b);
    }
}
