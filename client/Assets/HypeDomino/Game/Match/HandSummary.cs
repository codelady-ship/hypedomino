using System;
using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public sealed class HandSummary
    {
        readonly SeatTilesValue[] leftSeats = new SeatTilesValue[MatchSnapshot.MaxSeats];
        readonly TileFace[] leftTiles = new TileFace[MatchSnapshot.MaxTiles];
        int leftSeatCount;

        public uint Hand { get; private set; }
        public HandEnding Ending { get; private set; }
        public bool HasWinner { get; private set; }
        public Team Winner { get; private set; }
        public uint PipsA { get; private set; }
        public uint PipsB { get; private set; }
        public uint CountedPips { get; private set; }
        public uint HandPoints { get; private set; }
        public uint CarriedIn { get; private set; }
        public bool Locked { get; private set; }
        public uint CarryOverKept { get; private set; }
        public ReadOnlySpan<SeatTilesValue> LeftSeats => new(leftSeats, 0, leftSeatCount);

        public ReadOnlySpan<TileFace> LeftTiles(SeatTilesValue seat) => new(leftTiles, seat.Start, seat.Count);

        public bool Read(MatchSnapshot view)
        {
            if (!view.HasLastResult || view.LeftSeats.Length == 0 || view.HandsPlayed == Hand) return false;
            Hand = view.HandsPlayed;
            Explain(view.LastResult);
            CopyLeft(view);
            return true;
        }

        void Explain(HandResultValue result)
        {
            Ending = result.Ending;
            PipsA = result.Pips.TeamA;
            PipsB = result.Pips.TeamB;
            HasWinner = result.Winner != Team.Unspecified;
            Winner = result.Winner;
            Locked = result.HandPointsLocked;
            CarriedIn = HasWinner ? result.CarriedIn : 0;
            CountedPips = !HasWinner ? 0 : Winner == Team.A ? PipsB : PipsA;
            var awarded = Winner == Team.A ? result.Awarded.TeamA : result.Awarded.TeamB;
            HandPoints = HasWinner && awarded > CarriedIn ? awarded - CarriedIn : 0;
            CarryOverKept = HasWinner ? 0 : result.CarryOver;
        }

        void CopyLeft(MatchSnapshot view)
        {
            var seats = view.LeftSeats;
            leftSeatCount = 0;
            var tileCount = 0;
            for (var i = 0; i < seats.Length && i < leftSeats.Length; i++)
            {
                var tiles = view.LeftTiles(seats[i]);
                var count = Math.Min(tiles.Length, leftTiles.Length - tileCount);
                tiles[..count].CopyTo(new Span<TileFace>(leftTiles, tileCount, count));
                leftSeats[leftSeatCount++] = new SeatTilesValue(seats[i].Seat, tileCount, count);
                tileCount += count;
            }
        }
    }
}
