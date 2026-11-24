using System.Collections.Generic;
using HypeDomino.UI.Cosmetics;
using UnityEngine;

namespace HypeDomino.Content
{
    public sealed class CosmeticManifestUpdater
    {
        readonly ICosmeticManifestSource source;
        readonly ActiveCosmeticManifest manifest;
        readonly HashSet<string> downloading = new();

        public CosmeticManifestUpdater(ICosmeticManifestSource source, ActiveCosmeticManifest manifest)
        {
            this.source = source;
            this.manifest = manifest;
        }

        public void LoadStored()
        {
            if (!source.TryLoadStored(out var version, out var hash, out var content)) return;
            if (!manifest.Stage(version, hash, content))
            {
                Debug.LogWarning($"Cosmetics stored refused version={version}");
                return;
            }
            manifest.ApplyStaged(false);
        }

        public void Fetch(string baseUrl, int version, string hash)
        {
            if (string.IsNullOrEmpty(baseUrl) || string.IsNullOrEmpty(hash)) return;
            if (manifest.Holds(hash) || !downloading.Add(hash)) return;
            source.Download(ContentUrls.Cosmetics(baseUrl, hash), content => Downloaded(version, hash, content));
        }

        void Downloaded(int version, string hash, byte[] content)
        {
            downloading.Remove(hash);
            if (content == null)
            {
                Debug.LogWarning($"Cosmetics fetch failed version={version}");
                return;
            }
            if (!manifest.Stage(version, hash, content))
            {
                Debug.LogWarning($"Cosmetics fetch refused version={version}");
                return;
            }
            source.Store(version, hash, content);
        }
    }
}
