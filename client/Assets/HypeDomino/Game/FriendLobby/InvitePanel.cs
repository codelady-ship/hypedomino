using System;
using System.Collections.Generic;
using HypeDomino.Game.Friends;
using HypeDomino.Net;
using HypeDomino.Net.Api;

namespace HypeDomino.Game.FriendLobby
{
    public enum FriendInviteState
    {
        Invitable,
        Invited,
        AlreadyInRoom,
        InMatch,
        Offline,
        RoomClosed,
    }

    public sealed class InvitePanel
    {
        readonly FriendRoom room;
        readonly LobbyCommands commands;
        readonly FriendListLoader loader;
        readonly Action<FriendListLoadOutcome> onLoaded;
        readonly List<string> invited = new(FriendRoom.Capacity);
        Action<FriendListLoadOutcome> done;

        public InvitePanel(FriendRoom room, LobbyCommands commands, ApiClient api)
        {
            this.room = room ?? throw new ArgumentNullException(nameof(room));
            this.commands = commands ?? throw new ArgumentNullException(nameof(commands));
            loader = new FriendListLoader(api);
            onLoaded = Loaded;
        }

        public IReadOnlyList<Friend> Friends => loader.Friends;
        public string Code => room.Code;
        public bool Loading => loader.Busy;

        public event Action Changed;

        public bool Load(Action<FriendListLoadOutcome> done)
        {
            if (loader.Busy) return false;
            this.done = done;
            return loader.Load(onLoaded);
        }

        public void RoomChanged()
        {
            invited.Clear();
            Changed?.Invoke();
        }

        public FriendInviteState StateOf(Friend friend)
        {
            if (InRoom(friend.Account)) return FriendInviteState.AlreadyInRoom;
            if (!room.CanInvite) return FriendInviteState.RoomClosed;
            if (friend.Presence == FriendPresence.Offline) return FriendInviteState.Offline;
            if (friend.Presence == FriendPresence.InMatch) return FriendInviteState.InMatch;
            return invited.Contains(friend.Account) ? FriendInviteState.Invited : FriendInviteState.Invitable;
        }

        public bool TryInvite(Session session, Friend friend)
        {
            if (StateOf(friend) != FriendInviteState.Invitable) return false;
            if (!commands.TryInvite(session, friend.Account)) return false;
            invited.Add(friend.Account);
            Changed?.Invoke();
            return true;
        }

        bool InRoom(string account)
        {
            var members = room.Members;
            for (var index = 0; index < members.Count; index++)
            {
                if (members[index] == account) return true;
            }
            return false;
        }

        void Loaded(FriendListLoadOutcome outcome)
        {
            var answer = done;
            done = null;
            if (outcome == FriendListLoadOutcome.Loaded) Changed?.Invoke();
            answer?.Invoke(outcome);
        }
    }
}
