using System.Collections.Generic;
using HypeDomino.Game.Match;
using HypeDomino.UI.Audio;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.EventSystems;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class SoundPlayerTests
    {
        const int SampleRate = 48000;
        const float Tolerance = 0.001f;
        const float Half = 0.5f;

        readonly List<Object> created = new();

        GameObject root;
        SoundBank bank;
        AudioVolumes volumes;
        SoundPlayer player;
        AudioClip[] tilePlace;
        AudioClip[] sounds;
        AudioClip[] tracks;
        float now;

        [SetUp]
        public void Build()
        {
            tilePlace = new AudioClip[SoundBank.TilePlaceVariants];
            for (var i = 0; i < tilePlace.Length; i++) tilePlace[i] = NewClip("tile_place_" + i, 1);
            sounds = new AudioClip[SoundBank.SoundCount];
            for (var i = 1; i < sounds.Length; i++) sounds[i] = NewClip(((Sound)i).ToString(), 1);
            tracks = new AudioClip[SoundBank.MusicCount];
            tracks[(int)Music.Lobby] = NewClip("lobby", 2);
            tracks[(int)Music.Match] = NewClip("match", 2);
            bank = Keep(ScriptableObject.CreateInstance<SoundBank>());
            bank.Set(tilePlace, sounds, tracks);
            volumes = new AudioVolumes(Half, AudioVolumes.Full, Half);
            now = 0f;
            root = new GameObject("Audio");
            player = root.AddComponent<SoundPlayer>();
            player.Configure(bank, volumes, SoundPlayer.Voices, () => now);
        }

        [TearDown]
        public void DestroyAll()
        {
            Object.Destroy(root);
            foreach (var item in created) Object.Destroy(item);
            created.Clear();
        }

        AudioClip NewClip(string name, int channels) => Keep(AudioClip.Create(name, SampleRate / 10, channels, SampleRate, false));

        T Keep<T>(T item) where T : Object
        {
            created.Add(item);
            return item;
        }

        [Test]
        public void TheBankIsCompleteWithEveryClip()
        {
            Assert.IsTrue(bank.IsComplete);
            var missing = Keep(ScriptableObject.CreateInstance<SoundBank>());
            missing.Set(tilePlace, new AudioClip[SoundBank.SoundCount], tracks);
            Assert.IsFalse(missing.IsComplete);
        }

        [Test]
        public void ASoundPlaysOnTheNextVoiceAtItsGroupLevel()
        {
            Assert.IsTrue(player.Play(Sound.Score));
            Assert.IsTrue(player.Play(Sound.UiClick));

            Assert.AreSame(sounds[(int)Sound.Score], player.Voice(0).clip);
            Assert.AreEqual(AudioVolumes.Full, player.Voice(0).volume, Tolerance);
            Assert.AreSame(sounds[(int)Sound.UiClick], player.Voice(1).clip);
            Assert.AreEqual(Half, player.Voice(1).volume, Tolerance);
        }

        [Test]
        public void TheVoicesAreAFixedPoolReusedInTurn()
        {
            Assert.AreEqual(SoundPlayer.Voices, player.VoiceCount);
            var sources = root.GetComponents<AudioSource>().Length;
            for (var i = 0; i < SoundPlayer.Voices; i++) player.Play(Sound.Pass);
            player.Play(Sound.HandEnd);

            Assert.AreSame(sounds[(int)Sound.HandEnd], player.Voice(0).clip);
            Assert.AreSame(sounds[(int)Sound.Pass], player.Voice(1).clip);
            Assert.AreEqual(sources, root.GetComponents<AudioSource>().Length);
        }

        [Test]
        public void AGroupAtZeroStartsNothing()
        {
            volumes.Set(AudioGroup.Sfx, 0f);

            Assert.IsFalse(player.Play(Sound.TilePlace));
            Assert.IsTrue(player.Play(Sound.UiClick));
            Assert.AreSame(sounds[(int)Sound.UiClick], player.Voice(0).clip);
        }

        [Test]
        public void AVolumeChangeReachesThePlayingVoicesOfItsGroup()
        {
            player.Play(Sound.Score);
            player.Play(Sound.UiClick);

            volumes.Set(AudioGroup.Sfx, Half);

            Assert.AreEqual(Half, player.Voice(0).volume, Tolerance);
            Assert.AreEqual(Half, player.Voice(1).volume, Tolerance);
        }

        [Test]
        public void TheTileSoundNeverRepeatsTheSameVariantTwiceInARow()
        {
            var used = new HashSet<AudioClip>();
            AudioClip last = null;
            for (var i = 0; i < 200; i++)
            {
                player.Play(Sound.TilePlace);
                var clip = player.Voice(i % SoundPlayer.Voices).clip;
                Assert.AreNotSame(last, clip, $"play {i}");
                last = clip;
                used.Add(clip);
            }
            Assert.AreEqual(SoundBank.TilePlaceVariants, used.Count);
        }

        [Test]
        public void MusicFadesInOverTheCrossfade()
        {
            player.PlayMusic(Music.Lobby);

            Assert.AreEqual(Music.Lobby, player.Playing);
            Assert.AreSame(tracks[(int)Music.Lobby], player.CurrentMusic.clip);
            Assert.IsTrue(player.CurrentMusic.loop);
            Assert.AreEqual(0f, player.CurrentMusic.volume, Tolerance);
            now = SoundPlayer.CrossfadeMs / 2f;
            player.Tick();
            Assert.AreEqual(Half / 2f, player.CurrentMusic.volume, Tolerance);
            now = SoundPlayer.CrossfadeMs;
            player.Tick();
            Assert.AreEqual(Half, player.CurrentMusic.volume, Tolerance);
        }

        [Test]
        public void ANewTrackCrossfadesFromTheOldOne()
        {
            player.PlayMusic(Music.Lobby);
            now = SoundPlayer.CrossfadeMs;
            player.Tick();
            var lobby = player.CurrentMusic;

            player.PlayMusic(Music.Match);
            now += SoundPlayer.CrossfadeMs / 2f;
            player.Tick();

            Assert.AreSame(lobby, player.OutgoingMusic);
            Assert.AreSame(tracks[(int)Music.Match], player.CurrentMusic.clip);
            Assert.AreEqual(Half / 2f, player.CurrentMusic.volume, Tolerance);
            Assert.AreEqual(Half / 2f, player.OutgoingMusic.volume, Tolerance);
            now += SoundPlayer.CrossfadeMs / 2f;
            player.Tick();
            Assert.IsNull(player.OutgoingMusic.clip);
            Assert.AreEqual(Half, player.CurrentMusic.volume, Tolerance);
        }

        [Test]
        public void TheSameTrackKeepsPlaying()
        {
            player.PlayMusic(Music.Match);
            var source = player.CurrentMusic;
            player.PlayMusic(Music.Match);
            Assert.AreSame(source, player.CurrentMusic);
        }

        [Test]
        public void MutingMusicStopsItAndRaisingItAgainResumesTheWantedTrack()
        {
            player.PlayMusic(Music.Match);

            volumes.Set(AudioGroup.Music, 0f);
            Assert.AreEqual(Music.None, player.Playing);
            Assert.IsNull(player.CurrentMusic.clip);
            Assert.IsNull(player.OutgoingMusic.clip);

            volumes.Set(AudioGroup.Music, Half);
            Assert.AreEqual(Music.Match, player.Playing);
            Assert.AreSame(tracks[(int)Music.Match], player.CurrentMusic.clip);
        }

        [Test]
        public void WhilePausedNothingStartsAndTheMusicResumesAfter()
        {
            player.PlayMusic(Music.Lobby);
            player.SetPaused(true);

            Assert.IsTrue(player.Paused);
            Assert.IsFalse(player.Play(Sound.Score));
            player.PlayMusic(Music.Match);
            Assert.AreEqual(Music.Lobby, player.Playing);
            Assert.AreEqual(Half, player.CurrentMusic.volume, Tolerance, "the fade lands when the app pauses");

            player.SetPaused(false);
            Assert.AreEqual(Music.Match, player.Playing);
            Assert.IsTrue(player.Play(Sound.Score));
        }

        [Test]
        public void MatchSoundsPlayEachFlaggedSound()
        {
            SoundCues.Play(MatchSounds.TilePlace | MatchSounds.Score, player);

            CollectionAssert.Contains(tilePlace, player.Voice(0).clip);
            Assert.AreSame(sounds[(int)Sound.Score], player.Voice(1).clip);
            Assert.IsNull(player.Voice(2).clip);
        }

        [Test]
        public void TheTurnWarningFlagPlaysTheTurnWarningSound()
        {
            SoundCues.Play(MatchSounds.TurnWarning, player);

            Assert.AreSame(sounds[(int)Sound.TurnWarning], player.Voice(0).clip);
            Assert.IsNull(player.Voice(1).clip);
        }

        [Test]
        public void ClickingAnInteractableButtonPlaysTheClick()
        {
            player.MakeMain();
            var button = new GameObject("Button", typeof(RectTransform), typeof(Image), typeof(Button)).GetComponent<Button>();
            button.transform.SetParent(root.transform, false);
            var click = button.gameObject.AddComponent<ClickSound>();
            var press = new PointerEventData(null) { button = PointerEventData.InputButton.Left };

            button.interactable = false;
            click.OnPointerClick(press);
            Assert.IsNull(player.Voice(0).clip);

            button.interactable = true;
            click.OnPointerClick(press);
            Assert.AreSame(sounds[(int)Sound.UiClick], player.Voice(0).clip);
        }

        [Test]
        public void PlayingAndFadingAllocateNothing()
        {
            player.PlayMusic(Music.Lobby);
            player.Play(Sound.TilePlace);
            player.Play(Sound.Score);

            Assert.That(() =>
            {
                player.Play(Sound.TilePlace);
                player.Play(Sound.Score);
                now += 16f;
                player.Tick();
            }, Allocates.Nothing);
        }
    }
}
