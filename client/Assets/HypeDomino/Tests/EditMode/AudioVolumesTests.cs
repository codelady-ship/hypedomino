using HypeDomino.UI.Audio;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AudioVolumesTests
    {
        int changes;

        AudioVolumes Watched(float music, float sfx, float ui)
        {
            changes = 0;
            var volumes = new AudioVolumes(music, sfx, ui);
            volumes.Changed += () => changes++;
            return volumes;
        }

        [Test]
        public void EachGroupKeepsItsOwnLevel()
        {
            var volumes = Watched(0.25f, 0.5f, 0.75f);
            Assert.AreEqual(0.25f, volumes.Level(AudioGroup.Music));
            Assert.AreEqual(0.5f, volumes.Level(AudioGroup.Sfx));
            Assert.AreEqual(0.75f, volumes.Level(AudioGroup.Ui));
        }

        [TestCase(-0.5f, 0f)]
        [TestCase(1.5f, 1f)]
        [TestCase(float.NaN, 0f)]
        [TestCase(float.PositiveInfinity, 1f)]
        public void LevelsStayBetweenOffAndFull(float level, float expected)
        {
            Assert.AreEqual(expected, Watched(level, AudioVolumes.Full, AudioVolumes.Full).Level(AudioGroup.Music));
            var volumes = Watched(AudioVolumes.Full, AudioVolumes.Full, AudioVolumes.Full);
            volumes.Set(AudioGroup.Sfx, level);
            Assert.AreEqual(expected, volumes.Level(AudioGroup.Sfx));
        }

        [Test]
        public void AGroupAtZeroIsOffAndAReducedGroupStillPlays()
        {
            var volumes = Watched(0f, 0.1f, AudioVolumes.Full);
            Assert.IsFalse(volumes.Audible(AudioGroup.Music));
            Assert.IsTrue(volumes.Audible(AudioGroup.Sfx));
            Assert.IsTrue(volumes.Audible(AudioGroup.Ui));
            Assert.IsFalse(volumes.Silent);
        }

        [Test]
        public void SilentOnlyWhenEveryGroupIsOff()
        {
            var volumes = Watched(0f, 0f, AudioVolumes.Full);
            Assert.IsFalse(volumes.Silent);
            volumes.Set(AudioGroup.Ui, 0f);
            Assert.IsTrue(volumes.Silent);
            volumes.Set(AudioGroup.Music, 0.5f);
            Assert.IsFalse(volumes.Silent);
        }

        [Test]
        public void SettingTheSameLevelAgainChangesNothing()
        {
            var volumes = Watched(AudioVolumes.Full, AudioVolumes.Full, AudioVolumes.Full);
            volumes.Set(AudioGroup.Music, AudioVolumes.Full);
            volumes.Set(AudioGroup.Sfx, 2f);
            Assert.AreEqual(0, changes);
            volumes.Set(AudioGroup.Music, 0.5f);
            volumes.Set(AudioGroup.Music, 0.5f);
            Assert.AreEqual(1, changes);
        }
    }
}
