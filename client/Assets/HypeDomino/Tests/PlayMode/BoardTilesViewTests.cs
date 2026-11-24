using System.Collections.Generic;
using HypeDomino.UI.Board;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class BoardTilesViewTests
    {
        const float Tolerance = 0.001f;
        const int FaceWidth = 8;
        static readonly Vector2Int ShadowCanvas = new Vector2Int(11, 19);
        static readonly Vector2 Board = new Vector2(1516f, 521f);

        GameObject root;
        Sprite[] faces;
        Image[] images;
        Sprite shadow;
        Image[] shadows;
        BoardTilesView view;

        [SetUp]
        public void Build()
        {
            root = new GameObject("Board", typeof(RectTransform), typeof(Canvas));
            var faceTexture = new Texture2D(FaceWidth, FaceWidth * 2);
            faces = new Sprite[TileFaces.SetSize];
            for (var i = 0; i < faces.Length; i++) faces[i] = Sprite.Create(faceTexture, new Rect(0, 0, FaceWidth, FaceWidth * 2), new Vector2(0.5f, 0.5f));
            shadow = Sprite.Create(new Texture2D(ShadowCanvas.x, ShadowCanvas.y), new Rect(0, 0, ShadowCanvas.x, ShadowCanvas.y), new Vector2(0.5f, 0.5f));
            var set = ScriptableObject.CreateInstance<TileFaceSet>();
            set.SetFaces(faces);
            shadows = NewImages("Shadow");
            images = NewImages("Tile");
            view = root.AddComponent<BoardTilesView>();
            view.Configure(set, images, shadow, shadows);
        }

        Image[] NewImages(string name)
        {
            var created = new Image[TileFaces.SetSize];
            for (var i = 0; i < created.Length; i++)
            {
                created[i] = new GameObject(name + i, typeof(RectTransform), typeof(Image)).GetComponent<Image>();
                created[i].transform.SetParent(root.transform, false);
            }
            return created;
        }

        [TearDown]
        public void DestroyBoard() => Object.Destroy(root);

        static Domino D(byte first, byte second) => new Domino(first, second);

        static ChainLayout Arrange(List<Domino> line, int anchor)
        {
            var layout = new ChainLayout(TileFaces.SetSize);
            layout.Arrange(line, anchor, new List<Domino>(), new List<Domino>(), Board, float.MaxValue);
            return layout;
        }

        static ChainLayout FullBoard()
        {
            var all = new List<Domino>();
            for (byte low = 0; low <= TileFaces.HighestValue; low++)
            for (var high = low; high <= TileFaces.HighestValue; high++)
                all.Add(D(low, high));
            var layout = new ChainLayout(TileFaces.SetSize);
            var line = all.GetRange(0, 20);
            line[10] = D(6, 6);
            layout.Arrange(line, 10, all.GetRange(20, 4), all.GetRange(24, 4), Board, float.MaxValue);
            return layout;
        }

        [Test]
        public void EveryLaidTileShowsItsFaceUprightOnItsSlot()
        {
            var layout = Arrange(new List<Domino> { D(1, 2), D(2, 2), D(2, 5) }, 1);

            view.Show(layout);

            Assert.AreEqual(3, view.Count);
            for (var i = 0; i < layout.Count; i++)
            {
                var tile = layout[i];
                var image = images[i];
                Assert.IsTrue(image.enabled, $"tile {i}");
                Assert.AreSame(faces[TileFaces.Index(D(tile.Near, tile.Far))], image.sprite, $"tile {i}");
                Assert.AreEqual(layout.Scale, image.rectTransform.sizeDelta.x, Tolerance, $"tile {i}");
                Assert.AreEqual(layout.Scale * 2f, image.rectTransform.sizeDelta.y, Tolerance, $"tile {i}");
                Assert.Less((layout.OnBoard(tile.Center) - image.rectTransform.anchoredPosition).magnitude, Tolerance, $"tile {i}");
                Assert.AreEqual(0f, Mathf.DeltaAngle(TileFaces.Angle(tile), image.rectTransform.localEulerAngles.z), Tolerance, $"tile {i}");
                Assert.IsFalse(image.raycastTarget, $"tile {i}");
            }
            for (var i = layout.Count; i < images.Length; i++) Assert.IsFalse(images[i].enabled, $"slot {i}");
        }

        [Test]
        public void EveryLaidTileCastsTheSharedShadowStraightDownByTheRestingElevation()
        {
            var layout = Arrange(new List<Domino> { D(1, 2), D(2, 2), D(2, 5) }, 1);
            var tint = new Color32(TokenColor.Shadow.r, TokenColor.Shadow.g, TokenColor.Shadow.b, (byte)Mathf.RoundToInt(TokenElevation.E1Alpha * 255f));
            var scale = layout.Scale / FaceWidth;

            view.Show(layout);

            for (var i = 0; i < layout.Count; i++)
            {
                var cast = shadows[i];
                var tile = images[i].rectTransform;
                Assert.IsTrue(cast.enabled, $"shadow {i}");
                Assert.AreSame(shadow, cast.sprite, $"shadow {i}");
                Assert.AreEqual(tint, (Color32)cast.color, $"shadow {i}");
                Assert.IsFalse(cast.raycastTarget, $"shadow {i}");
                Assert.AreEqual(tile.anchoredPosition.x, cast.rectTransform.anchoredPosition.x, Tolerance, $"shadow {i}");
                Assert.AreEqual(tile.anchoredPosition.y - TokenElevation.E1OffsetY, cast.rectTransform.anchoredPosition.y, Tolerance, $"shadow {i}");
                Assert.AreEqual(ShadowCanvas.x * scale, cast.rectTransform.sizeDelta.x, Tolerance, $"shadow {i}");
                Assert.AreEqual(ShadowCanvas.y * scale, cast.rectTransform.sizeDelta.y, Tolerance, $"shadow {i}");
                Assert.AreEqual(0f, Mathf.DeltaAngle(tile.localEulerAngles.z, cast.rectTransform.localEulerAngles.z), Tolerance, $"shadow {i}");
            }
            for (var i = layout.Count; i < shadows.Length; i++) Assert.IsFalse(shadows[i].enabled, $"shadow slot {i}");
        }

        [Test]
        public void ThePoolNeedsOneShadowPerTile()
        {
            var set = ScriptableObject.CreateInstance<TileFaceSet>();
            set.SetFaces(faces);

            Assert.Throws<System.ArgumentException>(() => view.Configure(set, images, shadow, new Image[images.Length - 1]));
        }

        [Test]
        public void AShorterChainHidesTheTilesItNoLongerHas()
        {
            view.Show(FullBoard());
            Assert.AreEqual(TileFaces.SetSize, view.Count);

            view.Show(Arrange(new List<Domino> { D(3, 3) }, 0));

            Assert.AreEqual(1, view.Count);
            Assert.AreSame(faces[TileFaces.Index(D(3, 3))], images[0].sprite);
            for (var i = 1; i < images.Length; i++)
            {
                Assert.IsFalse(images[i].enabled, $"slot {i}");
                Assert.IsFalse(shadows[i].enabled, $"shadow slot {i}");
            }
            Assert.IsTrue(shadows[0].enabled);
        }

        [Test]
        public void ShowingTheBoardAllocatesNothing()
        {
            var full = FullBoard();
            var single = Arrange(new List<Domino> { D(0, 4) }, 0);
            view.Show(full);
            view.Show(single);

            Assert.That(() =>
            {
                view.Show(full);
                view.Show(single);
            }, Allocates.Nothing);
        }
    }
}
