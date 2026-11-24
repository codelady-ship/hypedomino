using System.Collections.Generic;
using HypeDomino.UI.Localization;
using UnityEngine;

namespace HypeDomino.Content
{
    public sealed class StringBundleUpdater
    {
        readonly IStringBundleSource source;
        readonly GameStrings strings;
        readonly HashSet<string> downloading = new();

        public StringBundleUpdater(IStringBundleSource source, GameStrings strings)
        {
            this.source = source;
            this.strings = strings;
        }

        public void LoadStored()
        {
            Restore(strings.Language);
            if (strings.Language != GameLanguage.English) Restore(GameLanguage.English);
            strings.ApplyStaged(false);
        }

        public void Fetch(string baseUrl, string language, int version, string hash)
        {
            if (string.IsNullOrEmpty(baseUrl) || string.IsNullOrEmpty(hash) || !GameLanguage.IsLaunch(language)) return;
            if (strings.Holds(language, hash) || !downloading.Add(hash)) return;
            source.Download(ContentUrls.Strings(baseUrl, language, hash), content => Downloaded(language, version, hash, content));
        }

        public void Restore(string language)
        {
            if (!source.TryLoadStored(language, out var version, out var hash, out var content) || strings.Holds(language, hash)) return;
            if (!strings.Stage(language, version, hash, content)) Debug.LogWarning($"Strings stored refused language={language} version={version}");
        }

        void Downloaded(string language, int version, string hash, byte[] content)
        {
            downloading.Remove(hash);
            if (content == null)
            {
                Debug.LogWarning($"Strings fetch failed language={language} version={version}");
                return;
            }
            if (!strings.Stage(language, version, hash, content))
            {
                Debug.LogWarning($"Strings fetch refused language={language} version={version}");
                return;
            }
            source.Store(language, version, hash, content);
        }
    }
}
