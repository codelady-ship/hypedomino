using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TakeBackBannerTests
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
        Connection connection;
        TakeBackBanner banner;

        FakeSocket Socket => sockets[sockets.Count - 1];

        [SetUp]
        public void CreateConnection()
        {
            sockets.Clear();
            connection = new Connection(OpenSocket, () => new ClientHello { SessionToken = "sess_01HZX" }, () => 10, new System.Random(1));
            banner = new TakeBackBanner();
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

        static MatchSnapshot View(Controller own, MatchState state = MatchState.Playing)
        {
            var view = new MatchView { State = state, Seat = Seat.X, Mode = Mode._1V1, AfkConsecutiveTimeouts = 3 };
            view.Seats.Add(new SeatView { Seat = Seat.X, Controller = own, ConsecutiveTimeouts = 3 });
            view.Seats.Add(new SeatView { Seat = Seat.Y, Controller = Controller.Player });
            var snapshot = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        [TestCase(Controller.Bot, MatchState.Playing, true)]
        [TestCase(Controller.Player, MatchState.Playing, false)]
        [TestCase(Controller.Bot, MatchState.Finished, false)]
        public void TheBannerShowsWhileTheOwnSeatCanBeTakenBack(Controller own, MatchState state, bool shown)
        {
            banner.Read(View(own, state));

            Assert.AreEqual(shown, banner.Shown);
        }

        [Test]
        public void ATapOnTheBannerRejoinsOnceAndIgnoresRepeatTaps()
        {
            GoOnline();
            banner.Read(View(Controller.Bot));

            Assert.IsTrue(banner.Tap(connection));
            Assert.AreEqual(2, sockets.Count);
            Assert.AreEqual(ConnectionState.Connecting, connection.State);

            Assert.IsFalse(banner.Tap(connection));
            AcceptHello();
            Assert.IsFalse(banner.Tap(connection));
            Assert.AreEqual(2, sockets.Count);
        }

        [Test]
        public void TheSeatHandedBackHidesTheBanner()
        {
            GoOnline();
            banner.Read(View(Controller.Bot));
            Assert.IsTrue(banner.Tap(connection));
            AcceptHello();

            banner.Read(View(Controller.Player));

            Assert.IsFalse(banner.Shown);
            Assert.IsFalse(banner.Tap(connection));
            Assert.AreEqual(2, sockets.Count);
        }

        [Test]
        public void ANewViewStillPlayedByTheBotOffersTheTapAgain()
        {
            GoOnline();
            banner.Read(View(Controller.Bot));
            Assert.IsTrue(banner.Tap(connection));
            AcceptHello();

            banner.Read(View(Controller.Bot));

            Assert.IsTrue(banner.Tap(connection));
            Assert.AreEqual(3, sockets.Count);
        }

        [Test]
        public void ATapWithoutTheBannerDoesNothing()
        {
            GoOnline();
            banner.Read(View(Controller.Player));

            Assert.IsFalse(banner.Tap(connection));
            Assert.AreEqual(1, sockets.Count);
            Assert.AreEqual(ConnectionState.Online, connection.State);
        }

        [Test]
        public void ATapWhileNotOnlineIsNotCountedAsSent()
        {
            connection.Start(ServerUri);
            banner.Read(View(Controller.Bot));

            Assert.IsFalse(banner.Tap(connection));
            Assert.AreEqual(1, sockets.Count);

            AcceptHello();
            Assert.IsTrue(banner.Tap(connection));
            Assert.AreEqual(2, sockets.Count);
        }

        [Test]
        public void ReadingAViewAndAnIgnoredTapAllocateNothing()
        {
            GoOnline();
            var view = View(Controller.Player);
            TestDelegate step = () =>
            {
                banner.Read(view);
                banner.Tap(connection);
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
