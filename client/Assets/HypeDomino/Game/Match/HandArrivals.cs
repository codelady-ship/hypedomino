using System;
using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public enum HandArrival
    {
        None,
        Deal,
        Draw,
    }

    public sealed class HandArrivals
    {
        readonly TileFace[] held = new TileFace[MatchSnapshot.MaxTiles];
        readonly int[] arrived = new int[MatchSnapshot.MaxTiles];
        readonly Seat[] seats = new Seat[MatchSnapshot.MaxSeats];
        readonly uint[] sizes = new uint[MatchSnapshot.MaxSeats];
        readonly uint[] grown = new uint[MatchSnapshot.MaxSeats];
        int heldCount;
        int arrivedCount;
        int seatCount;
        bool inHand;
        uint hand;

        public HandArrival Kind { get; private set; }

        public ReadOnlySpan<int> ArrivedTiles => new(arrived, 0, arrivedCount);

        public uint ArrivedAt(Seat seat)
        {
            for (var at = 0; at < seatCount; at++)
                if (seats[at] == seat) return grown[at];
            return 0;
        }

        public HandArrival Read(MatchSnapshot view)
        {
            Kind = HandArrival.None;
            arrivedCount = 0;
            if (!IsInHand(view.State))
            {
                inHand = false;
                Array.Clear(grown, 0, grown.Length);
                return Kind;
            }
            var fresh = !inHand || view.HandsPlayed != hand;
            if (fresh) Begin(view);
            else Compare(view);
            Keep(view);
            return Kind;
        }

        void Begin(MatchSnapshot view)
        {
            heldCount = 0;
            seatCount = 0;
            var dealing = view.Line.Length == 0 && view.Top.Length == 0 && view.Bottom.Length == 0;
            if (!dealing)
            {
                Array.Clear(grown, 0, grown.Length);
                return;
            }
            Compare(view);
            if (arrivedCount > 0 || AnySeatGrew()) Kind = HandArrival.Deal;
        }

        void Compare(MatchSnapshot view)
        {
            var tiles = view.Hand;
            for (var at = 0; at < tiles.Length && arrivedCount < arrived.Length; at++)
                if (!Held(tiles[at])) arrived[arrivedCount++] = at;
            var states = view.Seats;
            for (var at = 0; at < states.Length && at < grown.Length; at++)
            {
                var before = SizeBefore(states[at].Seat);
                grown[at] = states[at].HandSize > before ? states[at].HandSize - before : 0;
            }
            for (var at = states.Length; at < grown.Length; at++) grown[at] = 0;
            if (arrivedCount > 0 || AnySeatGrew()) Kind = HandArrival.Draw;
        }

        void Keep(MatchSnapshot view)
        {
            inHand = true;
            hand = view.HandsPlayed;
            var tiles = view.Hand;
            heldCount = Math.Min(tiles.Length, held.Length);
            tiles.Slice(0, heldCount).CopyTo(held);
            var states = view.Seats;
            seatCount = Math.Min(states.Length, seats.Length);
            for (var at = 0; at < seatCount; at++)
            {
                seats[at] = states[at].Seat;
                sizes[at] = states[at].HandSize;
            }
        }

        bool Held(TileFace tile)
        {
            for (var at = 0; at < heldCount; at++)
                if (held[at].Low == tile.Low && held[at].High == tile.High) return true;
            return false;
        }

        uint SizeBefore(Seat seat)
        {
            for (var at = 0; at < seatCount; at++)
                if (seats[at] == seat) return sizes[at];
            return 0;
        }

        bool AnySeatGrew()
        {
            for (var at = 0; at < grown.Length; at++)
                if (grown[at] > 0) return true;
            return false;
        }

        static bool IsInHand(MatchState state) => state == MatchState.Opening || state == MatchState.Playing;
    }
}
