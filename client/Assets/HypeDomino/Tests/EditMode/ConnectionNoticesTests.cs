using Google.Protobuf;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ConnectionNoticesTests
    {
        static MatchSnapshot Snapshot(MatchView view)
        {
            var snapshot = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        static MatchView TwoSeats(SeatView own, SeatView other, uint afkTimeouts = 3)
        {
            var view = new MatchView { State = MatchState.Playing, Seat = Seat.X, Mode = Mode._1V1, AfkConsecutiveTimeouts = afkTimeouts };
            own.Seat = Seat.X;
            other.Seat = Seat.Y;
            view.Seats.Add(own);
            view.Seats.Add(other);
            return view;
        }

        static SeatNotice OtherSeat(MatchView view)
        {
            var snapshot = Snapshot(view);
            return ConnectionNotices.Seat(snapshot, snapshot.Seats[1]);
        }

        [Test]
        public void AConnectedPlayerHasNoNotice() =>
            Assert.AreEqual(SeatNotice.None, OtherSeat(TwoSeats(new SeatView(), new SeatView { Controller = Controller.Player })));

        [Test]
        public void ADisconnectedPlayerStillInControlHasDropped() =>
            Assert.AreEqual(SeatNotice.Dropped, OtherSeat(TwoSeats(new SeatView(), new SeatView { Controller = Controller.Player, Disconnected = true })));

        [Test]
        public void AHumanSeatPlayedByTheBotSaysSo() =>
            Assert.AreEqual(SeatNotice.BotPlaying, OtherSeat(TwoSeats(new SeatView(), new SeatView { Controller = Controller.Bot, Disconnected = true })));

        [Test]
        public void ABotFromTheRosterHasNoNotice()
        {
            var view = TwoSeats(new SeatView(), new SeatView { Controller = Controller.Bot });
            view.Roster.Add(new RosterEntry { Seat = Seat.Y, Nickname = "bot", Bot = true });

            Assert.AreEqual(SeatNotice.None, OtherSeat(view));
        }

        [TestCase(2u, 3u, true)]
        [TestCase(1u, 3u, false)]
        [TestCase(0u, 1u, true)]
        [TestCase(0u, 0u, false)]
        public void TheWarningComesOneTimeoutBeforeTheBot(uint timeouts, uint afkTimeouts, bool warned)
        {
            var view = TwoSeats(new SeatView { Controller = Controller.Player, ConsecutiveTimeouts = timeouts }, new SeatView(), afkTimeouts);

            Assert.AreEqual(warned, ConnectionNotices.OneTimeoutFromBot(Snapshot(view)));
        }

        [TestCase(MatchState.Opening, true)]
        [TestCase(MatchState.Playing, true)]
        [TestCase(MatchState.Finished, false)]
        public void TheWarningEndsWithTheMatch(MatchState state, bool warned)
        {
            var view = TwoSeats(new SeatView { Controller = Controller.Player, ConsecutiveTimeouts = 2 }, new SeatView());
            view.State = state;

            Assert.AreEqual(warned, ConnectionNotices.OneTimeoutFromBot(Snapshot(view)));
        }

        [Test]
        public void ASeatAlreadyPlayedByTheBotIsNotWarned()
        {
            var view = TwoSeats(new SeatView { Controller = Controller.Bot, ConsecutiveTimeouts = 2 }, new SeatView());

            Assert.IsFalse(ConnectionNotices.OneTimeoutFromBot(Snapshot(view)));
        }

        [TestCase(Controller.Bot, MatchState.Playing, true)]
        [TestCase(Controller.Bot, MatchState.Opening, true)]
        [TestCase(Controller.Player, MatchState.Playing, false)]
        [TestCase(Controller.Bot, MatchState.Finished, false)]
        public void TheOwnSeatPlayedByTheBotCanBeTakenBackUntilTheMatchEnds(Controller controller, MatchState state, bool offered)
        {
            var view = TwoSeats(new SeatView { Controller = controller, ConsecutiveTimeouts = 3 }, new SeatView { Controller = Controller.Player });
            view.State = state;

            Assert.AreEqual(offered, ConnectionNotices.CanTakeBack(Snapshot(view)));
        }

        [Test]
        public void AnotherSeatPlayedByTheBotIsNotOffered()
        {
            var view = TwoSeats(new SeatView { Controller = Controller.Player }, new SeatView { Controller = Controller.Bot, Disconnected = true });

            Assert.IsFalse(ConnectionNotices.CanTakeBack(Snapshot(view)));
        }

        [Test]
        public void AViewerWithoutASeatIsNotOffered()
        {
            var view = TwoSeats(new SeatView { Controller = Controller.Bot }, new SeatView { Controller = Controller.Bot });
            view.Seat = Seat.Unspecified;

            Assert.IsFalse(ConnectionNotices.CanTakeBack(Snapshot(view)));
        }

        [Test]
        public void OnlyTheOwnSeatCountsForTheWarning()
        {
            var view = TwoSeats(new SeatView { Controller = Controller.Player }, new SeatView { Controller = Controller.Player, ConsecutiveTimeouts = 2 });

            Assert.IsFalse(ConnectionNotices.OneTimeoutFromBot(Snapshot(view)));
        }
    }
}
