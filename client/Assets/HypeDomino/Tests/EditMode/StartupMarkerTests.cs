using System.Collections.Generic;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Game;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class StartupMarkerTests
    {
        readonly List<string> lines = new List<string>();
        AppFlow flow;
        double seconds;

        [SetUp]
        public void SetUp()
        {
            lines.Clear();
            seconds = 0d;
            flow = new AppFlow();
            new StartupMarker(flow, SinceStartup, lines.Add, new StartupPhases(SinceStartup));
        }

        [Test]
        public void NothingIsLoggedWhileTheAppBoots()
        {
            seconds = 1.5d;
            Assert.IsEmpty(lines);
        }

        [Test]
        public void TheFirstScreenAfterBootIsLoggedWithTheTimeSinceStartup()
        {
            seconds = 2.5d;
            flow.Hello(new ServerHello { Status = HelloStatus.Accepted });
            CollectionAssert.AreEqual(new[] { "Startup state=Lobby ms=2500", "Startup phases Lobby=2500" }, lines);
        }

        [Test]
        public void OnlyTheFirstScreenIsLogged()
        {
            seconds = 0.75d;
            flow.SignInNeeded();
            seconds = 9d;
            flow.Hello(new ServerHello { Status = HelloStatus.Accepted, InMatch = true });
            flow.View(MatchState.Finished);
            flow.LeaveMatch();
            CollectionAssert.AreEqual(new[] { "Startup state=Login ms=750", "Startup phases Login=750" }, lines);
        }

        [TestCase(HelloStatus.UpdateRequired, "Startup state=UpdateRequired ms=1250", "Startup phases UpdateRequired=1250")]
        [TestCase(HelloStatus.Maintenance, "Startup state=Maintenance ms=1250", "Startup phases Maintenance=1250")]
        public void AScreenThatBlocksPlayIsLoggedAsItself(HelloStatus status, string marker, string trace)
        {
            seconds = 1.25d;
            flow.Hello(new ServerHello { Status = status });
            CollectionAssert.AreEqual(new[] { marker, trace }, lines);
        }

        [Test]
        public void ThePhasesBeforeTheFirstScreenAreLoggedAfterTheMarker()
        {
            var phases = new StartupPhases(SinceStartup);
            flow = new AppFlow();
            lines.Clear();
            new StartupMarker(flow, SinceStartup, lines.Add, phases);
            seconds = 0.25d;
            phases.Mark(StartupPhases.Assemblies);
            seconds = 0.5d;
            phases.Mark(StartupPhases.Scene);
            seconds = 2d;
            flow.SignInNeeded();
            CollectionAssert.AreEqual(new[] { "Startup state=Login ms=2000", "Startup phases assemblies=250 scene=500 Login=2000" }, lines);
        }

        [Test]
        public void ATraceClosedElsewhereIsNotLoggedAgain()
        {
            var phases = new StartupPhases(SinceStartup);
            phases.Close("Lobby");
            flow = new AppFlow();
            lines.Clear();
            new StartupMarker(flow, SinceStartup, lines.Add, phases);
            flow.SignInNeeded();
            CollectionAssert.AreEqual(new[] { "Startup state=Login ms=0" }, lines);
        }

        double SinceStartup() => seconds;
    }
}
