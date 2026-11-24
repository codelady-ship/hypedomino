using System;

namespace HypeDomino.Content
{
    public interface ICosmeticManifestSource
    {
        void Download(string url, Action<byte[]> done);

        bool TryLoadStored(out int version, out string hash, out byte[] content);

        void Store(int version, string hash, byte[] content);
    }
}
