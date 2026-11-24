using Hypedomino.V1;

namespace HypeDomino.Net
{
    public readonly struct TileFace
    {
        public readonly uint Low;
        public readonly uint High;

        public TileFace(uint low, uint high)
        {
            Low = low;
            High = high;
        }
    }

    public readonly struct OrientedTileFace
    {
        public readonly uint First;
        public readonly uint Second;

        public OrientedTileFace(uint first, uint second)
        {
            First = first;
            Second = second;
        }
    }

    public readonly struct OpenEndValue
    {
        public readonly BoardEnd End;
        public readonly uint Value;

        public OpenEndValue(BoardEnd end, uint value)
        {
            End = end;
            Value = value;
        }
    }

    public readonly struct SeatState
    {
        public readonly Seat Seat;
        public readonly uint HandSize;
        public readonly Controller Controller;
        public readonly uint LoadingProgress;
        public readonly bool Ready;
        public readonly bool Disconnected;
        public readonly uint ConsecutiveTimeouts;

        public SeatState(Seat seat, uint handSize, Controller controller, uint loadingProgress, bool ready, bool disconnected, uint consecutiveTimeouts)
        {
            Seat = seat;
            HandSize = handSize;
            Controller = controller;
            LoadingProgress = loadingProgress;
            Ready = ready;
            Disconnected = disconnected;
            ConsecutiveTimeouts = consecutiveTimeouts;
        }
    }

    public readonly struct LegalMoveValue
    {
        public readonly TileFace Tile;
        public readonly BoardEnd End;

        public LegalMoveValue(TileFace tile, BoardEnd end)
        {
            Tile = tile;
            End = end;
        }
    }

    public readonly struct OpeningValue
    {
        public readonly Seat Seat;
        public readonly TileFace Tile;

        public OpeningValue(Seat seat, TileFace tile)
        {
            Seat = seat;
            Tile = tile;
        }
    }

    public readonly struct TeamPointsValue
    {
        public readonly uint TeamA;
        public readonly uint TeamB;

        public TeamPointsValue(uint teamA, uint teamB)
        {
            TeamA = teamA;
            TeamB = teamB;
        }
    }

    public readonly struct HandResultValue
    {
        public readonly TeamPointsValue Pips;
        public readonly Team Winner;
        public readonly TeamPointsValue Awarded;
        public readonly uint CarryOver;
        public readonly TeamPointsValue Scores;
        public readonly Team NextOpener;
        public readonly HandEnding Ending;
        public readonly bool HandPointsLocked;
        public readonly uint CarriedIn;

        public HandResultValue(TeamPointsValue pips, Team winner, TeamPointsValue awarded, uint carryOver, TeamPointsValue scores, Team nextOpener, HandEnding ending, bool handPointsLocked, uint carriedIn)
        {
            Pips = pips;
            Winner = winner;
            Awarded = awarded;
            CarryOver = carryOver;
            Scores = scores;
            NextOpener = nextOpener;
            Ending = ending;
            HandPointsLocked = handPointsLocked;
            CarriedIn = carriedIn;
        }
    }

    public readonly struct SeatTilesValue
    {
        public readonly Seat Seat;
        public readonly int Start;
        public readonly int Count;

        public SeatTilesValue(Seat seat, int start, int count)
        {
            Seat = seat;
            Start = start;
            Count = count;
        }
    }
}
