using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LoadingCardsTests
    {
        MatchSnapshot snapshot;
        LoadingCards cards;

        [SetUp]
        public void CreateCards()
        {
            snapshot = new MatchSnapshot();
            cards = new LoadingCards();
        }

        static MatchView Loading(Mode mode, Seat seat, params SeatView[] seats)
        {
            var view = new MatchView { State = MatchState.Loading, Mode = mode, Seat = seat };
            view.Seats.Add(seats);
            return view;
        }

        static SeatView At(Seat seat, uint progress, bool ready = false) => new() { Seat = seat, LoadingProgress = progress, Ready = ready };

        void Read(MatchView view)
        {
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            cards.Read(snapshot);
        }

        [Test]
        public void InTwoVersusTwoTheOwnTeamFillsTheFirstRow()
        {
            Read(Loading(Mode._2V2, Seat.South, At(Seat.North, 0), At(Seat.East, 0), At(Seat.South, 0), At(Seat.West, 0)));

            Assert.AreEqual(4, cards.Count);
            Assert.AreEqual(Seat.South, cards[0].Seat);
            Assert.AreEqual(Seat.North, cards[1].Seat);
            Assert.AreEqual(Seat.West, cards[2].Seat);
            Assert.AreEqual(Seat.East, cards[3].Seat);
            Assert.AreEqual(new[] { 0, 0, 1, 1 }, new[] { cards[0].Row, cards[1].Row, cards[2].Row, cards[3].Row });
        }

        [Test]
        public void EastSeesItsPartnerWestAndTheOpponentsNextToIt()
        {
            Read(Loading(Mode._2V2, Seat.East));

            Assert.AreEqual(Seat.East, cards[0].Seat);
            Assert.AreEqual(Seat.West, cards[1].Seat);
            Assert.AreEqual(Seat.South, cards[2].Seat);
            Assert.AreEqual(Seat.North, cards[3].Seat);
        }

        [Test]
        public void InOneVersusOneThereAreTwoCards()
        {
            Read(Loading(Mode._1V1, Seat.Y, At(Seat.X, 40), At(Seat.Y, 20)));

            Assert.AreEqual(2, cards.Count);
            Assert.AreEqual(Seat.Y, cards[0].Seat);
            Assert.AreEqual(0, cards[0].Row);
            Assert.AreEqual(Seat.X, cards[1].Seat);
            Assert.AreEqual(1, cards[1].Row);
        }

        [Test]
        public void AViewerWithoutASeatSeesTheDefaultSide()
        {
            Read(Loading(Mode._2V2, Seat.Unspecified));

            Assert.AreEqual(Seat.South, cards[0].Seat);
        }

        [Test]
        public void EachCardShowsItsProgressAndReadiness()
        {
            Read(Loading(Mode._1V1, Seat.X, At(Seat.X, 40), At(Seat.Y, 100, ready: true)));

            Assert.AreEqual(40, cards[0].Progress);
            Assert.AreEqual(LoadingStatus.Loading, cards[0].Status);
            Assert.AreEqual(100, cards[1].Progress);
            Assert.AreEqual(LoadingStatus.Ready, cards[1].Status);
            Assert.AreEqual(0.7f, cards.Progress, 1e-6f);
        }

        [Test]
        public void ProgressAboveTheMaximumIsCapped()
        {
            Read(Loading(Mode._1V1, Seat.X, At(Seat.X, 250), At(Seat.Y, 0)));

            Assert.AreEqual(100, cards[0].Progress);
            Assert.AreEqual(0.5f, cards.Progress, 1e-6f);
        }

        [Test]
        public void APlayerNotReadyWhenThePhaseEndsShowsTheBot()
        {
            var view = Loading(Mode._1V1, Seat.X, At(Seat.X, 100, ready: true), At(Seat.Y, 60));
            view.State = MatchState.Opening;

            Read(view);

            Assert.AreEqual(LoadingStatus.Ready, cards[0].Status);
            Assert.AreEqual(LoadingStatus.BotPlaying, cards[1].Status);
        }

        [Test]
        public void ARosterBotIsAlwaysReady()
        {
            var view = Loading(Mode._1V1, Seat.X, At(Seat.X, 30), At(Seat.Y, 0));
            view.Roster.Add(new RosterEntry { Seat = Seat.X, Nickname = "me" });
            view.Roster.Add(new RosterEntry { Seat = Seat.Y, Nickname = "bot", Bot = true });

            Read(view);

            Assert.AreEqual(0, cards[0].RosterIndex);
            Assert.AreEqual(1, cards[1].RosterIndex);
            Assert.AreEqual(LoadingStatus.Ready, cards[1].Status);
            Assert.AreEqual(100, cards[1].Progress);
        }

        [Test]
        public void ASeatMissingFromTheRosterHasNoRosterIndex()
        {
            Read(Loading(Mode._1V1, Seat.X));

            Assert.AreEqual(-1, cards[0].RosterIndex);
            Assert.AreEqual(0f, cards.Progress);
        }

        [Test]
        public void AViewWithoutAModeHasNoCards()
        {
            Read(new MatchView { State = MatchState.Loading });

            Assert.AreEqual(0, cards.Count);
            Assert.AreEqual(0f, cards.Progress);
        }

        [Test]
        public void ReadingTheCardsAllocatesNothing()
        {
            var view = Loading(Mode._2V2, Seat.South, At(Seat.North, 10), At(Seat.East, 20), At(Seat.South, 30), At(Seat.West, 100, ready: true));
            view.Roster.Add(new RosterEntry { Seat = Seat.North, Nickname = "partner" });
            Read(view);

            TestDelegate step = () => cards.Read(snapshot);
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
