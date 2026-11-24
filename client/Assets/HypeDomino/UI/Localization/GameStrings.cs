using System;
using System.Collections.Generic;
using UnityEngine;

namespace HypeDomino.UI.Localization
{
    public sealed class GameStrings
    {
        const string UnavailableKey = "common.text.unavailable";

        readonly BuiltinStrings builtin;
        readonly LanguageChoice choice;
        readonly bool showMissingKeys;
        readonly Dictionary<string, string[]> chains = new();
        readonly Dictionary<string, StringBundle> active = new();
        readonly Dictionary<string, StringBundle> staged = new();
        readonly HashSet<string> reported = new();

        public GameStrings(BuiltinStrings builtin, LanguageChoice choice, bool showMissingKeys)
        {
            this.builtin = builtin;
            this.choice = choice;
            this.showMissingKeys = showMissingKeys;
            choice.Changed += OnLanguageChosen;
        }

        public event Action Changed;

        public string Language => choice.Language;

        public bool SetChain(string language, IReadOnlyList<string> chain)
        {
            if (chain == null || chain.Count == 0 || chain[0] != language || chain[chain.Count - 1] != GameLanguage.English) return false;
            var copy = new string[chain.Count];
            for (var i = 0; i < copy.Length; i++) copy[i] = chain[i];
            chains[language] = copy;
            return true;
        }

        public bool Holds(string language, string hash)
        {
            if (staged.TryGetValue(language, out var next)) return next.Hash == hash;
            return active.TryGetValue(language, out var held) && held.Hash == hash;
        }

        public bool Stage(string language, int version, string hash, byte[] content)
        {
            if (!StringBundle.TryRead(content, version, hash, out var bundle) || bundle.Language != language) return false;
            staged[language] = bundle;
            return true;
        }

        public bool ApplyStaged(bool inMatch)
        {
            if (inMatch || staged.Count == 0) return false;
            foreach (var pair in staged) active[pair.Key] = pair.Value;
            staged.Clear();
            Changed?.Invoke();
            return true;
        }

        public string Text(string key, IReadOnlyDictionary<string, object> args = null)
        {
            var chain = Chain(Language);
            for (var i = 0; i < chain.Length; i++)
            {
                if (!TryFormat(key, chain[i], args, out var text)) continue;
                if (i > 0) Report(key);
                return text;
            }
            Report(key);
            if (showMissingKeys) return "[!" + key + "]";
            return builtin.Value(UnavailableKey, GameLanguage.English) ?? string.Empty;
        }

        void OnLanguageChosen() => Changed?.Invoke();

        string[] Chain(string language) =>
            language != null && chains.TryGetValue(language, out var chain) ? chain : GameLanguage.DefaultChain(language);

        bool TryFormat(string key, string language, IReadOnlyDictionary<string, object> args, out string text)
        {
            text = null;
            var value = Value(key, language);
            return value != null && MessageFormat.TryFormat(value, language, Language, args, unitKey => Value(unitKey, language), out text);
        }

        string Value(string key, string language)
        {
            if (active.TryGetValue(language, out var bundle) && bundle.TryGet(key, out var value) && !string.IsNullOrEmpty(value)) return value;
            return builtin.Value(key, language);
        }

        void Report(string key)
        {
            if (!reported.Add(key)) return;
            var version = active.TryGetValue(Language, out var bundle) ? bundle.Version : 0;
            Debug.LogWarning($"Strings missing key={key} language={Language} version={version}");
        }
    }
}
