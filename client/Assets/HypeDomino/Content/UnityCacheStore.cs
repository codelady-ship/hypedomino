using System.IO;
using UnityEngine;

namespace HypeDomino.Content
{
    public sealed class UnityCacheStore : ICacheStore
    {
        const string ClearAtStartKey = "content.clearAtStart";

        readonly string catalogFolder;

        public UnityCacheStore(string catalogFolder) => this.catalogFolder = catalogFolder;

        public long OccupiedBytes => Caching.currentCacheForWriting.spaceOccupied + StoredFileBytes();

        public bool ClearAtStart
        {
            get => PlayerPrefs.GetInt(ClearAtStartKey, 0) == 1;
            set
            {
                PlayerPrefs.SetInt(ClearAtStartKey, value ? 1 : 0);
                PlayerPrefs.Save();
            }
        }

        public void ApplyLimits(long limitBytes, int unusedForSeconds)
        {
            var cache = Caching.currentCacheForWriting;
            cache.maximumAvailableStorageSpace = limitBytes;
            cache.expirationDelay = unusedForSeconds;
        }

        public bool TryClear() => Caching.ClearCache();

        long StoredFileBytes()
        {
            if (!Directory.Exists(catalogFolder)) return 0;
            long total = 0;
            foreach (var file in Directory.GetFiles(catalogFolder, "*", SearchOption.AllDirectories)) total += new FileInfo(file).Length;
            return total;
        }
    }
}
