using HypeDomino.Game.Tutorial;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TutorialRunTests
    {
        const int Unset = -1;

        TutorialRun run;
        int keptFinished;

        [SetUp]
        public void StartFresh()
        {
            run = new TutorialRun();
            keptFinished = PlayerPrefs.GetInt(TutorialProgressStore.FinishedKey, Unset);
            PlayerPrefs.DeleteKey(TutorialProgressStore.FinishedKey);
        }

        [TearDown]
        public void RestoreDeviceProgress()
        {
            if (keptFinished == Unset) PlayerPrefs.DeleteKey(TutorialProgressStore.FinishedKey);
            else PlayerPrefs.SetInt(TutorialProgressStore.FinishedKey, keptFinished);
            PlayerPrefs.Save();
        }

        [Test]
        public void ARunWaitsUntilItIsStarted()
        {
            Assert.AreEqual(TutorialState.NotStarted, run.State);
            Assert.IsNull(run.Current);
            Assert.IsFalse(run.Continue());
            Assert.AreEqual(TutorialMove.NotPlayerTurn, run.TryPlace(new TileFace(2, 3), BoardEnd.Unspecified));
            Assert.AreEqual(0, run.StepsOnBoard);
        }

        [Test]
        public void StartingShowsTheFirstStep()
        {
            run.Start();
            Assert.AreEqual(TutorialState.Running, run.State);
            Assert.AreSame(TutorialScript.Steps[0], run.Current);
            Assert.AreEqual(TutorialScript.Steps.Length, run.StepCount);
        }

        [Test]
        public void ACoachStepMovesOnWithContinue()
        {
            run.Start();
            Assert.AreEqual(TutorialActor.Coach, run.Current.Actor);
            Assert.IsTrue(run.Continue());
            Assert.AreEqual(1, run.StepIndex);
        }

        [Test]
        public void APlayerStepWaitsForTheScriptedTileAndEnd()
        {
            run.Start();
            ToNextPlayerStep();
            var step = run.Current;
            Assert.IsTrue(run.AwaitsPlayer);
            Assert.IsFalse(run.Continue());
            Assert.AreEqual(TutorialMove.WrongTile, run.TryPlace(new TileFace(6, 6), step.End));
            Assert.AreSame(step, run.Current);
            Assert.AreEqual(TutorialMove.Accepted, run.TryPlace(new TileFace(step.Tile.High, step.Tile.Low), step.End));
            Assert.AreNotSame(step, run.Current);
        }

        [Test]
        public void APlacementOnAnotherEndIsRefused()
        {
            run.Start();
            ToNextPlayerStep();
            run.TryPlace(run.Current.Tile, run.Current.End);
            ToNextPlayerStep();
            var step = run.Current;
            Assert.AreEqual(TutorialMove.WrongEnd, run.TryPlace(step.Tile, BoardEnd.Right));
            Assert.AreSame(step, run.Current);
            Assert.AreEqual(0u, run.PlayerPoints);
        }

        [Test]
        public void ABotStepIsOnTheBoardAndCountedAsSoonAsItIsShown()
        {
            run.Start();
            while (run.Current.Actor != TutorialActor.Bot || run.Current.Points == 0) Play();
            var step = run.Current;
            Assert.AreEqual(step.Points, run.BotPoints);
            Assert.AreEqual(run.StepIndex + 1, run.StepsOnBoard);
            Assert.IsTrue(run.Continue());
            Assert.AreEqual(step.Points, run.BotPoints);
        }

        [Test]
        public void PlayingTheWholeScriptCompletesItWithTheScriptedPoints()
        {
            run.Start();
            while (run.State == TutorialState.Running) Play();
            Assert.AreEqual(TutorialState.Completed, run.State);
            Assert.IsTrue(run.IsFinished);
            Assert.IsNull(run.Current);
            Assert.AreEqual(Sum(TutorialActor.Player), run.PlayerPoints);
            Assert.AreEqual(Sum(TutorialActor.Bot), run.BotPoints);
            Assert.AreEqual(run.StepCount, run.StepsOnBoard);
            Assert.IsFalse(run.Continue());
            Assert.IsFalse(run.Skip());
        }

        [Test]
        public void SkippingEndsTheRunAtAnyStep()
        {
            Assert.IsTrue(run.Skip());
            Assert.AreEqual(TutorialState.Skipped, run.State);
            run.Start();
            ToNextPlayerStep();
            Assert.IsTrue(run.Skip());
            Assert.AreEqual(TutorialState.Skipped, run.State);
            Assert.IsNull(run.Current);
            Assert.AreEqual(0, run.StepsOnBoard);
            Assert.AreEqual(TutorialMove.NotPlayerTurn, run.TryPlace(new TileFace(2, 3), BoardEnd.Unspecified));
            Assert.IsFalse(run.Skip());
        }

        [Test]
        public void ReplayingStartsAgainFromTheFirstStepWithNoPoints()
        {
            run.Start();
            while (run.State == TutorialState.Running) Play();
            run.Start();
            Assert.AreEqual(TutorialState.Running, run.State);
            Assert.AreEqual(0, run.StepIndex);
            Assert.AreEqual(0u, run.PlayerPoints);
            Assert.AreEqual(0u, run.BotPoints);
            run.Skip();
            run.Start();
            Assert.AreEqual(TutorialState.Running, run.State);
        }

        [Test]
        public void AnEmptyScriptCompletesAtOnce()
        {
            var empty = new TutorialRun(new TutorialStep[0]);
            empty.Start();
            Assert.AreEqual(TutorialState.Completed, empty.State);
        }

        [Test]
        public void ThePlayerIsOfferedTheTutorialUntilARunFinishes()
        {
            Assert.IsTrue(TutorialProgressStore.ShouldOffer);
            run.Start();
            Assert.IsFalse(TutorialProgressStore.Keep(run));
            Assert.IsTrue(TutorialProgressStore.ShouldOffer);
            run.Skip();
            Assert.IsTrue(TutorialProgressStore.Keep(run));
            Assert.IsFalse(TutorialProgressStore.ShouldOffer);
            Assert.IsFalse(TutorialProgressStore.Keep(run));
        }

        [Test]
        public void ACompletedRunIsKeptToo()
        {
            run.Start();
            while (run.State == TutorialState.Running) Play();
            Assert.IsTrue(TutorialProgressStore.Keep(run));
            Assert.IsFalse(TutorialProgressStore.ShouldOffer);
        }

        [Test]
        public void PlayingTheScriptAllocatesNothing()
        {
            PlayThrough();
            TestDelegate step = PlayThrough;
            Assert.That(step, Allocates.Nothing);
        }

        void PlayThrough()
        {
            run.Start();
            while (run.State == TutorialState.Running) Play();
        }

        void Play()
        {
            if (run.AwaitsPlayer) run.TryPlace(run.Current.Tile, run.Current.End);
            else run.Continue();
        }

        void ToNextPlayerStep()
        {
            while (!run.AwaitsPlayer) run.Continue();
        }

        static uint Sum(TutorialActor actor)
        {
            var total = 0u;
            foreach (var step in TutorialScript.Steps)
                if (step.Actor == actor) total += step.Points;
            return total;
        }
    }
}
