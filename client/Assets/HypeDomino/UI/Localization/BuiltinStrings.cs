using System;
using System.Collections.Generic;
using UnityEngine;

namespace HypeDomino.UI.Localization
{
    public sealed class BuiltinStrings
    {
        public const string ResourceName = "builtin-strings";

        [Serializable]
        sealed class Entry
        {
            public string key;
            public string en;
            public string az;
            public string tr;
            public string ru;
        }

        [Serializable]
        sealed class File
        {
            public Entry[] entries;
        }

        readonly Dictionary<string, Entry> entries = new();

        BuiltinStrings(File file)
        {
            foreach (var entry in file.entries ?? Array.Empty<Entry>())
            {
                entries[entry.key] = entry;
            }
        }

        public static BuiltinStrings Parse(string json) => new(JsonUtility.FromJson<File>(json));

        public static BuiltinStrings Load()
        {
            var asset = Resources.Load<TextAsset>(ResourceName);
            if (asset == null)
            {
                Debug.LogError($"Strings builtin missing resource={ResourceName}");
                return new BuiltinStrings(new File());
            }
            var strings = Parse(asset.text);
            Resources.UnloadAsset(asset);
            return strings;
        }

        public string Value(string key, string language)
        {
            if (!entries.TryGetValue(key, out var entry)) return null;
            var text = ValueIn(entry, language);
            return string.IsNullOrEmpty(text) ? null : text;
        }

        static string ValueIn(Entry entry, string language) => language switch
        {
            "az" => entry.az,
            "tr" => entry.tr,
            "ru" => entry.ru,
            "en" => entry.en,
            _ => null,
        };
    }
}
