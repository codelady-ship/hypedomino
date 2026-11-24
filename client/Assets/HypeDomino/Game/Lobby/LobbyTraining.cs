using HypeDomino.Game.Training;

namespace HypeDomino.Game.Lobby
{
    public sealed class LobbyTraining
    {
        readonly MatchQueue queue;
        readonly LobbyPlay play;
        readonly ITrainingGame game;

        public LobbyTraining(MatchQueue queue, LobbyPlay play, ITrainingGame game)
        {
            this.queue = queue;
            this.play = play;
            this.game = game;
        }

        public PlayCardState State => QueueBusy ? PlayCardState.Locked : PlayCardState.Ready;

        bool QueueBusy => play.ProvingDevice || queue.AwaitingReply || queue.Phase == QueuePhase.Searching || queue.Phase == QueuePhase.MatchFound;

        public bool Start() => State == PlayCardState.Ready && game.TryStart();
    }
}
