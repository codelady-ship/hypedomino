using System;
using HypeDomino.Net;
using HypeDomino.Net.Integrity;
using Hypedomino.V1;

namespace HypeDomino.Game.Lobby
{
    public enum PlayCardState
    {
        Ready,
        Starting,
        Searching,
        Locked,
    }

    public sealed class LobbyPlay
    {
        readonly MatchQueue queue;
        readonly Action<IntegrityPurpose, Action<IntegrityProof>> requestProof;
        ProofRequest proving;
        int joiningCard = -1;

        public LobbyPlay(MatchQueue queue, Action<IntegrityPurpose, Action<IntegrityProof>> requestProof)
        {
            this.queue = queue;
            this.requestProof = requestProof;
            queue.Changed += OnQueueChanged;
        }

        public bool ProvingDevice => proving != null;
        public bool CanCancel => proving != null || queue.CanLeave;

        public event Action Changed;

        public PlayCardState StateOf(int card)
        {
            if (queue.Phase == QueuePhase.MatchFound) return PlayCardState.Locked;
            if ((proving != null && proving.Card == card) || (queue.AwaitingReply && joiningCard == card)) return PlayCardState.Starting;
            if (proving != null || queue.AwaitingReply) return PlayCardState.Locked;
            if (queue.Phase == QueuePhase.Searching && PlayModes.At(card).Is(queue.Mode, queue.Kind)) return PlayCardState.Searching;
            return PlayCardState.Ready;
        }

        public bool Play(Session session, int card)
        {
            if (!session.Accepted || !PlayModes.IsCard(card) || StateOf(card) != PlayCardState.Ready) return false;
            var request = new ProofRequest(this, session, card);
            proving = request;
            Changed?.Invoke();
            requestProof(PlayModes.At(card).Purpose, request.Arrived);
            return true;
        }

        public bool Cancel(Session session)
        {
            if (proving != null)
            {
                proving = null;
                Changed?.Invoke();
                return true;
            }
            if (!queue.TryLeave(session)) return false;
            joiningCard = -1;
            Changed?.Invoke();
            return true;
        }

        void Proved(ProofRequest request, IntegrityProof proof)
        {
            if (request != proving) return;
            proving = null;
            var mode = PlayModes.At(request.Card);
            if (queue.TryJoin(request.Session, mode.Mode, mode.Kind, proof)) joiningCard = request.Card;
            Changed?.Invoke();
        }

        void OnQueueChanged()
        {
            if (!queue.AwaitingReply) joiningCard = -1;
            Changed?.Invoke();
        }

        sealed class ProofRequest
        {
            readonly LobbyPlay owner;

            public ProofRequest(LobbyPlay owner, Session session, int card)
            {
                this.owner = owner;
                Session = session;
                Card = card;
                Arrived = OnArrived;
            }

            public Session Session { get; }
            public int Card { get; }
            public Action<IntegrityProof> Arrived { get; }

            void OnArrived(IntegrityProof proof) => owner.Proved(this, proof);
        }
    }
}
