using System;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Emoji;
using Hypedomino.V1;

namespace HypeDomino.Game.FriendLobby
{
    public readonly struct MemberEmoji
    {
        public MemberEmoji(string account, string itemId)
        {
            Account = account;
            ItemId = itemId;
        }

        public string Account { get; }
        public string ItemId { get; }
    }

    public sealed class LobbyEmoji
    {
        static readonly string[] NothingWorn = Array.Empty<string>();

        readonly FriendRoom room;
        readonly EmojiWheel wheel;
        readonly Func<long> clockMs;
        readonly Action<LobbyState> onState;
        readonly Action<LobbyRelay> onRelay;
        readonly EmojiIntent intent = new();
        Session session;
        string[] worn = NothingWorn;
        int emojiSlots;
        string enteredRoom = string.Empty;

        public LobbyEmoji(FriendRoom room, EmojiWheel wheel, Func<long> clockMs)
        {
            this.room = room ?? throw new ArgumentNullException(nameof(room));
            this.wheel = wheel ?? throw new ArgumentNullException(nameof(wheel));
            this.clockMs = clockMs ?? throw new ArgumentNullException(nameof(clockMs));
            onState = Enter;
            onRelay = Receive;
        }

        public event Action<MemberEmoji> Shown;

        public bool CanSend => room.InRoom && room.Phase == LobbyPhase.Lobby && wheel.Visible;

        public void Wear(InventoryReply inventory)
        {
            if (inventory == null) return;
            worn = inventory.loadout?.emojis ?? NothingWorn;
            emojiSlots = inventory.emoji_slots;
        }

        public void Wear(EquipReply equipped)
        {
            if (equipped?.loadout == null) return;
            worn = equipped.loadout.emojis ?? NothingWorn;
        }

        public void Bind(Session source)
        {
            Unbind();
            session = source ?? throw new ArgumentNullException(nameof(source));
            session.LobbyStateReceived += onState;
            session.LobbyRelayReceived += onRelay;
        }

        public void Unbind()
        {
            if (session == null) return;
            session.LobbyStateReceived -= onState;
            session.LobbyRelayReceived -= onRelay;
            session = null;
        }

        public void Enter(LobbyState state)
        {
            if (state.Code == enteredRoom) return;
            enteredRoom = state.Code;
            wheel.Equip(enteredRoom.Length == 0 ? NothingWorn : worn, emojiSlots);
        }

        public bool TrySend(Session target, int slot, long nowMs)
        {
            if (!target.Accepted || !CanSend || !wheel.TrySend(slot, nowMs, out var item)) return false;
            intent.ItemId = item;
            target.Send(Envelope.PayloadOneofCase.EmojiIntent, intent);
            return true;
        }

        public void Receive(LobbyRelay relay) => TryShow(relay, clockMs());

        public bool TryShow(LobbyRelay relay, long nowMs)
        {
            if (string.IsNullOrEmpty(relay.ItemId) || !IsMember(relay.AccountId)) return false;
            if (relay.AccountId == room.LocalAccount) wheel.OwnEmojiShown(nowMs);
            Shown?.Invoke(new MemberEmoji(relay.AccountId, relay.ItemId));
            return true;
        }

        bool IsMember(string account)
        {
            if (string.IsNullOrEmpty(account)) return false;
            var members = room.Members;
            for (var index = 0; index < members.Count; index++)
            {
                if (members[index] == account) return true;
            }
            return false;
        }
    }
}
