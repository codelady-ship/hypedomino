using HypeDomino.Game;
using HypeDomino.Game.Tutorial;
using HypeDomino.UI.Localization;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TutorialOfferTests
    {
        const int Unset = -1;

        static readonly string[] OfferKeys = { TutorialOffer.Title, TutorialOffer.Body, TutorialOffer.Play, TutorialOffer.Skip };

        TutorialOffer offer;
        int changes;
        int keptFinished;

        [SetUp]
        public void StartFresh()
        {
            keptFinished = PlayerPrefs.GetInt(TutorialProgressStore.FinishedKey, Unset);
            PlayerPrefs.DeleteKey(TutorialProgressStore.FinishedKey);
            offer = new TutorialOffer();
            changes = 0;
            offer.Changed += CountChange;
        }

        [TearDown]
        public void RestoreDeviceProgress()
        {
            if (keptFinished == Unset) PlayerPrefs.DeleteKey(TutorialProgressStore.FinishedKey);
            else PlayerPrefs.SetInt(TutorialProgressStore.FinishedKey, keptFinished);
            PlayerPrefs.Save();
        }

        void CountChange() => changes++;

        static ServerHello Accepted(bool inMatch) => new ServerHello { Status = HelloStatus.Accepted, InMatch = inMatch };

        [Test]
        public void TheFirstLobbyEntryAsksWhenTheTutorialWasNeverFinished()
        {
            Assert.IsFalse(offer.Asking);
            Assert.IsTrue(offer.Read(AppState.Lobby));
            Assert.IsTrue(offer.Asking);
            Assert.IsTrue(offer.Offered);
            Assert.AreEqual(1, changes);
        }

        [TestCase(AppState.Boot)]
        [TestCase(AppState.Login)]
        [TestCase(AppState.Match)]
        [TestCase(AppState.Room)]
        [TestCase(AppState.Maintenance)]
        [TestCase(AppState.UpdateRequired)]
        public void NoOtherStateAsks(AppState state)
        {
            Assert.IsFalse(offer.Read(state));
            Assert.IsFalse(offer.Asking);
            Assert.IsFalse(offer.Offered);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void AFinishedDeviceIsNeverAsked()
        {
            var run = new TutorialRun();
            run.Skip();
            TutorialProgressStore.Keep(run);
            Assert.IsFalse(offer.Read(AppState.Lobby));
            Assert.IsFalse(offer.Asking);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void SkipKeepsTheDeviceFinishedSoItIsNotOfferedAgain()
        {
            offer.Read(AppState.Lobby);
            Assert.IsTrue(offer.Decline());
            Assert.IsFalse(offer.Asking);
            Assert.AreEqual(TutorialState.Skipped, offer.Run.State);
            Assert.IsFalse(TutorialProgressStore.ShouldOffer);
            Assert.IsFalse(new TutorialOffer().Read(AppState.Lobby));
            Assert.AreEqual(2, changes);
        }

        [Test]
        public void PlayStartsTheRunFromTheFirstStepWithoutKeepingTheDevice()
        {
            offer.Read(AppState.Lobby);
            Assert.IsTrue(offer.Accept());
            Assert.IsFalse(offer.Asking);
            Assert.AreEqual(TutorialState.Running, offer.Run.State);
            Assert.AreEqual(0, offer.Run.StepIndex);
            Assert.AreEqual(0u, offer.Run.PlayerPoints);
            Assert.IsTrue(TutorialProgressStore.ShouldOffer);
            Assert.AreEqual(2, changes);
        }

        [Test]
        public void AnswersCountOnlyWhileAsking()
        {
            Assert.IsFalse(offer.Accept());
            Assert.IsFalse(offer.Decline());
            offer.Read(AppState.Lobby);
            offer.Accept();
            Assert.IsFalse(offer.Decline());
            Assert.IsFalse(offer.Accept());
            Assert.IsTrue(TutorialProgressStore.ShouldOffer);
            Assert.AreEqual(TutorialState.Running, offer.Run.State);
        }

        [Test]
        public void ItAsksAtMostOncePerAppSession()
        {
            offer.Read(AppState.Lobby);
            offer.Accept();
            offer.Read(AppState.Match);
            Assert.IsFalse(offer.Read(AppState.Lobby));
            Assert.IsFalse(offer.Asking);
            Assert.AreEqual(2, changes);
        }

        [Test]
        public void LeavingTheLobbyWithdrawsAnUnansweredPromptWithoutKeepingTheDevice()
        {
            offer.Read(AppState.Lobby);
            Assert.IsTrue(offer.Read(AppState.Room));
            Assert.IsFalse(offer.Asking);
            Assert.IsTrue(TutorialProgressStore.ShouldOffer);
            Assert.IsFalse(offer.Read(AppState.Lobby));
            Assert.AreEqual(2, changes);
        }

        [Test]
        public void ItFollowsTheAppFlowIntoTheLobby()
        {
            var flow = new AppFlow();
            offer.Follow(flow);
            Assert.IsFalse(offer.Asking);
            flow.Hello(Accepted(false));
            Assert.IsTrue(offer.Asking);
        }

        [Test]
        public void AResumedMatchIsNotInterruptedAndTheLobbyAfterItAsks()
        {
            var flow = new AppFlow();
            offer.Follow(flow);
            flow.Hello(Accepted(true));
            Assert.IsFalse(offer.Asking);
            flow.View(MatchState.Finished);
            flow.LeaveMatch();
            Assert.AreEqual(AppState.Lobby, flow.State);
            Assert.IsTrue(offer.Asking);
        }

        [Test]
        public void FollowingAFlowAlreadyInTheLobbyAsks()
        {
            var flow = new AppFlow();
            flow.Hello(Accepted(false));
            offer.Follow(flow);
            Assert.IsTrue(offer.Asking);
        }

        [Test]
        public void TheShippedBundleHasThePromptInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            foreach (var key in OfferKeys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
