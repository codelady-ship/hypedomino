using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game.Lobby;
using HypeDomino.Game.Training;
using HypeDomino.Net;
using HypeDomino.Net.Integrity;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LobbyTrainingTests
    {
        const ulong Nonce = 0x5EED_0074_0000_0003;
        const int Ranked1V1 = 0;
        const int Casual2V2 = 3;

        sealed class FakeTransport : IFrameTransport
        {
            readonly Queue<byte[]> incoming = new();

            public List<byte[]> Sent { get; } = new();

            public void Send(ReadOnlySpan<byte> frame) => Sent.Add(frame.ToArray());

            public bool TryReceive(out ReadOnlySpan<byte> frame)
            {
                frame = default;
                if (incoming.Count == 0) return false;
                frame = incoming.Dequeue();
                return true;
            }

            public void Deliver(Envelope.PayloadOneofCase payload, IMessage message) =>
                incoming.Enqueue(new FrameWriter(Session.MaxSentFrameBytes).Write(new FrameHeader(Session.ProtocolVersion, 0, 0, Nonce), payload, message).ToArray());
        }

        sealed class ProofDesk
        {
            public readonly List<Action<IntegrityProof>> Answers = new();
            public bool AnswerAtOnce;

            public void Request(IntegrityPurpose purpose, Action<IntegrityProof> done)
            {
                if (AnswerAtOnce) done(null);
                else Answers.Add(done);
            }
        }

        sealed class BotGame : ITrainingGame
        {
            public int Starts;
            public bool Opens = true;

            public bool TryStart()
            {
                Starts++;
                return Opens;
            }
        }

        FakeTransport transport;
        Session session;
        MatchQueue queue;
        ProofDesk proofs;
        LobbyPlay play;
        BotGame game;
        LobbyTraining training;
        PlayCardState shown;

        [SetUp]
        public void Build()
        {
            transport = new FakeTransport();
            session = new Session();
            queue = new MatchQueue();
            proofs = new ProofDesk();
            play = new LobbyPlay(queue, proofs.Request);
            game = new BotGame();
            training = new LobbyTraining(queue, play, game);
            session.Hello(transport, new ClientHello());
            transport.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = Nonce });
            session.Poll();
            Assert.IsTrue(session.Accepted);
            transport.Sent.Clear();
        }

        void Search()
        {
            proofs.AnswerAtOnce = true;
            Assert.IsTrue(play.Play(session, Ranked1V1));
            queue.Read(new QueueState { Status = QueueStatus.Searching, Mode = Mode._1V1, Queue = QueueKind.Ranked });
            transport.Sent.Clear();
        }

        void AssertQueueCardsReady()
        {
            for (var card = 0; card < PlayModes.Count; card++) Assert.AreEqual(PlayCardState.Ready, play.StateOf(card), $"card {card}");
        }

        [Test]
        public void TrainingOpensTheBotGameAndNeverEntersTheQueue()
        {
            Assert.AreEqual(PlayCardState.Ready, training.State);

            Assert.IsTrue(training.Start());

            Assert.AreEqual(1, game.Starts);
            Assert.IsEmpty(transport.Sent);
            Assert.AreEqual(QueuePhase.Idle, queue.Phase);
            Assert.IsFalse(queue.AwaitingReply);
            AssertQueueCardsReady();
        }

        [Test]
        public void ABotGameThatDoesNotOpenIsNotStarted()
        {
            game.Opens = false;

            Assert.IsFalse(training.Start());

            Assert.AreEqual(1, game.Starts);
            Assert.IsEmpty(transport.Sent);
        }

        [Test]
        public void ARunningSearchLocksTrainingAndIsNeverLeftByIt()
        {
            Search();

            Assert.AreEqual(PlayCardState.Locked, training.State);
            Assert.IsFalse(training.Start());

            Assert.AreEqual(0, game.Starts);
            Assert.IsEmpty(transport.Sent);
            Assert.AreEqual(QueuePhase.Searching, queue.Phase);
        }

        [Test]
        public void TrainingIsFreeAgainOnceTheSearchIsLeft()
        {
            Search();
            Assert.IsTrue(play.Cancel(session));
            Assert.AreEqual(PlayCardState.Locked, training.State);

            queue.Read(new QueueState { Status = QueueStatus.Left });

            Assert.AreEqual(PlayCardState.Ready, training.State);
            Assert.IsTrue(training.Start());
            Assert.AreEqual(1, game.Starts);
        }

        [Test]
        public void ACardBeingProvedOrJoinedLocksTraining()
        {
            Assert.IsTrue(play.Play(session, Casual2V2));
            Assert.AreEqual(PlayCardState.Locked, training.State);
            Assert.IsFalse(training.Start());

            proofs.Answers[0](null);
            Assert.IsTrue(queue.AwaitingReply);
            Assert.AreEqual(PlayCardState.Locked, training.State);
            Assert.IsFalse(training.Start());

            Assert.AreEqual(0, game.Starts);
            Assert.AreEqual(1, transport.Sent.Count);
        }

        [Test]
        public void AFoundMatchLocksTrainingUntilTheLobbyIsShownAgain()
        {
            Search();

            queue.Read(new QueueState { Status = QueueStatus.MatchFound });

            Assert.AreEqual(PlayCardState.Locked, training.State);
            Assert.IsFalse(training.Start());
            queue.Reset();
            Assert.AreEqual(PlayCardState.Ready, training.State);
        }

        [Test]
        public void AQueueRefusalDoesNotLockTraining()
        {
            proofs.AnswerAtOnce = true;
            Assert.IsTrue(play.Play(session, Ranked1V1));

            queue.Read(new QueueState { Status = QueueStatus.Refused, Mode = Mode._1V1, Queue = QueueKind.Ranked, Refusal = QueueRefusal.RankedClosed });

            Assert.AreEqual(PlayCardState.Ready, training.State);
            Assert.IsTrue(training.Start());
        }

        [Test]
        public void ReadingTheTrainingCardAllocatesNothing()
        {
            Search();
            TestDelegate read = ReadState;
            read();

            Assert.That(read, Allocates.Nothing);
            Assert.AreEqual(PlayCardState.Locked, shown);
        }

        void ReadState() => shown = training.State;
    }
}
