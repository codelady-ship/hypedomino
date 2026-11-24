using System;
using System.Collections.Generic;
using HypeDomino.UI.Board;
using NUnit.Framework;
using UnityEngine;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TileFacesTests
    {
        const float Tolerance = 0.001f;

        readonly List<Object> created = new List<Object>();

        [TearDown]
        public void DestroyCreated()
        {
            foreach (var item in created) Object.DestroyImmediate(item);
            created.Clear();
        }

        static Domino D(byte first, byte second) => new Domino(first, second);

        static Vector2 Top(float angle) => Quaternion.Euler(0f, 0f, angle) * Vector3.up;

        static void AssertSameDirection(Vector2 expected, Vector2 actual, string message) =>
            Assert.Less((expected.normalized - actual).magnitude, Tolerance, $"{message}: expected {expected}, got {actual}");

        [Test]
        public void EveryTileOfTheSetHasItsOwnIndexInSetOrder()
        {
            var expected = 0;
            for (byte low = 0; low <= 6; low++)
            for (var high = low; high <= 6; high++)
                Assert.AreEqual(expected++, TileFaces.Index(D(low, high)), $"{low}-{high}");
            Assert.AreEqual(TileFaces.SetSize, expected);
        }

        [Test]
        public void BothOrdersOfATileShareOneIndex()
        {
            for (byte first = 0; first <= 6; first++)
            for (byte second = 0; second <= 6; second++)
                Assert.AreEqual(TileFaces.Index(D(first, second)), TileFaces.Index(D(second, first)));
        }

        [Test]
        public void AHalfAboveSixHasNoFace() =>
            Assert.Throws<ArgumentOutOfRangeException>(() => TileFaces.Index(D(2, 7)));

        [TestCase(1f, 0f, 2, 3, 90f)]
        [TestCase(1f, 0f, 3, 2, -90f)]
        [TestCase(-1f, 0f, 1, 4, -90f)]
        [TestCase(0f, 1f, 2, 5, 180f)]
        [TestCase(0f, -1f, 2, 5, 0f)]
        [TestCase(0f, -1f, 5, 2, 180f)]
        public void TheTopOfAFaceTurnsTowardTheLowerHalf(float facingX, float facingY, int near, int far, float angle)
        {
            var tile = new PlacedDomino(Vector2.zero, new Vector2(facingX, facingY), (byte)near, (byte)far, false);

            Assert.AreEqual(0f, Mathf.DeltaAngle(angle, TileFaces.Angle(tile)), Tolerance);
        }

        [TestCase(1f, 0f, 0f)]
        [TestCase(-1f, 0f, 0f)]
        [TestCase(0f, 1f, 90f)]
        [TestCase(0f, -1f, 90f)]
        public void ADoubleStandsAcrossItsArm(float facingX, float facingY, float angle)
        {
            var tile = new PlacedDomino(Vector2.zero, new Vector2(facingX, facingY), 4, 4, true);

            Assert.AreEqual(0f, Mathf.Abs(Mathf.DeltaAngle(angle, TileFaces.Angle(tile))) % 180f, Tolerance);
        }

        [TestCase(1080f, 1920f)]
        [TestCase(1920f, 1080f)]
        public void EveryLaidTileTurnsTheTopOfItsFaceTowardItsLowerHalf(float width, float height)
        {
            var layout = new ChainLayout(TileFaces.SetSize);
            var set = FullSet();
            var line = set.GetRange(0, 20).ToArray();
            line[10] = D(6, 6);
            var top = set.GetRange(20, 4).ToArray();
            var bottom = set.GetRange(24, 4).ToArray();

            layout.Arrange(line, 10, top, bottom, new Vector2(width, height), float.MaxValue);

            Assert.AreEqual(TileFaces.SetSize, layout.Count);
            for (var i = 0; i < layout.Count; i++)
            {
                var tile = layout[i];
                var faceTop = Top(TileFaces.Angle(tile));
                if (tile.Across)
                {
                    Assert.AreEqual(0f, Vector2.Dot(faceTop, tile.Facing), Tolerance, $"double {i} stands across its arm");
                    continue;
                }
                var lowerHalf = tile.Near <= tile.Far ? -tile.Facing : tile.Facing;
                AssertSameDirection(lowerHalf, faceTop, $"tile {i}");
            }
        }

        static List<Domino> FullSet()
        {
            var all = new List<Domino>();
            for (byte low = 0; low <= 6; low++)
            for (var high = low; high <= 6; high++)
                all.Add(D(low, high));
            return all;
        }

        [Test]
        public void ASetGivesTheFaceOfEachTile()
        {
            var faces = new Sprite[TileFaces.SetSize];
            for (var i = 0; i < faces.Length; i++) faces[i] = Keep(Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero));
            var set = Keep(ScriptableObject.CreateInstance<TileFaceSet>());

            set.SetFaces(faces);

            Assert.IsTrue(set.IsComplete);
            Assert.AreSame(faces[0], set.Face(D(0, 0)));
            Assert.AreSame(faces[7], set.Face(D(1, 1)));
            Assert.AreSame(faces[11], set.Face(D(5, 1)));
            Assert.AreSame(faces[27], set.Face(D(6, 6)));
        }

        [Test]
        public void ASetMissingAFaceIsNotComplete()
        {
            var set = Keep(ScriptableObject.CreateInstance<TileFaceSet>());
            Assert.IsFalse(set.IsComplete);

            set.SetFaces(new Sprite[TileFaces.SetSize]);
            Assert.IsFalse(set.IsComplete);

            set.SetFaces(new Sprite[TileFaces.SetSize - 1]);
            Assert.IsFalse(set.IsComplete);
        }

        T Keep<T>(T item) where T : Object
        {
            created.Add(item);
            return item;
        }
    }
}
