using System;

namespace HypeDomino.Content
{
    public sealed class ContentCache
    {
        public const long LimitBytes = 1_000_000_000;
        public const int UnusedForSeconds = 30 * 24 * 60 * 60;

        readonly ICacheStore store;
        readonly Func<bool> inMatch;

        public ContentCache(ICacheStore store, Func<bool> inMatch)
        {
            this.store = store;
            this.inMatch = inMatch;
        }

        public long OccupiedBytes => store.OccupiedBytes;

        public void Start()
        {
            store.ApplyLimits(LimitBytes, UnusedForSeconds);
            if (store.ClearAtStart && store.TryClear()) store.ClearAtStart = false;
        }

        public CacheClearResult Clear()
        {
            if (inMatch()) return CacheClearResult.NotInMatch;
            if (store.TryClear())
            {
                if (store.ClearAtStart) store.ClearAtStart = false;
                return CacheClearResult.Cleared;
            }
            store.ClearAtStart = true;
            return CacheClearResult.AtNextStart;
        }
    }
}
