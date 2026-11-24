using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public enum LoadingStatus
    {
        Loading,
        Ready,
        BotPlaying,
    }

    public readonly struct LoadingCard
    {
        public readonly Seat Seat;
        public readonly int Row;
        public readonly int RosterIndex;
        public readonly uint Progress;
        public readonly LoadingStatus Status;

        public LoadingCard(Seat seat, int row, int rosterIndex, uint progress, LoadingStatus status)
        {
            Seat = seat;
            Row = row;
            RosterIndex = rosterIndex;
            Progress = progress;
            Status = status;
        }
    }

    public sealed class LoadingCards
    {
        public const uint FullProgress = 100;

        static readonly Seat[] TwoVersusTwo = { Seat.North, Seat.East, Seat.South, Seat.West };
        static readonly Seat[] OneVersusOne = { Seat.X, Seat.Y };

        readonly LoadingCard[] cards = new LoadingCard[MatchSnapshot.MaxSeats];

        public int Count { get; private set; }

        public float Progress { get; private set; }

        public LoadingCard this[int index] => cards[index];

        public void Read(MatchSnapshot view)
        {
            Count = 0;
            Progress = 0f;
            var order = view.Mode == Mode._1V1 ? OneVersusOne : view.Mode == Mode._2V2 ? TwoVersusTwo : null;
            if (order == null) return;
            var viewer = SeatOrder.Viewer(view.Seat, view.Mode);
            var phaseOver = view.State != MatchState.Loading;
            Add(view, order[viewer % order.Length], 0, phaseOver);
            if (order.Length == TwoVersusTwo.Length) Add(view, order[(viewer + 2) % order.Length], 0, phaseOver);
            Add(view, order[(viewer + 1) % order.Length], 1, phaseOver);
            if (order.Length == TwoVersusTwo.Length) Add(view, order[(viewer + 3) % order.Length], 1, phaseOver);
            Progress = SumProgress() / (float)(Count * FullProgress);
        }

        void Add(MatchSnapshot view, Seat seat, int row, bool phaseOver)
        {
            var rosterIndex = RosterIndexOf(view, seat);
            var rosterBot = rosterIndex >= 0 && view.Roster[rosterIndex].Bot;
            var state = SeatStateOf(view, seat);
            var ready = rosterBot || state.Ready;
            var status = ready ? LoadingStatus.Ready : phaseOver ? LoadingStatus.BotPlaying : LoadingStatus.Loading;
            var progress = ready ? FullProgress : state.LoadingProgress > FullProgress ? FullProgress : state.LoadingProgress;
            cards[Count++] = new LoadingCard(seat, row, rosterIndex, progress, status);
        }

        uint SumProgress()
        {
            var sum = 0u;
            for (var at = 0; at < Count; at++) sum += cards[at].Progress;
            return sum;
        }

        static SeatState SeatStateOf(MatchSnapshot view, Seat seat)
        {
            foreach (var state in view.Seats)
                if (state.Seat == seat) return state;
            return default;
        }

        static int RosterIndexOf(MatchSnapshot view, Seat seat)
        {
            var roster = view.Roster;
            for (var at = 0; at < roster.Count; at++)
                if (roster[at].Seat == seat) return at;
            return -1;
        }
    }
}
