using HypeDomino.Core.Diagnostics;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class StartupPhasesTests
    {
        StartupPhases phases;
        double seconds;

        [SetUp]
        public void SetUp()
        {
            seconds = 0d;
            phases = new StartupPhases(SinceStartup);
        }

        [Test]
        public void ThePhasesAreListedInTheOrderTheyEndedWithTheTimeSinceStartup()
        {
            seconds = 0.3125d;
            phases.Mark(StartupPhases.Assemblies);
            seconds = 0.5d;
            phases.Mark("GraphicsBoot");
            seconds = 0.625d;
            phases.Mark(StartupPhases.Scene);
            seconds = 2.75d;
            Assert.AreEqual("Startup phases assemblies=312 GraphicsBoot=500 scene=625 Lobby=2750", phases.Close("Lobby"));
        }

        [Test]
        public void AFirstScreenWithoutEarlierPhasesStillCloses()
        {
            seconds = 1.5d;
            Assert.AreEqual("Startup phases Login=1500", phases.Close("Login"));
            Assert.IsTrue(phases.Closed);
        }

        [Test]
        public void NothingIsAddedOnceClosed()
        {
            phases.Mark(StartupPhases.Scene);
            phases.Close("Lobby");
            seconds = 9d;
            phases.Mark("AudioBoot");
            Assert.IsNull(phases.Close("Match"));
        }

        double SinceStartup() => seconds;
    }
}
