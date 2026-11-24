using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game.FriendLobby;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Emoji;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LobbyEmojiTests
    {
        const ulong Nonce = 0x5EED_0000_0000_0279;
        const string Leader = "acc-leader";
        const string Friend = "acc-friend";
        static readonly string[] Equipped = { "emoji.laugh", "emoji.cry" };

        sealed class FakeTransport : IFrameTransport
        {
            readonly Queue<byte[]> incoming = new();

            public List<byte[]> Sent { get; } = new();

            public void Send(ReadOnlySpan<byte> frame) => Sent.Add(frame.ToArray());

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                if (incoming.Count == 0) return false;
                frame = incoming.Dequeue();
                return true;
            }

            public void Deliver(Envelope.PayloadOneofCase payload, IMessage message) =>
                incoming.Enqueue(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, Nonce), payload, message).ToArray());
        }

        FriendRoom room;
        EmojiWheel wheel;
        LobbyEmoji emoji;
        List<MemberEmoji> shown;
        long now;

        [SetUp]
        public void Build()
        {
            room = new FriendRoom();
            wheel = new EmojiWheel(Equipped.Length);
            emoji = new LobbyEmoji(room, wheel, Now);
            shown = new List<MemberEmoji>();
            emoji.Shown += shown.Add;
            emoji.Wear(Inventory(Equipped, Equipped.Length));
            now = 0;
        }

        [TearDown]
        public void Release() => emoji.Unbind();

        long Now() => now;

        static InventoryReply Inventory(string[] emojis, int slots) =>
            new() { loadout = new LoadoutReply { emojis = emojis }, emoji_slots = slots };

        static LobbyState Lobby(string code = "K7QM3X", LobbyPhase phase = LobbyPhase.Lobby)
        {
            var state = new LobbyState { Code = code, LeaderAccountId = Leader, Phase = phase, Mode = Mode._1V1 };
            state.MemberAccountIds.Add(new[] { Leader, Friend });
            return state;
        }

        static Session AcceptedSession(FakeTransport transport)
        {
            var session = new Session();
            session.Hello(transport, new ClientHello());
            transport.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = Nonce });
            session.Poll();
            Assert.IsTrue(session.Accepted);
            transport.Sent.Clear();
            return session;
        }

        void Enter(LobbyState state)
        {
            room.Read(state, Leader);
            emoji.Enter(state);
        }

        [Test]
        public void EnteringARoomPutsTheWornEmojiOnTheWheelUpToTheSlots()
        {
            emoji.Wear(Inventory(new[] { "emoji.party", "emoji.laugh", "emoji.cry" }, 1));
            Assert.AreEqual(0, wheel.Count);

            Enter(Lobby());

            Assert.AreEqual(1, wheel.Count);
            Assert.AreEqual("emoji.party", wheel.Slot(0));
            Assert.IsTrue(emoji.CanSend);
        }

        [Test]
        public void LeavingTheRoomEmptiesTheWheel()
        {
            Enter(Lobby());
            wheel.Toggle();

            Enter(new LobbyState());

            Assert.AreEqual(0, wheel.Count);
            Assert.IsFalse(wheel.IsOpen);
            Assert.IsFalse(emoji.CanSend);
        }

        [Test]
        public void AnEquipInsideTheRoomReachesTheWheelInTheNextRoom()
        {
            Enter(Lobby());

            emoji.Wear(new EquipReply { loadout = new LoadoutReply { emojis = new[] { "emoji.party", "emoji.laugh" } } });
            Enter(Lobby());
            Assert.AreEqual("emoji.laugh", wheel.Slot(0), "the server keeps the emoji the member joined with");

            Enter(new LobbyState());
            Enter(Lobby("Z23456"));
            Assert.AreEqual("emoji.party", wheel.Slot(0));
            Assert.AreEqual("emoji.laugh", wheel.Slot(1));
        }

        [Test]
        public void ATapInTheLobbySendsTheSlotAsAnUnsequencedIntent()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);
            Enter(Lobby());
            wheel.Toggle();

            Assert.IsTrue(emoji.TrySend(session, 1, 500));

            Assert.AreEqual(1, transport.Sent.Count);
            var envelope = Envelope.Parser.ParseFrom(transport.Sent[0]);
            Assert.AreEqual(Envelope.PayloadOneofCase.EmojiIntent, envelope.PayloadCase);
            Assert.AreEqual("emoji.cry", envelope.EmojiIntent.ItemId);
            Assert.AreEqual(0ul, envelope.Sequence);
            Assert.IsTrue(wheel.Calm(500));
        }

        [TestCase(LobbyPhase.Teams)]
        [TestCase(LobbyPhase.Match)]
        public void NothingIsSentOutsideTheLobbyPhase(LobbyPhase phase)
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);
            Enter(Lobby(phase: phase));
            wheel.Toggle();

            Assert.IsFalse(emoji.CanSend);
            Assert.IsFalse(emoji.TrySend(session, 0, 0));
            Assert.IsEmpty(transport.Sent);
            Assert.IsTrue(wheel.IsOpen);
        }

        [Test]
        public void NothingIsSentWithoutAnAcceptedSessionOrARoom()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);
            wheel.Equip(Equipped);
            wheel.Toggle();

            Assert.IsFalse(emoji.TrySend(session, 0, 0), "no room");
            Enter(Lobby());
            Assert.IsTrue(wheel.IsOpen);
            Assert.IsFalse(emoji.TrySend(new Session(), 0, 0), "no accepted session");
            Assert.IsEmpty(transport.Sent);
            Assert.IsFalse(wheel.Calm(0));
        }

        [Test]
        public void AMembersEmojiIsShownOverThatMember()
        {
            Enter(Lobby());

            Assert.IsTrue(emoji.TryShow(new LobbyRelay { AccountId = Friend, ItemId = "emoji.party" }, 100));

            Assert.AreEqual(1, shown.Count);
            Assert.AreEqual(Friend, shown[0].Account);
            Assert.AreEqual("emoji.party", shown[0].ItemId);
            Assert.IsFalse(wheel.Calm(100), "another member's emoji does not calm the own wheel");
        }

        [Test]
        public void TheOwnEmojiComingBackCalmsTheWheel()
        {
            Enter(Lobby());

            Assert.IsTrue(emoji.TryShow(new LobbyRelay { AccountId = Leader, ItemId = "emoji.laugh" }, 100));

            Assert.AreEqual(Leader, shown[0].Account);
            Assert.IsTrue(wheel.Calm(100));
        }

        [Test]
        public void ASeatRequestOrAStrangersEmojiShowsNothing()
        {
            Enter(Lobby());

            Assert.IsFalse(emoji.TryShow(new LobbyRelay { AccountId = Friend, RequestedSeat = Seat.Y }, 0));
            Assert.IsFalse(emoji.TryShow(new LobbyRelay { AccountId = "acc-stranger", ItemId = "emoji.party" }, 0));
            Assert.IsFalse(emoji.TryShow(new LobbyRelay { ItemId = "emoji.party" }, 0));
            Assert.IsEmpty(shown);
        }

        [Test]
        public void TheBoundSessionFillsTheWheelAndShowsRelaysUntilUnbound()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);
            emoji.Bind(session);
            now = 700;
            room.Read(Lobby(), Leader);
            transport.Deliver(Envelope.PayloadOneofCase.LobbyState, Lobby());
            transport.Deliver(Envelope.PayloadOneofCase.LobbyRelay, new LobbyRelay { AccountId = Leader, ItemId = "emoji.cry" });

            session.Poll();

            Assert.AreEqual(Equipped.Length, wheel.Count);
            Assert.AreEqual(1, shown.Count);
            Assert.AreEqual("emoji.cry", shown[0].ItemId);
            Assert.IsTrue(wheel.Calm(700));

            emoji.Unbind();
            transport.Deliver(Envelope.PayloadOneofCase.LobbyRelay, new LobbyRelay { AccountId = Friend, ItemId = "emoji.laugh" });
            session.Poll();
            Assert.AreEqual(1, shown.Count);
        }
    }
}
