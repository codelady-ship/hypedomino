using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using HypeDomino.Content;
using HypeDomino.Core.Diagnostics;
using HypeDomino.UI.Localization;
using UnityEngine;

namespace HypeDomino.Game
{
    public static class LanguageBoot
    {
        const string ChoiceKey = "language.choice";

        public static LanguageChoice Choice { get; private set; }

        public static GameStrings Strings { get; private set; }

        public static StringBundleUpdater Bundles { get; private set; }

        [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]
        static void Start()
        {
            Build();
            StartupPhases.App.Mark(nameof(LanguageBoot));
        }

        static void Build()
        {
            var saved = PlayerPrefs.GetString(ChoiceKey, string.Empty);
            Choice = LanguageChoice.Load(saved, DeviceLocales);
            Choice.Changed += Save;
            if (Choice.Language != saved) Save();
            else Log();
            Strings = new GameStrings(BuiltinStrings.Load(), Choice, Debug.isDebugBuild);
            if (Application.isEditor) return;
            Bundles = new StringBundleUpdater(new UnityStringBundleSource(ContentBoot.Folder), Strings);
            Bundles.LoadStored();
        }

        static void Save()
        {
            PlayerPrefs.SetString(ChoiceKey, Choice.Language);
            PlayerPrefs.Save();
            Log();
        }

        static void Log() => Debug.Log($"Language {Choice.Language}");

        static IReadOnlyList<string> DeviceLocales()
        {
            if (Application.isEditor) return Array.Empty<string>();
            if (Application.platform == RuntimePlatform.Android) return GameLanguage.SplitLocales(AndroidLocaleTags());
            if (Application.platform == RuntimePlatform.IPhonePlayer) return GameLanguage.SplitLocales(IosPreferredLanguages());
            return Array.Empty<string>();
        }

        static string AndroidLocaleTags()
        {
            using var localeList = new AndroidJavaClass("android.os.LocaleList");
            using var locales = localeList.CallStatic<AndroidJavaObject>("getDefault");
            return locales.Call<string>("toLanguageTags");
        }

#if UNITY_IOS && !UNITY_EDITOR
        [DllImport("__Internal")]
        static extern string HypeDominoPreferredLanguages();

        static string IosPreferredLanguages() => HypeDominoPreferredLanguages();
#else
        static string IosPreferredLanguages() => string.Empty;
#endif
    }
}
