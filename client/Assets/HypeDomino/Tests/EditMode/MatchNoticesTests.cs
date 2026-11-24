using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Match;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchNoticesTests
    {
        static readonly Uri ServerUri = new("wss://play.test/ws");

        sealed class FakeSocket : ISocketTransport
        {
            readonly Queue<byte[]> incoming = new();

            public TransportState State { get; set; } = TransportState.Idle;

            public void Connect(Uri uri) => State = TransportState.Connecting;

            public void Send(ReadOnlySpan<byte> frame) { }

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                if (incoming.Count == 0) return false;
                frame = incoming.Dequeue();
                return true;
            }

            public void Deliver(byte[] frame) => incoming.Enqueue(frame);

            public void Dispose() => State = TransportState.Closed;
        }

        readonly List<FakeSocket> sockets = new();
        double now;
        Connection connection;
        MatchNotices notices;

        FakeSocket Socket => sockets[sockets.Count - 1];

        [SetUp]
        public void CreateConnection()
        {
            sockets.Clear();
            now = 10;
            connection = new Connection(OpenSocket, () => new ClientHello { SessionToken = "sess_01HZX" }, () => now, new System.Random(1));
            notices = new MatchNotices();
        }

        ISocketTransport OpenSocket()
        {
            var socket = new FakeSocket();
            sockets.Add(socket);
            return socket;
        }

        void AcceptHello()
        {
            Socket.State = TransportState.Open;
            var hello = new ServerHello { Status = HelloStatus.Accepted, SessionNonce = 77 };
            Socket.Deliver(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, 0), Envelope.PayloadOneofCase.ServerHello, hello).ToArray());
            connection.Tick();
            Assert.AreEqual(ConnectionState.Online, connection.State);
        }

        void GoOnline()
        {
            connection.Start(ServerUri);
            AcceptHello();
        }

        void DropAndTick()
        {
            Socket.State = TransportState.Closed;
            connection.Tick();
        }

        static MatchSnapshot View(Controller own, uint ownTimeouts, bool otherDisconnected, MatchState state = MatchState.Playing)
        {
            var view = new MatchView { State = state, Seat = Seat.X, Mode = Mode._1V1, AfkConsecutiveTimeouts = 3 };
            view.Seats.Add(new SeatView { Seat = Seat.X, Controller = own, ConsecutiveTimeouts = ownTimeouts });
            view.Seats.Add(new SeatView { Seat = Seat.Y, Controller = Controller.Player, Disconnected = otherDisconnected });
            var snapshot = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        [Test]
        public void NothingShowsBeforeAnyViewOrConnection()
        {
            Assert.AreEqual(NetworkNotice.None, notices.Network);
            Assert.IsFalse(notices.AfkWarning);
            Assert.IsFalse(notices.TakeBack.Shown);
            Assert.AreEqual(SeatNotice.None, notices.Seats.At(TableSide.Top));
        }

        [Test]
        public void AViewReportsAChangeOnlyWhenWhatShowsChanges()
        {
            Assert.IsTrue(notices.Read(View(Controller.Player, 0, true)));
            Assert.AreEqual(SeatNotice.Dropped, notices.Seats.At(TableSide.Top));

            Assert.IsFalse(notices.Read(View(Controller.Player, 1, true)));

            Assert.IsTrue(notices.Read(View(Controller.Player, 1, false)));
            Assert.AreEqual(SeatNotice.None, notices.Seats.At(TableSide.Top));
        }

        [Test]
        public void TheAfkWarningGivesWayToTheTakeBackBanner()
        {
            Assert.IsFalse(notices.Read(View(Controller.Player, 1, false)));

            Assert.IsTrue(notices.Read(View(Controller.Player, 2, false)));
            Assert.IsTrue(notices.AfkWarning);
            Assert.IsFalse(notices.TakeBack.Shown);

            Assert.IsTrue(notices.Read(View(Controller.Bot, 3, false)));
            Assert.IsFalse(notices.AfkWarning);
            Assert.IsTrue(notices.TakeBack.Shown);
            Assert.AreEqual(SeatNotice.BotPlaying, notices.Seats.At(TableSide.Bottom));

            Assert.IsTrue(notices.Read(View(Controller.Player, 0, false)));
            Assert.IsFalse(notices.AfkWarning);
            Assert.IsFalse(notices.TakeBack.Shown);
            Assert.AreEqual(SeatNotice.None, notices.Seats.At(TableSide.Bottom));
        }

        [Test]
        public void TheFinishedMatchHidesTheAfkWarning()
        {
            notices.Read(View(Controller.Player, 2, false));

            Assert.IsTrue(notices.Read(View(Controller.Player, 2, false, MatchState.Finished)));
            Assert.IsFalse(notices.AfkWarning);
        }

        [Test]
        public void ATickReportsAChangeOnlyWhenTheNetworkNoticeChanges()
        {
            connection.Start(ServerUri);
            Assert.IsFalse(notices.Tick(connection, now));
            AcceptHello();
            Assert.IsFalse(notices.Tick(connection, now));

            DropAndTick();
            Assert.IsTrue(notices.Tick(connection, now));
            Assert.AreEqual(NetworkNotice.Reconnecting, notices.Network);
            connection.Tick();
            Assert.IsFalse(notices.Tick(connection, now));

            now += 60;
            connection.Tick();
            AcceptHello();
            Assert.IsTrue(notices.Tick(connection, now));
            Assert.AreEqual(NetworkNotice.None, notices.Network);
        }

        [Test]
        public void AnIntentWaitingTooLongShowsWeakUntilItIsAcked()
        {
            GoOnline();
            connection.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent());

            now += 2.5;
            Assert.IsTrue(notices.Tick(connection, now));
            Assert.AreEqual(NetworkNotice.Weak, notices.Network);
            Assert.IsFalse(notices.Tick(connection, now + 1));
        }

        [Test]
        public void ReadingViewsAndTickingAllocateNothing()
        {
            GoOnline();
            var warned = View(Controller.Player, 2, true);
            var botPlaying = View(Controller.Bot, 3, false);
            TestDelegate frame = () =>
            {
                notices.Read(warned);
                notices.Tick(connection, now);
                notices.Read(botPlaying);
                notices.Tick(connection, now);
            };
            frame();

            Assert.That(frame, Allocates.Nothing);
        }
    }
}
