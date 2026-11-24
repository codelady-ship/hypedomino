using System.Collections;
using System.Collections.Generic;
using HypeDomino.Core.Pooling;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.Pool;
using UnityEngine.TestTools;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class PoolingTests
    {
        const int Prewarm = 28;
        const int Cycles = 1000;

        GameObject root;
        Transform prefab;

        [SetUp]
        public void CreatePrefab()
        {
            root = new GameObject("PoolRoot");
            prefab = new GameObject("Tile").transform;
            prefab.SetParent(root.transform);
        }

        [TearDown]
        public void DestroyPrefab() => Object.Destroy(root);

        [Test]
        public void PrewarmCreatesInactiveInstancesUnderTheParent()
        {
            var pool = new PrefabPool<Transform>(prefab, root.transform, Prewarm, Prewarm);
            Assert.AreEqual(Prewarm, pool.CountInactive);
            Assert.AreEqual(0, pool.CountActive);
            Assert.AreEqual(Prewarm + 1, root.transform.childCount);
        }

        [Test]
        public void GetActivatesAndReleaseDeactivatesTheSameInstance()
        {
            var pool = new PrefabPool<Transform>(prefab, root.transform, 1, 1);
            var tile = pool.Get();
            Assert.IsTrue(tile.gameObject.activeSelf);
            Assert.AreEqual(1, pool.CountActive);
            pool.Release(tile);
            Assert.IsFalse(tile.gameObject.activeSelf);
            Assert.AreSame(tile, pool.Get());
        }

        [UnityTest]
        public IEnumerator ReleaseBeyondMaxSizeDestroysTheInstance()
        {
            var pool = new PrefabPool<Transform>(prefab, root.transform, 0, 1);
            var kept = pool.Get();
            var extra = pool.Get();
            pool.Release(kept);
            pool.Release(extra);
            yield return null;
            Assert.IsTrue(kept);
            Assert.IsFalse(extra);
        }

        [Test]
        public void PrefabPoolCyclesWithoutAllocating()
        {
            var pool = new PrefabPool<Transform>(prefab, root.transform, Prewarm, Prewarm);
            var taken = new Transform[Prewarm];
            Assert.That(() =>
            {
                for (var cycle = 0; cycle < Cycles; cycle++)
                {
                    for (var i = 0; i < Prewarm; i++) taken[i] = pool.Get();
                    for (var i = 0; i < Prewarm; i++) pool.Release(taken[i]);
                }
            }, Allocates.Nothing);
        }

        [Test]
        public void GenericPoolsCycleWithoutAllocatingOnceWarm()
        {
            var tiles = new ObjectPool<List<int>>(() => new List<int>(Prewarm), actionOnRelease: list => list.Clear());

            void Cycle()
            {
                var list = tiles.Get();
                for (var i = 0; i < Prewarm; i++) list.Add(i);
                var scratch = ListPool<int>.Get();
                scratch.Add(list.Count);
                ListPool<int>.Release(scratch);
                tiles.Release(list);
            }

            Cycle();
            Assert.That(() =>
            {
                for (var cycle = 0; cycle < Cycles; cycle++) Cycle();
            }, Allocates.Nothing);
        }
    }
}
