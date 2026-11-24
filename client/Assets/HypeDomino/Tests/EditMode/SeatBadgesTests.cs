using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Match;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SeatBadgesTests
    {
        static MatchSnapshot Snapshot(MatchView view)
        {
            var snapshot = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            return snapshot;
        }

        static MatchView FourSeats(Seat own)
        {
            var view = new MatchView { State = MatchState.Playing, Seat = own, Mode = Mode._2V2 };
            view.Seats.Add(new SeatView { Seat = Seat.North, Controller = Controller.Bot, Disconnected = true });
            view.Seats.Add(new SeatView { Seat = Seat.East, Controller = Controller.Player });
            view.Seats.Add(new SeatView { Seat = Seat.South, Controller = Controller.Player });
            view.Seats.Add(new SeatView { Seat = Seat.West, Controller = Controller.Player, Disconnected = true });
            return view;
        }

        static MatchView TwoSeats(SeatView x, SeatView y)
        {
            var view = new MatchView { State = MatchState.Playing, Seat = Seat.Y, Mode = Mode._1V1 };
            x.Seat = Seat.X;
            y.Seat = Seat.Y;
            view.Seats.Add(x);
            view.Seats.Add(y);
            return view;
        }

        static void AssertSides(SeatBadges badges, SeatNotice bottom, SeatNotice left, SeatNotice top, SeatNotice right)
        {
            Assert.AreEqual(bottom, badges.At(TableSide.Bottom));
            Assert.AreEqual(left, badges.At(TableSide.Left));
            Assert.AreEqual(top, badges.At(TableSide.Top));
            Assert.AreEqual(right, badges.At(TableSide.Right));
        }

        [Test]
        public void EachSeatNoticeSitsOnTheTableSideOfItsSeat()
        {
            var badges = new SeatBadges();

            Assert.IsTrue(badges.Read(Snapshot(FourSeats(Seat.South))));

            AssertSides(badges, SeatNotice.None, SeatNotice.Dropped, SeatNotice.BotPlaying, SeatNotice.None);
        }

        [Test]
        public void TheSidesTurnWithTheLocalSeat()
        {
            var badges = new SeatBadges();

            badges.Read(Snapshot(FourSeats(Seat.East)));

            AssertSides(badges, SeatNotice.None, SeatNotice.None, SeatNotice.Dropped, SeatNotice.BotPlaying);
        }

        [Test]
        public void AViewerWithoutASeatSeesTheSidesFromSouth()
        {
            var badges = new SeatBadges();

            badges.Read(Snapshot(FourSeats(Seat.Unspecified)));

            AssertSides(badges, SeatNotice.None, SeatNotice.Dropped, SeatNotice.BotPlaying, SeatNotice.None);
        }

        [Test]
        public void TheOpponentOfAOneVersusOneSitsOnTop()
        {
            var badges = new SeatBadges();

            badges.Read(Snapshot(TwoSeats(new SeatView { Controller = Controller.Player, Disconnected = true }, new SeatView { Controller = Controller.Player })));

            AssertSides(badges, SeatNotice.None, SeatNotice.None, SeatNotice.Dropped, SeatNotice.None);
        }

        [Test]
        public void TheOwnSeatPlayedByTheBotShowsItsBadge()
        {
            var badges = new SeatBadges();

            badges.Read(Snapshot(TwoSeats(new SeatView { Controller = Controller.Player }, new SeatView { Controller = Controller.Bot })));

            AssertSides(badges, SeatNotice.BotPlaying, SeatNotice.None, SeatNotice.None, SeatNotice.None);
        }

        [Test]
        public void ARosterBotHasNoBadge()
        {
            var view = TwoSeats(new SeatView { Controller = Controller.Bot }, new SeatView { Controller = Controller.Player });
            view.Roster.Add(new RosterEntry { Seat = Seat.X, Nickname = "bot", Bot = true });
            var badges = new SeatBadges();

            Assert.IsFalse(badges.Read(Snapshot(view)));

            AssertSides(badges, SeatNotice.None, SeatNotice.None, SeatNotice.None, SeatNotice.None);
        }

        [Test]
        public void ASeatOutsideTheModeIsSkipped()
        {
            var view = TwoSeats(new SeatView { Controller = Controller.Player }, new SeatView { Controller = Controller.Player });
            view.Seats.Add(new SeatView { Seat = Seat.North, Controller = Controller.Player, Disconnected = true });
            var badges = new SeatBadges();

            Assert.IsFalse(badges.Read(Snapshot(view)));

            AssertSides(badges, SeatNotice.None, SeatNotice.None, SeatNotice.None, SeatNotice.None);
        }

        [Test]
        public void AViewWithoutAModeClearsEveryBadge()
        {
            var badges = new SeatBadges();
            badges.Read(Snapshot(FourSeats(Seat.South)));
            var view = FourSeats(Seat.South);
            view.Mode = Mode.Unspecified;

            Assert.IsTrue(badges.Read(Snapshot(view)));

            AssertSides(badges, SeatNotice.None, SeatNotice.None, SeatNotice.None, SeatNotice.None);
        }

        [Test]
        public void ReadReportsAChangeOnlyWhenABadgeChanged()
        {
            var badges = new SeatBadges();
            var dropped = Snapshot(TwoSeats(new SeatView { Controller = Controller.Player, Disconnected = true }, new SeatView { Controller = Controller.Player }));
            var back = Snapshot(TwoSeats(new SeatView { Controller = Controller.Player }, new SeatView { Controller = Controller.Player }));

            Assert.IsTrue(badges.Read(dropped));
            Assert.IsFalse(badges.Read(dropped));
            Assert.IsTrue(badges.Read(back));
            Assert.AreEqual(SeatNotice.None, badges.At(TableSide.Top));
            Assert.IsFalse(badges.Read(back));
        }

        [Test]
        public void ReadingAViewAllocatesNothing()
        {
            var badges = new SeatBadges();
            var view = FourSeats(Seat.South);
            view.Roster.Add(new RosterEntry { Seat = Seat.East, Nickname = "east" });
            var snapshot = Snapshot(view);
            TestDelegate read = () => badges.Read(snapshot);
            read();

            Assert.That(read, Allocates.Nothing);
        }

        [TestCase(Seat.X, 2, true)]
        [TestCase(Seat.Y, 2, true)]
        [TestCase(Seat.North, 2, false)]
        [TestCase(Seat.West, 4, true)]
        [TestCase(Seat.X, 4, false)]
        [TestCase(Seat.Unspecified, 4, false)]
        public void ASeatBelongsOnlyToItsMode(Seat seat, int seatCount, bool inMode) =>
            Assert.AreEqual(inMode, SeatOrder.IsIn(seat, seatCount));
    }
}
