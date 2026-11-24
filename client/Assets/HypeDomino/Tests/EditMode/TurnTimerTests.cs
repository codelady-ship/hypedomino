using Google.Protobuf;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TurnTimerTests
    {
        const ulong ServerStartMs = 1_800_000_000_000;

        ServerClock clock;
        MatchSnapshot snapshot;
        TurnTimer timer;

        [SetUp]
        public void StartFresh()
        {
            clock = new ServerClock();
            clock.Sync(ServerStartMs, 0, 0);
            snapshot = new MatchSnapshot();
            timer = new TurnTimer();
        }

        void Read(MatchState state, uint turnSeconds, uint warningSeconds)
        {
            var view = new MatchView
            {
                State = state,
                Seat = Seat.X,
                Mode = Mode._1V1,
                Current = Seat.X,
                TurnDeadlineUnixMs = ServerStartMs + turnSeconds * 1000UL,
                TurnSeconds = turnSeconds,
                TurnWarningSeconds = warningSeconds,
            };
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));
            timer.Read(snapshot);
        }

        [TestCase(0.0, 20f, 1f, false)]
        [TestCase(10.0, 10f, 0.5f, false)]
        [TestCase(14.9, 5.1f, 0.255f, false)]
        [TestCase(15.0, 5f, 0.25f, true)]
        [TestCase(19.0, 1f, 0.05f, true)]
        [TestCase(25.0, 0f, 0f, true)]
        public void TheRingFollowsServerTimeAndWarnsAtTheThreshold(double nowSeconds, float remaining, float fill, bool warning)
        {
            Read(MatchState.Playing, 20, 5);

            timer.Tick(clock, nowSeconds);

            Assert.IsTrue(timer.Running);
            Assert.AreEqual(remaining, timer.Remaining, 0.001f);
            Assert.AreEqual(fill, timer.Fill, 0.001f);
            Assert.AreEqual(warning, timer.IsWarning);
        }

        [Test]
        public void TheThresholdComesFromTheView()
        {
            Read(MatchState.Opening, 30, 8);

            timer.Tick(clock, 21.5);
            Assert.IsFalse(timer.IsWarning);
            Assert.AreEqual(0.283f, timer.Fill, 0.001f);

            timer.Tick(clock, 22.0);
            Assert.IsTrue(timer.IsWarning);
        }

        [Test]
        public void AServerClockAheadOfTheDeviceCountsFromServerTime()
        {
            clock.Sync(ServerStartMs + 12_000, 0, 0);
            Read(MatchState.Playing, 20, 5);

            timer.Tick(clock, 0);

            Assert.AreEqual(8f, timer.Remaining, 0.001f);
            Assert.AreEqual(0.4f, timer.Fill, 0.001f);
            Assert.IsFalse(timer.IsWarning);
        }

        [TestCase(MatchState.Loading)]
        [TestCase(MatchState.Waiting)]
        [TestCase(MatchState.Finished)]
        public void NoTurnRunsOutsideOpeningAndPlay(MatchState state)
        {
            Read(state, 20, 5);

            timer.Tick(clock, 19.0);

            Assert.IsFalse(timer.Running);
            Assert.AreEqual(0f, timer.Fill);
            Assert.IsFalse(timer.IsWarning);
        }

        [Test]
        public void NothingRunsWithoutTheTurnLengthOrASyncedClock()
        {
            Read(MatchState.Playing, 0, 5);
            timer.Tick(clock, 1.0);
            Assert.IsFalse(timer.Running);

            Read(MatchState.Playing, 20, 5);
            timer.Tick(new ServerClock(), 1.0);
            Assert.IsFalse(timer.Running);
            Assert.IsFalse(timer.IsWarning);
        }

        [Test]
        public void TickingAllocatesNothing()
        {
            Read(MatchState.Playing, 20, 5);
            var now = 0.0;
            TestDelegate step = () =>
            {
                timer.Tick(clock, now);
                now += 0.016;
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
