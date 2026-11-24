using System;
using System.Collections.Generic;
using Google.Protobuf;
using HypeDomino.Game;
using HypeDomino.Game.Lobby;
using HypeDomino.Net;
using HypeDomino.UI.Localization;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchQueueTests
    {
        const ulong Nonce = 0x5EED_0074_0000_0001;
        const ulong JoinedMs = 1_800_000_000_000;

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

        FakeTransport transport;
        Session session;
        MatchQueue queue;

        [SetUp]
        public void Connect()
        {
            transport = new FakeTransport();
            session = new Session();
            queue = new MatchQueue();
        }

        void Accept()
        {
            session.Hello(transport, new ClientHello());
            transport.Deliver(Envelope.PayloadOneofCase.ServerHello, new ServerHello { Status = HelloStatus.Accepted, SessionNonce = Nonce });
            session.Poll();
            Assert.IsTrue(session.Accepted);
            transport.Sent.Clear();
        }

        Envelope SentAt(int index) => Envelope.Parser.ParseFrom(transport.Sent[index]);

        static QueueState Searching(Mode mode, QueueKind kind, ulong searchStartsMs) => new()
        {
            Status = QueueStatus.Searching,
            Mode = mode,
            Queue = kind,
            JoinedUnixMs = JoinedMs,
            SearchStartsUnixMs = searchStartsMs,
        };

        static QueueState Left() => new() { Status = QueueStatus.Left };

        static QueueState Found() => new() { Status = QueueStatus.MatchFound };

        static QueueState Refused(QueueRefusal refusal, ulong untilMs = 0) => new()
        {
            Status = QueueStatus.Refused,
            Mode = Mode._1V1,
            Queue = QueueKind.Ranked,
            Refusal = refusal,
            RefusedUntilUnixMs = untilMs,
        };

        static ServerClock ClockAt(ulong serverMs)
        {
            var clock = new ServerClock();
            clock.Sync(serverMs, 0, 0);
            return clock;
        }

        [Test]
        public void JoinSendsTheModeQueueAndProofAsAnUnsequencedIntent()
        {
            Accept();
            var proof = new IntegrityProof { PlayIntegrityToken = "token" };

            Assert.IsTrue(queue.TryJoin(session, Mode._2V2, QueueKind.Casual, proof));

            Assert.AreEqual(1, transport.Sent.Count);
            var envelope = SentAt(0);
            Assert.AreEqual(Envelope.PayloadOneofCase.JoinQueueIntent, envelope.PayloadCase);
            Assert.AreEqual(Mode._2V2, envelope.JoinQueueIntent.Mode);
            Assert.AreEqual(QueueKind.Casual, envelope.JoinQueueIntent.Queue);
            Assert.AreEqual("token", envelope.JoinQueueIntent.Integrity.PlayIntegrityToken);
            Assert.AreEqual(0ul, envelope.Sequence);
            Assert.AreEqual(Nonce, envelope.SessionNonce);
            Assert.IsTrue(queue.AwaitingReply);
            Assert.AreEqual(QueuePhase.Idle, queue.Phase);
        }

        [Test]
        public void NothingIsSentBeforeAnAcceptedHello()
        {
            Assert.IsFalse(queue.TryJoin(session, Mode._1V1, QueueKind.Ranked, null));
            Assert.IsFalse(queue.TryLeave(session));
            Assert.IsEmpty(transport.Sent);
            Assert.IsFalse(queue.AwaitingReply);
        }

        [TestCase(Mode.Unspecified, QueueKind.Ranked)]
        [TestCase(Mode._1V1, QueueKind.Unspecified)]
        [TestCase((Mode)7, QueueKind.Casual)]
        [TestCase(Mode._2V2, (QueueKind)9)]
        public void AModeOrQueueTheProtocolDoesNotNameIsNotSent(Mode mode, QueueKind kind)
        {
            Accept();

            Assert.IsFalse(queue.TryJoin(session, mode, kind, null));

            Assert.IsEmpty(transport.Sent);
        }

        [Test]
        public void ASecondJoinWaitsForTheServerReply()
        {
            Accept();
            Assert.IsTrue(queue.TryJoin(session, Mode._1V1, QueueKind.Ranked, null));

            Assert.IsFalse(queue.TryJoin(session, Mode._1V1, QueueKind.Ranked, null));
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked, JoinedMs));
            Assert.IsTrue(queue.TryJoin(session, Mode._2V2, QueueKind.Ranked, null));

            Assert.AreEqual(2, transport.Sent.Count);
            Assert.AreEqual(Mode._2V2, SentAt(1).JoinQueueIntent.Mode);
        }

        [Test]
        public void SearchingKeepsWhatTheServerSent()
        {
            var changes = 0;
            queue.Changed += () => changes++;

            queue.Read(Searching(Mode._2V2, QueueKind.Casual, JoinedMs + 20_000));

            Assert.AreEqual(QueuePhase.Searching, queue.Phase);
            Assert.AreEqual(Mode._2V2, queue.Mode);
            Assert.AreEqual(QueueKind.Casual, queue.Kind);
            Assert.AreEqual(JoinedMs, queue.JoinedUnixMs);
            Assert.AreEqual(JoinedMs + 20_000, queue.SearchStartsUnixMs);
            Assert.IsNull(queue.RefusalKey);
            Assert.IsTrue(queue.CanLeave);
            Assert.AreEqual(1, changes);
        }

        [Test]
        public void SearchTimesCountFromServerTime()
        {
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked, JoinedMs + 20_000));
            var clock = ClockAt(JoinedMs + 5_000);

            Assert.AreEqual(5d, queue.SecondsSearching(clock, 0), 1e-6);
            Assert.AreEqual(15d, queue.SecondsUntilSearch(clock, 0), 1e-6);
            Assert.AreEqual(8d, queue.SecondsSearching(clock, 3), 1e-6);
            Assert.AreEqual(0d, queue.SecondsUntilSearch(clock, 30), 1e-6);
            Assert.AreEqual(0d, queue.SecondsUntilBanEnds(clock, 0));
        }

        [Test]
        public void AnUnsyncedClockCountsNothing()
        {
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked, JoinedMs + 20_000));

            Assert.AreEqual(0d, queue.SecondsSearching(new ServerClock(), 100));
            Assert.AreEqual(0d, queue.SecondsUntilSearch(new ServerClock(), 0));
        }

        [Test]
        public void LeaveIsSentOnceWhileSearching()
        {
            Accept();
            Assert.IsFalse(queue.CanLeave);
            Assert.IsFalse(queue.TryLeave(session));
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked, JoinedMs));

            Assert.IsTrue(queue.TryLeave(session));
            Assert.IsFalse(queue.TryLeave(session));

            Assert.AreEqual(1, transport.Sent.Count);
            Assert.AreEqual(Envelope.PayloadOneofCase.LeaveQueueIntent, SentAt(0).PayloadCase);
            Assert.AreEqual(0ul, SentAt(0).Sequence);
            queue.Read(Left());
            Assert.AreEqual(QueuePhase.Idle, queue.Phase);
            Assert.AreEqual(Mode.Unspecified, queue.Mode);
            Assert.AreEqual(0ul, queue.JoinedUnixMs);
            Assert.IsFalse(queue.AwaitingReply);
            Assert.IsFalse(queue.CanLeave);
        }

        [Test]
        public void ALeaveBeforeTheJoinReplyEndsIdle()
        {
            Accept();
            Assert.IsTrue(queue.TryJoin(session, Mode._1V1, QueueKind.Casual, null));

            Assert.IsTrue(queue.TryLeave(session));
            queue.Read(Searching(Mode._1V1, QueueKind.Casual, JoinedMs));
            Assert.IsTrue(queue.AwaitingReply);
            Assert.IsFalse(queue.TryJoin(session, Mode._1V1, QueueKind.Casual, null));
            queue.Read(Left());

            Assert.AreEqual(QueuePhase.Idle, queue.Phase);
            Assert.IsFalse(queue.AwaitingReply);
            Assert.IsTrue(queue.TryJoin(session, Mode._1V1, QueueKind.Casual, null));
        }

        [TestCase(QueueRefusal.InMatch, "lobby.queue.refused.in_match")]
        [TestCase(QueueRefusal.MatchmakingBan, "lobby.queue.refused.matchmaking_ban")]
        [TestCase(QueueRefusal.RankedClosed, "lobby.queue.refused.ranked_closed")]
        [TestCase(QueueRefusal.Unavailable, "lobby.queue.refused.unavailable")]
        [TestCase(QueueRefusal.IntegrityFailed, "lobby.queue.refused.integrity_failed")]
        [TestCase(QueueRefusal.IntegrityUnavailable, "lobby.queue.refused.integrity_unavailable")]
        [TestCase(QueueRefusal.RankedBan, "lobby.queue.refused.ranked_ban")]
        [TestCase(QueueRefusal.Unspecified, "notice.generic")]
        [TestCase((QueueRefusal)42, "notice.generic")]
        public void ARefusalShowsTheKeyOfItsReason(QueueRefusal refusal, string key)
        {
            Accept();
            Assert.IsTrue(queue.TryJoin(session, Mode._1V1, QueueKind.Ranked, null));

            queue.Read(Refused(refusal));

            Assert.AreEqual(QueuePhase.Refused, queue.Phase);
            Assert.AreEqual(refusal, queue.Refusal);
            Assert.AreEqual(key, queue.RefusalKey);
            Assert.IsFalse(queue.AwaitingReply);
            Assert.IsFalse(queue.CanLeave);
            Assert.IsTrue(queue.TryJoin(session, Mode._1V1, QueueKind.Casual, null));
        }

        [Test]
        public void ABanCountsDownToTheEndTheServerSent()
        {
            queue.Read(Refused(QueueRefusal.MatchmakingBan, JoinedMs + 3_600_000));

            Assert.AreEqual(JoinedMs + 3_600_000, queue.RefusedUntilUnixMs);
            Assert.AreEqual(3_600d, queue.SecondsUntilBanEnds(ClockAt(JoinedMs), 0), 1e-6);
            Assert.AreEqual(0d, queue.SecondsUntilBanEnds(ClockAt(JoinedMs + 4_000_000), 0));
            Assert.AreEqual(0d, queue.SecondsSearching(ClockAt(JoinedMs), 0));
        }

        [TestCase(BanReason.Cheating, "ban.reason.cheating")]
        [TestCase(BanReason.Collusion, "ban.reason.collusion")]
        [TestCase(BanReason.AfkLeaving, "ban.reason.afk_leaving")]
        [TestCase(BanReason.OffensiveName, "ban.reason.offensive_name")]
        [TestCase(BanReason.PaymentFraud, "ban.reason.payment_fraud")]
        [TestCase(BanReason.Other, "ban.reason.other")]
        [TestCase((BanReason)42, "ban.reason.other")]
        public void ARankedBanShowsItsReasonAndCountsDownToItsEnd(BanReason reason, string reasonKey)
        {
            var state = Refused(QueueRefusal.RankedBan, JoinedMs + 7_200_000);
            state.BanReason = reason;

            queue.Read(state);

            Assert.AreEqual(QueueRefusalKeys.RankedBan, queue.RefusalKey);
            Assert.AreEqual(reason, queue.BanReason);
            Assert.AreEqual(reasonKey, queue.BanReasonKey);
            Assert.AreEqual(7_200d, queue.SecondsUntilBanEnds(ClockAt(JoinedMs), 0), 1e-6);
        }

        [Test]
        public void ARefusalWithoutABanReasonShowsNoReason()
        {
            queue.Read(Refused(QueueRefusal.RankedClosed));

            Assert.AreEqual(BanReason.Unspecified, queue.BanReason);
            Assert.IsNull(queue.BanReasonKey);
        }

        [TestCase(QueueRefusal.RankedBan)]
        [TestCase(QueueRefusal.MatchmakingBan)]
        public void ABanRefusalOffersItsAppealUntilTheQueueMovesOn(QueueRefusal refusal)
        {
            var state = Refused(refusal, JoinedMs + 60_000);
            state.Appeal = new AppealGrant { SanctionId = 42, Token = "appeal-1", ExpiresUnixMs = JoinedMs + 1_800_000 };

            queue.Read(state);

            Assert.IsTrue(queue.Appeal.Offered);
            Assert.AreEqual(42, queue.Appeal.SanctionId);
            Assert.AreEqual("appeal-1", queue.Appeal.Token);
            Assert.AreEqual(JoinedMs + 1_800_000, queue.Appeal.ExpiresUnixMs);

            queue.Read(Searching(Mode._1V1, QueueKind.Casual, JoinedMs));
            Assert.IsFalse(queue.Appeal.Offered);

            queue.Read(state);
            queue.Reset();
            Assert.IsFalse(queue.Appeal.Offered);
        }

        [Test]
        public void ARefusalWithoutAGrantOffersNoAppeal()
        {
            queue.Read(Refused(QueueRefusal.RankedBan, JoinedMs + 60_000));

            Assert.IsFalse(queue.Appeal.Offered);
        }

        [Test]
        public void ReadingABanRefusalWithItsAppealAllocatesNothing()
        {
            var refused = Refused(QueueRefusal.RankedBan, JoinedMs + 60_000);
            refused.Appeal = new AppealGrant { SanctionId = 42, Token = "appeal-1", ExpiresUnixMs = JoinedMs + 1_800_000 };
            var left = Left();
            TestDelegate step = () =>
            {
                queue.Read(refused);
                queue.Read(left);
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }

        [Test]
        public void TheBanReasonIsGoneOnceTheQueueMovesOn()
        {
            var state = Refused(QueueRefusal.MatchmakingBan, JoinedMs + 60_000);
            state.BanReason = BanReason.AfkLeaving;
            queue.Read(state);

            queue.Read(Searching(Mode._1V1, QueueKind.Casual, JoinedMs));

            Assert.AreEqual(BanReason.Unspecified, queue.BanReason);
            Assert.IsNull(queue.BanReasonKey);
        }

        [Test]
        public void ABanWithoutAnEndCountsNothing()
        {
            queue.Read(Refused(QueueRefusal.MatchmakingBan));

            Assert.AreEqual(0d, queue.SecondsUntilBanEnds(ClockAt(JoinedMs), 0));
            Assert.AreEqual(QueueRefusalKeys.MatchmakingBan, queue.RefusalKey);
        }

        [Test]
        public void AFoundMatchAnswersNoIntentAndHoldsTheQueue()
        {
            Accept();
            Assert.IsTrue(queue.TryJoin(session, Mode._1V1, QueueKind.Ranked, null));
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked, JoinedMs));
            Assert.IsTrue(queue.TryLeave(session));

            queue.Read(Found());

            Assert.AreEqual(QueuePhase.MatchFound, queue.Phase);
            Assert.AreEqual(Mode._1V1, queue.Mode);
            Assert.IsTrue(queue.AwaitingReply);
            queue.Read(Refused(QueueRefusal.Unavailable));
            Assert.IsFalse(queue.AwaitingReply);
            queue.Read(Found());
            Assert.IsFalse(queue.TryJoin(session, Mode._1V1, QueueKind.Ranked, null));
            Assert.IsFalse(queue.TryLeave(session));
        }

        [Test]
        public void AStatusThisClientDoesNotKnowAnswersAndChangesNothing()
        {
            Accept();
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked, JoinedMs));
            Assert.IsTrue(queue.TryLeave(session));
            var changes = 0;
            queue.Changed += () => changes++;

            queue.Read(new QueueState { Status = (QueueStatus)9 });

            Assert.AreEqual(QueuePhase.Searching, queue.Phase);
            Assert.IsFalse(queue.AwaitingReply);
            Assert.AreEqual(0, changes);
        }

        [Test]
        public void FollowingASessionReadsItsQueueStates()
        {
            var flow = new AppFlow();
            queue.Follow(session, flow);
            Accept();

            transport.Deliver(Envelope.PayloadOneofCase.QueueState, Searching(Mode._2V2, QueueKind.Ranked, JoinedMs));
            session.Poll();

            Assert.AreEqual(QueuePhase.Searching, queue.Phase);
            Assert.AreEqual(Mode._2V2, queue.Mode);
            Assert.AreEqual(JoinedMs, queue.JoinedUnixMs);
        }

        [Test]
        public void ANewHelloForgetsTheEntryTheClosedConnectionLost()
        {
            var flow = new AppFlow();
            queue.Follow(session, flow);
            Accept();
            Assert.IsTrue(queue.TryJoin(session, Mode._1V1, QueueKind.Ranked, null));
            queue.Read(Searching(Mode._1V1, QueueKind.Ranked, JoinedMs));
            Assert.IsTrue(queue.TryLeave(session));

            Accept();

            Assert.AreEqual(QueuePhase.Idle, queue.Phase);
            Assert.IsFalse(queue.AwaitingReply);
            Assert.IsFalse(queue.CanLeave);
        }

        [Test]
        public void ReturningToTheLobbyAfterAMatchClearsTheFoundMatch()
        {
            var flow = new AppFlow();
            queue.Follow(session, flow);
            flow.Hello(new ServerHello { Status = HelloStatus.Accepted });
            queue.Read(Found());
            flow.View(MatchState.Playing);
            Assert.AreEqual(QueuePhase.MatchFound, queue.Phase);

            flow.View(MatchState.Finished);
            Assert.IsTrue(flow.LeaveMatch());

            Assert.AreEqual(QueuePhase.Idle, queue.Phase);
        }

        [Test]
        public void ReadingAQueueStateAllocatesNothing()
        {
            var searching = Searching(Mode._1V1, QueueKind.Ranked, JoinedMs + 1_000);
            var left = Left();
            var clock = ClockAt(JoinedMs);
            var seconds = 0d;
            TestDelegate step = () =>
            {
                queue.Read(searching);
                seconds = queue.SecondsSearching(clock, 1) + queue.SecondsUntilSearch(clock, 1);
                queue.Read(left);
            };
            step();

            Assert.That(step, Allocates.Nothing);
            Assert.AreEqual(1d, seconds, 1e-6);
        }

        [TestCase(BanReason.Cheating, "ban.reason.cheating")]
        [TestCase(BanReason.Collusion, "ban.reason.collusion")]
        [TestCase(BanReason.AfkLeaving, "ban.reason.afk_leaving")]
        [TestCase(BanReason.OffensiveName, "ban.reason.offensive_name")]
        [TestCase(BanReason.PaymentFraud, "ban.reason.payment_fraud")]
        [TestCase(BanReason.Other, "ban.reason.other")]
        [TestCase((BanReason)42, "ban.reason.other")]
        public void ABanReasonNamesItsCategoryKey(BanReason reason, string key)
        {
            Assert.AreEqual(key, BanReasonKeys.Of(reason));
        }

        [Test]
        public void NoBanReasonHasNoKey()
        {
            Assert.IsNull(BanReasonKeys.Of(BanReason.Unspecified));
        }

        [Test]
        public void TheShippedBundleHasEveryBanReasonAndTheRankedBanRefusalInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            var keys = new List<string> { QueueRefusalKeys.RankedBan };
            foreach (BanReason reason in Enum.GetValues(typeof(BanReason)))
            {
                var key = BanReasonKeys.Of(reason);
                if (key != null)
                {
                    keys.Add(key);
                }
            }
            foreach (var key in keys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
