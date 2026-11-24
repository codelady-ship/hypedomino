using HypeDomino.Core.Diagnostics;
using HypeDomino.Game.Match;
using HypeDomino.UI.Audio;
using UnityEngine;

namespace HypeDomino.Game
{
    public static class AudioBoot
    {
        static AudioLevelStore store;

        public static AudioVolumes Volumes { get; private set; }

        public static SoundPlayer Player { get; private set; }

        [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.BeforeSceneLoad)]
        static void Start()
        {
            Volumes = AudioLevelStore.Load();
            store = new AudioLevelStore(Volumes);
            Volumes.Changed += OnChanged;
            Log();
            var driver = new GameObject(nameof(AudioBoot)) { hideFlags = HideFlags.HideInHierarchy };
            Object.DontDestroyOnLoad(driver);
            Player = driver.AddComponent<SoundPlayer>();
            Player.Configure(SoundBank.Load(), Volumes, SoundPlayer.Voices, () => Time.unscaledTime * 1000f);
            Player.MakeMain();
            driver.AddComponent<Driver>();
            if (!Application.isEditor) ApplyOutput();
            StartupPhases.App.Mark(nameof(AudioBoot));
        }

        public static void FollowState(AppState state)
        {
            if (Player != null) Player.PlayMusic(MusicFor(state));
        }

        public static Music MusicFor(AppState state) => state switch
        {
            AppState.Match => Music.Match,
            AppState.Lobby => Music.Lobby,
            AppState.Room => Music.Lobby,
            AppState.Login => Music.Lobby,
            _ => Music.None,
        };

        public static void PlayMatch(MatchSounds sounds) => SoundCues.Play(sounds, Player);

        static void OnChanged()
        {
            if (!Application.isEditor) ApplyOutput();
        }

        static void SaveLevels()
        {
            if (store.Save()) Log();
        }

        static void ApplyOutput()
        {
            var started = AudioSettings.Mobile.audioOutputStarted;
            if (Volumes.Silent && started) AudioSettings.Mobile.StopAudioOutput();
            if (!Volumes.Silent && !started) AudioSettings.Mobile.StartAudioOutput();
        }

        static void Log() => Debug.Log($"Audio music={Volumes.Level(AudioGroup.Music)} sfx={Volumes.Level(AudioGroup.Sfx)} ui={Volumes.Level(AudioGroup.Ui)} silent={Volumes.Silent}");

        sealed class Driver : MonoBehaviour
        {
            void OnApplicationFocus(bool hasFocus)
            {
                if (hasFocus && !Application.isEditor) ApplyOutput();
            }

            void OnApplicationPause(bool pause)
            {
                if (pause) SaveLevels();
            }

            void OnApplicationQuit() => SaveLevels();
        }
    }
}
