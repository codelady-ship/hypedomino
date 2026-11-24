using System;
using System.Collections.Generic;
using HypeDomino.UI.Cosmetics;
using UnityEngine.AddressableAssets;
using UnityEngine.ResourceManagement.AsyncOperations;

namespace HypeDomino.Content
{
    public sealed class AddressablesArtSource<T> : ICosmeticArtSource<T> where T : class
    {
        readonly Dictionary<string, AsyncOperationHandle<T>> handles = new();

        public int Held => handles.Count;

        public void Load(string address, Action<T> done)
        {
            var handle = Addressables.LoadAssetAsync<T>(address);
            handles[address] = handle;
            handle.Completed += loaded =>
            {
                if (loaded.Status == AsyncOperationStatus.Succeeded)
                {
                    done(loaded.Result);
                    return;
                }
                handles.Remove(address);
                Addressables.Release(loaded);
                done(null);
            };
        }

        public void Release(string address)
        {
            if (handles.Remove(address, out var handle)) Addressables.Release(handle);
        }
    }
}
