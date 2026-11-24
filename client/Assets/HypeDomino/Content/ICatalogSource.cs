using System;

namespace HypeDomino.Content
{
    public interface ICatalogSource
    {
        string StoredVersion { get; }

        string StoredBaseUrl { get; }

        void LoadStored(Action<bool> done);

        void Download(string url, Action<bool> done);

        void Commit(string version, string baseUrl, Action<bool> done);
    }
}
