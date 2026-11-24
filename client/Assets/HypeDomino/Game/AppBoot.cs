using System;
using System.Collections.Generic;
using HypeDomino.Content;
using HypeDomino.Core.Diagnostics;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Graphics;
using HypeDomino.UI.Localization;
using Hypedomino.V1;
using UnityEngine;
using UnityEngine.SceneManagement;
using AnnouncedBundle = Hypedomino.V1.StringBundle;

namespace HypeDomino.Game
{
    public static class AppBoot
    {
        public static AppFlow Flow { get; private set; }

        [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterAssembliesLoaded)]
        static void AssembliesLoaded() => StartupPhases.App.Mark(StartupPhases.Assemblies);

        [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]
        static void Start()
        {
            Flow = new AppFlow();
            Flow.Changed += OnChanged;
            new StartupMarker(Flow, () => Time.realtimeSinceStartupAsDouble, line => Debug.Log(line), StartupPhases.App);
            SceneManager.sceneLoaded += FirstSceneLoaded;
            StartupPhases.App.Mark(nameof(AppBoot));
        }

        static void FirstSceneLoaded(Scene scene, LoadSceneMode mode)
        {
            SceneManager.sceneLoaded -= FirstSceneLoaded;
            StartupPhases.App.Mark(StartupPhases.Scene);
        }

        public static void Listen(Session session, AppFlow flow, CatalogUpdater catalog, GameStrings strings, StringBundleUpdater bundles, CosmeticManifestUpdater cosmetics, Action<HapticCue> playHaptic, Action<MatchSounds> playSounds)
        {
            var cues = new HapticCues();
            var sounds = new SoundCues();
            ServerHello announced = null;
            var followed = strings.Language;
            session.HelloReceived += hello =>
            {
                cues.StartConnection();
                sounds.StartConnection();
                if (Hello(hello, flow, catalog, strings, bundles, cosmetics)) announced = hello;
            };
            strings.Changed += () =>
            {
                if (strings.Language == followed) return;
                followed = strings.Language;
                FollowLanguage(announced, strings, bundles);
            };
            session.LobbyStateReceived += flow.Room;
            session.ViewReceived += view =>
            {
                flow.View(view.State);
                playHaptic(cues.Read(view));
                playSounds(sounds.Read(view));
            };
        }

        public static void FollowMatch(AppState state, FrameRatePolicy frameRate, CatalogUpdater catalog)
        {
            var inMatch = state == AppState.Match;
            frameRate.SetInMatch(inMatch);
            catalog?.SetInMatch(inMatch);
        }

        public static void ApplyStagedStrings(AppState state, GameStrings strings) => strings?.ApplyStaged(state == AppState.Match);

        public static List<AnnouncedBundle> FollowStrings(ServerHello hello, GameStrings strings)
        {
            var chains = hello.FallbackChains;
            for (var i = 0; i < chains.Count; i++)
                if (!strings.SetChain(chains[i].Language, chains[i].Languages)) Debug.LogWarning($"Strings chain refused language={chains[i].Language}");
            return Missing(hello, strings);
        }

        static List<AnnouncedBundle> Missing(ServerHello hello, GameStrings strings)
        {
            var bundles = hello.StringBundles;
            var missing = new List<AnnouncedBundle>(bundles.Count);
            for (var i = 0; i < bundles.Count; i++)
                if (IsWanted(bundles[i].Language, strings.Language) && !strings.Holds(bundles[i].Language, bundles[i].Hash)) missing.Add(bundles[i]);
            return missing;
        }

        static bool IsWanted(string language, string selected) => language == selected || language == GameLanguage.English;

        static bool Hello(ServerHello hello, AppFlow flow, CatalogUpdater catalog, GameStrings strings, StringBundleUpdater bundles, CosmeticManifestUpdater cosmetics)
        {
            flow.Hello(hello);
            if (hello.Status != HelloStatus.Accepted) return false;
            catalog?.Announce(hello.CatalogVersion, hello.ContentBaseUrl);
            Fetch(hello.ContentBaseUrl, FollowStrings(hello, strings), bundles);
            var manifest = hello.CosmeticsManifest;
            if (manifest != null) cosmetics?.Fetch(hello.ContentBaseUrl, (int)manifest.Version, manifest.Hash);
            return true;
        }

        static void FollowLanguage(ServerHello announced, GameStrings strings, StringBundleUpdater bundles)
        {
            bundles?.Restore(strings.Language);
            if (announced != null) Fetch(announced.ContentBaseUrl, Missing(announced, strings), bundles);
        }

        static void Fetch(string baseUrl, List<AnnouncedBundle> missing, StringBundleUpdater bundles)
        {
            for (var i = 0; i < missing.Count; i++)
            {
                Debug.Log($"Strings fetch language={missing[i].Language} version={missing[i].Version}");
                bundles?.Fetch(baseUrl, missing[i].Language, (int)missing[i].Version, missing[i].Hash);
            }
        }

        static void OnChanged(AppState state)
        {
            FollowMatch(state, GraphicsBoot.FrameRate, ContentBoot.Catalog);
            ContentBoot.Cosmetics?.ApplyStaged(state == AppState.Match);
            ApplyStagedStrings(state, LanguageBoot.Strings);
            AudioBoot.FollowState(state);
            Debug.Log($"App state={state}");
        }
    }
}
