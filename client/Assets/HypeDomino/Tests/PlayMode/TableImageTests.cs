using HypeDomino.UI.Cosmetics;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class TableImageTests
    {
        const float Tolerance = 0.01f;
        static readonly Vector4 Slice = new(20f, 8f, 20f, 8f);

        GameObject root;
        Texture2D texture;
        Sprite first;
        Sprite second;
        TableImage image;
        MeshProbe probe;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Screen", typeof(RectTransform), typeof(Canvas));
            texture = new Texture2D(64, 36, TextureFormat.RGBA32, false);
            first = Sprite.Create(texture, new Rect(0f, 0f, 64f, 36f), Vector2.zero, 100f, 0, SpriteMeshType.FullRect, Slice);
            second = Sprite.Create(texture, new Rect(0f, 0f, 64f, 36f), Vector2.zero, 100f, 0, SpriteMeshType.FullRect, Slice);
            image = new GameObject("Table", typeof(RectTransform)).AddComponent<TableImage>();
            image.transform.SetParent(root.transform, false);
            probe = image.gameObject.AddComponent<MeshProbe>();
        }

        [TearDown]
        public void DestroyScreen()
        {
            Object.Destroy(root);
            Object.Destroy(first);
            Object.Destroy(second);
            Object.Destroy(texture);
        }

        void Draw(Vector2 size, Sprite sprite)
        {
            image.rectTransform.sizeDelta = size;
            image.sprite = sprite;
            image.Rebuild(CanvasUpdate.PreRender);
        }

        [TestCase(2400f, 1080f)]
        [TestCase(1920f, 1080f)]
        [TestCase(1440f, 1080f)]
        [TestCase(1080f, 2400f)]
        public void EveryQuadIsDrawnInsideTheRectFromInsideTheSprite(float width, float height)
        {
            Draw(new Vector2(width, height), first);

            Assert.AreEqual(TableCloth.MaxQuads, image.QuadCount);
            Assert.AreEqual(TableCloth.MaxQuads * 6, probe.Vertices.Count);
            foreach (var vertex in probe.Vertices)
            {
                Assert.That(Mathf.Abs(vertex.position.x), Is.LessThanOrEqualTo(width / 2f + Tolerance));
                Assert.That(Mathf.Abs(vertex.position.y), Is.LessThanOrEqualTo(height / 2f + Tolerance));
                Assert.That(vertex.uv0.x, Is.InRange(-Tolerance, 1f + Tolerance));
                Assert.That(vertex.uv0.y, Is.InRange(-Tolerance, 1f + Tolerance));
            }
        }

        [Test]
        public void ATallRectTurnsTheTableAndAWideOneDoesNot()
        {
            Draw(new Vector2(1080f, 2400f), first);
            Assert.IsTrue(image.IsTurned);

            Draw(new Vector2(2400f, 1080f), first);
            Assert.IsFalse(image.IsTurned);
        }

        [Test]
        public void TheFieldFadeIsInTheVertexAlpha()
        {
            image.color = new Color(1f, 1f, 1f, 1f);
            Draw(new Vector2(2400f, 1080f), first);

            var transparent = 0;
            foreach (var vertex in probe.Vertices)
                if (vertex.color.a == 0) transparent++;
            Assert.That(transparent, Is.GreaterThan(0));
        }

        [Test]
        public void WithoutASpriteTheImageDrawsItsPlainRect()
        {
            Draw(new Vector2(2400f, 1080f), null);

            Assert.AreEqual(0, image.QuadCount);
            Assert.AreEqual(6, probe.Vertices.Count);
        }

        [Test]
        public void ChangingTheFrameAllocatesNothing()
        {
            Object.DestroyImmediate(probe);
            Draw(new Vector2(2400f, 1080f), first);
            Draw(new Vector2(2400f, 1080f), second);
            Draw(new Vector2(2400f, 1080f), first);

            Assert.That(() =>
            {
                image.sprite = second;
                image.Rebuild(CanvasUpdate.PreRender);
                image.sprite = first;
                image.Rebuild(CanvasUpdate.PreRender);
            }, Allocates.Nothing);
        }
    }
}
