using System;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Emoji;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Match;
using Hypedomino.V1;

namespace HypeDomino.Game.Match
{
    public sealed class MatchEmoji
    {
        public const string MuteKey = "match.seat.mute";
        public const string UnmuteKey = "match.seat.unmute";

        static readonly string[] NothingWorn = Array.Empty<string>();

        readonly EmojiWheel wheel;
        readonly SeatMessages messages;
        readonly GameStrings strings;
        readonly EmojiArtShelf shelf;
        readonly Func<long> clockMs;
        readonly Action<MatchSnapshot> onView;
        readonly Action<EmojiShown> onEmoji;
        readonly Action<QuickMessageShown> onMessage;
        readonly EmojiIntent intent = new();
        readonly QuickMessageIntent messageIntent = new();
        readonly bool[] muted = new bool[(int)Seat.Y + 1];
        Session session;
        string[] worn = NothingWorn;
        int emojiSlots;
        Seat local;
        int viewer;
        int seatCount;

        public MatchEmoji(EmojiWheel wheel, SeatMessages messages, GameStrings strings, EmojiArtShelf shelf, Func<long> clockMs)
        {
            this.wheel = wheel ?? throw new ArgumentNullException(nameof(wheel));
            this.messages = messages ?? throw new ArgumentNullException(nameof(messages));
            this.strings = strings ?? throw new ArgumentNullException(nameof(strings));
            this.shelf = shelf ?? throw new ArgumentNullException(nameof(shelf));
            this.clockMs = clockMs ?? throw new ArgumentNullException(nameof(clockMs));
            onView = Show;
            onEmoji = shown => Place(shown);
            onMessage = shown => TryShowMessage(shown, this.clockMs());
        }

        public bool Begun { get; private set; }

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
            session.ViewReceived += onView;
            session.EmojiReceived += onEmoji;
            session.QuickMessageReceived += onMessage;
        }

        public void Unbind()
        {
            if (session == null) return;
            session.ViewReceived -= onView;
            session.EmojiReceived -= onEmoji;
            session.QuickMessageReceived -= onMessage;
            session = null;
            End();
        }

        public void Show(MatchSnapshot view)
        {
            if (view.State == MatchState.Finished)
            {
                if (Begun) End();
                return;
            }
            if (!Begun) Begin(view.Seat, view.Mode);
        }

        public void Begin(Seat localSeat, Mode mode)
        {
            local = localSeat;
            seatCount = mode == Mode._1V1 || mode == Mode._2V2 ? SeatOrder.Size(mode) : 0;
            viewer = seatCount == 0 ? 0 : SeatOrder.Viewer(localSeat, mode);
            Array.Clear(muted, 0, muted.Length);
            messages.Clear();
            Begun = seatCount != 0;
            if (!Begun) return;
            wheel.Equip(worn, emojiSlots);
            for (var slot = 0; slot < wheel.Count; slot++) shelf.Prefetch(wheel.Slot(slot));
        }

        public void End()
        {
            Begun = false;
            Array.Clear(muted, 0, muted.Length);
            shelf.Clear();
        }

        public bool TrySend(Session session, int slot, long nowMs)
        {
            if (!session.Accepted || !wheel.TrySend(slot, nowMs, out var item)) return false;
            intent.ItemId = item;
            session.Send(Envelope.PayloadOneofCase.EmojiIntent, intent);
            return true;
        }

        public bool TrySendMessage(Session session, string key, bool teamOnly)
        {
            if (!session.Accepted) return false;
            messageIntent.Key = key;
            messageIntent.TeamOnly = teamOnly;
            session.Send(Envelope.PayloadOneofCase.QuickMessageIntent, messageIntent);
            return true;
        }

        public bool Place(EmojiShown shown) => TryPlace(shown.Seat, clockMs(), out var side) && shelf.Show(side, shown.ItemId);

        public bool TryPlace(Seat sender, long nowMs, out TableSide side)
        {
            if (!TryRoute(sender, out side)) return false;
            if (sender == local) wheel.OwnEmojiShown(nowMs);
            return true;
        }

        public bool TryShowMessage(QuickMessageShown shown, long nowMs)
        {
            if (!TryRoute(shown.Seat, out var side)) return false;
            messages.Show(side, strings.Text(shown.Key), nowMs);
            return true;
        }

        public bool CanMute(Seat seat) => InMatch(seat) && seat != local;

        public bool IsMuted(Seat seat) => InMatch(seat) && muted[(int)seat];

        public string MuteLabelKey(Seat seat) => IsMuted(seat) ? UnmuteKey : MuteKey;

        public bool ToggleMute(Seat seat)
        {
            if (!CanMute(seat)) return false;
            muted[(int)seat] = !muted[(int)seat];
            return true;
        }

        bool TryRoute(Seat sender, out TableSide side)
        {
            side = TableSide.Bottom;
            if (!InMatch(sender) || muted[(int)sender]) return false;
            side = TableSides.Of(viewer, SeatOrder.Clockwise(sender), seatCount);
            return true;
        }

        bool InMatch(Seat seat) =>
            seatCount != 0 && seat >= Seat.North && seat <= Seat.Y && IsTwoSeat(seat) == (seatCount == 2);

        static bool IsTwoSeat(Seat seat) => seat == Seat.X || seat == Seat.Y;
    }
}
