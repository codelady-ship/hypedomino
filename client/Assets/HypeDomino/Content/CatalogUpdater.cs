namespace HypeDomino.Content
{
    public sealed class CatalogUpdater
    {
        static readonly int[] RetryDelaysMs = { 2000, 8000, 30000 };

        readonly ICatalogSource source;
        readonly string platform;
        string wantedVersion;
        string wantedBaseUrl;
        bool inMatch;
        bool busy;
        int failures;
        float retryInMs;

        public CatalogUpdater(ICatalogSource source, string platform)
        {
            this.source = source;
            this.platform = platform;
            BaseUrl = source.StoredBaseUrl;
        }

        public string LoadedVersion { get; private set; }

        public string BaseUrl { get; private set; }

        public float RetryInMs => retryInMs;

        public bool InMatch => inMatch;

        public void Start()
        {
            var stored = source.StoredVersion;
            if (string.IsNullOrEmpty(stored)) return;
            busy = true;
            source.LoadStored(ok =>
            {
                busy = false;
                if (ok) LoadedVersion = stored;
                TryStart();
            });
        }

        public void Announce(string version, string baseUrl)
        {
            if (string.IsNullOrEmpty(version) || string.IsNullOrEmpty(baseUrl)) return;
            BaseUrl = baseUrl;
            if (version == LoadedVersion && !busy)
            {
                wantedVersion = null;
                return;
            }
            if (version == wantedVersion && baseUrl == wantedBaseUrl && (busy || retryInMs > 0f)) return;
            wantedVersion = version;
            wantedBaseUrl = baseUrl;
            failures = 0;
            retryInMs = 0f;
            TryStart();
        }

        public void SetInMatch(bool on)
        {
            inMatch = on;
            TryStart();
        }

        public void Advance(float deltaMs)
        {
            if (retryInMs <= 0f) return;
            retryInMs -= deltaMs;
            if (retryInMs > 0f) return;
            retryInMs = 0f;
            TryStart();
        }

        void TryStart()
        {
            if (busy || inMatch || retryInMs > 0f || wantedVersion == null) return;
            if (wantedVersion == LoadedVersion)
            {
                wantedVersion = null;
                return;
            }
            busy = true;
            var version = wantedVersion;
            var baseUrl = wantedBaseUrl;
            source.Download(ContentUrls.Catalog(baseUrl, platform, version), ok => Downloaded(version, baseUrl, ok));
        }

        void Downloaded(string version, string baseUrl, bool ok)
        {
            if (!ok || version != wantedVersion)
            {
                Finish(version, false);
                return;
            }
            source.Commit(version, baseUrl, loaded => Finish(version, loaded));
        }

        void Finish(string version, bool ok)
        {
            busy = false;
            if (ok) LoadedVersion = version;
            if (version == wantedVersion)
            {
                if (ok) wantedVersion = null;
                else Retry();
            }
            TryStart();
        }

        void Retry()
        {
            if (failures >= RetryDelaysMs.Length)
            {
                wantedVersion = null;
                return;
            }
            retryInMs = RetryDelaysMs[failures++];
        }
    }
}
