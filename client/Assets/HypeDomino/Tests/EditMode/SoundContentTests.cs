using System.IO;
using System.Linq;
using HypeDomino.Editor;
using HypeDomino.UI.Audio;
using NUnit.Framework;
using UnityEditor;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SoundContentTests
    {
        static readonly string[] SfxNames =
        {
            "combo_1", "combo_2", "combo_3", "combo_4", "combo_5",
            "hand_end", "match_lose", "match_win", "pass", "score", "tile_draw",
            "tile_place_01", "tile_place_02", "tile_place_03", "tile_place_04", "turn_warning", "ui_click",
        };

        static readonly string[] MusicNames = { "lobby", "match" };

        [Test]
        public void TheClientHoldsEverySoundOfTheSpec()
        {
            CollectionAssert.AreEqual(SfxNames, SoundContent.SfxPaths().Select(Path.GetFileNameWithoutExtension).ToArray());
            CollectionAssert.AreEqual(MusicNames, SoundContent.MusicPaths().Select(Path.GetFileNameWithoutExtension).ToArray());
        }

        [Test]
        public void ShortSoundsAreMonoAdpcmDecompressedAndPreloaded()
        {
            foreach (var path in SoundContent.SfxPaths())
            {
                var importer = (AudioImporter)AssetImporter.GetAtPath(path);
                var settings = importer.defaultSampleSettings;
                Assert.IsTrue(importer.forceToMono, path);
                Assert.IsFalse(SoundContent.Normalizes(importer), path);
                Assert.AreEqual(AudioClipLoadType.DecompressOnLoad, settings.loadType, path);
                Assert.AreEqual(AudioCompressionFormat.ADPCM, settings.compressionFormat, path);
                Assert.IsTrue(settings.preloadAudioData, path);
            }
        }

        [Test]
        public void MusicIsStereoVorbisStreamedWithoutPreload()
        {
            foreach (var path in SoundContent.MusicPaths())
            {
                var importer = (AudioImporter)AssetImporter.GetAtPath(path);
                var settings = importer.defaultSampleSettings;
                Assert.IsFalse(importer.forceToMono, path);
                Assert.AreEqual(AudioClipLoadType.Streaming, settings.loadType, path);
                Assert.AreEqual(AudioCompressionFormat.Vorbis, settings.compressionFormat, path);
                Assert.IsFalse(settings.preloadAudioData, path);
                Assert.AreEqual(2, AssetDatabase.LoadAssetAtPath<AudioClip>(path).channels, path);
            }
        }

        [Test]
        public void TheBankGivesEachSoundAndTrackItsFile()
        {
            var bank = AssetDatabase.LoadAssetAtPath<SoundBank>(SoundContent.BankPath);
            Assert.IsNotNull(bank, SoundContent.BankPath);
            Assert.IsTrue(bank.IsComplete);
            Assert.AreEqual(SoundBank.TilePlaceVariants, bank.Variants(Sound.TilePlace));
            for (var i = 0; i < SoundBank.TilePlaceVariants; i++)
                Assert.AreSame(Clip(SoundContent.TilePlacePath(i)), bank.Clip(Sound.TilePlace, i), $"tile_place {i}");
            for (var sound = Sound.TileDraw; sound <= Sound.Combo5; sound++)
                Assert.AreSame(Clip(SoundContent.SoundPath(sound)), bank.Clip(sound, 0), sound.ToString());
            Assert.AreSame(Clip(SoundContent.TrackPath(Music.Lobby)), bank.Track(Music.Lobby));
            Assert.AreSame(Clip(SoundContent.TrackPath(Music.Match)), bank.Track(Music.Match));
        }

        [Test]
        public void TheBankLoadsFromResources() => Assert.AreSame(AssetDatabase.LoadAssetAtPath<SoundBank>(SoundContent.BankPath), SoundBank.Load());

        static AudioClip Clip(string path)
        {
            var clip = AssetDatabase.LoadAssetAtPath<AudioClip>(path);
            Assert.IsNotNull(clip, path);
            return clip;
        }
    }
}
