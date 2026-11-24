using System.Collections.Generic;
using HypeDomino.Game;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AppFlowTests
    {
        static ServerHello Accepted(bool inMatch) => new ServerHello { Status = HelloStatus.Accepted, InMatch = inMatch };

        [Test]
        public void StartsInBootAndGoesToLoginWithoutASession()
        {
            var flow = new AppFlow();
            Assert.AreEqual(AppState.Boot, flow.State);
            flow.SignInNeeded();
            Assert.AreEqual(AppState.Login, flow.State);
        }

        [Test]
        public void AnAcceptedHelloOpensTheLobbyOrTheRunningMatch()
        {
            var lobby = new AppFlow();
            lobby.Hello(Accepted(false));
            Assert.AreEqual(AppState.Lobby, lobby.State);

            var match = new AppFlow();
            match.Hello(Accepted(true));
            Assert.AreEqual(AppState.Match, match.State);
        }

        [Test]
        public void UpdateRequiredBlocksPlayAndIgnoresSignIn()
        {
            var flow = new AppFlow();
            flow.Hello(new ServerHello { Status = HelloStatus.UpdateRequired });
            Assert.AreEqual(AppState.UpdateRequired, flow.State);
            flow.SignInNeeded();
            Assert.AreEqual(AppState.UpdateRequired, flow.State);
        }

        [Test]
        public void MaintenanceShowsTheMaintenanceScreenUntilAnAcceptedHello()
        {
            var flow = new AppFlow();
            flow.Hello(new ServerHello { Status = HelloStatus.Maintenance });
            Assert.AreEqual(AppState.Maintenance, flow.State);
            flow.Hello(Accepted(false));
            Assert.AreEqual(AppState.Lobby, flow.State);
        }

        [Test]
        public void AnUnknownHelloStatusChangesNothing()
        {
            var flow = new AppFlow();
            flow.Hello(new ServerHello { Status = (HelloStatus)99 });
            Assert.AreEqual(AppState.Boot, flow.State);
        }

        [Test]
        public void AMatchViewInTheLobbyEntersTheMatchUnlessItIsFinished()
        {
            var flow = new AppFlow();
            flow.Hello(Accepted(false));
            flow.View(MatchState.Finished);
            Assert.AreEqual(AppState.Lobby, flow.State);
            flow.View(MatchState.Loading);
            Assert.AreEqual(AppState.Match, flow.State);
        }

        [Test]
        public void TheMatchCanBeLeftOnlyOnceItIsFinished()
        {
            var flow = new AppFlow();
            flow.Hello(Accepted(true));
            Assert.IsFalse(flow.LeaveMatch());
            flow.View(MatchState.Playing);
            Assert.IsFalse(flow.LeaveMatch());
            Assert.AreEqual(AppState.Match, flow.State);
            flow.View(MatchState.Finished);
            Assert.IsTrue(flow.LeaveMatch());
            Assert.AreEqual(AppState.Lobby, flow.State);
        }

        static LobbyState InRoom(LobbyPhase phase) => new LobbyState { Code = "K7QM3X", LeaderAccountId = "a", Phase = phase };

        static readonly LobbyState NoRoom = new LobbyState();

        static AppFlow InCustomMatch()
        {
            var flow = new AppFlow();
            flow.Hello(Accepted(false));
            flow.Room(InRoom(LobbyPhase.Lobby));
            flow.Room(InRoom(LobbyPhase.Match));
            flow.View(MatchState.Loading);
            return flow;
        }

        [Test]
        public void EnteringARoomOpensItAndLeavingItReturnsToTheLobby()
        {
            var flow = new AppFlow();
            flow.Hello(Accepted(false));
            flow.Room(InRoom(LobbyPhase.Lobby));
            Assert.AreEqual(AppState.Room, flow.State);
            flow.Room(InRoom(LobbyPhase.Teams));
            Assert.AreEqual(AppState.Room, flow.State);
            flow.Room(NoRoom);
            Assert.AreEqual(AppState.Lobby, flow.State);
        }

        [Test]
        public void ACustomMatchStartsFromTheRoom()
        {
            var flow = InCustomMatch();
            Assert.AreEqual(AppState.Match, flow.State);
            flow.Room(InRoom(LobbyPhase.Match));
            Assert.AreEqual(AppState.Match, flow.State);
        }

        [TestCase(true)]
        [TestCase(false)]
        public void AMemberStillInTheRoomReturnsToItWhenTheMatchEnds(bool resultFirst)
        {
            var flow = InCustomMatch();
            flow.View(MatchState.Playing);
            if (resultFirst) flow.View(MatchState.Finished);
            flow.Room(InRoom(LobbyPhase.Lobby));
            Assert.AreEqual(AppState.Room, flow.State);
            Assert.IsFalse(flow.LeaveMatch());
        }

        [Test]
        public void AMemberWhoLeftTheRoomDuringTheMatchGoesToTheLobby()
        {
            var flow = InCustomMatch();
            flow.Room(NoRoom);
            Assert.AreEqual(AppState.Match, flow.State);
            flow.View(MatchState.Finished);
            Assert.IsTrue(flow.LeaveMatch());
            Assert.AreEqual(AppState.Lobby, flow.State);
        }

        [Test]
        public void LeavingAFinishedMatchBeforeTheRoomReturnsGoesToTheRoom()
        {
            var flow = InCustomMatch();
            flow.View(MatchState.Finished);
            Assert.IsTrue(flow.LeaveMatch());
            Assert.AreEqual(AppState.Room, flow.State);
        }

        [Test]
        public void ARoomFromBeforeAReconnectIsForgotten()
        {
            var flow = InCustomMatch();
            flow.Hello(Accepted(true));
            flow.View(MatchState.Finished);
            Assert.IsTrue(flow.LeaveMatch());
            Assert.AreEqual(AppState.Lobby, flow.State);
        }

        [Test]
        public void ARoomStateBeforeTheHelloOrOnABlockedScreenChangesNothing()
        {
            var boot = new AppFlow();
            boot.Room(InRoom(LobbyPhase.Lobby));
            Assert.AreEqual(AppState.Boot, boot.State);
            var maintenance = new AppFlow();
            maintenance.Hello(new ServerHello { Status = HelloStatus.Maintenance });
            maintenance.Room(InRoom(LobbyPhase.Lobby));
            Assert.AreEqual(AppState.Maintenance, maintenance.State);
        }

        [Test]
        public void ChangedFiresOncePerNewState()
        {
            var flow = new AppFlow();
            var seen = new List<AppState>();
            flow.Changed += seen.Add;
            flow.Hello(Accepted(false));
            flow.Hello(Accepted(false));
            flow.View(MatchState.Opening);
            flow.View(MatchState.Playing);
            CollectionAssert.AreEqual(new[] { AppState.Lobby, AppState.Match }, seen);
        }
    }
}
