using System;
using System.Text;
using HypeDomino.UI.Localization;

namespace HypeDomino.UI.Cosmetics
{
    public sealed class ActiveCosmeticManifest
    {
        CosmeticManifest staged;
        string stagedHash;

        public ActiveCosmeticManifest(CosmeticManifest builtin) => Current = builtin;

        public event Action Changed;

        public CosmeticManifest Current { get; private set; }

        public string Hash { get; private set; }

        public bool Holds(string hash) => (staged != null ? stagedHash : Hash) == hash;

        public bool Stage(int version, string hash, byte[] content)
        {
            if (content == null || StringBundle.Sha256Hex(content) != hash) return false;
            if (!CosmeticManifest.TryParse(Encoding.UTF8.GetString(content), out var manifest) || manifest.ManifestVersion != version) return false;
            staged = manifest;
            stagedHash = hash;
            return true;
        }

        public bool ApplyStaged(bool inMatch)
        {
            if (inMatch || staged == null) return false;
            Current = staged;
            Hash = stagedHash;
            staged = null;
            stagedHash = null;
            Changed?.Invoke();
            return true;
        }
    }
}
