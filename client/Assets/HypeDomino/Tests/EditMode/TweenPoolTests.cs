using HypeDomino.UI.Board;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TweenPoolTests
    {
        const int Channels = 4;

        struct RecordingSink : ITweenSink
        {
            public Vector3[] Values;
            public int Applied;

            public void Apply(int target, TweenChannel channel, Vector3 value)
            {
                Values[target * Channels + (int)channel] = value;
                Applied++;
            }

            public Vector3 Of(int target, TweenChannel channel) => Values[target * Channels + (int)channel];
        }

        RecordingSink sink;

        [SetUp]
        public void CreateSink() => sink = new RecordingSink { Values = new Vector3[16 * Channels] };

        static Tween Move(int target, float startMs, float durationMs, bool cosmetic = false, TweenChannel channel = TweenChannel.Position) =>
            new Tween(target, channel, Vector3.zero, new Vector3(10f, 20f, 0f), startMs, durationMs, TokenEasing.Linear, cosmetic);

        [Test]
        public void ATweenMovesAlongItsCurveAndEndsOnItsTarget()
        {
            var pool = new TweenPool(4);
            pool.Start(Move(1, 100f, 200f), ref sink);

            pool.Step(100f, ref sink);
            Assert.AreEqual(Vector3.zero, sink.Of(1, TweenChannel.Position));
            pool.Step(200f, ref sink);
            Assert.AreEqual(5f, sink.Of(1, TweenChannel.Position).x, 1e-3f);
            Assert.AreEqual(10f, sink.Of(1, TweenChannel.Position).y, 1e-3f);
            pool.Step(300f, ref sink);
            Assert.AreEqual(new Vector3(10f, 20f, 0f), sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(0, pool.Count);
        }

        [Test]
        public void AFrameLateTweenStillLandsExactlyOnItsTarget()
        {
            var pool = new TweenPool(4);
            pool.Start(Move(1, 0f, 100f), ref sink);

            pool.Step(5000f, ref sink);

            Assert.AreEqual(new Vector3(10f, 20f, 0f), sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(0, pool.Count);
        }

        [TestCase(EffectLevel.Low, TokenEffectLevel.LowMaxTweens)]
        [TestCase(EffectLevel.Medium, TokenEffectLevel.MediumMaxTweens)]
        [TestCase(EffectLevel.High, TokenEffectLevel.HighMaxTweens)]
        public void TheCapacityComesFromTheEffectLevel(EffectLevel level, int capacity) => Assert.AreEqual(capacity, TweenPool.For(level).Capacity);

        [Test]
        public void AFullPoolFinishesItsOldestCosmeticTween()
        {
            var pool = new TweenPool(3);
            pool.Start(Move(1, 0f, 100f), ref sink);
            pool.Start(Move(2, 0f, 100f, cosmetic: true), ref sink);
            pool.Start(Move(3, 0f, 100f, cosmetic: true), ref sink);

            pool.Start(Move(4, 0f, 100f), ref sink);

            Assert.AreEqual(new Vector3(10f, 20f, 0f), sink.Of(2, TweenChannel.Position));
            Assert.IsFalse(pool.IsMoving(2, TweenChannel.Position));
            Assert.IsTrue(pool.IsMoving(1, TweenChannel.Position));
            Assert.IsTrue(pool.IsMoving(3, TweenChannel.Position));
            Assert.IsTrue(pool.IsMoving(4, TweenChannel.Position));
        }

        [Test]
        public void WithoutACosmeticTweenTheOldestFinishes()
        {
            var pool = new TweenPool(2);
            pool.Start(Move(1, 0f, 100f), ref sink);
            pool.Start(Move(2, 0f, 100f), ref sink);

            pool.Start(Move(3, 0f, 100f), ref sink);

            Assert.AreEqual(new Vector3(10f, 20f, 0f), sink.Of(1, TweenChannel.Position));
            Assert.IsFalse(pool.IsMoving(1, TweenChannel.Position));
            Assert.AreEqual(2, pool.Count);
        }

        [Test]
        public void APoolWithoutRoomJumpsTheNewTweenToItsEnd()
        {
            var pool = new TweenPool(0);

            pool.Start(Move(1, 0f, 100f), ref sink);

            Assert.AreEqual(new Vector3(10f, 20f, 0f), sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(0, pool.Count);
        }

        [Test]
        public void ANewTweenOnTheSameChannelReplacesTheOldOne()
        {
            var pool = new TweenPool(4);
            pool.Start(Move(1, 0f, 100f), ref sink);
            pool.Start(Move(1, 0f, 100f, channel: TweenChannel.Scale), ref sink);

            pool.Start(new Tween(1, TweenChannel.Position, Vector3.one, Vector3.one * 2f, 0f, 100f, TokenEasing.Linear, false), ref sink);
            pool.Step(100f, ref sink);

            Assert.AreEqual(2, sink.Applied);
            Assert.AreEqual(Vector3.one * 2f, sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(new Vector3(10f, 20f, 0f), sink.Of(1, TweenChannel.Scale));
        }

        [Test]
        public void FinishingAllLandsEveryTweenAndEmptiesThePool()
        {
            var pool = new TweenPool(4);
            pool.Start(Move(1, 0f, 100f), ref sink);
            pool.Start(Move(2, 0f, 900f, cosmetic: true), ref sink);

            pool.FinishAll(ref sink);

            Assert.AreEqual(new Vector3(10f, 20f, 0f), sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(new Vector3(10f, 20f, 0f), sink.Of(2, TweenChannel.Position));
            Assert.AreEqual(0, pool.Count);
        }

        [Test]
        public void FinishedTweensLeaveTheOthersRunningInOrder()
        {
            var pool = new TweenPool(3);
            pool.Start(Move(1, 0f, 300f), ref sink);
            pool.Start(Move(2, 0f, 100f), ref sink);
            pool.Start(Move(3, 0f, 300f, cosmetic: true), ref sink);

            pool.Step(150f, ref sink);
            pool.Start(Move(4, 150f, 100f), ref sink);
            pool.Start(Move(5, 150f, 100f), ref sink);

            Assert.IsFalse(pool.IsMoving(3, TweenChannel.Position));
            Assert.IsTrue(pool.IsMoving(1, TweenChannel.Position));
            Assert.AreEqual(3, pool.Count);
        }

        [Test]
        public void TileTravelUsesTheBaseDurationAndTheShortestTurn()
        {
            var pool = new TweenPool(4);

            sink.Values[1 * Channels + (int)TweenChannel.Scale] = Vector3.one;
            TileTravel.Start(pool, ref sink, 1, Vector3.zero, 350f, new Vector3(4f, 2f, 0f), 10f, 0f, false);
            pool.Step(TokenDurationMs.Base - 1f, ref sink);
            Assert.AreEqual(Vector3.one, sink.Of(1, TweenChannel.Scale));
            pool.Step(TokenDurationMs.Base, ref sink);

            Assert.AreEqual(new Vector3(4f, 2f, 0f), sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(370f, sink.Of(1, TweenChannel.Rotation).z, 1e-3f);
            Assert.AreEqual(1, pool.Count);
        }

        [Test]
        public void ThePlacedTileSettlesFromASquashAfterItLands()
        {
            var pool = new TweenPool(4);
            TileTravel.Start(pool, ref sink, 1, Vector3.zero, 0f, Vector3.one, 0f, 0f, false);

            pool.Step(TokenDurationMs.Base, ref sink);
            Assert.AreEqual(TokenMotion.PressScale, sink.Of(1, TweenChannel.Scale).x, 1e-5f);
            pool.Step(TokenDurationMs.Base + TokenDurationMs.Micro, ref sink);

            Assert.AreEqual(Vector3.one, sink.Of(1, TweenChannel.Scale));
            Assert.AreEqual(0, pool.Count);
            Assert.AreEqual(TokenEffect.TilePlaceDurationMs, TileTravel.PlaceMs(false));
        }

        [Test]
        public void ATweenThatHasNotStartedLeavesItsTargetAlone()
        {
            var pool = new TweenPool(4);
            pool.Start(Move(1, 100f, 100f), ref sink);

            pool.Step(50f, ref sink);

            Assert.AreEqual(0, sink.Applied);
            Assert.AreEqual(1, pool.Count);
        }

        [Test]
        public void DealtTilesArriveOneAfterAnother()
        {
            var pool = new TweenPool(8);
            for (var order = 0; order < 3; order++) TileTravel.Arrive(pool, ref sink, order, order, Vector3.zero, Vector3.one * (order + 1), 0f, false);

            pool.Step(2 * TokenDurationMs.Micro - 1f, ref sink);
            Assert.AreNotEqual(Vector3.zero, sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(Vector3.zero, sink.Of(2, TweenChannel.Position));
            pool.Step(TokenDurationMs.Base, ref sink);
            Assert.AreEqual(Vector3.one, sink.Of(0, TweenChannel.Position));
            Assert.AreNotEqual(Vector3.one * 2f, sink.Of(1, TweenChannel.Position));
            pool.Step(TokenDurationMs.Base + 2 * TokenDurationMs.Micro, ref sink);

            Assert.AreEqual(Vector3.one * 2f, sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(Vector3.one * 3f, sink.Of(2, TweenChannel.Position));
            Assert.AreEqual(0, pool.Count);
        }

        [Test]
        public void ADrawnTileArrivesAtOnceAndFastWithReduceMotion()
        {
            var pool = new TweenPool(4);
            TileTravel.Arrive(pool, ref sink, 1, 0, Vector3.zero, Vector3.one, 0f, true);

            pool.Step(TokenDurationMs.Fast, ref sink);

            Assert.AreEqual(Vector3.one, sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(0, pool.Count);
        }

        [Test]
        public void TileTravelWithReduceMotionIsFast()
        {
            var pool = new TweenPool(4);

            TileTravel.Start(pool, ref sink, 1, Vector3.zero, 0f, Vector3.one, 90f, 0f, true);
            pool.Step(TokenDurationMs.Fast, ref sink);

            Assert.AreEqual(TokenDurationMs.Fast, TileTravel.PlaceMs(true));
            Assert.AreEqual(Vector3.zero, sink.Of(1, TweenChannel.Scale));

            Assert.AreEqual(TokenDurationMs.Fast, TileTravel.DurationMs(true));
            Assert.AreEqual(TokenDurationMs.Base, TileTravel.DurationMs(false));
            Assert.AreEqual(Vector3.one, sink.Of(1, TweenChannel.Position));
            Assert.AreEqual(0, pool.Count);
        }

        [Test]
        public void StartingAndSteppingAllocateNothing()
        {
            var pool = TweenPool.For(EffectLevel.High);
            var now = 0f;

            TestDelegate frame = () =>
            {
                for (var tile = 0; tile < 8; tile++) TileTravel.Start(pool, ref sink, tile, Vector3.zero, 0f, Vector3.one, 90f, now, false);
                for (var tile = 8; tile < 16; tile++) TileTravel.Arrive(pool, ref sink, tile, tile - 8, Vector3.zero, Vector3.one, now, false);
                pool.Start(Move(0, now, 50f, cosmetic: true, channel: TweenChannel.Alpha), ref sink);
                pool.Step(now + 100f, ref sink);
                pool.FinishAll(ref sink);
                now += 16f;
            };
            frame();

            Assert.That(frame, Allocates.Nothing);
        }
    }
}
