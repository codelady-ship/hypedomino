using HypeDomino.UI.Board;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TilePlaceQueueTests
    {
        const int Channels = 4;
        const float PlaceMs = TokenDurationMs.Base + TokenDurationMs.Micro;
        const float ReducedPlaceMs = TokenDurationMs.Fast;

        struct RecordingSink : ITweenSink
        {
            public Vector3[] Values;

            public void Apply(int target, TweenChannel channel, Vector3 value) => Values[target * Channels + (int)channel] = value;

            public Vector3 Of(int target, TweenChannel channel) => Values[target * Channels + (int)channel];
        }

        static readonly Vector3 Hand = new Vector3(0f, -500f, 0f);

        RecordingSink sink;
        TweenPool pool;
        TilePlaceQueue queue;

        [SetUp]
        public void CreateQueue()
        {
            sink = new RecordingSink { Values = new Vector3[16 * Channels] };
            pool = TweenPool.For(EffectLevel.Medium);
            queue = default;
        }

        static Vector3 Slot(int tile) => new Vector3(tile * 100f, 0f, 0f);

        void Place(int tile, float nowMs, bool reduceMotion = false) => queue.Place(pool, ref sink, tile, Hand, 0f, Slot(tile), 90f, nowMs, reduceMotion);

        [Test]
        public void APlacementOnAnIdleBoardStartsAtOnceAndSettlesOnItsSlot()
        {
            Place(1, 1000f);

            Assert.AreEqual(1000f + PlaceMs, queue.SettledAtMs);
            Assert.AreEqual(Hand, sink.Of(1, TweenChannel.Position));
            pool.Step(1000f + TokenDurationMs.Base / 2f, ref sink);
            Assert.AreNotEqual(Hand, sink.Of(1, TweenChannel.Position));
            pool.Step(1000f + PlaceMs, ref sink);
            Assert.AreEqual(Slot(1), sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(90f, sink.Of(1, TweenChannel.Rotation).z, 1e-3f);
            Assert.AreEqual(Vector3.one, sink.Of(1, TweenChannel.Scale));
            Assert.AreEqual(0, pool.Count);
        }

        [Test]
        public void APlacementArrivingWhileAnotherMovesWaitsAtItsSourceUntilThatOneSettles()
        {
            Place(1, 0f);
            Place(2, 100f);

            Assert.AreEqual(2f * PlaceMs, queue.SettledAtMs);
            pool.Step(PlaceMs - 1f, ref sink);
            Assert.AreEqual(Hand, sink.Of(2, TweenChannel.Position));
            pool.Step(PlaceMs + TokenDurationMs.Base / 2f, ref sink);
            Assert.AreEqual(Slot(1), sink.Of(1, TweenChannel.Position));
            Assert.AreNotEqual(Hand, sink.Of(2, TweenChannel.Position));
            pool.Step(2f * PlaceMs, ref sink);
            Assert.AreEqual(Slot(2), sink.Of(2, TweenChannel.Position));
        }

        [Test]
        public void ViewsArrivingTogetherPlayOneAfterAnotherWithinTheLag()
        {
            Place(1, 0f);
            Place(2, 0f);

            Assert.AreEqual(2f * PlaceMs, queue.SettledAtMs);
            Assert.LessOrEqual(queue.SettledAtMs, TokenMotion.MaxPresentationLagMs);
            Assert.IsTrue(pool.IsMoving(2, TweenChannel.Position));
        }

        [Test]
        public void APlacementThatWouldLagTooFarJumpsEveryTileToItsFinalState()
        {
            Place(1, 0f);
            Place(2, 0f);
            Place(3, 0f);

            Assert.AreEqual(0, pool.Count);
            Assert.AreEqual(0f, queue.SettledAtMs);
            for (var tile = 1; tile <= 3; tile++)
            {
                Assert.AreEqual(Slot(tile), sink.Of(tile, TweenChannel.Position));
                Assert.AreEqual(90f, sink.Of(tile, TweenChannel.Rotation).z, 1e-3f);
                Assert.AreEqual(Vector3.one, sink.Of(tile, TweenChannel.Scale));
            }
        }

        [Test]
        public void AfterAJumpTheNextPlacementMovesAgain()
        {
            Place(1, 0f);
            Place(2, 0f);
            Place(3, 0f);

            Place(4, 10f);

            Assert.AreEqual(10f + PlaceMs, queue.SettledAtMs);
            Assert.IsTrue(pool.IsMoving(4, TweenChannel.Position));
        }

        [Test]
        public void ReduceMotionPlacesInFastSoMorePlacementsFitTheLag()
        {
            var fitting = (int)(TokenMotion.MaxPresentationLagMs / ReducedPlaceMs);
            for (var tile = 1; tile <= fitting; tile++) Place(tile, 0f, true);

            Assert.AreEqual(fitting * ReducedPlaceMs, queue.SettledAtMs);
            Assert.IsFalse(pool.IsMoving(1, TweenChannel.Scale));
            Place(fitting + 1, 0f, true);
            Assert.AreEqual(0, pool.Count);
            Assert.AreEqual(Slot(fitting + 1), sink.Of(fitting + 1, TweenChannel.Position));
        }

        [Test]
        public void PlacingAndSteppingDoesNotAllocate()
        {
            var now = 0f;
            Place(1, now);
            pool.Step(now += PlaceMs, ref sink);
            Assert.That(() =>
            {
                for (var turn = 0; turn < 200; turn++)
                {
                    Place(1 + turn % 3, now);
                    Place(4 + turn % 3, now);
                    Place(7, now);
                    pool.Step(now += 16f, ref sink);
                    pool.Step(now += PlaceMs, ref sink);
                }
            }, Allocates.Nothing);
        }
    }
}
