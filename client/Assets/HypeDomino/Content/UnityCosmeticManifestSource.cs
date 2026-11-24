using System;
using System.Globalization;
using System.IO;
using UnityEngine.Networking;

namespace HypeDomino.Content
{
    public sealed class UnityCosmeticManifestSource : ICosmeticManifestSource
    {
        const string CosmeticsFolder = "cosmetics";
        const string Extension = ".json";
        const char VersionSeparator = '_';

        readonly string folder;

        public UnityCosmeticManifestSource(string contentFolder) => folder = Path.Combine(contentFolder, CosmeticsFolder);

        public void Download(string url, Action<byte[]> done)
        {
            var request = UnityWebRequest.Get(url);
            request.SendWebRequest().completed += _ =>
            {
                var content = request.result == UnityWebRequest.Result.Success ? request.downloadHandler.data : null;
                request.Dispose();
                done(content);
            };
        }

        public bool TryLoadStored(out int version, out string hash, out byte[] content)
        {
            version = 0;
            hash = null;
            content = null;
            if (!Directory.Exists(folder)) return false;
            var files = Directory.GetFiles(folder, "*" + Extension);
            if (files.Length != 1 || !TryReadName(Path.GetFileNameWithoutExtension(files[0]), out version, out hash)) return false;
            try
            {
                content = File.ReadAllBytes(files[0]);
                return true;
            }
            catch (IOException)
            {
                return false;
            }
        }

        public void Store(int version, string hash, byte[] content)
        {
            try
            {
                Directory.CreateDirectory(folder);
                foreach (var previous in Directory.GetFiles(folder)) File.Delete(previous);
                File.WriteAllBytes(Path.Combine(folder, version.ToString(CultureInfo.InvariantCulture) + VersionSeparator + hash + Extension), content);
            }
            catch (IOException)
            {
            }
        }

        static bool TryReadName(string name, out int version, out string hash)
        {
            version = 0;
            hash = null;
            var separator = name.IndexOf(VersionSeparator);
            if (separator <= 0 || !int.TryParse(name.Substring(0, separator), NumberStyles.None, CultureInfo.InvariantCulture, out version)) return false;
            hash = name.Substring(separator + 1);
            return hash.Length > 0;
        }
    }
}
