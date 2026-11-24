using System;
using HypeDomino.UI.Tokens;
using UnityEngine;
using Random = UnityEngine.Random;

namespace HypeDomino.UI.Audio
{
    public sealed class SoundPlayer : MonoBehaviour
    {
        public const int Voices = 8;
        public const float CrossfadeMs = TokenDurationMs.Slow;
        const int NoVariant = -1;

        readonly AudioSource[] music = new AudioSource[2];
        readonly int[] lastVariant = new int[SoundBank.SoundCount];

        SoundBank bank;
        AudioVolumes volumes;
        Func<float> clockMs;
        AudioSource[] voices = Array.Empty<AudioSource>();
        AudioGroup[] voiceGroups = Array.Empty<AudioGroup>();
        int nextVoice;
        int current;
        Music wanted;
        float fadeStartMs;
        float outgoingFrom;
        bool fading;

        public static SoundPlayer Main { get; private set; }

        public Music Playing { get; private set; }

        public bool Paused { get; private set; }

        public int VoiceCount => voices.Length;

        public AudioSource Voice(int index) => voices[index];

        public AudioSource CurrentMusic => music[current];

        public AudioSource OutgoingMusic => music[1 - current];

        public void Configure(SoundBank soundBank, AudioVolumes audioVolumes, int voiceCount, Func<float> nowMs)
        {
            bank = soundBank;
            volumes = audioVolumes;
            clockMs = nowMs;
            voices = new AudioSource[voiceCount];
            voiceGroups = new AudioGroup[voiceCount];
            for (var i = 0; i < voices.Length; i++) voices[i] = NewSource(false);
            for (var i = 0; i < music.Length; i++) music[i] = NewSource(true);
            for (var i = 0; i < lastVariant.Length; i++) lastVariant[i] = NoVariant;
            volumes.Changed += ApplyVolumes;
        }

        public void MakeMain() => Main = this;

        public bool Play(Sound sound)
        {
            var group = SoundBank.GroupOf(sound);
            if (Paused || bank == null || voices.Length == 0 || !volumes.Audible(group)) return false;
            var voice = nextVoice;
            nextVoice = (nextVoice + 1) % voices.Length;
            var source = voices[voice];
            source.Stop();
            source.clip = bank.Clip(sound, Variant(sound));
            source.volume = volumes.Level(group);
            voiceGroups[voice] = group;
            source.Play();
            return true;
        }

        public void PlayMusic(Music track)
        {
            wanted = track;
            FollowWanted();
        }

        public void SetPaused(bool paused)
        {
            if (paused == Paused) return;
            Paused = paused;
            if (!paused)
            {
                CurrentMusic.UnPause();
                FollowWanted();
                return;
            }
            for (var i = 0; i < voices.Length; i++) voices[i].Stop();
            FinishFade();
            CurrentMusic.Pause();
        }

        public void Tick()
        {
            if (!fading) return;
            var progress = Mathf.Clamp01((clockMs() - fadeStartMs) / CrossfadeMs);
            CurrentMusic.volume = progress * volumes.Level(AudioGroup.Music);
            OutgoingMusic.volume = (1f - progress) * outgoingFrom;
            if (progress >= 1f) FinishFade();
        }

        void Update()
        {
            if (!Paused && clockMs != null) Tick();
        }

        void OnApplicationPause(bool pause) => SetPaused(pause);

        void OnDestroy()
        {
            if (volumes != null) volumes.Changed -= ApplyVolumes;
            if (Main == this) Main = null;
        }

        AudioSource NewSource(bool loop)
        {
            var source = gameObject.AddComponent<AudioSource>();
            source.playOnAwake = false;
            source.loop = loop;
            source.spatialBlend = 0f;
            source.volume = 0f;
            return source;
        }

        int Variant(Sound sound)
        {
            var count = bank.Variants(sound);
            if (count < 2) return 0;
            var last = lastVariant[(int)sound];
            var pick = last == NoVariant ? Random.Range(0, count) : (last + 1 + Random.Range(0, count - 1)) % count;
            lastVariant[(int)sound] = pick;
            return pick;
        }

        void FollowWanted()
        {
            if (bank == null || Paused) return;
            var target = volumes.Audible(AudioGroup.Music) ? wanted : Music.None;
            if (target == Playing) return;
            FinishFade();
            outgoingFrom = CurrentMusic.volume;
            current = 1 - current;
            var incoming = CurrentMusic;
            incoming.clip = target == Music.None ? null : bank.Track(target);
            incoming.volume = 0f;
            if (target != Music.None) incoming.Play();
            Playing = target;
            fadeStartMs = clockMs();
            fading = true;
        }

        void FinishFade()
        {
            if (!fading) return;
            fading = false;
            OutgoingMusic.Stop();
            OutgoingMusic.clip = null;
            CurrentMusic.volume = volumes.Level(AudioGroup.Music);
        }

        void ApplyVolumes()
        {
            for (var i = 0; i < voices.Length; i++)
            {
                if (!volumes.Audible(voiceGroups[i])) voices[i].Stop();
                voices[i].volume = volumes.Level(voiceGroups[i]);
            }
            if (!volumes.Audible(AudioGroup.Music)) StopMusic();
            if (!fading) CurrentMusic.volume = volumes.Level(AudioGroup.Music);
            FollowWanted();
        }

        void StopMusic()
        {
            fading = false;
            for (var i = 0; i < music.Length; i++)
            {
                music[i].Stop();
                music[i].clip = null;
            }
            Playing = Music.None;
        }
    }
}
