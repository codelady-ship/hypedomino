using System;
using System.Collections.Generic;
using System.IO;
using HypeDomino.Game.FriendLobby;
using HypeDomino.Net;
using HypeDomino.UI.Localization;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class FriendLobbyTests
    {
        const ulong Nonce = 0x5EED_0000_0000_0154;
        const string Leader = "0b7c3f2e-5a41-4d8e-9c1f-2a6b8d4e7f10";
        const string Friend = "5e2d9a14-7c3b-4f60-8a1e-9d4c2b6f3a87";
        const int DeprecatedUnknownAction = 15;

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

            public void Deliver(byte[] frame) => incoming.Enqueue(frame);
        }

        static LobbyState GoldenState() =>
            Envelope.Parser.ParseFrom(GoldenFrame("lobby_state")).LobbyState;

        static byte[] GoldenFrame(string name) =>
            File.ReadAllBytes(Path.GetFullPath(Path.Combine(Application.dataPath, "..", "..", "proto", "testdata", "frames", name + ".bin")));

        static LobbyState FullTwoVersusTwo()
        {
            var state = new LobbyState { Code = "K7QM3X", LeaderAccountId = "a", Phase = LobbyPhase.Teams, Mode = Mode._2V2 };
            state.MemberAccountIds.Add(new[] { "a", "b", "c", "d" });
            state.Seats.Add(new LobbySeat { Seat = Seat.North, AccountId = "a" });
            state.Seats.Add(new LobbySeat { Seat = Seat.East, AccountId = "b" });
            state.Seats.Add(new LobbySeat { Seat = Seat.South, AccountId = "c" });
            state.Seats.Add(new LobbySeat { Seat = Seat.West, AccountId = "d" });
            return state;
        }

        static Session AcceptedSession(FakeTransport transport)
        {
            var session = new Session();
            session.Hello(transport, new ClientHello());
            var hello = new ServerHello { Status = HelloStatus.Accepted, SessionNonce = Nonce };
            transport.Deliver(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, Nonce), Envelope.PayloadOneofCase.ServerHello, hello).ToArray());
            session.Poll();
            Assert.IsTrue(session.Accepted);
            transport.Sent.Clear();
            return session;
        }

        [TestCase("K7QM3X", "K7QM3X")]
        [TestCase("k7qm3x", "K7QM3X")]
        [TestCase(" K7Q M3X ", "K7QM3X")]
        [TestCase("LLLLLL", "LLLLLL")]
        [TestCase("z23456", "Z23456")]
        public void ATypedRoomCodeIsNormalisedToTheServersFormat(string typed, string expected)
        {
            Assert.IsTrue(RoomCode.TryNormalize(typed, out var code));
            Assert.AreEqual(expected, code);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("      ")]
        [TestCase("K7QM3")]
        [TestCase("K7QM3XA")]
        [TestCase("K7QM3O")]
        [TestCase("K7QM30")]
        [TestCase("K7QM31")]
        [TestCase("K7QM3I")]
        [TestCase("K7-QM3X")]
        [TestCase("K7QM3X\n")]
        public void ACodeNoRoomCanHaveIsRefused(string typed)
        {
            Assert.IsFalse(RoomCode.TryNormalize(typed, out var code));
            Assert.IsNull(code);
        }

        [Test]
        public void TheLeaderSeesTheTeamColumnsAndCannotStartWithAnOpenSeat()
        {
            var room = new FriendRoom();

            room.Read(GoldenState(), Leader);

            Assert.IsTrue(room.InRoom);
            Assert.IsTrue(room.IsLeader);
            Assert.AreEqual("K7QM3X", room.Code);
            Assert.AreEqual(LobbyPhase.Teams, room.Phase);
            Assert.AreEqual(Mode._1V1, room.Mode);
            Assert.AreEqual(LobbyRefusal.SeatsOpen, room.Refusal);
            CollectionAssert.AreEqual(new[] { Leader, Friend }, room.Members);
            Assert.AreEqual(1, room.TeamA.Count);
            Assert.AreEqual(Seat.X, room.TeamA[0].Seat);
            Assert.IsTrue(room.TeamA[0].Local);
            Assert.AreEqual(1, room.TeamB.Count);
            Assert.AreEqual(Seat.Y, room.TeamB[0].Seat);
            Assert.IsTrue(room.TeamB[0].Free);
            Assert.AreEqual(Seat.X, room.LocalSeat);
            CollectionAssert.AreEqual(new[] { Friend }, room.Waiting);
            Assert.IsFalse(room.SeatsFilled);
            Assert.IsFalse(room.CanStart);
            Assert.IsTrue(room.CanGoBack);
            Assert.IsTrue(room.CanChooseMode);
            Assert.IsTrue(room.CanInvite);
            Assert.IsFalse(room.CanRequestSeat(Seat.Y));
        }

        [Test]
        public void AMemberCanAskForAnotherSeatOfTheModeOnly()
        {
            var room = new FriendRoom();

            room.Read(GoldenState(), Friend);

            Assert.IsFalse(room.IsLeader);
            Assert.AreEqual(Seat.Unspecified, room.LocalSeat);
            Assert.IsTrue(room.CanRequestSeat(Seat.X));
            Assert.IsTrue(room.CanRequestSeat(Seat.Y));
            Assert.IsFalse(room.CanRequestSeat(Seat.North));
            Assert.IsFalse(room.CanStart);
            Assert.IsFalse(room.CanGoBack);
            Assert.IsFalse(room.CanChooseMode);
        }

        [Test]
        public void OnlyTheLeaderCanStartOnceEverySeatIsFilled()
        {
            var room = new FriendRoom();

            room.Read(FullTwoVersusTwo(), "a");

            Assert.IsTrue(room.SeatsFilled);
            Assert.IsTrue(room.CanStart);
            Assert.IsTrue(room.Full);
            Assert.IsFalse(room.CanInvite);
            Assert.IsEmpty(room.Waiting);
            Assert.AreEqual(new[] { Seat.North, Seat.South }, new[] { room.TeamA[0].Seat, room.TeamA[1].Seat });
            Assert.AreEqual(new[] { Seat.East, Seat.West }, new[] { room.TeamB[0].Seat, room.TeamB[1].Seat });
            Assert.AreEqual(Team.B, room.TeamB[1].Team);

            room.Read(FullTwoVersusTwo(), "c");

            Assert.AreEqual(Seat.South, room.LocalSeat);
            Assert.IsFalse(room.CanStart);
        }

        [Test]
        public void TheLobbyPhaseHasNoSeatsAndNobodyWaits()
        {
            var state = new LobbyState { Code = "K7QM3X", LeaderAccountId = Leader, Phase = LobbyPhase.Lobby };
            state.MemberAccountIds.Add(new[] { Leader, Friend });
            var room = new FriendRoom();

            room.Read(state, Leader);

            Assert.IsEmpty(room.TeamA);
            Assert.IsEmpty(room.TeamB);
            Assert.IsEmpty(room.Waiting);
            Assert.IsFalse(room.SeatsFilled);
            Assert.IsFalse(room.CanStart);
            Assert.IsFalse(room.CanGoBack);
            Assert.IsTrue(room.CanChooseMode);
            Assert.IsTrue(room.CanInvite);
        }

        [Test]
        public void ARunningMatchTakesNoInvitesOrModeChanges()
        {
            var state = FullTwoVersusTwo();
            state.Phase = LobbyPhase.Match;
            state.MemberAccountIds.RemoveAt(3);
            var room = new FriendRoom();

            room.Read(state, "a");

            Assert.IsFalse(room.CanInvite);
            Assert.IsFalse(room.CanChooseMode);
            Assert.IsFalse(room.CanStart);
        }

        [Test]
        public void AnEmptyStateMeansThePlayerIsInNoRoom()
        {
            var room = new FriendRoom();
            room.Read(GoldenState(), Leader);

            room.Read(new LobbyState(), Leader);

            Assert.IsFalse(room.InRoom);
            Assert.IsFalse(room.IsLeader);
            Assert.IsEmpty(room.Members);
            Assert.IsEmpty(room.TeamA);
            Assert.IsEmpty(room.TeamB);
            Assert.IsEmpty(room.Waiting);
            Assert.AreEqual(Seat.Unspecified, room.LocalSeat);
            Assert.IsFalse(room.CanInvite);
        }

        [Test]
        public void CommandsWaitForAnAcceptedSession()
        {
            var transport = new FakeTransport();
            var session = new Session();
            session.Hello(transport, new ClientHello());
            transport.Sent.Clear();

            Assert.IsFalse(new LobbyCommands().TryCreate(session));
            Assert.IsEmpty(transport.Sent);
        }

        [Test]
        public void AJoinSendsTheNormalisedCodeUnsequenced()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);

            Assert.IsTrue(new LobbyCommands().TryJoin(session, " k7q m3x"));

            var sent = Envelope.Parser.ParseFrom(transport.Sent[0]);
            Assert.AreEqual(LobbyCommand.Join, sent.LobbyIntent.Command);
            Assert.AreEqual("K7QM3X", sent.LobbyIntent.Code);
            Assert.AreEqual(0UL, sent.Sequence);
            Assert.AreEqual(Nonce, sent.SessionNonce);
        }

        [Test]
        public void AJoinWithACodeNoRoomCanHaveIsNotSent()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);

            Assert.IsFalse(new LobbyCommands().TryJoin(session, "K7QM30"));
            Assert.IsEmpty(transport.Sent);
        }

        [Test]
        public void EachCommandCarriesOnlyItsOwnFields()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);
            var commands = new LobbyCommands();

            Assert.IsTrue(commands.TryJoin(session, "K7QM3X"));
            Assert.IsTrue(commands.TryChooseMode(session, Mode._2V2));
            Assert.IsTrue(commands.TryPlace(session, Friend, Seat.East));
            Assert.IsTrue(commands.TryAdd(session, Friend, Seat.West));
            Assert.IsTrue(commands.TryRequestSeat(session, Seat.North));
            Assert.IsTrue(commands.TryInvite(session, Friend));
            Assert.IsTrue(commands.TryRemove(session, Friend));
            Assert.IsTrue(commands.TryBack(session));
            Assert.IsTrue(commands.TryStart(session));
            Assert.IsTrue(commands.TryLeave(session));
            Assert.IsTrue(commands.TryCreate(session));

            var expected = new[]
            {
                new LobbyIntent { Command = LobbyCommand.Join, Code = "K7QM3X" },
                new LobbyIntent { Command = LobbyCommand.Teams, Mode = Mode._2V2 },
                new LobbyIntent { Command = LobbyCommand.Place, AccountId = Friend, Seat = Seat.East },
                new LobbyIntent { Command = LobbyCommand.Add, AccountId = Friend, Seat = Seat.West },
                new LobbyIntent { Command = LobbyCommand.RequestSeat, Seat = Seat.North },
                new LobbyIntent { Command = LobbyCommand.Invite, AccountId = Friend },
                new LobbyIntent { Command = LobbyCommand.Remove, AccountId = Friend },
                new LobbyIntent { Command = LobbyCommand.Back },
                new LobbyIntent { Command = LobbyCommand.Start },
                new LobbyIntent { Command = LobbyCommand.Leave },
                new LobbyIntent { Command = LobbyCommand.Create },
            };
            Assert.AreEqual(expected.Length, transport.Sent.Count);
            for (var index = 0; index < expected.Length; index++)
                Assert.AreEqual(expected[index], Envelope.Parser.ParseFrom(transport.Sent[index]).LobbyIntent, $"command {index}");
        }

        static LobbyInvite Invite(string code, string from) => new() { Code = code, FromAccountId = from };

        [Test]
        public void AnInGameInviteFromTheSessionWaitsForAnAnswer()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);
            var invites = new RoomInvites();
            var changes = 0;
            invites.Changed += () => changes++;
            invites.Follow(session);

            transport.Deliver(GoldenFrame("lobby_invite"));
            session.Poll();

            Assert.AreEqual(1, changes);
            Assert.AreEqual(1, invites.Pending.Count);
            Assert.AreEqual("K7QM3X", invites.Pending[0].Code);
            Assert.AreEqual(Leader, invites.Pending[0].From);
            Assert.IsTrue(invites.CanAccept);
        }

        [Test]
        public void AnotherInviteToTheSameRoomReplacesTheEarlierOne()
        {
            var invites = new RoomInvites();

            invites.Read(Invite("K7QM3X", Leader));
            invites.Read(Invite("Z23456", Leader));
            invites.Read(Invite("K7QM3X", Friend));

            Assert.AreEqual(2, invites.Pending.Count);
            Assert.AreEqual("Z23456", invites.Pending[0].Code);
            Assert.AreEqual("K7QM3X", invites.Pending[1].Code);
            Assert.AreEqual(Friend, invites.Pending[1].From);
        }

        [TestCase("", Leader)]
        [TestCase("K7QM30", Leader)]
        [TestCase("K7QM3X", "")]
        public void AnInviteNoRoomCanHaveIsDropped(string code, string from)
        {
            var invites = new RoomInvites();
            var changes = 0;
            invites.Changed += () => changes++;

            invites.Read(Invite(code, from));

            Assert.IsEmpty(invites.Pending);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void AcceptingSendsAJoinWithTheInvitedCodeAndClearsTheInvite()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);
            var invites = new RoomInvites();
            invites.Read(Invite("K7QM3X", Leader));
            var changes = 0;
            invites.Changed += () => changes++;

            Assert.IsTrue(invites.TryAccept(session, new LobbyCommands(), "K7QM3X"));

            Assert.AreEqual(new LobbyIntent { Command = LobbyCommand.Join, Code = "K7QM3X" }, Envelope.Parser.ParseFrom(transport.Sent[0]).LobbyIntent);
            Assert.IsEmpty(invites.Pending);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void AnInviteIsKeptWhenTheJoinCannotBeSent()
        {
            var transport = new FakeTransport();
            var session = new Session();
            session.Hello(transport, new ClientHello());
            transport.Sent.Clear();
            var invites = new RoomInvites();
            invites.Read(Invite("K7QM3X", Leader));

            Assert.IsFalse(invites.TryAccept(session, new LobbyCommands(), "K7QM3X"));

            Assert.IsEmpty(transport.Sent);
            Assert.AreEqual(1, invites.Pending.Count);
        }

        [Test]
        public void OnlyAPendingInviteCanBeAccepted()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);
            var invites = new RoomInvites();
            invites.Read(Invite("K7QM3X", Leader));

            Assert.IsFalse(invites.TryAccept(session, new LobbyCommands(), "Z23456"));

            Assert.IsEmpty(transport.Sent);
            Assert.AreEqual(1, invites.Pending.Count);
        }

        [Test]
        public void DecliningDropsOnlyThatInvite()
        {
            var invites = new RoomInvites();
            invites.Read(Invite("K7QM3X", Leader));
            invites.Read(Invite("Z23456", Friend));
            var changes = 0;
            invites.Changed += () => changes++;

            invites.Decline("K7QM3X");
            invites.Decline("K7QM3X");

            Assert.AreEqual(1, invites.Pending.Count);
            Assert.AreEqual("Z23456", invites.Pending[0].Code);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void InsideARoomInvitesWaitUntilThePlayerLeavesIt()
        {
            var transport = new FakeTransport();
            var session = AcceptedSession(transport);
            var invites = new RoomInvites();
            invites.Read(Invite("Z23456", Friend));

            invites.Read(new LobbyState { Code = "K7QM3X", LeaderAccountId = Leader });

            Assert.IsFalse(invites.CanAccept);
            Assert.IsFalse(invites.TryAccept(session, new LobbyCommands(), "Z23456"));
            Assert.IsEmpty(transport.Sent);
            Assert.AreEqual(1, invites.Pending.Count);

            invites.Read(new LobbyState());

            Assert.IsTrue(invites.CanAccept);
            Assert.IsTrue(invites.TryAccept(session, new LobbyCommands(), "Z23456"));
        }

        [Test]
        public void AnInviteToTheRoomThePlayerIsInIsDropped()
        {
            var invites = new RoomInvites();
            invites.Read(Invite("K7QM3X", Leader));

            invites.Read(GoldenState());
            invites.Read(Invite("K7QM3X", Friend));

            Assert.AreEqual("K7QM3X", invites.CurrentRoom);
            Assert.IsEmpty(invites.Pending);
        }

        [Test]
        public void ARefusedCommandNamesTheTextOfItsReason()
        {
            var room = new FriendRoom();

            room.Read(GoldenState(), Leader);

            Assert.AreEqual(LobbyRefusalKeys.SeatsOpen, room.RefusalKey);
        }

        [Test]
        public void ARankedBanOffersItsAppealUntilTheNextState()
        {
            var room = new FriendRoom();
            var refused = new LobbyState { Refusal = LobbyRefusal.RankedBan, BanReason = BanReason.Cheating };
            refused.Appeal = new AppealGrant { SanctionId = 42, Token = "appeal-1", ExpiresUnixMs = 1_800_000_000_000 };

            room.Read(refused, Leader);

            Assert.IsTrue(room.Appeal.Offered);
            Assert.AreEqual(42, room.Appeal.SanctionId);
            Assert.AreEqual("appeal-1", room.Appeal.Token);

            room.Read(new LobbyState(), Leader);
            Assert.IsFalse(room.Appeal.Offered);
        }

        [TestCase(BanReason.Cheating, "ban.reason.cheating")]
        [TestCase(BanReason.Collusion, "ban.reason.collusion")]
        [TestCase(BanReason.AfkLeaving, "ban.reason.afk_leaving")]
        [TestCase(BanReason.OffensiveName, "ban.reason.offensive_name")]
        [TestCase(BanReason.PaymentFraud, "ban.reason.payment_fraud")]
        [TestCase(BanReason.Other, "ban.reason.other")]
        [TestCase((BanReason)42, "ban.reason.other")]
        public void ARankedBanClosesCustomRoomsWithItsReasonAndEnd(BanReason reason, string reasonKey)
        {
            var room = new FriendRoom();
            const ulong now = 1_800_000_000_000;

            room.Read(new LobbyState { Refusal = LobbyRefusal.RankedBan, RefusedUntilUnixMs = now + 3_600_000, BanReason = reason }, Leader);

            Assert.IsFalse(room.InRoom);
            Assert.AreEqual(LobbyRefusalKeys.RankedBan, room.RefusalKey);
            Assert.AreEqual(reasonKey, room.BanReasonKey);
            Assert.AreEqual(now + 3_600_000, room.RefusedUntilUnixMs);
            var clock = new ServerClock();
            clock.Sync(now, 0, 0);
            Assert.AreEqual(3_600d, room.SecondsUntilBanEnds(clock, 0), 1e-6);
            Assert.AreEqual(0d, room.SecondsUntilBanEnds(new ServerClock(), 0));
        }

        [Test]
        public void AStateWithoutABanNamesNoReasonAndCountsNothing()
        {
            var room = new FriendRoom();
            var clock = new ServerClock();
            clock.Sync(1_800_000_000_000, 0, 0);

            room.Read(GoldenState(), Leader);

            Assert.IsNull(room.BanReasonKey);
            Assert.AreEqual(0d, room.SecondsUntilBanEnds(clock, 0));
        }

        [Test]
        public void AStateWithoutARefusalNamesNoText()
        {
            var room = new FriendRoom();

            room.Read(FullTwoVersusTwo(), "a");

            Assert.IsNull(room.RefusalKey);
        }

        [Test]
        public void EveryReasonTheServerSendsHasItsOwnText()
        {
            var keys = new HashSet<string>();
            foreach (LobbyRefusal refusal in Enum.GetValues(typeof(LobbyRefusal)))
            {
                var key = LobbyRefusalKeys.Of(refusal);
                if (refusal == LobbyRefusal.Unspecified)
                {
                    Assert.IsNull(key);
                    continue;
                }
                if ((int)refusal == DeprecatedUnknownAction)
                {
                    Assert.AreEqual(LobbyRefusalKeys.Unknown, key);
                    continue;
                }
                Assert.IsTrue(key.StartsWith("lobby.room.refused.", StringComparison.Ordinal), refusal.ToString());
                Assert.IsTrue(keys.Add(key), refusal.ToString());
            }
            Assert.AreEqual(LobbyRefusalKeys.Unknown, LobbyRefusalKeys.Of((LobbyRefusal)999));
        }

        [Test]
        public void TheShippedBundleHasEveryRefusalInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            foreach (LobbyRefusal refusal in Enum.GetValues(typeof(LobbyRefusal)))
            {
                var key = LobbyRefusalKeys.Of(refusal);
                if (key == null)
                {
                    continue;
                }
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
            Assert.AreEqual("Bu kodla otaq yoxdur", strings.Value(LobbyRefusalKeys.UnknownCode, "az"));
            Assert.AreEqual("The room is full", strings.Value(LobbyRefusalKeys.Full, GameLanguage.English));
        }

        [Test]
        public void NamingARefusalTextAllocatesNothing()
        {
            var room = new FriendRoom();
            room.Read(GoldenState(), Leader);
            string key = null;
            TestDelegate name = () => key = room.RefusalKey;
            name();

            Assert.That(name, Allocates.Nothing);
            Assert.AreEqual(LobbyRefusalKeys.SeatsOpen, key);
        }
    }
}
