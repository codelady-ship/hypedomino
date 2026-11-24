using System;
using HypeDomino.Game.Friends;
using HypeDomino.Game.Inventory;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Emoji;
using Hypedomino.V1;

namespace HypeDomino.Game.FriendLobby
{
    public sealed class FriendLobbyLink
    {
        readonly FriendRoom room;
        readonly InventoryLoader inventory;
        readonly Func<long> clockMs;
        readonly Action<ServerHello> onHello;
        readonly Action<LobbyState> onState;
        readonly Action<InventoryLoadResult> onInventory;
        readonly Action<MemberEmoji> onShown;
        readonly Action<FriendListLoadOutcome> onFriends;
        Session session;
        LobbyState lastState;
        bool inRoom;
        int wheelSlots;

        public FriendLobbyLink(FriendRoom room, LobbyCommands commands, ApiClient api, Func<long> clockMs)
        {
            this.room = room ?? throw new ArgumentNullException(nameof(room));
            if (commands == null) throw new ArgumentNullException(nameof(commands));
            if (api == null) throw new ArgumentNullException(nameof(api));
            this.clockMs = clockMs ?? throw new ArgumentNullException(nameof(clockMs));
            Members = new RoomMembers(api);
            Invites = new InvitePanel(room, commands, api);
            inventory = new InventoryLoader(api);
            onHello = Hello;
            onState = Read;
            onInventory = Loaded;
            onShown = Show;
            onFriends = FriendsAnswered;
        }

        public RoomMembers Members { get; }
        public InvitePanel Invites { get; }
        public EmojiWheel Wheel { get; private set; }
        public LobbyEmoji Emoji { get; private set; }

        public event Action<MemberEmoji> EmojiShown;
        public event Action<InventoryLoadResult> InventoryLoaded;
        public event Action<FriendListLoadOutcome> FriendsLoaded;

        public void Follow(Session source)
        {
            Unfollow();
            session = source ?? throw new ArgumentNullException(nameof(source));
            session.HelloReceived += onHello;
            session.LobbyStateReceived += onState;
            Emoji?.Bind(session);
        }

        public void Unfollow()
        {
            if (session == null) return;
            session.HelloReceived -= onHello;
            session.LobbyStateReceived -= onState;
            Emoji?.Unbind();
            session = null;
        }

        public bool TryInvite(Friend friend) => session != null && Invites.TryInvite(session, friend);

        public bool TrySend(int slot) => session != null && Emoji != null && Emoji.TrySend(session, slot, clockMs());

        public void Wear(InventoryReply worn)
        {
            if (worn == null) return;
            if (Wheel == null || worn.emoji_slots > wheelSlots) Build(worn.emoji_slots);
            Emoji.Wear(worn);
            if (lastState != null) Emoji.Enter(lastState);
        }

        public void Wear(EquipReply equipped) => Emoji?.Wear(equipped);

        void Build(int slots)
        {
            if (Emoji != null)
            {
                Emoji.Unbind();
                Emoji.Shown -= onShown;
            }
            wheelSlots = Math.Max(0, slots);
            Wheel = new EmojiWheel(wheelSlots);
            Emoji = new LobbyEmoji(room, Wheel, clockMs);
            Emoji.Shown += onShown;
            if (session != null) Emoji.Bind(session);
        }

        void Hello(ServerHello hello)
        {
            if (hello.Status == HelloStatus.Accepted) inventory.Load(onInventory);
        }

        void Read(LobbyState state)
        {
            lastState = state;
            Members.Read(room);
            Invites.RoomChanged();
            if (room.InRoom && !inRoom) Invites.Load(onFriends);
            inRoom = room.InRoom;
        }

        void Loaded(InventoryLoadResult result)
        {
            Wear(result.Inventory);
            InventoryLoaded?.Invoke(result);
        }

        void Show(MemberEmoji shown) => EmojiShown?.Invoke(shown);

        void FriendsAnswered(FriendListLoadOutcome outcome) => FriendsLoaded?.Invoke(outcome);
    }
}
