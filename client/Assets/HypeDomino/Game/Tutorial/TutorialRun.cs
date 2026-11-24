using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.Tutorial
{
    public enum TutorialState
    {
        NotStarted,
        Running,
        Skipped,
        Completed,
    }

    public enum TutorialMove
    {
        Accepted,
        NotPlayerTurn,
        WrongTile,
        WrongEnd,
    }

    public sealed class TutorialRun
    {
        readonly TutorialStep[] steps;

        public TutorialRun() : this(TutorialScript.Steps)
        {
        }

        public TutorialRun(TutorialStep[] steps)
        {
            this.steps = steps;
        }

        public TutorialState State { get; private set; }
        public int StepIndex { get; private set; }
        public int StepCount => steps.Length;
        public uint PlayerPoints { get; private set; }
        public uint BotPoints { get; private set; }
        public bool IsFinished => State == TutorialState.Skipped || State == TutorialState.Completed;
        public TutorialStep Current => State == TutorialState.Running ? steps[StepIndex] : null;
        public bool AwaitsPlayer => State == TutorialState.Running && steps[StepIndex].Actor == TutorialActor.Player;

        public int StepsOnBoard
        {
            get
            {
                if (State == TutorialState.Completed) return steps.Length;
                if (State != TutorialState.Running) return 0;
                return steps[StepIndex].Actor == TutorialActor.Bot ? StepIndex + 1 : StepIndex;
            }
        }

        public void Start()
        {
            PlayerPoints = 0;
            BotPoints = 0;
            StepIndex = 0;
            State = steps.Length == 0 ? TutorialState.Completed : TutorialState.Running;
            if (State == TutorialState.Running) Enter();
        }

        public bool Continue()
        {
            if (State != TutorialState.Running || AwaitsPlayer) return false;
            Advance();
            return true;
        }

        public TutorialMove TryPlace(TileFace tile, BoardEnd end)
        {
            if (!AwaitsPlayer) return TutorialMove.NotPlayerTurn;
            var step = steps[StepIndex];
            if (!Same(tile, step.Tile)) return TutorialMove.WrongTile;
            if (end != step.End) return TutorialMove.WrongEnd;
            PlayerPoints += step.Points;
            Advance();
            return TutorialMove.Accepted;
        }

        public bool Skip()
        {
            if (IsFinished) return false;
            State = TutorialState.Skipped;
            return true;
        }

        void Advance()
        {
            StepIndex++;
            if (StepIndex >= steps.Length)
            {
                State = TutorialState.Completed;
                return;
            }
            Enter();
        }

        void Enter()
        {
            var step = steps[StepIndex];
            if (step.Actor == TutorialActor.Bot) BotPoints += step.Points;
        }

        static bool Same(TileFace a, TileFace b) =>
            (a.Low == b.Low && a.High == b.High) || (a.Low == b.High && a.High == b.Low);
    }
}
