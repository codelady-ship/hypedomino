using System;
using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.Tutorial
{
    public enum TutorialActor
    {
        Coach,
        Player,
        Bot,
    }

    public enum TutorialTopic
    {
        Ends,
        MultiplesOfFive,
        Spinner,
        TargetScore,
    }

    public readonly struct TutorialEnd
    {
        public readonly BoardEnd End;
        public readonly uint Pips;
        public readonly bool Double;

        public TutorialEnd(BoardEnd end, uint pips, bool isDouble)
        {
            End = end;
            Pips = pips;
            Double = isDouble;
        }

        public uint Value => Double ? Pips * 2 : Pips;
    }

    public sealed class TutorialStep
    {
        readonly TutorialEnd[] ends;

        TutorialStep(TutorialActor actor, TutorialTopic topic, string promptKey, TileFace tile, BoardEnd end, uint points, TutorialEnd[] ends)
        {
            Actor = actor;
            Topic = topic;
            PromptKey = promptKey;
            Tile = tile;
            End = end;
            Points = points;
            this.ends = ends;
        }

        public TutorialActor Actor { get; }
        public TutorialTopic Topic { get; }
        public string PromptKey { get; }
        public TileFace Tile { get; }
        public BoardEnd End { get; }
        public uint Points { get; }
        public bool Places => Actor != TutorialActor.Coach;
        public ReadOnlySpan<TutorialEnd> Ends => ends;

        public uint EndsTotal
        {
            get
            {
                var total = 0u;
                for (var i = 0; i < ends.Length; i++) total += ends[i].Value;
                return total;
            }
        }

        public static TutorialStep Coach(TutorialTopic topic, string promptKey) =>
            new(TutorialActor.Coach, topic, promptKey, default, BoardEnd.Unspecified, 0, Array.Empty<TutorialEnd>());

        public static TutorialStep Placement(TutorialActor actor, TutorialTopic topic, string promptKey, TileFace tile, BoardEnd end, uint points, params TutorialEnd[] ends) =>
            new(actor, topic, promptKey, tile, end, points, ends);
    }

    public static class TutorialScript
    {
        public const int BotHandSize = 7;

        public const string Welcome = "tutorial.welcome";
        public const string Open = "tutorial.open";
        public const string Ends = "tutorial.ends";
        public const string BotDouble = "tutorial.bot_double";
        public const string ScoreTen = "tutorial.score_ten";
        public const string Multiples = "tutorial.multiples";
        public const string BotSpinner = "tutorial.bot_spinner";
        public const string Spinner = "tutorial.spinner";
        public const string SpinnerPlay = "tutorial.spinner_play";
        public const string Target = "tutorial.target";
        public const string Done = "tutorial.done";
        public const string Skip = "tutorial.skip";
        public const string Replay = "tutorial.replay";
        public const string Continue = "tutorial.continue";
        public const string WrongTile = "tutorial.wrong_tile";
        public const string WrongEnd = "tutorial.wrong_end";

        static readonly TileFace[] playerHand =
        {
            new(2, 3), new(2, 4), new(3, 5), new(6, 6), new(0, 4), new(1, 5), new(2, 6),
        };

        static readonly TutorialStep[] steps =
        {
            TutorialStep.Coach(TutorialTopic.Ends, Welcome),
            TutorialStep.Placement(TutorialActor.Player, TutorialTopic.Ends, Open, new TileFace(2, 3), BoardEnd.Unspecified, 0,
                End(BoardEnd.Left, 2), End(BoardEnd.Right, 3)),
            TutorialStep.Coach(TutorialTopic.Ends, Ends),
            TutorialStep.Placement(TutorialActor.Bot, TutorialTopic.Ends, BotDouble, new TileFace(3, 3), BoardEnd.Right, 0,
                End(BoardEnd.Left, 2), DoubleEnd(BoardEnd.Right, 3)),
            TutorialStep.Placement(TutorialActor.Player, TutorialTopic.MultiplesOfFive, ScoreTen, new TileFace(2, 4), BoardEnd.Left, 10,
                End(BoardEnd.Left, 4), DoubleEnd(BoardEnd.Right, 3)),
            TutorialStep.Coach(TutorialTopic.MultiplesOfFive, Multiples),
            TutorialStep.Placement(TutorialActor.Bot, TutorialTopic.Spinner, BotSpinner, new TileFace(1, 3), BoardEnd.Right, 5,
                End(BoardEnd.Left, 4), End(BoardEnd.Right, 1)),
            TutorialStep.Coach(TutorialTopic.Spinner, Spinner),
            TutorialStep.Placement(TutorialActor.Player, TutorialTopic.Spinner, SpinnerPlay, new TileFace(3, 5), BoardEnd.Top, 10,
                End(BoardEnd.Left, 4), End(BoardEnd.Right, 1), End(BoardEnd.Top, 5)),
            TutorialStep.Coach(TutorialTopic.TargetScore, Target),
            TutorialStep.Coach(TutorialTopic.TargetScore, Done),
        };

        public static ReadOnlySpan<TileFace> PlayerHand => playerHand;
        public static TutorialStep[] Steps => steps;

        static TutorialEnd End(BoardEnd end, uint pips) => new(end, pips, false);

        static TutorialEnd DoubleEnd(BoardEnd end, uint pips) => new(end, pips, true);
    }
}
