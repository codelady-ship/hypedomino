using System;
using System.IO;
using UnityEngine;
using UnityEngine.AddressableAssets;
using UnityEngine.AddressableAssets.ResourceLocators;
using UnityEngine.Networking;
using UnityEngine.ResourceManagement.AsyncOperations;

namespace HypeDomino.Content
{
    public sealed class AddressablesCatalogSource : ICatalogSource
    {
        const string VersionKey = "content.catalogVersion";
        const string BaseUrlKey = "content.baseUrl";
        const int TimeoutSeconds = 20;

        readonly string folder;
        readonly string downloadPath;
        IResourceLocator remote;

        public AddressablesCatalogSource(string folder)
        {
            Directory.CreateDirectory(folder);
            this.folder = folder;
            downloadPath = Path.Combine(folder, "catalog-new.bin");
        }

        public string StoredVersion
        {
            get
            {
                var version = PlayerPrefs.GetString(VersionKey, null);
                return string.IsNullOrEmpty(version) || !File.Exists(StoredPath(version)) ? null : version;
            }
        }

        public string StoredBaseUrl => PlayerPrefs.GetString(BaseUrlKey, null);

        public void LoadStored(Action<bool> done) => Load(StoredPath(StoredVersion), done);

        public void Download(string url, Action<bool> done)
        {
            var request = new UnityWebRequest(url, UnityWebRequest.kHttpVerbGET, new DownloadHandlerFile(downloadPath) { removeFileOnAbort = true }, null)
            {
                timeout = TimeoutSeconds,
            };
            request.SendWebRequest().completed += _ =>
            {
                var ok = request.result == UnityWebRequest.Result.Success;
                request.Dispose();
                done(ok);
            };
        }

        public void Commit(string version, string baseUrl, Action<bool> done)
        {
            var previous = StoredVersion;
            var target = StoredPath(version);
            if (!TryMove(downloadPath, target))
            {
                done(false);
                return;
            }
            Load(target, ok =>
            {
                if (!ok)
                {
                    TryDelete(target);
                    done(false);
                    return;
                }
                if (previous != null && previous != version) TryDelete(StoredPath(previous));
                PlayerPrefs.SetString(VersionKey, version);
                PlayerPrefs.SetString(BaseUrlKey, baseUrl);
                PlayerPrefs.Save();
                done(true);
            });
        }

        string StoredPath(string version) => Path.Combine(folder, ContentUrls.CatalogFile(version));

        void Load(string path, Action<bool> done)
        {
            Addressables.LoadContentCatalogAsync(path, false).Completed += loaded =>
            {
                var ok = loaded.Status == AsyncOperationStatus.Succeeded;
                if (ok)
                {
                    if (remote != null) Addressables.RemoveResourceLocator(remote);
                    remote = loaded.Result;
                    Addressables.CleanBundleCache().Completed += cleaned => Addressables.Release(cleaned);
                }
                Addressables.Release(loaded);
                done(ok);
            };
        }

        static bool TryMove(string from, string to)
        {
            try
            {
                File.Copy(from, to, true);
                File.Delete(from);
                return true;
            }
            catch (IOException)
            {
                return false;
            }
        }

        static void TryDelete(string path)
        {
            try
            {
                File.Delete(path);
            }
            catch (IOException)
            {
            }
        }
    }
}
