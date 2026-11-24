using System;
using System.Collections.Generic;
using HypeDomino.Net;
using Hypedomino.V1;

namespace HypeDomino.Game.FriendLobby
{
    public readonly struct RoomInvite
    {
        public RoomInvite(string code, string from)
        {
            Code = code;
            From = from;
        }

        public string Code { get; }
        public string From { get; }
    }

    public sealed class RoomInvites
    {
        readonly List<RoomInvite> pending = new(FriendRoom.Capacity);

        public string CurrentRoom { get; private set; } = string.Empty;
        public IReadOnlyList<RoomInvite> Pending => pending;
        public bool CanAccept => CurrentRoom.Length == 0;

        public event Action Changed;

        public void Follow(Session session)
        {
            session.LobbyInviteReceived += Read;
            session.LobbyStateReceived += Read;
        }

        public void Read(LobbyInvite invite)
        {
            if (!RoomCode.TryNormalize(invite.Code, out var code)) return;
            if (string.IsNullOrEmpty(invite.FromAccountId) || code == CurrentRoom) return;
            Drop(code);
            pending.Add(new RoomInvite(code, invite.FromAccountId));
            Changed?.Invoke();
        }

        public void Read(LobbyState state)
        {
            CurrentRoom = state.Code;
            if (Drop(CurrentRoom)) Changed?.Invoke();
        }

        public bool TryAccept(Session session, LobbyCommands commands, string code)
        {
            if (!CanAccept || IndexOf(code) < 0) return false;
            if (!commands.TryJoin(session, code)) return false;
            Decline(code);
            return true;
        }

        public void Decline(string code)
        {
            if (Drop(code)) Changed?.Invoke();
        }

        bool Drop(string code)
        {
            var index = IndexOf(code);
            if (index < 0) return false;
            pending.RemoveAt(index);
            return true;
        }

        int IndexOf(string code)
        {
            for (var index = 0; index < pending.Count; index++)
            {
                if (pending[index].Code == code) return index;
            }
            return -1;
        }
    }
}
