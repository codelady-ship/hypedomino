using System;
using System.Collections.Generic;
using HypeDomino.Game.Profile;
using HypeDomino.Net.Api;
using HypeDomino.UI.Profile;
using Hypedomino.V1;

namespace HypeDomino.Game.FriendLobby
{
    public readonly struct RoomMember
    {
        public RoomMember(string account, PlayerProfile profile, bool leader, bool local, Seat seat)
        {
            Account = account;
            Profile = profile;
            IsLeader = leader;
            IsLocal = local;
            Seat = seat;
        }

        public string Account { get; }
        public PlayerProfile Profile { get; }
        public bool IsLeader { get; }
        public bool IsLocal { get; }
        public Seat Seat { get; }
        public bool Seated => Seat != Seat.Unspecified;
        public bool ProfileLoaded => Profile != null;
        public string Nickname => Profile?.Nickname;
        public string AvatarId => Profile?.AvatarId;
        public string FrameId => Profile?.FrameId;
    }

    public sealed class RoomMembers
    {
        readonly ProfileLoader loader;
        readonly Action<ProfileLoadResult> onLoaded;
        readonly List<RoomMember> members = new(FriendRoom.Capacity);
        readonly Dictionary<string, PlayerProfile> profiles = new(FriendRoom.Capacity);
        readonly List<string> unloaded = new(FriendRoom.Capacity);
        readonly List<string> left = new(FriendRoom.Capacity);
        string loading;

        public RoomMembers(ApiClient api)
        {
            loader = new ProfileLoader(api ?? throw new ArgumentNullException(nameof(api)));
            onLoaded = Loaded;
        }

        public IReadOnlyList<RoomMember> All => members;

        public event Action Changed;

        public void Read(FriendRoom room)
        {
            members.Clear();
            foreach (var account in room.Members)
            {
                var profile = profiles.TryGetValue(account, out var known) ? known : null;
                members.Add(new RoomMember(account, profile, account == room.Leader, account == room.LocalAccount, SeatOf(room, account)));
            }
            ForgetLeft();
            QueueUnloaded();
            Changed?.Invoke();
            LoadNext();
        }

        public int IndexOf(string account)
        {
            for (var index = 0; index < members.Count; index++)
            {
                if (members[index].Account == account) return index;
            }
            return -1;
        }

        void ForgetLeft()
        {
            left.Clear();
            foreach (var account in profiles.Keys)
            {
                if (IndexOf(account) < 0) left.Add(account);
            }
            foreach (var account in left) profiles.Remove(account);
        }

        void QueueUnloaded()
        {
            unloaded.Clear();
            foreach (var member in members)
            {
                if (!member.ProfileLoaded && member.Account != loading) unloaded.Add(member.Account);
            }
        }

        void LoadNext()
        {
            while (loading == null && unloaded.Count > 0)
            {
                var account = unloaded[0];
                unloaded.RemoveAt(0);
                if (IndexOf(account) < 0 || profiles.ContainsKey(account)) continue;
                loading = account;
                if (!loader.LoadPublic(account, onLoaded)) loading = null;
            }
        }

        void Loaded(ProfileLoadResult result)
        {
            var account = loading;
            loading = null;
            var index = IndexOf(account);
            if (index >= 0 && result.Profile != null)
            {
                profiles[account] = result.Profile;
                var member = members[index];
                members[index] = new RoomMember(account, result.Profile, member.IsLeader, member.IsLocal, member.Seat);
                Changed?.Invoke();
            }
            LoadNext();
        }

        static Seat SeatOf(FriendRoom room, string account)
        {
            foreach (var seat in room.TeamA)
            {
                if (seat.Account == account) return seat.Seat;
            }
            foreach (var seat in room.TeamB)
            {
                if (seat.Account == account) return seat.Seat;
            }
            return Seat.Unspecified;
        }
    }
}
