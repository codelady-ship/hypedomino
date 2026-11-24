using HypeDomino.UI.Audio;
using UnityEngine;

namespace HypeDomino.Game
{
    public sealed class AudioLevelStore
    {
        public const string MusicKey = "audio.musicVolume";
        public const string SfxKey = "audio.sfxVolume";
        public const string UiKey = "audio.uiVolume";

        readonly AudioVolumes volumes;

        public AudioLevelStore(AudioVolumes volumes)
        {
            this.volumes = volumes;
            volumes.Changed += MarkUnsaved;
        }

        public bool Unsaved { get; private set; }

        public static AudioVolumes Load() => new(
            PlayerPrefs.GetFloat(MusicKey, AudioVolumes.Full),
            PlayerPrefs.GetFloat(SfxKey, AudioVolumes.Full),
            PlayerPrefs.GetFloat(UiKey, AudioVolumes.Full));

        public bool Save()
        {
            if (!Unsaved) return false;
            Unsaved = false;
            PlayerPrefs.SetFloat(MusicKey, volumes.Level(AudioGroup.Music));
            PlayerPrefs.SetFloat(SfxKey, volumes.Level(AudioGroup.Sfx));
            PlayerPrefs.SetFloat(UiKey, volumes.Level(AudioGroup.Ui));
            PlayerPrefs.Save();
            return true;
        }

        void MarkUnsaved() => Unsaved = true;
    }
}
