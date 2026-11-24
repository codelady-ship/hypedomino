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
    public sealed class ConnectionTests
    {
        const ulong ServerTimeUnixMs = 1_761_480_000_000;
        static readonly Uri ServerUri = new("wss://play.test/ws");

        sealed class FakeSocket : ISocketTransport
        {
            readonly Queue<byte[]> incoming = new();

            public List<byte[]> Sent { get; } = new();
            public Uri Uri { get; private set; }
            public bool Disposed { get; private set; }
            public TransportState State { get; set; } = TransportState.Idle;

            public void Connect(Uri uri)
            {
                Uri = uri;
                State = TransportState.Connecting;
            }

            public void Send(ReadOnlySpan<byte> frame) => Sent.Add(frame.ToArray());

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                if (incoming.Count == 0) return false;
                frame = incoming.Dequeue();
                return true;
            }

            public void Deliver(byte[] frame) => incoming.Enqueue(frame);

            public void Dispose()
            {
                Disposed = true;
                State = TransportState.Closed;
            }
        }

        sealed class FixedRandom : System.Random
        {
            readonly double value;

            public FixedRandom(double value) => this.value = value;

            public override double NextDouble() => value;
        }

        readonly List<FakeSocket> sockets = new();
        double now;
        Connection connection;

        FakeSocket Socket => sockets[sockets.Count - 1];

        [SetUp]
        public void CreateConnection()
        {
            sockets.Clear();
            now = 10;
            debuggerAttached = false;
            connection = new Connection(OpenSocket, () => new ClientHello { SessionToken = "sess_01HZX" }, () => now, new FixedRandom(0.5));
        }

        ISocketTransport OpenSocket()
        {
            var socket = new FakeSocket();
            sockets.Add(socket);
            return socket;
        }

        static byte[] ServerFrame(ulong acked, Envelope.PayloadOneofCase payload, IMessage message) =>
            new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, acked, 0), payload, message).ToArray();

        static byte[] AcceptedHello(ulong acked = 0) =>
            ServerFrame(acked, Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, ServerTimeUnixMs = ServerTimeUnixMs, SessionNonce = 77 });

        static byte[] PlayingView(ulong acked) =>
            ServerFrame(acked, Envelope.PayloadOneofCase.MatchView, new MatchView { State = MatchState.Playing, Seat = Seat.X });

        static byte[] Golden(string name) =>
            File.ReadAllBytes(Path.GetFullPath(Path.Combine(Application.dataPath, "..", "..", "proto", "testdata", "frames", name + ".bin")));

        void GoOnline(ulong acked = 0)
        {
            connection.Start(ServerUri);
            Socket.State = TransportState.Open;
            now += 0.2;
            Socket.Deliver(AcceptedHello(acked));
            connection.Tick();
            Assert.AreEqual(ConnectionState.Online, connection.State);
        }

        void DropAndTick()
        {
            Socket.State = TransportState.Closed;
            connection.Tick();
        }

        [Test]
        public void StartDialsTheServerAndSendsTheHello()
        {
            connection.Start(ServerUri);

            Assert.AreEqual(ConnectionState.Connecting, connection.State);
            Assert.AreEqual(ServerUri, Socket.Uri);
            Assert.AreEqual("sess_01HZX", Envelope.Parser.ParseFrom(Socket.Sent[0]).ClientHello.SessionToken);
        }

        bool debuggerAttached;

        void CollectHooks(List<string> signals)
        {
            if (debuggerAttached) signals.Add(HypeDomino.Core.DeviceRisk.HookSignals.Debugger);
        }

        [Test]
        public void ARedialSendsTheHookSignalsCollectedForIt()
        {
            connection.Session.CollectHookSignals = CollectHooks;
            GoOnline();
            debuggerAttached = true;

            DropAndTick();
            now = connection.RetryAtSeconds;
            connection.Tick();

            Assert.AreEqual(2, sockets.Count);
            CollectionAssert.IsEmpty(Envelope.Parser.ParseFrom(sockets[0].Sent[0]).ClientHello.HookSignals);
            CollectionAssert.AreEqual(new[] { "debugger" }, Envelope.Parser.ParseFrom(Socket.Sent[0]).ClientHello.HookSignals);
        }

        [Test]
        public void AnAcceptedHelloGoesOnlineAndSetsTheServerClock()
        {
            GoOnline();

            Assert.IsTrue(connection.Clock.Synced);
            Assert.AreEqual(ServerTimeUnixMs, connection.Clock.UnixMs(10.1), 0.001);
            Assert.AreEqual(1.5, connection.Clock.SecondsUntil(ServerTimeUnixMs + 2000, 10.6), 0.001);
        }

        [Test]
        public void ADroppedSessionRedialsAtOnceThenBacksOff()
        {
            GoOnline();
            var waits = new List<double>();

            for (var attempt = 0; attempt < 7; attempt++)
            {
                DropAndTick();
                Assert.AreEqual(ConnectionState.Waiting, connection.State);
                waits.Add(Math.Round(connection.RetryAtSeconds - now, 3));
                now = connection.RetryAtSeconds - 0.01;
                connection.Tick();
                Assert.AreEqual(ConnectionState.Waiting, connection.State);
                now = connection.RetryAtSeconds;
                connection.Tick();
                Assert.AreEqual(ConnectionState.Connecting, connection.State);
            }

            CollectionAssert.AreEqual(new[] { 0, 0.5, 1, 2, 4, 5, 5 }, waits);
            Assert.AreEqual(6, connection.FailedAttempts);
            Assert.AreEqual(8, sockets.Count);
            Assert.IsTrue(sockets[0].Disposed);
        }

        [Test]
        public void AnAcceptedResumeStartsTheScheduleAgain()
        {
            GoOnline();
            DropAndTick();
            connection.Tick();
            DropAndTick();
            now = connection.RetryAtSeconds;
            connection.Tick();
            Socket.Deliver(AcceptedHello());
            connection.Tick();

            DropAndTick();

            Assert.AreEqual(0, connection.FailedAttempts);
            Assert.AreEqual(now, connection.RetryAtSeconds);
        }

        [TestCase(0.0, 0.8)]
        [TestCase(1.0, 1.2)]
        public void JitterSpreadsEveryDelayByAFifth(double random, double factor)
        {
            var schedule = new ReconnectSchedule();
            schedule.NextDelaySeconds(random);

            Assert.AreEqual(0.5 * factor, schedule.NextDelaySeconds(random), 1e-9);
        }

        [Test]
        public void TheResumeHelloCarriesTheLastAckedSequence()
        {
            GoOnline();
            Socket.Deliver(PlayingView(8));
            connection.Tick();

            DropAndTick();
            connection.Tick();

            var hello = Envelope.Parser.ParseFrom(Socket.Sent[0]);
            Assert.AreEqual(8UL, hello.ClientHello.LastAckedSequence);
            Assert.AreEqual(9UL, hello.Sequence);
        }

        [Test]
        public void FramesReadBeforeTheCloseStillCount()
        {
            GoOnline();
            Socket.Deliver(PlayingView(3));
            Socket.State = TransportState.Closed;

            connection.Tick();

            Assert.AreEqual(MatchState.Playing, connection.Session.View.State);
            Assert.AreEqual(3UL, connection.Session.AckedSequence);
            Assert.AreEqual(ConnectionState.Waiting, connection.State);
        }

        [Test]
        public void ABrokenSessionIsDialledAgain()
        {
            GoOnline();
            Socket.Deliver(Golden("place_tile_intent"));

            connection.Tick();

            Assert.AreEqual(ConnectionState.Waiting, connection.State);
        }

        [TestCase("server_hello_update_required")]
        public void ARefusedHelloStopsForGood(string frame)
        {
            connection.Start(ServerUri);
            Socket.Deliver(Golden(frame));

            connection.Tick();
            now += 60;
            connection.Tick();

            Assert.AreEqual(ConnectionState.Stopped, connection.State);
            Assert.AreEqual(1, sockets.Count);
            Assert.IsTrue(Socket.Disposed);
        }

        [TestCase(ErrorCode.Unauthenticated)]
        [TestCase(ErrorCode.IntegrityFailed)]
        [TestCase(ErrorCode.UnsupportedProtocol)]
        public void AnErrorTheClientCannotFixStopsForGood(ErrorCode code)
        {
            connection.Start(ServerUri);
            Socket.Deliver(ServerFrame(0, Envelope.PayloadOneofCase.Error, new Error { Code = code }));

            connection.Tick();
            Socket.State = TransportState.Closed;
            connection.Tick();

            Assert.AreEqual(ConnectionState.Stopped, connection.State);
            Assert.AreEqual(1, sockets.Count);
        }

        [Test]
        public void AStoppedConnectionStartsAgain()
        {
            GoOnline();
            connection.Stop();

            connection.Start(ServerUri);

            Assert.AreEqual(ConnectionState.Connecting, connection.State);
            Assert.AreEqual(2, sockets.Count);
        }

        [Test]
        public void ARunningConnectionDoesNotStartTwice()
        {
            connection.Start(ServerUri);

            Assert.Throws<InvalidOperationException>(() => connection.Start(ServerUri));
        }

        [Test]
        public void AnAcknowledgedTurnIntentMeasuresTheRoundTrip()
        {
            GoOnline();
            var place = new PlaceTileIntent { Tile = new Tile { Low = 2, High = 3 }, End = BoardEnd.Left };

            now = 20;
            connection.Send(Envelope.PayloadOneofCase.PlaceTileIntent, place);
            now = 20.08;
            Socket.Deliver(PlayingView(1));
            connection.Tick();
            Assert.AreEqual(80, connection.RoundTrip.LastMs, 1e-6);

            now = 21;
            connection.Send(Envelope.PayloadOneofCase.PlaceTileIntent, place);
            Socket.Deliver(ServerFrame(1, Envelope.PayloadOneofCase.Error, new Error { Code = ErrorCode.IllegalIntent, RejectedSequence = 2 }));
            Socket.Deliver(PlayingView(1));
            connection.Tick();
            now = 22;
            connection.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent());
            now = 22.05;
            Socket.Deliver(PlayingView(3));
            connection.Tick();

            Assert.AreEqual(50, connection.RoundTrip.LastMs, 1e-6);
            Assert.AreEqual(80 + (50 - 80) / 8.0, connection.RoundTrip.SmoothedMs, 1e-6);
        }

        [Test]
        public void IntentsWithoutASequenceAreNotTimed()
        {
            GoOnline();

            connection.Send(Envelope.PayloadOneofCase.EmojiIntent, new EmojiIntent { ItemId = "emoji.laugh" });
            Socket.Deliver(PlayingView(0));
            connection.Tick();

            Assert.IsFalse(connection.RoundTrip.HasSample);
        }

        [Test]
        public void ReconnectingShowsOnlyAfterTheSessionWasOnline()
        {
            connection.Start(ServerUri);
            Assert.AreEqual(NetworkNotice.None, ConnectionNotices.Network(connection, now));
            Socket.State = TransportState.Open;
            Socket.Deliver(AcceptedHello());
            connection.Tick();
            Assert.AreEqual(NetworkNotice.None, ConnectionNotices.Network(connection, now));

            DropAndTick();
            Assert.AreEqual(NetworkNotice.Reconnecting, ConnectionNotices.Network(connection, now));
            connection.Tick();
            Assert.AreEqual(ConnectionState.Connecting, connection.State);
            Assert.AreEqual(NetworkNotice.Reconnecting, ConnectionNotices.Network(connection, now));

            connection.Stop();
            connection.Start(ServerUri);
            Assert.AreEqual(NetworkNotice.None, ConnectionNotices.Network(connection, now));
        }

        [Test]
        public void RejoinDropsTheSocketAndResumesAtOnce()
        {
            GoOnline();
            Socket.Deliver(PlayingView(5));
            connection.Tick();

            Assert.IsTrue(connection.Rejoin());

            Assert.AreEqual(2, sockets.Count);
            Assert.IsTrue(sockets[0].Disposed);
            Assert.AreEqual(ConnectionState.Connecting, connection.State);
            Assert.AreEqual(NetworkNotice.Reconnecting, ConnectionNotices.Network(connection, now));
            Assert.AreEqual(5UL, Envelope.Parser.ParseFrom(Socket.Sent[0]).ClientHello.LastAckedSequence);

            Socket.Deliver(AcceptedHello(5));
            connection.Tick();
            Assert.AreEqual(ConnectionState.Online, connection.State);
        }

        [Test]
        public void RejoinWorksOnlyWhileOnline()
        {
            connection.Start(ServerUri);

            Assert.IsFalse(connection.Rejoin());
            Assert.AreEqual(1, sockets.Count);
            Assert.AreEqual(ConnectionState.Connecting, connection.State);
        }

        [TestCase(0.25, NetworkNotice.None)]
        [TestCase(0.35, NetworkNotice.Weak)]
        public void ASlowRoundTripIsWeak(double ackAfterSeconds, NetworkNotice expected)
        {
            GoOnline();
            now = 20;
            connection.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent());
            now += ackAfterSeconds;
            Socket.Deliver(PlayingView(1));
            connection.Tick();

            Assert.AreEqual(expected, ConnectionNotices.Network(connection, now));
        }

        [TestCase(1.9, NetworkNotice.None)]
        [TestCase(2.1, NetworkNotice.Weak)]
        public void AnIntentWaitingTooLongIsWeak(double waitedSeconds, NetworkNotice expected)
        {
            GoOnline();
            now = 20;
            connection.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent());

            now += waitedSeconds;
            connection.Tick();

            Assert.AreEqual(expected, ConnectionNotices.Network(connection, now));
        }

        [TestCase(ErrorCode.IllegalIntent)]
        [TestCase(ErrorCode.RateLimited)]
        [TestCase(ErrorCode.ReplayRejected)]
        public void ADroppedTurnIntentNeitherWaitsNorIsTimed(ErrorCode code)
        {
            GoOnline();
            now = 20;
            connection.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent());
            Socket.Deliver(ServerFrame(0, Envelope.PayloadOneofCase.Error, new Error { Code = code, RejectedSequence = 1 }));
            connection.Tick();

            now += 3;
            connection.Tick();
            Assert.IsFalse(connection.RoundTrip.HasPending);
            Assert.AreEqual(NetworkNotice.None, ConnectionNotices.Network(connection, now));

            connection.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent());
            now += 0.06;
            Socket.Deliver(PlayingView(2));
            connection.Tick();

            Assert.AreEqual(ConnectionState.Online, connection.State);
            Assert.AreEqual(60, connection.RoundTrip.LastMs, 1e-6);
            Assert.AreEqual(60, connection.RoundTrip.SmoothedMs, 1e-6);
        }

        [Test]
        public void TickingAQuietConnectionAllocatesNothing()
        {
            GoOnline();
            TestDelegate step = connection.Tick;
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
