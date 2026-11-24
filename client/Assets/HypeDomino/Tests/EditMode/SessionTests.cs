using System;
using System.Collections.Generic;
using System.IO;
using Google.Protobuf;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SessionTests
    {
        const ulong Nonce = 0x5EED_CAFE_F00D_0001;

        sealed class FakeTransport : IFrameTransport
        {
            readonly List<byte[]> incoming = new(16);
            int next;

            public List<byte[]> Sent { get; } = new();
            public bool Recording { get; set; } = true;

            public void Send(ReadOnlySpan<byte> frame)
            {
                if (Recording) Sent.Add(frame.ToArray());
            }

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                if (next == incoming.Count)
                {
                    incoming.Clear();
                    next = 0;
                    return false;
                }
                frame = incoming[next++];
                return true;
            }

            public void Deliver(byte[] frame) => incoming.Add(frame);
        }

        FakeTransport transport;
        Session session;

        [SetUp]
        public void CreateSession()
        {
            transport = new FakeTransport();
            session = new Session();
            hookCollections = 0;
        }

        static byte[] Golden(string name) =>
            File.ReadAllBytes(Path.GetFullPath(Path.Combine(Application.dataPath, "..", "..", "proto", "testdata", "frames", name + ".bin")));

        static byte[] ServerFrame(FrameHeader header, Envelope.PayloadOneofCase payload, IMessage message) =>
            new FrameWriter(Session.MaxSentFrameBytes).Write(header, payload, message).ToArray();

        static byte[] AcceptedHello(ulong acked, ulong nonce) =>
            ServerFrame(new FrameHeader(Session.ProtocolVersion, 0, acked, nonce), Envelope.PayloadOneofCase.ServerHello,
                new ServerHello { Status = HelloStatus.Accepted, SessionNonce = nonce });

        static Envelope SentEnvelope(byte[] frame) => Envelope.Parser.ParseFrom(frame);

        void Accept(ulong acked = 0)
        {
            session.Hello(transport, new ClientHello());
            transport.Deliver(AcceptedHello(acked, Nonce));
            session.Poll();
            Assert.IsTrue(session.Accepted);
        }

        [Test]
        public void AFreshHelloIsTheGoldenClientHello()
        {
            var golden = Golden("client_hello");

            session.Hello(transport, Envelope.Parser.ParseFrom(golden).ClientHello);

            CollectionAssert.AreEqual(golden, transport.Sent[0]);
        }

        [Test]
        public void EveryHelloCarriesTheDeviceRiskSignalsOnce()
        {
            session.DeviceRiskSignals.Add(HypeDomino.Core.DeviceRisk.DeviceRiskSignals.Root);
            session.DeviceRiskSignals.Add(HypeDomino.Core.DeviceRisk.DeviceRiskSignals.Emulator);
            var hello = new ClientHello();

            session.Hello(transport, hello);
            session.Hello(transport, hello);

            var expected = new[] { "root", "emulator" };
            CollectionAssert.AreEqual(expected, SentEnvelope(transport.Sent[0]).ClientHello.DeviceRiskSignals);
            CollectionAssert.AreEqual(expected, SentEnvelope(transport.Sent[1]).ClientHello.DeviceRiskSignals);
        }

        [Test]
        public void EveryHelloCarriesTheHookSignalsOnce()
        {
            session.HookSignals.Add(HypeDomino.Core.DeviceRisk.HookSignals.Instrumentation);
            session.HookSignals.Add(HypeDomino.Core.DeviceRisk.HookSignals.Debugger);
            var hello = new ClientHello();

            session.Hello(transport, hello);
            session.Hello(transport, hello);

            var expected = new[] { "instrumentation", "debugger" };
            CollectionAssert.AreEqual(expected, SentEnvelope(transport.Sent[0]).ClientHello.HookSignals);
            CollectionAssert.AreEqual(expected, SentEnvelope(transport.Sent[1]).ClientHello.HookSignals);
            CollectionAssert.IsEmpty(SentEnvelope(transport.Sent[0]).ClientHello.DeviceRiskSignals);
        }

        int hookCollections;

        void CollectDebuggerAfterFirstHello(List<string> signals)
        {
            if (hookCollections++ > 0) signals.Add(HypeDomino.Core.DeviceRisk.HookSignals.Debugger);
        }

        [Test]
        public void EveryHelloCollectsTheHookSignalsAgain()
        {
            session.CollectHookSignals = CollectDebuggerAfterFirstHello;
            var hello = new ClientHello();

            session.Hello(transport, hello);
            session.Hello(transport, hello);

            Assert.AreEqual(2, hookCollections);
            CollectionAssert.IsEmpty(SentEnvelope(transport.Sent[0]).ClientHello.HookSignals);
            CollectionAssert.AreEqual(new[] { "debugger" }, SentEnvelope(transport.Sent[1]).ClientHello.HookSignals);
        }

        [Test]
        public void ACollectedHelloDropsHookSignalsThatAreGone()
        {
            session.HookSignals.Add(HypeDomino.Core.DeviceRisk.HookSignals.Instrumentation);
            session.CollectHookSignals = CollectDebuggerAfterFirstHello;

            session.Hello(transport, new ClientHello());

            CollectionAssert.IsEmpty(session.HookSignals);
            CollectionAssert.IsEmpty(SentEnvelope(transport.Sent[0]).ClientHello.HookSignals);
        }

        [Test]
        public void AResumeHelloReportsTheLastAckedSequence()
        {
            Accept(41);
            var golden = Golden("client_hello_resume");

            session.Hello(transport, Envelope.Parser.ParseFrom(golden).ClientHello);

            CollectionAssert.AreEqual(golden, transport.Sent[1]);
            Assert.IsFalse(session.Accepted);
        }

        [Test]
        public void AnAcceptedHelloStampsItsNonceOnEveryLaterFrame()
        {
            ServerHello received = null;
            session.HelloReceived += hello => received = hello;
            Accept();

            session.Send(Envelope.PayloadOneofCase.LoadingIntent, new LoadingIntent { Progress = 100, Ready = true });
            session.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent());

            Assert.AreEqual(HelloStatus.Accepted, received.Status);
            Assert.AreEqual(Nonce, session.SessionNonce);
            Assert.AreEqual(Nonce, SentEnvelope(transport.Sent[1]).SessionNonce);
            Assert.AreEqual(Nonce, SentEnvelope(transport.Sent[2]).SessionNonce);
        }

        [Test]
        public void OnlyTurnIntentsTakeASequence()
        {
            Accept();

            var place = session.Send(Envelope.PayloadOneofCase.PlaceTileIntent, new PlaceTileIntent { Tile = new Tile { Low = 2, High = 3 }, End = BoardEnd.Left });
            var emoji = session.Send(Envelope.PayloadOneofCase.EmojiIntent, new EmojiIntent { ItemId = "emoji.laugh" });
            var quick = session.Send(Envelope.PayloadOneofCase.QuickMessageIntent, new QuickMessageIntent { Key = "qm.nice" });
            var join = session.Send(Envelope.PayloadOneofCase.JoinQueueIntent, new JoinQueueIntent { Mode = Mode._1V1 });
            var draw = session.Send(Envelope.PayloadOneofCase.DrawIntent, new DrawIntent());
            var end = session.Send(Envelope.PayloadOneofCase.EndTurnIntent, new EndTurnIntent());

            Assert.AreEqual(new ulong[] { 1, 0, 0, 0, 2, 3 }, new[] { place, emoji, quick, join, draw, end });
            Assert.AreEqual(3UL, SentEnvelope(transport.Sent[6]).Sequence);
        }

        [Test]
        public void AResumedSessionNumbersAfterTheServersAckedSequence()
        {
            session.Hello(transport, new ClientHello());
            transport.Deliver(Golden("server_hello_in_match"));
            session.Poll();

            var sequence = session.Send(Envelope.PayloadOneofCase.PlaceTileIntent, new PlaceTileIntent { Tile = new Tile { Low = 2, High = 3 }, End = BoardEnd.Left });

            Assert.AreEqual(7UL, session.AckedSequence);
            Assert.AreEqual(8UL, sequence);
        }

        [Test]
        public void ARefusedHelloKeepsIntentsBack()
        {
            ServerHello received = null;
            session.HelloReceived += hello => received = hello;
            session.Hello(transport, new ClientHello());
            transport.Deliver(Golden("server_hello_update_required"));

            session.Poll();

            Assert.AreEqual(HelloStatus.UpdateRequired, received.Status);
            Assert.IsFalse(session.Accepted);
            Assert.Throws<InvalidOperationException>(() => session.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent()));
        }

        [Test]
        public void AnIntentNeedsAnAcceptedHello() =>
            Assert.Throws<InvalidOperationException>(() => session.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent()));

        [Test]
        public void OnlyIntentsCanBeSent()
        {
            Accept();

            Assert.Throws<ArgumentException>(() => session.Send(Envelope.PayloadOneofCase.ClientHello, new ClientHello()));
        }

        [Test]
        public void AMatchViewUpdatesTheSnapshotAndTheAckedSequence()
        {
            Accept();
            MatchSnapshot received = null;
            session.ViewReceived += view => received = view;
            transport.Deliver(Golden("match_view"));

            session.Poll();

            Assert.AreSame(session.View, received);
            Assert.AreEqual(MatchState.Playing, received.State);
            Assert.AreEqual(3, received.Hand.Length);
            Assert.AreEqual(8UL, session.AckedSequence);
        }

        [Test]
        public void ServerPayloadsReachTheirEvents()
        {
            Accept();
            Error error = null;
            EmojiShown emoji = null;
            QuickMessageShown quick = null;
            QueueState queue = null;
            session.ErrorReceived += value => error = value;
            session.EmojiReceived += value => emoji = value;
            session.QuickMessageReceived += value => quick = value;
            session.QueueStateReceived += value => queue = value;
            foreach (var name in new[] { "error_illegal_intent", "emoji_shown", "quick_message_shown", "queue_state_searching" })
                transport.Deliver(Golden(name));

            session.Poll();

            Assert.AreEqual(ErrorCode.IllegalIntent, error.Code);
            Assert.AreEqual(7UL, error.RejectedSequence);
            Assert.AreEqual("emoji.laugh", emoji.ItemId);
            Assert.IsNotNull(quick);
            Assert.IsNotNull(queue);
            Assert.IsFalse(session.Broken);
        }

        [Test]
        public void LobbyPayloadsReachTheirEvents()
        {
            Accept();
            LobbyState state = null;
            LobbyRelay relay = null;
            LobbyInvite invite = null;
            session.LobbyStateReceived += value => state = value;
            session.LobbyRelayReceived += value => relay = value;
            session.LobbyInviteReceived += value => invite = value;
            foreach (var name in new[] { "lobby_state", "lobby_relay", "lobby_invite" })
                transport.Deliver(Golden(name));

            session.Poll();

            Assert.AreEqual("K7QM3X", state.Code);
            Assert.AreEqual(LobbyRefusal.SeatsOpen, state.Refusal);
            Assert.AreEqual("emoji.laugh", relay.ItemId);
            Assert.AreEqual("K7QM3X", invite.Code);
            Assert.IsFalse(session.Broken);
        }

        [Test]
        public void AFrameWithoutAProtocolVersionIsRead()
        {
            Accept();
            var view = Envelope.Parser.ParseFrom(Golden("match_view"));
            transport.Deliver(ServerFrame(new FrameHeader(0, 0, 8, 0), Envelope.PayloadOneofCase.MatchView, view.MatchView));

            session.Poll();

            Assert.IsFalse(session.Broken);
            Assert.AreEqual(MatchState.Playing, session.View.State);
        }

        [TestCase(new byte[] { 0x6A, 0x05, 0x08 }, TestName = "a malformed frame breaks the session")]
        [TestCase(new byte[] { 0x08, 0x02, 0x72, 0x00 }, TestName = "a newer protocol version breaks the session")]
        public void ABrokenSessionStopsReading(byte[] frame)
        {
            Accept();
            EmojiShown emoji = null;
            session.EmojiReceived += value => emoji = value;
            transport.Deliver(frame);
            transport.Deliver(Golden("emoji_shown"));

            session.Poll();

            Assert.IsTrue(session.Broken);
            Assert.IsFalse(session.Accepted);
            Assert.IsNull(emoji);
        }

        [Test]
        public void AClientPayloadFromTheServerBreaksTheSession()
        {
            Accept();
            transport.Deliver(Golden("place_tile_intent"));

            session.Poll();

            Assert.IsTrue(session.Broken);
            Assert.Throws<InvalidOperationException>(() => session.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent()));
        }

        [Test]
        public void ANewHelloMendsABrokenSession()
        {
            Accept();
            transport.Deliver(Golden("place_tile_intent"));
            session.Poll();

            Accept();

            Assert.IsFalse(session.Broken);
        }

        [Test]
        public void PollingAMatchViewWithoutARosterAllocatesNothing()
        {
            Accept();
            var golden = Envelope.Parser.ParseFrom(Golden("match_view"));
            golden.MatchView.Roster.Clear();
            golden.MatchView.Table = string.Empty;
            var view = golden.ToByteArray();

            TestDelegate step = () =>
            {
                transport.Deliver(view);
                session.Poll();
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }

        [Test]
        public void SendingAReusedTurnIntentAllocatesNothing()
        {
            Accept();
            var intent = new PlaceTileIntent { Tile = new Tile { Low = 2, High = 3 }, End = BoardEnd.Left };
            transport.Recording = false;
            TestDelegate step = () => session.Send(Envelope.PayloadOneofCase.PlaceTileIntent, intent);
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
