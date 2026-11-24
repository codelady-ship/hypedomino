using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using HypeDomino.UI.Match;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchScreenTests
    {
        const ulong ServerNowMs = 1_000_000;
        static readonly Uri ServerUri = new("wss://play.test/ws");

        sealed class Table : IMatchTable
        {
            public Vector2 BoardSize => new(400f, 300f);
            public float BoardMaxScale => 1f;
            public int BackgroundsShown;
            public MatchSnapshot Background;
            public int SeatsShown;
            public int ScoresShown;
            public int BoardsShown;
            public MatchSnapshot Seats;
            public ScoreBoard Score;
            public ChainLayout Board;

            public void ShowBackground(MatchSnapshot view)
            {
                BackgroundsShown++;
                Background = view;
            }

            public void ShowSeats(MatchSnapshot view)
            {
                SeatsShown++;
                Seats = view;
            }

            public void ShowScore(ScoreBoard score)
            {
                ScoresShown++;
                Score = score;
            }

            public void ShowBoard(ChainLayout board)
            {
                BoardsShown++;
                Board = board;
            }
        }

        sealed class Hud : IMatchHud
        {
            public int HandsShown;
            public HandChoice Hand;
            public int TurnsShown;
            public float TurnRemaining;
            public int PassesShown;
            public TableSide PassSide;
            public float PassAlpha;
            public int PassesHidden;
            public TableSide HiddenSide;
            public int NoticesShown;
            public MatchNotices Notices;

            public void ShowHand(HandChoice hand)
            {
                HandsShown++;
                Hand = hand;
            }

            public void ShowTurn(TurnTimer timer)
            {
                TurnsShown++;
                TurnRemaining = timer.Remaining;
            }

            public void ShowPass(TableSide side, PassNotice notice)
            {
                PassesShown++;
                PassSide = side;
                PassAlpha = notice.Alpha;
            }

            public void HidePass(TableSide side)
            {
                PassesHidden++;
                HiddenSide = side;
            }

            public void ShowNotices(MatchNotices notices)
            {
                NoticesShown++;
                Notices = notices;
            }
        }

        sealed class FakeSocket : ISocketTransport
        {
            readonly Queue<byte[]> incoming = new();

            public TransportState State { get; set; } = TransportState.Idle;

            public void Connect(Uri uri) => State = TransportState.Connecting;

            public void Send(ReadOnlySpan<byte> frame)
            {
            }

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

        sealed class ServerFrames : IFrameTransport
        {
            readonly byte[] lastSent = new byte[Session.MaxSentFrameBytes];
            int lastSentLength;
            byte[] pending;

            public int SentCount { get; private set; }

            public void Send(ReadOnlySpan<byte> frame)
            {
                SentCount++;
                frame.CopyTo(lastSent);
                lastSentLength = frame.Length;
            }

            public Envelope.PayloadOneofCase LastSentPayload =>
                FrameReader.TryRead(lastSent.AsSpan(0, lastSentLength), out var frame) ? frame.Payload : Envelope.PayloadOneofCase.None;

            public PlaceTileIntent LastSentPlacement()
            {
                Assert.IsTrue(FrameReader.TryRead(lastSent.AsSpan(0, lastSentLength), out var frame));
                Assert.AreEqual(Envelope.PayloadOneofCase.PlaceTileIntent, frame.Payload);
                return PlaceTileIntent.Parser.ParseFrom(frame.PayloadBytes);
            }

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = pending;
                var has = pending != null;
                pending = null;
                return has;
            }

            public void Deliver(Envelope.PayloadOneofCase payload, IMessage message) =>
                pending = new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, 1), payload, message).ToArray();
        }

        readonly MatchSnapshot snapshot = new();
        Table table;
        Hud hud;
        MatchScreen screen;
        readonly List<FakeSocket> sockets = new();
        int warningsPlayed;
        MatchSounds soundsPlayed;

        [SetUp]
        public void CreateScreen()
        {
            table = new Table();
            hud = new Hud();
            warningsPlayed = 0;
            soundsPlayed = MatchSounds.None;
            sockets.Clear();
            screen = new MatchScreen(table, hud, PlaySounds);
        }

        void PlaySounds(MatchSounds sounds)
        {
            soundsPlayed |= sounds;
            if ((sounds & MatchSounds.TurnWarning) != 0) warningsPlayed++;
        }

        static Tile T(uint low, uint high) => new() { Low = low, High = high };

        static OrientedTile O(uint first, uint second) => new() { First = first, Second = second };

        static MatchView Playing(Seat current, params LegalMove[] legal)
        {
            var view = new MatchView
            {
                State = MatchState.Playing,
                Seat = Seat.South,
                Mode = Mode._2V2,
                Current = current,
                Scores = new TeamPoints { TeamA = 40, TeamB = 25 },
                TargetScore = 150,
                TurnDeadlineUnixMs = ServerNowMs + 10_000,
                TurnSeconds = 15,
                TurnWarningSeconds = 5,
                Board = new BoardView { Line = { O(2, 3), O(3, 6), O(6, 1) }, OpeningIndex = 1 },
            };
            view.Hand.Add(new[] { T(1, 2), T(3, 3), T(4, 6) });
            view.LegalMoves.Add(legal);
            return view;
        }

        static LegalMove Move(uint low, uint high, BoardEnd end) => new() { Tile = T(low, high), End = end };

        MatchSnapshot Read(MatchView view)
        {
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        void Show(MatchView view) => screen.Show(Read(view));

        Connection OfflineConnection() =>
            new(OpenSocket, () => new ClientHello { SessionToken = "sess_01HZX" }, () => 0d, new System.Random(1));

        Connection SyncedConnection()
        {
            var connection = OfflineConnection();
            connection.Clock.Sync(ServerNowMs, 0d, 0d);
            return connection;
        }

        Connection OnlineConnection()
        {
            var connection = OfflineConnection();
            connection.Start(ServerUri);
            var socket = sockets[sockets.Count - 1];
            socket.State = TransportState.Open;
            var hello = new ServerHello { Status = HelloStatus.Accepted, SessionNonce = 77 };
            socket.Deliver(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, 0), Envelope.PayloadOneofCase.ServerHello, hello).ToArray());
            connection.Tick();
            Assert.AreEqual(ConnectionState.Online, connection.State);
            return connection;
        }

        Session BoundSession()
        {
            var connection = OfflineConnection();
            screen.Bind(connection);
            return connection.Session;
        }

        ISocketTransport OpenSocket()
        {
            var socket = new FakeSocket();
            sockets.Add(socket);
            return socket;
        }

        static MatchView WithSeat(MatchView view, Seat seat, Controller controller, bool disconnected)
        {
            view.Seats.Add(new SeatView { Seat = seat, Controller = controller, Disconnected = disconnected });
            return view;
        }

        [Test]
        public void AViewReachesTheBackgroundTheSeatsTheScoreTheBoardAndTheHand()
        {
            Show(Playing(Seat.South, Move(1, 2, BoardEnd.Right)));

            Assert.AreEqual(1, table.BackgroundsShown);
            Assert.AreSame(snapshot, table.Background);
            Assert.AreEqual(1, table.SeatsShown);
            Assert.AreSame(snapshot, table.Seats);
            Assert.AreEqual(1, table.ScoresShown);
            Assert.AreSame(screen.Score, table.Score);
            Assert.AreEqual(Team.A, screen.Score.Mine.Team);
            Assert.AreEqual(40u, screen.Score.Mine.Score);
            Assert.AreEqual(25u, screen.Score.Theirs.Score);
            Assert.AreEqual(150u, screen.Score.TargetScore);
            Assert.AreEqual(1, table.BoardsShown);
            Assert.AreSame(screen.Board, table.Board);
            Assert.AreEqual(3, screen.Board.Count);
            Assert.Greater(screen.Board.Scale, 0f);
            Assert.AreEqual(1, hud.HandsShown);
            Assert.AreSame(screen.Hand, hud.Hand);
        }

        [Test]
        public void TheHandHighlightsOnlyWhatTheServerListsAsLegal()
        {
            Show(Playing(Seat.South, Move(3, 3, BoardEnd.Left)));

            Assert.AreEqual(3, screen.Hand.HandCount);
            Assert.IsFalse(screen.Hand.IsSelectable(0));
            Assert.IsTrue(screen.Hand.IsSelectable(1));
            Assert.IsFalse(screen.Hand.IsSelectable(2));
        }

        [Test]
        public void NoLegalMovesFromTheServerMeansNoHighlightEvenWhenATileWouldFit()
        {
            Show(Playing(Seat.South));

            for (var i = 0; i < screen.Hand.HandCount; i++) Assert.IsFalse(screen.Hand.IsSelectable(i));
            Assert.IsTrue(screen.Hand.CanPass);
        }

        [Test]
        public void TheTimerTicksFromTheServerDeadline()
        {
            Show(Playing(Seat.South));
            screen.Tick(SyncedConnection(), 4d, false);

            Assert.AreEqual(1, hud.TurnsShown);
            Assert.IsTrue(screen.Timer.Running);
            Assert.AreEqual(6f, hud.TurnRemaining, 0.001f);
            Assert.IsFalse(screen.Timer.IsWarning);

            screen.Tick(SyncedConnection(), 6d, false);
            Assert.IsTrue(screen.Timer.IsWarning);
        }

        [Test]
        public void TheTurnWarningPlaysOnceWhenTheLocalTurnReachesTheWarning()
        {
            Show(Playing(Seat.South));
            screen.Tick(SyncedConnection(), 4d, false);
            Assert.AreEqual(0, warningsPlayed);

            screen.Tick(SyncedConnection(), 6d, false);
            Assert.AreEqual(1, warningsPlayed);
            Assert.AreEqual(MatchSounds.TurnWarning, soundsPlayed);

            screen.Tick(SyncedConnection(), 7d, false);
            screen.Tick(SyncedConnection(), 11d, false);
            Assert.AreEqual(1, warningsPlayed);
        }

        [Test]
        public void TheTurnWarningPlaysAgainOnTheNextLocalTurn()
        {
            Show(Playing(Seat.South));
            screen.Tick(SyncedConnection(), 6d, false);
            Show(Playing(Seat.West));
            screen.Tick(SyncedConnection(), 6d, false);
            Assert.AreEqual(1, warningsPlayed);

            Show(Playing(Seat.South));
            screen.Tick(SyncedConnection(), 6d, false);
            Assert.AreEqual(2, warningsPlayed);
        }

        [Test]
        public void AnotherSeatsTurnPlaysNoTurnWarning()
        {
            Show(Playing(Seat.West));
            screen.Tick(SyncedConnection(), 6d, false);

            Assert.IsTrue(screen.Timer.IsWarning);
            Assert.AreEqual(0, warningsPlayed);
        }

        [Test]
        public void AViewerWithoutASeatHearsNoTurnWarning()
        {
            var view = Playing(Seat.Unspecified);
            view.Seat = Seat.Unspecified;
            Show(view);
            screen.Tick(SyncedConnection(), 6d, false);

            Assert.IsTrue(screen.Timer.IsWarning);
            Assert.AreEqual(0, warningsPlayed);
        }

        [Test]
        public void TheTurnWarningStaysSilentWithoutASyncedConnection()
        {
            Show(Playing(Seat.South));
            screen.Tick(OfflineConnection(), 6d, false);

            Assert.AreEqual(0, warningsPlayed);
        }

        [Test]
        public void TheTimerStaysStillWithoutASyncedConnection()
        {
            Show(Playing(Seat.South));
            screen.Tick(OfflineConnection(), 4d, false);

            Assert.AreEqual(1, hud.TurnsShown);
            Assert.IsFalse(screen.Timer.Running);
            Assert.AreEqual(0f, hud.TurnRemaining);
        }

        [Test]
        public void APassShowsAtThePassingSeatAndLeavesAfterItsLifetime()
        {
            Show(Playing(Seat.West));
            screen.Tick(SyncedConnection(), 0d, false);
            Assert.AreEqual(0, hud.PassesShown);

            Show(Playing(Seat.North));
            screen.Tick(SyncedConnection(), 10d, false);
            Assert.AreEqual(1, hud.PassesShown);
            Assert.AreEqual(TableSide.Left, hud.PassSide);

            screen.Tick(SyncedConnection(), 10d + Seconds(PassNotice.LifetimeMs) - 0.05d, false);
            Assert.AreEqual(2, hud.PassesShown);
            Assert.AreEqual(0, hud.PassesHidden);

            screen.Tick(SyncedConnection(), 10d + Seconds(PassNotice.LifetimeMs) + 0.01d, false);
            Assert.AreEqual(1, hud.PassesHidden);
            Assert.AreEqual(TableSide.Left, hud.HiddenSide);

            screen.Tick(SyncedConnection(), 20d, false);
            Assert.AreEqual(2, hud.PassesShown);
            Assert.AreEqual(1, hud.PassesHidden);
        }

        [Test]
        public void APlacementIsNotAPass()
        {
            Show(Playing(Seat.West));
            var placed = Playing(Seat.North);
            placed.Board.Line.Add(O(1, 4));
            Show(placed);
            screen.Tick(SyncedConnection(), 0d, false);

            Assert.AreEqual(0, hud.PassesShown);
        }

        [Test]
        public void TheFirstViewAfterAHelloShowsNoPass()
        {
            var session = BoundSession();
            var frames = new ServerFrames();
            session.Hello(frames, new ClientHello());
            frames.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = 1 });
            session.Poll();
            frames.Deliver(Envelope.PayloadOneofCase.MatchView, Playing(Seat.West));
            session.Poll();
            frames.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = 1 });
            session.Poll();
            frames.Deliver(Envelope.PayloadOneofCase.MatchView, Playing(Seat.North));
            session.Poll();
            screen.Tick(SyncedConnection(), 0d, false);

            Assert.AreEqual(2, table.SeatsShown);
            Assert.AreEqual(0, hud.PassesShown);
        }

        [Test]
        public void ViewsFromTheSessionStopAfterUnbind()
        {
            var session = BoundSession();
            var frames = new ServerFrames();
            session.Hello(frames, new ClientHello());
            frames.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = 1 });
            session.Poll();
            frames.Deliver(Envelope.PayloadOneofCase.MatchView, Playing(Seat.South, Move(1, 2, BoardEnd.Right)));
            session.Poll();

            Assert.AreEqual(1, table.SeatsShown);
            Assert.IsTrue(screen.Hand.IsSelectable(0));

            screen.Unbind();
            frames.Deliver(Envelope.PayloadOneofCase.MatchView, Playing(Seat.West));
            session.Poll();
            Assert.AreEqual(1, table.SeatsShown);
        }

        [Test]
        public void AViewOutsideAKnownModeShowsNoPass()
        {
            var first = Playing(Seat.West);
            first.Mode = Mode.Unspecified;
            Show(first);
            var second = Playing(Seat.North);
            second.Mode = Mode.Unspecified;
            Show(second);
            screen.Tick(SyncedConnection(), 0d, false);

            Assert.AreEqual(0, hud.PassesShown);
        }

        [Test]
        public void ShowingAViewAndTickingAllocateNothing()
        {
            var connection = SyncedConnection();
            var west = Playing(Seat.West, Move(1, 2, BoardEnd.Right)).ToByteArray();
            var north = Playing(Seat.North).ToByteArray();
            var now = 0d;
            TestDelegate frame = () =>
            {
                snapshot.TryRead(west);
                screen.Show(snapshot);
                screen.Tick(connection, now += 0.1d, false);
                snapshot.TryRead(north);
                screen.Show(snapshot);
                screen.Tick(connection, now += 0.1d, true);
            };
            frame();

            Assert.That(frame, Allocates.Nothing);
            Assert.Greater(hud.PassesShown, 0);
        }

        [Test]
        public void TickingIntoTheTurnWarningAllocatesNothing()
        {
            var connection = SyncedConnection();
            var south = Playing(Seat.South).ToByteArray();
            var west = Playing(Seat.West).ToByteArray();
            TestDelegate frame = () =>
            {
                snapshot.TryRead(south);
                screen.Show(snapshot);
                screen.Tick(connection, 6d, false);
                snapshot.TryRead(west);
                screen.Show(snapshot);
                screen.Tick(connection, 6d, false);
            };
            frame();

            Assert.That(frame, Allocates.Nothing);
            Assert.Greater(warningsPlayed, 1);
        }

        [Test]
        public void AViewReachesTheHudNoticesOnlyWhenWhatShowsChanges()
        {
            Show(Playing(Seat.South));
            Assert.AreEqual(0, hud.NoticesShown);

            Show(WithSeat(Playing(Seat.South), Seat.North, Controller.Player, true));
            Assert.AreEqual(1, hud.NoticesShown);
            Assert.AreSame(screen.Notices, hud.Notices);
            Assert.AreEqual(SeatNotice.Dropped, screen.Notices.Seats.At(TableSide.Top));

            Show(WithSeat(Playing(Seat.West), Seat.North, Controller.Player, true));
            Assert.AreEqual(1, hud.NoticesShown);

            Show(Playing(Seat.South));
            Assert.AreEqual(2, hud.NoticesShown);
            Assert.AreEqual(SeatNotice.None, screen.Notices.Seats.At(TableSide.Top));
        }

        [Test]
        public void ATickReachesTheHudNoticesOnlyWhenTheNetworkNoticeChanges()
        {
            var connection = OnlineConnection();
            screen.Tick(connection, 1d, false);
            Assert.AreEqual(0, hud.NoticesShown);

            sockets[sockets.Count - 1].State = TransportState.Closed;
            connection.Tick();
            screen.Tick(connection, 2d, false);
            Assert.AreEqual(1, hud.NoticesShown);
            Assert.AreSame(screen.Notices, hud.Notices);
            Assert.AreEqual(NetworkNotice.Reconnecting, screen.Notices.Network);

            screen.Tick(connection, 3d, false);
            Assert.AreEqual(1, hud.NoticesShown);
        }

        [Test]
        public void ATapOnTheTakeBackBannerRejoinsOnceUntilTheNextView()
        {
            var connection = OnlineConnection();
            Assert.IsFalse(screen.TapTakeBack(connection));

            Show(WithSeat(Playing(Seat.West), Seat.South, Controller.Bot, true));
            Assert.IsTrue(screen.Notices.TakeBack.Shown);
            Assert.AreEqual(1, hud.NoticesShown);

            Assert.IsTrue(screen.TapTakeBack(connection));
            Assert.AreEqual(2, sockets.Count);
            Assert.IsFalse(screen.TapTakeBack(connection));
            Assert.AreEqual(2, sockets.Count);
        }

        [Test]
        public void TickingTheConnectionNoticesAllocatesNothing()
        {
            var connection = OnlineConnection();
            connection.Send(Envelope.PayloadOneofCase.PassIntent, new PassIntent());
            var dropped = WithSeat(Playing(Seat.West), Seat.North, Controller.Player, true).ToByteArray();
            var settled = Playing(Seat.West).ToByteArray();
            TestDelegate frame = () =>
            {
                snapshot.TryRead(dropped);
                screen.Show(snapshot);
                screen.Tick(connection, 1d, false);
                snapshot.TryRead(settled);
                screen.Show(snapshot);
                screen.Tick(connection, 5d, false);
            };
            frame();
            var noticesShown = hud.NoticesShown;

            Assert.That(frame, Allocates.Nothing);
            Assert.Greater(hud.NoticesShown, noticesShown);
        }

        [Test]
        public void AMoveWaitingForItsAckShowsTheWeakNoticeUntilTheAckArrives()
        {
            var connection = OnlineConnection();
            screen.Bind(connection);
            Show(Playing(Seat.South, Move(1, 2, BoardEnd.Right)));

            Assert.AreEqual(HandTap.Placed, screen.TapTile(0));
            Assert.IsTrue(connection.RoundTrip.HasPending);
            screen.Tick(connection, ConnectionNotices.WeakUnackedSeconds - 0.5d, false);
            Assert.AreEqual(NetworkNotice.None, screen.Notices.Network);

            screen.Tick(connection, ConnectionNotices.WeakUnackedSeconds + 0.5d, false);
            Assert.AreEqual(NetworkNotice.Weak, screen.Notices.Network);
            Assert.AreEqual(1, hud.NoticesShown);

            var view = Playing(Seat.West);
            sockets[sockets.Count - 1].Deliver(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 1, 77), Envelope.PayloadOneofCase.MatchView, view).ToArray());
            connection.Tick();
            screen.Tick(connection, ConnectionNotices.WeakUnackedSeconds + 1d, false);

            Assert.IsFalse(connection.RoundTrip.HasPending);
            Assert.AreEqual(NetworkNotice.None, screen.Notices.Network);
            Assert.AreEqual(2, hud.NoticesShown);
            Assert.AreEqual(Seat.West, table.Seats.Current);
        }

        ServerFrames BindAccepted()
        {
            var session = BoundSession();
            var frames = new ServerFrames();
            session.Hello(frames, new ClientHello());
            frames.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = 1 });
            session.Poll();
            return frames;
        }

        [Test]
        public void ATapOnATileWithOneLegalEndSendsItsPlacementOnceUntilTheNextView()
        {
            var frames = BindAccepted();
            Show(Playing(Seat.South, Move(1, 2, BoardEnd.Right)));
            var sentBefore = frames.SentCount;
            var handsShown = hud.HandsShown;

            Assert.AreEqual(HandTap.Placed, screen.TapTile(0));
            Assert.AreEqual(sentBefore + 1, frames.SentCount);
            var placement = frames.LastSentPlacement();
            Assert.AreEqual(1u, placement.Tile.Low);
            Assert.AreEqual(2u, placement.Tile.High);
            Assert.AreEqual(BoardEnd.Right, placement.End);
            Assert.AreEqual(handsShown + 1, hud.HandsShown);

            Assert.AreEqual(HandTap.Ignored, screen.TapTile(0));
            Assert.IsFalse(screen.ChooseEnd(BoardEnd.Right));
            Assert.IsFalse(screen.Pass());
            Assert.AreEqual(sentBefore + 1, frames.SentCount);

            Show(Playing(Seat.South, Move(1, 2, BoardEnd.Right)));
            Assert.AreEqual(HandTap.Placed, screen.TapTile(0));
            Assert.AreEqual(sentBefore + 2, frames.SentCount);
        }

        [Test]
        public void ATileWithTwoLegalEndsWaitsForTheChosenEnd()
        {
            var frames = BindAccepted();
            Show(Playing(Seat.South, Move(3, 3, BoardEnd.Left), Move(3, 3, BoardEnd.Right)));
            var sentBefore = frames.SentCount;

            Assert.AreEqual(HandTap.Selected, screen.TapTile(1));
            Assert.AreEqual(sentBefore, frames.SentCount);
            Assert.IsFalse(screen.ChooseEnd(BoardEnd.Top));
            Assert.AreEqual(sentBefore, frames.SentCount);

            Assert.IsTrue(screen.ChooseEnd(BoardEnd.Left));
            Assert.AreEqual(sentBefore + 1, frames.SentCount);
            var placement = frames.LastSentPlacement();
            Assert.AreEqual(3u, placement.Tile.Low);
            Assert.AreEqual(BoardEnd.Left, placement.End);
            Assert.IsFalse(screen.ChooseEnd(BoardEnd.Right));
            Assert.AreEqual(sentBefore + 1, frames.SentCount);
        }

        [Test]
        public void ATileTheServerDidNotListSendsNothing()
        {
            var frames = BindAccepted();
            Show(Playing(Seat.South, Move(1, 2, BoardEnd.Right)));
            var sentBefore = frames.SentCount;

            Assert.AreEqual(HandTap.Ignored, screen.TapTile(2));
            Assert.AreEqual(HandTap.Ignored, screen.TapTile(5));
            Assert.AreEqual(sentBefore, frames.SentCount);
        }

        [Test]
        public void PassIsSentOnlyWhenTheViewOffersIt()
        {
            var frames = BindAccepted();
            Show(Playing(Seat.West));
            var sentBefore = frames.SentCount;
            Assert.IsFalse(screen.Pass());
            Assert.AreEqual(sentBefore, frames.SentCount);

            Show(Playing(Seat.South));
            Assert.IsFalse(screen.Draw());
            Assert.IsFalse(screen.EndTurn());
            Assert.IsTrue(screen.Pass());
            Assert.AreEqual(Envelope.PayloadOneofCase.PassIntent, frames.LastSentPayload);
            Assert.IsFalse(screen.Pass());
            Assert.AreEqual(sentBefore + 1, frames.SentCount);
        }

        [Test]
        public void DrawIsSentInOneVersusOneWhileTheBoneyardHoldsMoreThanOneTile()
        {
            var frames = BindAccepted();
            var view = Playing(Seat.South);
            view.Mode = Mode._1V1;
            view.BoneyardSize = 2;
            Show(view);
            var sentBefore = frames.SentCount;

            Assert.IsFalse(screen.Pass());
            Assert.IsTrue(screen.Draw());
            Assert.AreEqual(Envelope.PayloadOneofCase.DrawIntent, frames.LastSentPayload);
            Assert.IsFalse(screen.Draw());
            Assert.AreEqual(sentBefore + 1, frames.SentCount);

            view.BoneyardSize = 1;
            Show(view);
            Assert.IsFalse(screen.Draw());
            Assert.IsTrue(screen.Pass());
            Assert.AreEqual(Envelope.PayloadOneofCase.PassIntent, frames.LastSentPayload);
        }

        [Test]
        public void EndTurnIsSentOnlyAfterAPlacementThisTurn()
        {
            var frames = BindAccepted();
            Show(Playing(Seat.South, Move(3, 3, BoardEnd.Left)));
            Assert.IsFalse(screen.EndTurn());

            var view = Playing(Seat.South, Move(3, 3, BoardEnd.Left));
            view.TurnPlacements = 1;
            Show(view);
            var sentBefore = frames.SentCount;
            Assert.IsTrue(screen.EndTurn());
            Assert.AreEqual(Envelope.PayloadOneofCase.EndTurnIntent, frames.LastSentPayload);
            Assert.AreEqual(HandTap.Ignored, screen.TapTile(1));
            Assert.AreEqual(sentBefore + 1, frames.SentCount);
        }

        [Test]
        public void NothingIsSentWithoutAnAcceptedSession()
        {
            Show(Playing(Seat.South, Move(1, 2, BoardEnd.Right)));
            Assert.AreEqual(HandTap.Ignored, screen.TapTile(0));
            Assert.IsTrue(screen.Hand.IsSelectable(0));

            var session = BoundSession();
            var frames = new ServerFrames();
            session.Hello(frames, new ClientHello());
            frames.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Maintenance });
            session.Poll();
            Show(Playing(Seat.South, Move(1, 2, BoardEnd.Right)));

            Assert.AreEqual(HandTap.Ignored, screen.TapTile(0));
            Assert.AreEqual(1, frames.SentCount);
        }

        [Test]
        public void ATapThatSendsAnIntentAllocatesNothing()
        {
            var frames = BindAccepted();
            var placing = Playing(Seat.South, Move(1, 2, BoardEnd.Right)).ToByteArray();
            var choosing = Playing(Seat.South, Move(3, 3, BoardEnd.Left), Move(3, 3, BoardEnd.Right)).ToByteArray();
            var passing = Playing(Seat.South).ToByteArray();
            TestDelegate input = () =>
            {
                snapshot.TryRead(placing);
                screen.Show(snapshot);
                screen.TapTile(0);
                snapshot.TryRead(choosing);
                screen.Show(snapshot);
                screen.TapTile(1);
                screen.ChooseEnd(BoardEnd.Right);
                snapshot.TryRead(passing);
                screen.Show(snapshot);
                screen.Pass();
            };
            input();
            var sentBefore = frames.SentCount;

            Assert.That(input, Allocates.Nothing);
            Assert.AreEqual(sentBefore + 3, frames.SentCount);
        }

        static double Seconds(float ms) => ms / 1000d;
    }
}
