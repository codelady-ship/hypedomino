using System;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Game;
using HypeDomino.Net;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class NetStartPhasesTests
    {
        sealed class FakeSocket : ISocketTransport
        {
            public TransportState State { get; set; } = TransportState.Connecting;

            public void Connect(Uri uri) => State = TransportState.Connecting;

            public void Send(ReadOnlySpan<byte> frame) { }

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                return false;
            }

            public void Dispose() => State = TransportState.Closed;
        }

        StartupPhases startup;
        NetStartPhases phases;
        double seconds;

        [SetUp]
        public void SetUp()
        {
            seconds = 0d;
            startup = new StartupPhases(SinceStartup);
            phases = new NetStartPhases(startup);
        }

        [Test]
        public void EachNetworkStepIsListedWhenItEnds()
        {
            seconds = 1d;
            phases.Resumed();
            seconds = 1.25d;
            phases.Proved();
            var socket = new FakeSocket();
            phases.Dialing(socket);
            seconds = 1.375d;
            phases.Tick();
            seconds = 1.5d;
            socket.State = TransportState.Open;
            phases.Tick();
            seconds = 1.75d;
            Assert.AreEqual("Startup phases resume=1000 proof=1250 dial=1500 Lobby=1750", startup.Close("Lobby"));
        }

        [Test]
        public void ARepeatedStepIsListedOnlyTheFirstTime()
        {
            seconds = 0.5d;
            phases.Resumed();
            phases.Proved();
            var socket = new FakeSocket { State = TransportState.Open };
            phases.Dialing(socket);
            phases.Tick();
            seconds = 0.75d;
            phases.Resumed();
            phases.Proved();
            phases.Dialing(new FakeSocket { State = TransportState.Open });
            phases.Tick();
            Assert.AreEqual("Startup phases resume=500 proof=500 dial=500 Login=750", startup.Close("Login"));
        }

        [Test]
        public void ARetriedDialIsListedWhenTheLatestSocketOpens()
        {
            var dropped = new FakeSocket();
            phases.Dialing(dropped);
            dropped.Dispose();
            seconds = 0.25d;
            phases.Tick();
            var retried = new FakeSocket();
            phases.Dialing(retried);
            seconds = 2d;
            retried.State = TransportState.Open;
            phases.Tick();
            Assert.AreEqual("Startup phases dial=2000 Lobby=2000", startup.Close("Lobby"));
        }

        [Test]
        public void DialingHandsBackTheSocketItWasGiven()
        {
            var socket = new FakeSocket();
            Assert.AreSame(socket, phases.Dialing(socket));
        }

        double SinceStartup() => seconds;
    }
}
