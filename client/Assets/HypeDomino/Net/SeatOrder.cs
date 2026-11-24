using System;
using Hypedomino.V1;

namespace HypeDomino.Net
{
    public static class SeatOrder
    {
        public static int Size(Mode mode) => mode switch
        {
            Mode._1V1 => 2,
            Mode._2V2 => 4,
            _ => throw new ArgumentOutOfRangeException(nameof(mode), mode, null),
        };

        public static int Clockwise(Seat seat) => seat switch
        {
            Seat.North => 0,
            Seat.East => 1,
            Seat.South => 2,
            Seat.West => 3,
            Seat.X => 0,
            Seat.Y => 1,
            _ => throw new ArgumentOutOfRangeException(nameof(seat), seat, null),
        };

        public static Team TeamOf(Seat seat) => seat switch
        {
            Seat.North or Seat.South or Seat.X => Team.A,
            Seat.East or Seat.West or Seat.Y => Team.B,
            _ => throw new ArgumentOutOfRangeException(nameof(seat), seat, null),
        };

        public static bool IsIn(Seat seat, int seatCount)
        {
            if (seat == Seat.X || seat == Seat.Y) return seatCount == 2;
            return seatCount == 4 && seat >= Seat.North && seat <= Seat.West;
        }

        public static int Viewer(Seat local, Mode mode)
        {
            if (local != Seat.Unspecified) return Clockwise(local);
            return Clockwise(mode == Mode._1V1 ? Seat.X : Seat.South);
        }
    }
}
