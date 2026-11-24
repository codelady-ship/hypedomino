using UnityEngine;
using UnityEngine.Pool;

namespace HypeDomino.Core.Pooling
{
    public sealed class PrefabPool<T> where T : Component
    {
        readonly T prefab;
        readonly Transform parent;
        readonly ObjectPool<T> pool;

        public PrefabPool(T prefab, Transform parent, int prewarm, int maxSize)
        {
            this.prefab = prefab;
            this.parent = parent;
            pool = new ObjectPool<T>(Create, Activate, Deactivate, DestroyInstance, true, prewarm, maxSize);
            Prewarm(prewarm);
        }

        public int CountActive => pool.CountActive;
        public int CountInactive => pool.CountInactive;

        public T Get() => pool.Get();

        public void Release(T instance) => pool.Release(instance);

        public void Clear() => pool.Clear();

        void Prewarm(int count)
        {
            var warmed = new T[count];
            for (var i = 0; i < count; i++) warmed[i] = pool.Get();
            for (var i = 0; i < count; i++) pool.Release(warmed[i]);
        }

        T Create()
        {
            var instance = Object.Instantiate(prefab, parent);
            instance.gameObject.SetActive(false);
            return instance;
        }

        static void Activate(T instance) => instance.gameObject.SetActive(true);

        static void Deactivate(T instance) => instance.gameObject.SetActive(false);

        static void DestroyInstance(T instance) => Object.Destroy(instance.gameObject);
    }
}
