using System;
using UnityEngine;

namespace HypeDomino.UI.Audio
{
    public sealed class SoundBank : ScriptableObject
    {
        public const string ResourceName = "SoundBank";
        public const int TilePlaceVariants = 4;
        public const int SoundCount = (int)Sound.Combo5 + 1;
        public const int MusicCount = (int)Music.Match + 1;

        [SerializeField] AudioClip[] tilePlace = Array.Empty<AudioClip>();
        [SerializeField] AudioClip[] sounds = Array.Empty<AudioClip>();
        [SerializeField] AudioClip[] tracks = Array.Empty<AudioClip>();

        public static SoundBank Load() => Resources.Load<SoundBank>(ResourceName);

        public static AudioGroup GroupOf(Sound sound) => sound == Sound.UiClick ? AudioGroup.Ui : AudioGroup.Sfx;

        public bool IsComplete => tilePlace.Length == TilePlaceVariants && AllSet(tilePlace, -1) && sounds.Length == SoundCount && AllSet(sounds, (int)Sound.TilePlace) && tracks.Length == MusicCount && AllSet(tracks, (int)Music.None);

        public int Variants(Sound sound) => sound == Sound.TilePlace ? tilePlace.Length : 1;

        public AudioClip Clip(Sound sound, int variant) => sound == Sound.TilePlace ? tilePlace[variant] : sounds[(int)sound];

        public AudioClip Track(Music music) => tracks[(int)music];

        public void Set(AudioClip[] tilePlaceClips, AudioClip[] soundClips, AudioClip[] musicTracks)
        {
            tilePlace = tilePlaceClips ?? Array.Empty<AudioClip>();
            sounds = soundClips ?? Array.Empty<AudioClip>();
            tracks = musicTracks ?? Array.Empty<AudioClip>();
        }

        static bool AllSet(AudioClip[] clips, int emptySlot)
        {
            for (var i = 0; i < clips.Length; i++)
                if ((clips[i] == null) != (i == emptySlot)) return false;
            return true;
        }
    }
}
