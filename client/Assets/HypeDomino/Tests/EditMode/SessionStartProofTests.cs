using System;
using System.Collections.Generic;
using HypeDomino.Net;
using HypeDomino.Net.Integrity;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SessionStartProofTests
    {
        readonly List<IntegrityPurpose> purposes = new();
        readonly List<Action<IntegrityProof>> pending = new();
        readonly List<ClientHello> hellos = new();
        SessionStartProof startProof;

        [SetUp]
        public void Create()
        {
            purposes.Clear();
            pending.Clear();
            hellos.Clear();
            startProof = new SessionStartProof(RequestProof, Dial);
        }

        void RequestProof(IntegrityPurpose purpose, Action<IntegrityProof> done)
        {
            purposes.Add(purpose);
            pending.Add(done);
        }

        void Dial()
        {
            var hello = new ClientHello { SessionToken = "access-1" };
            startProof.Attach(hello);
            hellos.Add(hello);
        }

        static IntegrityProof Proof(string token) => new() { PlayIntegrityToken = token };

        static ServerHello Hello(HelloStatus status) => new() { Status = status };

        static Error Refused(ErrorCode code) => new() { Code = code };

        [Test]
        public void ANewSessionAsksForASessionStartProofBeforeItDials()
        {
            Assert.IsTrue(startProof.Start());

            CollectionAssert.AreEqual(new[] { IntegrityPurpose.SessionStart }, purposes);
            Assert.IsTrue(startProof.Proving);
            Assert.AreEqual(0, hellos.Count);
        }

        [Test]
        public void TheFirstHelloCarriesTheProof()
        {
            startProof.Start();
            pending[0](Proof("token-1"));

            Assert.AreEqual(1, hellos.Count);
            Assert.AreEqual("token-1", hellos[0].Integrity.PlayIntegrityToken);
            Assert.IsTrue(startProof.LastHelloProved);
            Assert.IsFalse(startProof.Proving);
        }

        [Test]
        public void ADeviceWithoutAProofStillDialsAndLeavesTheDecisionToTheServer()
        {
            startProof.Start();
            pending[0](null);

            Assert.AreEqual(1, hellos.Count);
            Assert.IsNull(hellos[0].Integrity);
            Assert.IsFalse(startProof.LastHelloProved);
        }

        [Test]
        public void ASecondStartWhileProvingAsksForNothingMore()
        {
            startProof.Start();

            Assert.IsFalse(startProof.Start());
            Assert.AreEqual(1, purposes.Count);
        }

        [Test]
        public void RedialsBeforeTheHelloIsAcceptedSendTheSameProof()
        {
            startProof.Start();
            pending[0](Proof("token-1"));

            Dial();

            Assert.AreEqual("token-1", hellos[1].Integrity.PlayIntegrityToken);
        }

        [Test]
        public void AnAcceptedHelloUsesTheProofUpSoReconnectsCarryNone()
        {
            startProof.Start();
            pending[0](Proof("token-1"));
            startProof.Read(Hello(HelloStatus.Accepted));

            Dial();

            Assert.IsNull(hellos[1].Integrity);
            Assert.IsFalse(startProof.LastHelloProved);
        }

        [TestCase(HelloStatus.UpdateRequired)]
        [TestCase(HelloStatus.Maintenance)]
        public void AHelloThatIsNotAcceptedKeepsTheProof(HelloStatus status)
        {
            startProof.Start();
            pending[0](Proof("token-1"));
            startProof.Read(Hello(status));

            Dial();

            Assert.AreEqual("token-1", hellos[1].Integrity.PlayIntegrityToken);
        }

        [Test]
        public void EveryNewSessionGetsAFreshProof()
        {
            startProof.Start();
            pending[0](Proof("token-1"));
            startProof.Read(Hello(HelloStatus.Accepted));

            startProof.Start();
            pending[1](Proof("token-2"));

            Assert.AreEqual(2, purposes.Count);
            Assert.AreEqual("token-2", hellos[1].Integrity.PlayIntegrityToken);
        }

        [Test]
        public void AnIntegrityFailureAfterAnUnprovedReconnectRetriesOnceWithAProof()
        {
            startProof.Start();
            pending[0](Proof("token-1"));
            startProof.Read(Hello(HelloStatus.Accepted));
            Dial();

            startProof.Read(Refused(ErrorCode.IntegrityFailed));
            pending[1](Proof("token-2"));

            CollectionAssert.AreEqual(new[] { IntegrityPurpose.SessionStart, IntegrityPurpose.SessionStart }, purposes);
            Assert.AreEqual(3, hellos.Count);
            Assert.AreEqual("token-2", hellos[2].Integrity.PlayIntegrityToken);
        }

        [Test]
        public void AnIntegrityFailureWithoutANewProofDoesNotRedial()
        {
            startProof.Start();
            pending[0](null);

            startProof.Read(Refused(ErrorCode.IntegrityFailed));
            pending[1](null);

            Assert.AreEqual(1, hellos.Count);
        }

        [Test]
        public void AnIntegrityFailureOfAProvedHelloIsTheServersAnswer()
        {
            startProof.Start();
            pending[0](Proof("token-1"));

            startProof.Read(Refused(ErrorCode.IntegrityFailed));

            Assert.AreEqual(1, purposes.Count);
            Assert.AreEqual(1, hellos.Count);
        }

        [TestCase(ErrorCode.Unauthenticated)]
        [TestCase(ErrorCode.IllegalIntent)]
        [TestCase(ErrorCode.UnsupportedProtocol)]
        public void OtherErrorsAskForNoProof(ErrorCode code)
        {
            startProof.Start();
            pending[0](null);

            startProof.Read(Refused(code));

            Assert.AreEqual(1, purposes.Count);
        }

        [Test]
        public void TheConnectionSendsTheProofInItsHello()
        {
            var socket = new LoopSocket();
            Connection connection = null;
            var proof = new SessionStartProof(RequestProof, () => connection.Start(new Uri("wss://play.test/ws")));
            connection = new Connection(() => socket, () =>
            {
                var hello = new ClientHello { SessionToken = "access-1" };
                proof.Attach(hello);
                return hello;
            }, () => 0d, new Random(1));

            proof.Start();
            pending[0](Proof("token-1"));

            Assert.AreEqual(ConnectionState.Connecting, connection.State);
            var sent = Envelope.Parser.ParseFrom(socket.Sent[0]);
            Assert.AreEqual("token-1", sent.ClientHello.Integrity.PlayIntegrityToken);
            Assert.AreEqual("access-1", sent.ClientHello.SessionToken);
        }

        sealed class LoopSocket : ISocketTransport
        {
            public List<byte[]> Sent { get; } = new();
            public TransportState State { get; private set; } = TransportState.Idle;

            public void Connect(Uri uri) => State = TransportState.Connecting;

            public void Send(ReadOnlySpan<byte> frame) => Sent.Add(frame.ToArray());

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                return false;
            }

            public void Dispose() => State = TransportState.Closed;
        }
    }
}
