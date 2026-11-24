using HypeDomino.UI.Emoji;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using Unity.Profiling;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class SeatEmojiLayerTests
    {
        const float Duration = TokenEffect.EmojiDurationMs / 1000f;
        const int MarkedTicks = 3;
        const int RecordedSamplesCap = 64;

        GameObject root;
        SeatEmojiLayer layer;
        RectTransform[] seats;
        Sprite still;
        Sprite frame;
        LoopArt art;

        [SetUp]
        public void BuildLayer()
        {
            root = new GameObject("Hud", typeof(RectTransform));
            var prefab = new GameObject("Emoji", typeof(RectTransform), typeof(Image)).AddComponent<EmojiPopup>();
            prefab.transform.SetParent(root.transform);
            seats = new RectTransform[4];
            for (var i = 0; i < seats.Length; i++)
            {
                seats[i] = new GameObject("Seat" + i, typeof(RectTransform)).GetComponent<RectTransform>();
                seats[i].SetParent(root.transform);
                seats[i].position = new Vector3(100f * i, 50f, 0f);
            }
            layer = new GameObject("Emoji layer", typeof(RectTransform)).AddComponent<SeatEmojiLayer>();
            layer.transform.SetParent(root.transform);
            layer.Build(prefab, seats);
            still = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);
            frame = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 2, 2), Vector2.zero);
            art = new LoopArt(still, new[] { frame }, 12f);
        }

        [TearDown]
        public void DestroyLayer() => Object.Destroy(root);

        [Test]
        public void BuildPrewarmsThePooledEmoji()
        {
            Assert.AreEqual(TokenEffect.EmojiPooled, layer.PooledCount);
            Assert.AreEqual(0, layer.LiveCount);
        }

        [Test]
        public void EachSeatShowsOneEmojiThatANewOneReplaces()
        {
            layer.Show(1, art, 10f);
            var first = layer.LiveAt(1);
            Assert.AreEqual(seats[1].position, first.transform.position);
            layer.Tick(10.5f);
            layer.Show(1, art, 10.5f);
            Assert.AreSame(first, layer.LiveAt(1));
            Assert.AreEqual(0f, first.transform.localScale.x, "a replaced emoji starts its pop again");
            layer.Show(3, art, 10.5f);
            Assert.AreEqual(2, layer.LiveCount);

            layer.Tick(10.5f + Duration - 0.01f);
            Assert.AreEqual(2, layer.LiveCount);
            layer.Tick(10.5f + Duration);
            Assert.AreEqual(0, layer.LiveCount);
            Assert.IsNull(layer.LiveAt(1));
            Assert.AreEqual(TokenEffect.EmojiPooled, layer.PooledCount);
        }

        [Test]
        public void TheLowLevelShowsStaticArt()
        {
            layer.Configure(EffectLevel.Low, false);
            layer.Show(0, art, 0f);
            Assert.AreSame(still, layer.LiveAt(0).Shown);
            layer.Configure(EffectLevel.Medium, false);
            layer.Show(2, art, 0f);
            Assert.AreSame(frame, layer.LiveAt(2).Shown);
        }

        [Test]
        public void ReduceMotionFadesInAtFullSize()
        {
            layer.Configure(EffectLevel.High, true);
            layer.Show(0, art, 0f);
            Assert.AreEqual(1f, layer.LiveAt(0).transform.localScale.x);
            Assert.AreEqual(0f, layer.LiveAt(0).Alpha);
        }

        [Test]
        public void ShowingAndExpiringEmojiDoesNotAllocate()
        {
            var now = 0f;
            for (var seat = 0; seat < seats.Length; seat++) layer.Show(seat, art, now);
            layer.Tick(now += Duration);
            Assert.That(() =>
            {
                for (var cycle = 0; cycle < 200; cycle++)
                {
                    for (var seat = 0; seat < seats.Length; seat++) layer.Show(seat, art, now);
                    layer.Tick(now += 0.1f);
                    layer.Tick(now += Duration);
                }
            }, Allocates.Nothing);
        }

        [Test]
        public void EveryTickIsOneSampleOfTheEmojiEffectMarker()
        {
            layer.Show(0, art, 0f);
            layer.Tick(0f);
            var recorder = new ProfilerRecorder(SeatEmojiLayer.ProfilerMarkerName, RecordedSamplesCap, ProfilerRecorderOptions.CollectOnlyOnCurrentThread);
            try
            {
                Assert.IsTrue(recorder.Valid);
                recorder.Start();
                for (var tick = 1; tick <= MarkedTicks; tick++) layer.Tick(tick * Duration / MarkedTicks);
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
