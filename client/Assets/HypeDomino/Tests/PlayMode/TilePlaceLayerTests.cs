using HypeDomino.UI.Board;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using Unity.Profiling;
using UnityEngine;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class TilePlaceLayerTests
    {
        const float PlaceMs = TokenDurationMs.Base + TokenDurationMs.Micro;
        const int MarkedTicks = 3;
        const int RecordedSamplesCap = 64;

        static readonly Vector3 Hand = new Vector3(0f, -500f, 0f);
        static readonly Vector3 Slot = new Vector3(200f, 40f, 0f);

        GameObject root;
        TilePlaceLayer layer;
        Transform[] tiles;

        [SetUp]
        public void BuildLayer()
        {
            root = new GameObject("Board");
            tiles = new Transform[4];
            for (var i = 0; i < tiles.Length; i++)
            {
                tiles[i] = new GameObject("Tile" + i).transform;
                tiles[i].SetParent(root.transform, false);
            }
            layer = root.AddComponent<TilePlaceLayer>();
            layer.Build(tiles);
        }

        [TearDown]
        public void DestroyLayer() => Object.Destroy(root);

        [Test]
        public void APlacedTileTravelsTurnsAndSettlesOnItsSlot()
        {
            layer.Place(2, Hand, 0f, Slot, 90f, 0f);

            Assert.AreEqual(Hand, tiles[2].localPosition);
            layer.Tick(TokenDurationMs.Base);
            Assert.AreEqual(Slot, tiles[2].localPosition);
            Assert.AreEqual(90f, tiles[2].localEulerAngles.z, 1e-3f);
            Assert.AreEqual(TokenMotion.PressScale, tiles[2].localScale.x, 1e-3f);
            layer.Tick(PlaceMs);
            Assert.AreEqual(Vector3.one, tiles[2].localScale);
            Assert.AreEqual(0, layer.MovingCount);
        }

        [Test]
        public void ReduceMotionLandsInFastWithoutTheSettle()
        {
            layer.Configure(EffectLevel.Medium, true);
            layer.Place(1, Hand, 0f, Slot, -90f, 0f);

            layer.Tick(TokenDurationMs.Fast);

            Assert.AreEqual(Slot, tiles[1].localPosition);
            Assert.AreEqual(Vector3.one, tiles[1].localScale);
            Assert.AreEqual(0, layer.MovingCount);
        }

        [TestCase(EffectLevel.Low, TokenEffectLevel.LowMaxTweens)]
        [TestCase(EffectLevel.Medium, TokenEffectLevel.MediumMaxTweens)]
        [TestCase(EffectLevel.High, TokenEffectLevel.HighMaxTweens)]
        public void ThePoolFollowsTheEffectLevel(EffectLevel level, int capacity)
        {
            layer.Configure(level, false);
            Assert.AreEqual(capacity, layer.Capacity);
        }

        [Test]
        public void ChangingTheEffectLevelLandsTheMovingTiles()
        {
            layer.Place(0, Hand, 0f, Slot, 90f, 0f);

            layer.Configure(EffectLevel.Low, false);

            Assert.AreEqual(Slot, tiles[0].localPosition);
            Assert.AreEqual(0, layer.MovingCount);
        }

        [Test]
        public void PlacingAndTickingDoesNotAllocate()
        {
            var now = 0f;
            layer.Place(0, Hand, 0f, Slot, 90f, now);
            layer.Tick(now += PlaceMs);
            Assert.That(() =>
            {
                for (var turn = 0; turn < 200; turn++)
                {
                    layer.Place(turn % tiles.Length, Hand, 0f, Slot, 90f, now);
                    layer.Tick(now += 16f);
                    layer.Tick(now += PlaceMs);
                }
            }, Allocates.Nothing);
        }

        [Test]
        public void EveryTickIsOneSampleOfTheTilePlaceEffectMarker()
        {
            layer.Place(0, Hand, 0f, Slot, 90f, 0f);
            layer.Tick(0f);
            var recorder = new ProfilerRecorder(TilePlaceLayer.ProfilerMarkerName, RecordedSamplesCap, ProfilerRecorderOptions.CollectOnlyOnCurrentThread);
            try
            {
                Assert.IsTrue(recorder.Valid);
                recorder.Start();
                for (var tick = 1; tick <= MarkedTicks; tick++) layer.Tick(tick * PlaceMs / MarkedTicks);
                recorder.Stop();
                Assert.AreEqual(MarkedTicks, recorder.Count);
            }
            finally
            {
                recorder.Dispose();
            }
        }
    }
}
