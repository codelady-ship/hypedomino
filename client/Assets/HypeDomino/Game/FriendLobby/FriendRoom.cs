using System;
using System.Collections.Generic;
using HypeDomino.Game.Lobby;
using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.FriendLobby
{
    public readonly struct RoomSeat
    {
        public RoomSeat(Seat seat, string account, bool local)
        {
            Seat = seat;
            Account = account;
            Local = local;
        }

        public Seat Seat { get; }
        public string Account { get; }
        public bool Local { get; }
        public bool Free => string.IsNullOrEmpty(Account);
        public Team Team => SeatOrder.TeamOf(Seat);
    }

    public sealed class FriendRoom
    {
        public static readonly int Capacity = SeatOrder.Size(Mode._2V2);

        readonly List<string> members = new(Capacity);
        readonly List<string> waiting = new(Capacity);
        readonly List<RoomSeat> teamA = new(Capacity / 2);
        readonly List<RoomSeat> teamB = new(Capacity / 2);

        public string LocalAccount { get; private set; } = string.Empty;
        public string Code { get; private set; } = string.Empty;
        public string Leader { get; private set; } = string.Empty;
        public LobbyPhase Phase { get; private set; }
        public Mode Mode { get; private set; }
        public LobbyRefusal Refusal { get; private set; }
        public string RefusalKey => LobbyRefusalKeys.Of(Refusal);
        public ulong RefusedUntilUnixMs { get; private set; }
        public BanReason BanReason { get; private set; }
        public string BanReasonKey => BanReasonKeys.Of(BanReason);
        public SanctionAppeal Appeal { get; private set; }
        public Seat LocalSeat { get; private set; }
        public IReadOnlyList<string> Members => members;
        public IReadOnlyList<string> Waiting => waiting;
        public IReadOnlyList<RoomSeat> TeamA => teamA;
        public IReadOnlyList<RoomSeat> TeamB => teamB;

        public bool InRoom => Code.Length > 0;
        public bool IsLeader => InRoom && Leader == LocalAccount;
        public bool Full => members.Count >= Capacity;
        public bool SeatsFilled => teamA.Count + teamB.Count > 0 && AllTaken(teamA) && AllTaken(teamB);
        public bool CanInvite => InRoom && Phase != LobbyPhase.Match && !Full;
        public bool CanChooseMode => IsLeader && Phase != LobbyPhase.Match;
        public bool CanGoBack => IsLeader && Phase == LobbyPhase.Teams;
        public bool CanStart => IsLeader && Phase == LobbyPhase.Teams && SeatsFilled;

        public bool CanRequestSeat(Seat seat) =>
            InRoom && !IsLeader && Phase == LobbyPhase.Teams && seat != LocalSeat && HasSeat(seat);

        public void Read(LobbyState state, string localAccount)
        {
            LocalAccount = localAccount ?? string.Empty;
            Code = state.Code;
            Leader = state.LeaderAccountId;
            Phase = state.Phase;
            Mode = state.Mode;
            Refusal = state.Refusal;
            RefusedUntilUnixMs = state.RefusedUntilUnixMs;
            BanReason = state.BanReason;
            Appeal = SanctionAppeal.FromWire(state.Appeal);
            LocalSeat = Seat.Unspecified;
            members.Clear();
            members.AddRange(state.MemberAccountIds);
            ReadSeats(state);
            ReadWaiting();
        }

        public double SecondsUntilBanEnds(ServerClock clock, double nowSeconds) =>
            RefusedUntilUnixMs != 0 && clock.Synced ? Math.Max(0d, clock.SecondsUntil(RefusedUntilUnixMs, nowSeconds)) : 0d;

        void ReadSeats(LobbyState state)
        {
            teamA.Clear();
            teamB.Clear();
            foreach (var seat in state.Seats)
            {
                var local = LocalAccount.Length > 0 && seat.AccountId == LocalAccount;
                if (local) LocalSeat = seat.Seat;
                var placed = new RoomSeat(seat.Seat, seat.AccountId, local);
                (placed.Team == Team.A ? teamA : teamB).Add(placed);
            }
        }

        void ReadWaiting()
        {
            waiting.Clear();
            if (teamA.Count + teamB.Count == 0) return;
            foreach (var member in members)
            {
                if (!Seated(member)) waiting.Add(member);
            }
        }

        bool Seated(string account) => Holds(teamA, account) || Holds(teamB, account);

        bool HasSeat(Seat seat) => Contains(teamA, seat) || Contains(teamB, seat);

        static bool Holds(List<RoomSeat> seats, string account)
        {
            foreach (var seat in seats)
            {
                if (seat.Account == account) return true;
            }
            return false;
        }

        static bool Contains(List<RoomSeat> seats, Seat wanted)
        {
            foreach (var seat in seats)
            {
                if (seat.Seat == wanted) return true;
            }
            return false;
        }

        static bool AllTaken(List<RoomSeat> seats)
        {
            foreach (var seat in seats)
            {
                if (seat.Free) return false;
            }
            return true;
        }
    }
}
