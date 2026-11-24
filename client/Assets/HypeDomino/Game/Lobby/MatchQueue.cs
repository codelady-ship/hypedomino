using System;
using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.Lobby
{
    public enum QueuePhase
    {
        Idle,
        Searching,
        MatchFound,
        Refused,
    }

    public sealed class MatchQueue
    {
        readonly JoinQueueIntent join = new();
        readonly LeaveQueueIntent leave = new();
        int repliesDue;
        bool leaveSent;

        public QueuePhase Phase { get; private set; }
        public Mode Mode { get; private set; }
        public QueueKind Kind { get; private set; }
        public ulong JoinedUnixMs { get; private set; }
        public ulong SearchStartsUnixMs { get; private set; }
        public QueueRefusal Refusal { get; private set; }
        public ulong RefusedUntilUnixMs { get; private set; }
        public BanReason BanReason { get; private set; }
        public SanctionAppeal Appeal { get; private set; }
        public bool AwaitingReply => repliesDue > 0;
        public bool CanLeave => !leaveSent && (Phase == QueuePhase.Searching || repliesDue > 0);
        public string RefusalKey => Phase == QueuePhase.Refused ? QueueRefusalKeys.Of(Refusal) : null;
        public string BanReasonKey => Phase == QueuePhase.Refused ? BanReasonKeys.Of(BanReason) : null;

        public event Action Changed;

        public event Action<Mode, QueueKind> JoinSent;

        public void Follow(Session session, AppFlow flow)
        {
            session.HelloReceived += OnHello;
            session.QueueStateReceived += Read;
            flow.Changed += OnFlowChanged;
        }

        public bool TryJoin(Session session, Mode mode, QueueKind kind, IntegrityProof integrity)
        {
            if (!session.Accepted || repliesDue > 0 || Phase == QueuePhase.MatchFound) return false;
            if (!IsOffered(mode) || !IsOffered(kind)) return false;
            join.Mode = mode;
            join.Queue = kind;
            join.Integrity = integrity;
            session.Send(Envelope.PayloadOneofCase.JoinQueueIntent, join);
            repliesDue++;
            JoinSent?.Invoke(mode, kind);
            return true;
        }

        public bool TryLeave(Session session)
        {
            if (!session.Accepted || !CanLeave) return false;
            session.Send(Envelope.PayloadOneofCase.LeaveQueueIntent, leave);
            repliesDue++;
            leaveSent = true;
            return true;
        }

        public void Read(QueueState state)
        {
            if (state.Status != QueueStatus.MatchFound) Answered();
            switch (state.Status)
            {
                case QueueStatus.Searching:
                    Show(QueuePhase.Searching, state.Mode, state.Queue);
                    JoinedUnixMs = state.JoinedUnixMs;
                    SearchStartsUnixMs = state.SearchStartsUnixMs;
                    break;
                case QueueStatus.Left:
                    Show(QueuePhase.Idle, Mode.Unspecified, QueueKind.Unspecified);
                    break;
                case QueueStatus.MatchFound:
                    Phase = QueuePhase.MatchFound;
                    break;
                case QueueStatus.Refused:
                    Show(QueuePhase.Refused, state.Mode, state.Queue);
                    Refusal = state.Refusal;
                    RefusedUntilUnixMs = state.RefusedUntilUnixMs;
                    BanReason = state.BanReason;
                    Appeal = SanctionAppeal.FromWire(state.Appeal);
                    break;
                default:
                    return;
            }
            Changed?.Invoke();
        }

        public void Reset()
        {
            repliesDue = 0;
            leaveSent = false;
            Show(QueuePhase.Idle, Mode.Unspecified, QueueKind.Unspecified);
            Changed?.Invoke();
        }

        public double SecondsSearching(ServerClock clock, double nowSeconds) =>
            Phase == QueuePhase.Searching && clock.Synced ? Math.Max(0d, -clock.SecondsUntil(JoinedUnixMs, nowSeconds)) : 0d;

        public double SecondsUntilSearch(ServerClock clock, double nowSeconds) =>
            Phase == QueuePhase.Searching && clock.Synced ? Math.Max(0d, clock.SecondsUntil(SearchStartsUnixMs, nowSeconds)) : 0d;

        public double SecondsUntilBanEnds(ServerClock clock, double nowSeconds) =>
            Phase == QueuePhase.Refused && RefusedUntilUnixMs != 0 && clock.Synced ? Math.Max(0d, clock.SecondsUntil(RefusedUntilUnixMs, nowSeconds)) : 0d;

        void Answered()
        {
            if (repliesDue > 0) repliesDue--;
            if (repliesDue == 0) leaveSent = false;
        }

        void Show(QueuePhase phase, Mode mode, QueueKind kind)
        {
            Phase = phase;
            Mode = mode;
            Kind = kind;
            JoinedUnixMs = 0;
            SearchStartsUnixMs = 0;
            Refusal = QueueRefusal.Unspecified;
            RefusedUntilUnixMs = 0;
            BanReason = BanReason.Unspecified;
            Appeal = default;
        }

        void OnHello(ServerHello hello) => Reset();

        void OnFlowChanged(AppState state)
        {
            if (state == AppState.Lobby) Reset();
        }

        static bool IsOffered(Mode mode) => mode == Mode._1V1 || mode == Mode._2V2;

        static bool IsOffered(QueueKind kind) => kind == QueueKind.Ranked || kind == QueueKind.Casual;
    }
}
