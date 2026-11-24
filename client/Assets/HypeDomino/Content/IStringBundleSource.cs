using System;

namespace HypeDomino.Content
{
    public interface IStringBundleSource
    {
        void Download(string url, Action<byte[]> done);

        bool TryLoadStored(string language, out int version, out string hash, out byte[] content);

        void Store(string language, int version, string hash, byte[] content);
    }
}
