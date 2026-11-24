using HypeDomino.Game;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MaintenanceScreenTests
    {
        const ulong ServerNowMs = 1_800_000_000_000;
        const ulong EndsMs = ServerNowMs + 60_000;

        MaintenanceScreen screen;
        double now;
        int dials;
        int changes;

        void Dial() => dials++;

        double Now() => now;

        void Count() => changes++;

        static ServerHello Maintenance(ulong serverMs, ulong endsMs) => new() { Status = HelloStatus.Maintenance, ServerTimeUnixMs = serverMs, MaintenanceEndsUnixMs = endsMs };

        static ServerHello Status(HelloStatus status) => new() { Status = status, ServerTimeUnixMs = ServerNowMs };

        [SetUp]
        public void Build()
        {
            now = 100;
            dials = 0;
            changes = 0;
            screen = new MaintenanceScreen(Dial, Now);
            screen.Changed += Count;
        }

        [Test]
        public void StartsInactiveAndNeitherRetriesNorTicks()
        {
            Assert.IsFalse(screen.Active);
            Assert.IsFalse(screen.Retry());
            now += 1_000_000;
            screen.Tick();
            Assert.AreEqual(0, dials);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void AMaintenanceHelloShowsTheEndTimeTheServerSent()
        {
            screen.Read(Maintenance(ServerNowMs, EndsMs));

            Assert.IsTrue(screen.Active);
            Assert.IsTrue(screen.HasEnd);
            Assert.AreEqual(EndsMs, screen.EndsUnixMs);
            Assert.IsFalse(screen.Checking);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void AMaintenanceHelloWithoutAnEndShowsNoEndAndNeverRetriesOnItsOwn()
        {
            screen.Read(Maintenance(ServerNowMs, 0));
            now += 1_000_000;
            screen.Tick();

            Assert.IsTrue(screen.Active);
            Assert.IsFalse(screen.HasEnd);
            Assert.AreEqual(0, dials);
        }

        [Test]
        public void RetryDialsOnceUntilTheServerAnswers()
        {
            screen.Read(Maintenance(ServerNowMs, EndsMs));

            Assert.IsTrue(screen.Retry());
            Assert.IsTrue(screen.Checking);
            Assert.IsFalse(screen.Retry());
            Assert.AreEqual(1, dials);
            Assert.AreEqual(2, changes);

            screen.Read(Maintenance(ServerNowMs + 5_000, EndsMs));
            Assert.IsFalse(screen.Checking);
            Assert.AreEqual(3, changes);
            Assert.IsTrue(screen.Retry());
            Assert.AreEqual(2, dials);
        }

        [Test]
        public void TheEndTimeIsCountedOnServerTimeNotTheDeviceClock()
        {
            screen.Read(Maintenance(ServerNowMs, EndsMs));

            now += 59.5;
            screen.Tick();
            Assert.AreEqual(0, dials);

            now += 0.5;
            screen.Tick();
            Assert.AreEqual(1, dials);
            Assert.IsTrue(screen.Checking);
        }

        [Test]
        public void ThePassedEndTimeRetriesOnlyOnceUntilTheServerMovesIt()
        {
            screen.Read(Maintenance(ServerNowMs, EndsMs));
            now += 60;
            screen.Tick();
            screen.Read(Maintenance(EndsMs + 1_000, EndsMs));
            now += 1_000;
            screen.Tick();
            Assert.AreEqual(1, dials);

            const ulong movedMs = EndsMs + 600_000;
            screen.Read(Maintenance(EndsMs + 2_000, movedMs));
            Assert.AreEqual(movedMs, screen.EndsUnixMs);
            now += 597.5;
            screen.Tick();
            Assert.AreEqual(1, dials);
            now += 0.5;
            screen.Tick();
            Assert.AreEqual(2, dials);
        }

        [Test]
        public void AnEndAlreadyPassedWhenShownRetriesAtTheNextTick()
        {
            screen.Read(Maintenance(EndsMs + 30_000, EndsMs));
            screen.Tick();

            Assert.AreEqual(1, dials);
        }

        [TestCase(HelloStatus.Accepted)]
        [TestCase(HelloStatus.UpdateRequired)]
        public void AnyOtherAnswerClosesTheScreen(HelloStatus status)
        {
            screen.Read(Maintenance(ServerNowMs, EndsMs));
            screen.Retry();

            screen.Read(Status(status));

            Assert.IsFalse(screen.Active);
            Assert.IsFalse(screen.Checking);
            Assert.IsFalse(screen.HasEnd);
            Assert.AreEqual(3, changes);
            now += 1_000;
            screen.Tick();
            Assert.IsFalse(screen.Retry());
            Assert.AreEqual(1, dials);
        }

        [Test]
        public void TheSameEndAfterTheScreenClosedRetriesOnItsOwnAgain()
        {
            screen.Read(Maintenance(ServerNowMs, EndsMs));
            now += 60;
            screen.Tick();
            screen.Read(Status(HelloStatus.Accepted));

            screen.Read(Maintenance(EndsMs + 1_000, EndsMs));
            screen.Tick();

            Assert.AreEqual(2, dials);
        }

        [Test]
        public void AnUnknownStatusChangesNothing()
        {
            screen.Read(Maintenance(ServerNowMs, EndsMs));

            screen.Read(Status(HelloStatus.Unspecified));

            Assert.IsTrue(screen.Active);
            Assert.AreEqual(EndsMs, screen.EndsUnixMs);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void TickingBeforeTheEndAllocatesNothing()
        {
            screen.Read(Maintenance(ServerNowMs, EndsMs));
            TestDelegate step = screen.Tick;
            step();

            Assert.That(step, Allocates.Nothing);
            Assert.AreEqual(0, dials);
        }
    }
}
