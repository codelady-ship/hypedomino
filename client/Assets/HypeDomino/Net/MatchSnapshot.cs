using System;
using System.Collections.Generic;
using System.Text;
using Google.Protobuf;
using Hypedomino.V1;

namespace HypeDomino.Net
{
    public sealed class MatchSnapshot
    {
        public const int MaxTiles = 28;
        public const int MaxSeats = 4;
        public const int MaxOpenEnds = 4;
        public const int MaxLegalMoves = MaxTiles * MaxOpenEnds;

        const int Varint = WireFormat.Varint;
        const int Bytes = WireFormat.LengthDelimited;

        readonly TileFace[] hand = new TileFace[MaxTiles];
        readonly SeatState[] seats = new SeatState[MaxSeats];
        readonly OrientedTileFace[] line = new OrientedTileFace[MaxTiles];
        readonly OrientedTileFace[] top = new OrientedTileFace[MaxTiles];
        readonly OrientedTileFace[] bottom = new OrientedTileFace[MaxTiles];
        readonly OpenEndValue[] openEnds = new OpenEndValue[MaxOpenEnds];
        readonly LegalMoveValue[] legalMoves = new LegalMoveValue[MaxLegalMoves];
        readonly SeatTilesValue[] leftSeats = new SeatTilesValue[MaxSeats];
        readonly TileFace[] leftTiles = new TileFace[MaxTiles];
        readonly List<RosterEntry> roster = new(MaxSeats);
        int handCount;
        int seatCount;
        int lineCount;
        int topCount;
        int bottomCount;
        int openEndCount;
        int legalMoveCount;
        int leftSeatCount;
        int leftTileCount;
        bool lockedA;
        bool lockedB;
        bool rosterRead;

        public MatchState State { get; private set; }
        public Seat Seat { get; private set; }
        public Mode Mode { get; private set; }
        public uint HandsPlayed { get; private set; }
        public TeamPointsValue Scores { get; private set; }
        public uint CarryOver { get; private set; }
        public uint BoneyardSize { get; private set; }
        public int OpeningIndex { get; private set; }
        public bool HasSpinner { get; private set; }
        public TileFace Spinner { get; private set; }
        public Seat Current { get; private set; }
        public Team OpeningTeam { get; private set; }
        public bool HasForcedOpening { get; private set; }
        public OpeningValue ForcedOpening { get; private set; }
        public uint TurnPlacements { get; private set; }
        public ulong TurnDeadlineUnixMs { get; private set; }
        public bool HasLastResult { get; private set; }
        public HandResultValue LastResult { get; private set; }
        public Team Winner { get; private set; }
        public ulong LoadingDeadlineUnixMs { get; private set; }
        public uint AfkConsecutiveTimeouts { get; private set; }
        public uint TargetScore { get; private set; }
        public uint HandPointsLockThreshold { get; private set; }
        public uint TurnSeconds { get; private set; }
        public uint TurnWarningSeconds { get; private set; }
        public string Table { get; private set; } = string.Empty;

        public ReadOnlySpan<TileFace> Hand => new(hand, 0, handCount);
        public ReadOnlySpan<SeatState> Seats => new(seats, 0, seatCount);
        public ReadOnlySpan<OrientedTileFace> Line => new(line, 0, lineCount);
        public ReadOnlySpan<OrientedTileFace> Top => new(top, 0, topCount);
        public ReadOnlySpan<OrientedTileFace> Bottom => new(bottom, 0, bottomCount);
        public ReadOnlySpan<OpenEndValue> OpenEnds => new(openEnds, 0, openEndCount);
        public ReadOnlySpan<LegalMoveValue> LegalMoves => new(legalMoves, 0, legalMoveCount);
        public ReadOnlySpan<SeatTilesValue> LeftSeats => new(leftSeats, 0, leftSeatCount);

        public ReadOnlySpan<TileFace> LeftTiles(SeatTilesValue seat) => new(leftTiles, seat.Start, seat.Count);

        public bool IsHandPointsLocked(Team team) => team == Team.A ? lockedA : team == Team.B && lockedB;
        public IReadOnlyList<RosterEntry> Roster => roster;

        public bool TryRead(ReadOnlySpan<byte> view)
        {
            Reset();
            var at = 0;
            while (at < view.Length)
            {
                if (!TryReadTag(view, ref at, out var tag)) return false;
                if (!TryReadField(view, ref at, tag)) return false;
            }
            return OpeningIndex == 0 || OpeningIndex < lineCount;
        }

        void Reset()
        {
            State = default;
            Seat = default;
            Mode = default;
            HandsPlayed = 0;
            Scores = default;
            CarryOver = 0;
            BoneyardSize = 0;
            OpeningIndex = 0;
            HasSpinner = false;
            Spinner = default;
            Current = default;
            OpeningTeam = default;
            HasForcedOpening = false;
            ForcedOpening = default;
            TurnPlacements = 0;
            TurnDeadlineUnixMs = 0;
            HasLastResult = false;
            LastResult = default;
            Winner = default;
            LoadingDeadlineUnixMs = 0;
            AfkConsecutiveTimeouts = 0;
            TargetScore = 0;
            HandPointsLockThreshold = 0;
            TurnSeconds = 0;
            TurnWarningSeconds = 0;
            lockedA = lockedB = false;
            handCount = seatCount = lineCount = topCount = bottomCount = openEndCount = legalMoveCount = leftSeatCount = leftTileCount = 0;
            rosterRead = false;
        }

        bool TryReadField(ReadOnlySpan<byte> view, ref int at, int tag)
        {
            ulong value;
            ReadOnlySpan<byte> bytes;
            switch (tag)
            {
                case (1 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    State = (MatchState)(int)value;
                    return true;
                case (2 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    Seat = (Seat)(int)value;
                    return true;
                case (3 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    Mode = (Mode)(int)value;
                    return true;
                case (4 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    HandsPlayed = (uint)value;
                    return true;
                case (5 << 3) | Bytes:
                    if (!WireFormat.TryReadBytes(view, ref at, out bytes) || !TryReadTeamPoints(bytes, out var scores)) return false;
                    Scores = scores;
                    return true;
                case (6 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    CarryOver = (uint)value;
                    return true;
                case (7 << 3) | Bytes:
                    if (!WireFormat.TryReadBytes(view, ref at, out bytes) || handCount == MaxTiles || !TryReadTile(bytes, out var tile)) return false;
                    hand[handCount++] = tile;
                    return true;
                case (8 << 3) | Bytes:
                    if (!WireFormat.TryReadBytes(view, ref at, out bytes) || seatCount == MaxSeats || !TryReadSeat(bytes, out var seat)) return false;
                    seats[seatCount++] = seat;
                    return true;
                case (9 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    BoneyardSize = (uint)value;
                    return true;
                case (10 << 3) | Bytes:
                    return WireFormat.TryReadBytes(view, ref at, out bytes) && TryReadBoard(bytes);
                case (11 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    Current = (Seat)(int)value;
                    return true;
                case (12 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    OpeningTeam = (Team)(int)value;
                    return true;
                case (13 << 3) | Bytes:
                    if (!WireFormat.TryReadBytes(view, ref at, out bytes) || !TryReadOpening(bytes, out var opening)) return false;
                    ForcedOpening = opening;
                    HasForcedOpening = true;
                    return true;
                case (14 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    TurnPlacements = (uint)value;
                    return true;
                case (15 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    TurnDeadlineUnixMs = value;
                    return true;
                case (16 << 3) | Bytes:
                    if (!WireFormat.TryReadBytes(view, ref at, out bytes) || legalMoveCount == MaxLegalMoves || !TryReadLegalMove(bytes, out var move)) return false;
                    legalMoves[legalMoveCount++] = move;
                    return true;
                case (17 << 3) | Bytes:
                    if (!WireFormat.TryReadBytes(view, ref at, out bytes) || !TryReadHandResult(bytes, out var result)) return false;
                    LastResult = result;
                    HasLastResult = true;
                    return true;
                case (18 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    Winner = (Team)(int)value;
                    return true;
                case (19 << 3) | Bytes:
                    return WireFormat.TryReadBytes(view, ref at, out bytes) && TryReadRosterEntry(bytes);
                case (20 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    LoadingDeadlineUnixMs = value;
                    return true;
                case (21 << 3) | Bytes:
                    if (!WireFormat.TryReadBytes(view, ref at, out bytes)) return false;
                    Table = Encoding.UTF8.GetString(bytes);
                    return true;
                case (22 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    AfkConsecutiveTimeouts = (uint)value;
                    return true;
                case (23 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    TargetScore = (uint)value;
                    return true;
                case (24 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    HandPointsLockThreshold = (uint)value;
                    return true;
                case (26 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    TurnSeconds = (uint)value;
                    return true;
                case (27 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    TurnWarningSeconds = (uint)value;
                    return true;
                case (25 << 3) | Varint:
                    if (!WireFormat.TryReadVarint(view, ref at, out value)) return false;
                    Lock((Team)(int)value);
                    return true;
                case (25 << 3) | Bytes:
                    if (!WireFormat.TryReadBytes(view, ref at, out bytes)) return false;
                    var packed = 0;
                    while (packed < bytes.Length)
                    {
                        if (!WireFormat.TryReadVarint(bytes, ref packed, out value)) return false;
                        Lock((Team)(int)value);
                    }
                    return true;
                default:
                    return WireFormat.TrySkip(view, ref at, tag & 7);
            }
        }

        bool TryReadBoard(ReadOnlySpan<byte> board)
        {
            var at = 0;
            while (at < board.Length)
            {
                if (!TryReadTag(board, ref at, out var tag)) return false;
                ReadOnlySpan<byte> bytes;
                switch (tag)
                {
                    case (1 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(board, ref at, out bytes) || !TryAddOriented(bytes, line, ref lineCount)) return false;
                        break;
                    case (2 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(board, ref at, out bytes) || !TryReadTile(bytes, out var spinner)) return false;
                        Spinner = spinner;
                        HasSpinner = true;
                        break;
                    case (3 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(board, ref at, out bytes) || !TryAddOriented(bytes, top, ref topCount)) return false;
                        break;
                    case (4 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(board, ref at, out bytes) || !TryAddOriented(bytes, bottom, ref bottomCount)) return false;
                        break;
                    case (5 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(board, ref at, out bytes) || openEndCount == MaxOpenEnds || !TryReadOpenEnd(bytes, out var end)) return false;
                        openEnds[openEndCount++] = end;
                        break;
                    case (6 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(board, ref at, out var opening) || opening >= MaxTiles) return false;
                        OpeningIndex = (int)opening;
                        break;
                    default:
                        if (!WireFormat.TrySkip(board, ref at, tag & 7)) return false;
                        break;
                }
            }
            return true;
        }

        bool TryReadRosterEntry(ReadOnlySpan<byte> bytes)
        {
            if (!rosterRead)
            {
                roster.Clear();
                rosterRead = true;
            }
            if (roster.Count == MaxSeats) return false;
            var entry = new RosterEntry();
            try
            {
                entry.MergeFrom(bytes);
            }
            catch (InvalidProtocolBufferException)
            {
                return false;
            }
            roster.Add(entry);
            return true;
        }

        static bool TryAddOriented(ReadOnlySpan<byte> bytes, OrientedTileFace[] target, ref int count)
        {
            if (count == target.Length || !TryReadPair(bytes, out var first, out var second)) return false;
            target[count++] = new OrientedTileFace(first, second);
            return true;
        }

        static bool TryReadTile(ReadOnlySpan<byte> bytes, out TileFace tile)
        {
            var read = TryReadPair(bytes, out var low, out var high);
            tile = new TileFace(low, high);
            return read;
        }

        static bool TryReadTeamPoints(ReadOnlySpan<byte> bytes, out TeamPointsValue points)
        {
            var read = TryReadPair(bytes, out var teamA, out var teamB);
            points = new TeamPointsValue(teamA, teamB);
            return read;
        }

        static bool TryReadPair(ReadOnlySpan<byte> bytes, out uint first, out uint second)
        {
            first = second = 0;
            var at = 0;
            while (at < bytes.Length)
            {
                if (!TryReadTag(bytes, ref at, out var tag)) return false;
                switch (tag)
                {
                    case (1 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out var a)) return false;
                        first = (uint)a;
                        break;
                    case (2 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out var b)) return false;
                        second = (uint)b;
                        break;
                    default:
                        if (!WireFormat.TrySkip(bytes, ref at, tag & 7)) return false;
                        break;
                }
            }
            return true;
        }

        static bool TryReadOpenEnd(ReadOnlySpan<byte> bytes, out OpenEndValue end)
        {
            var read = TryReadPair(bytes, out var side, out var value);
            end = new OpenEndValue((BoardEnd)(int)side, value);
            return read;
        }

        static bool TryReadSeat(ReadOnlySpan<byte> bytes, out SeatState seat)
        {
            seat = default;
            Seat which = default;
            Controller controller = default;
            uint handSize = 0, loadingProgress = 0, consecutiveTimeouts = 0;
            bool ready = false, disconnected = false;
            var at = 0;
            while (at < bytes.Length)
            {
                if (!TryReadTag(bytes, ref at, out var tag)) return false;
                if ((tag & 7) != Varint || tag >> 3 > 7)
                {
                    if (!WireFormat.TrySkip(bytes, ref at, tag & 7)) return false;
                    continue;
                }
                if (!WireFormat.TryReadVarint(bytes, ref at, out var value)) return false;
                switch (tag >> 3)
                {
                    case 1: which = (Seat)(int)value; break;
                    case 2: handSize = (uint)value; break;
                    case 3: controller = (Controller)(int)value; break;
                    case 4: loadingProgress = (uint)value; break;
                    case 5: ready = value != 0; break;
                    case 6: disconnected = value != 0; break;
                    case 7: consecutiveTimeouts = (uint)value; break;
                }
            }
            seat = new SeatState(which, handSize, controller, loadingProgress, ready, disconnected, consecutiveTimeouts);
            return true;
        }

        static bool TryReadLegalMove(ReadOnlySpan<byte> bytes, out LegalMoveValue move)
        {
            move = default;
            TileFace tile = default;
            BoardEnd end = default;
            var at = 0;
            while (at < bytes.Length)
            {
                if (!TryReadTag(bytes, ref at, out var tag)) return false;
                switch (tag)
                {
                    case (1 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(bytes, ref at, out var tileBytes) || !TryReadTile(tileBytes, out tile)) return false;
                        break;
                    case (2 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out var value)) return false;
                        end = (BoardEnd)(int)value;
                        break;
                    default:
                        if (!WireFormat.TrySkip(bytes, ref at, tag & 7)) return false;
                        break;
                }
            }
            move = new LegalMoveValue(tile, end);
            return true;
        }

        static bool TryReadOpening(ReadOnlySpan<byte> bytes, out OpeningValue opening)
        {
            opening = default;
            Seat seat = default;
            TileFace tile = default;
            var at = 0;
            while (at < bytes.Length)
            {
                if (!TryReadTag(bytes, ref at, out var tag)) return false;
                switch (tag)
                {
                    case (1 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out var value)) return false;
                        seat = (Seat)(int)value;
                        break;
                    case (2 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(bytes, ref at, out var tileBytes) || !TryReadTile(tileBytes, out tile)) return false;
                        break;
                    default:
                        if (!WireFormat.TrySkip(bytes, ref at, tag & 7)) return false;
                        break;
                }
            }
            opening = new OpeningValue(seat, tile);
            return true;
        }

        bool TryReadHandResult(ReadOnlySpan<byte> bytes, out HandResultValue result)
        {
            result = default;
            TeamPointsValue pips = default, awarded = default, scores = default;
            Team winner = default, nextOpener = default;
            var ending = HandEnding.Unspecified;
            var locked = false;
            uint carryOver = 0, carriedIn = 0;
            leftSeatCount = leftTileCount = 0;
            var at = 0;
            while (at < bytes.Length)
            {
                if (!TryReadTag(bytes, ref at, out var tag)) return false;
                ReadOnlySpan<byte> points;
                ulong value;
                switch (tag)
                {
                    case (1 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(bytes, ref at, out points) || !TryReadTeamPoints(points, out pips)) return false;
                        break;
                    case (2 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out value)) return false;
                        winner = (Team)(int)value;
                        break;
                    case (3 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(bytes, ref at, out points) || !TryReadTeamPoints(points, out awarded)) return false;
                        break;
                    case (4 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out value)) return false;
                        carryOver = (uint)value;
                        break;
                    case (5 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(bytes, ref at, out points) || !TryReadTeamPoints(points, out scores)) return false;
                        break;
                    case (6 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out value)) return false;
                        nextOpener = (Team)(int)value;
                        break;
                    case (7 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out value)) return false;
                        ending = (HandEnding)(int)value;
                        break;
                    case (8 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out value)) return false;
                        locked = value != 0;
                        break;
                    case (9 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out value)) return false;
                        carriedIn = (uint)value;
                        break;
                    case (10 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(bytes, ref at, out points) || leftSeatCount == MaxSeats || !TryReadSeatTiles(points)) return false;
                        break;
                    default:
                        if (!WireFormat.TrySkip(bytes, ref at, tag & 7)) return false;
                        break;
                }
            }
            result = new HandResultValue(pips, winner, awarded, carryOver, scores, nextOpener, ending, locked, carriedIn);
            return true;
        }

        void Lock(Team team)
        {
            if (team == Team.A) lockedA = true;
            if (team == Team.B) lockedB = true;
        }

        bool TryReadSeatTiles(ReadOnlySpan<byte> bytes)
        {
            Seat seat = default;
            var start = leftTileCount;
            var at = 0;
            while (at < bytes.Length)
            {
                if (!TryReadTag(bytes, ref at, out var tag)) return false;
                switch (tag)
                {
                    case (1 << 3) | Varint:
                        if (!WireFormat.TryReadVarint(bytes, ref at, out var value)) return false;
                        seat = (Seat)(int)value;
                        break;
                    case (2 << 3) | Bytes:
                        if (!WireFormat.TryReadBytes(bytes, ref at, out var tileBytes) || leftTileCount == MaxTiles || !TryReadTile(tileBytes, out var tile)) return false;
                        leftTiles[leftTileCount++] = tile;
                        break;
                    default:
                        if (!WireFormat.TrySkip(bytes, ref at, tag & 7)) return false;
                        break;
                }
            }
            leftSeats[leftSeatCount++] = new SeatTilesValue(seat, start, leftTileCount - start);
            return true;
        }

        static bool TryReadTag(ReadOnlySpan<byte> source, ref int at, out int tag)
        {
            tag = 0;
            if (!WireFormat.TryReadVarint(source, ref at, out var raw) || raw > int.MaxValue || raw >> 3 == 0) return false;
            tag = (int)raw;
            return true;
        }
    }
}
