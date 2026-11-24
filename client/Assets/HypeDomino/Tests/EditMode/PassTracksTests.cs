using HypeDomino.Game.Pass;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PassTracksTests
    {
        const string ReplyJson = @"{""season"":""season_1"",""starts_at"":""2026-12-01T00:00:00Z"",""ends_at"":""2027-03-01T00:00:00Z"",""config_version"":4,""levels"":100,""xp"":1700,""level"":5,""level_xp"":1500,""next_level_xp"":2000,""premium"":false,""tracks"":{""free"":{""1"":{""id"":""bundle.pass_season_1_free_1"",""revision"":1,""coins"":30,""cosmetics"":null,""medals"":null,""passXp"":0},""10"":{""id"":""bundle.pass_season_1_free_10"",""revision"":1,""coins"":0,""cosmetics"":[""emoji.s1_wave""],""medals"":null,""passXp"":0}},""premium"":{""1"":{""id"":""bundle.pass_season_1_premium_1"",""revision"":1,""coins"":0,""crystals"":6,""cosmetics"":[""avatar.s1_navigator""],""medals"":null,""passXp"":0},""2"":{""id"":""bundle.pass_season_1_premium_2"",""revision"":1,""coins"":0,""crystals"":6,""cosmetics"":null,""medals"":null,""passXp"":0}}}}";

        const string RewardJson = @"{""id"":""bundle.pass_season_1_free_1"",""revision"":1,""coins"":30,""cosmetics"":null,""medals"":null,""passXp"":0}";

        static PassTracks Read(string json)
        {
            Assert.IsTrue(PassTracks.TryRead(json, 100, out var tracks));
            return tracks;
        }

        static string WithFreeLevel(string key, string reward) => $@"{{""levels"":100,""tracks"":{{""free"":{{""{key}"":{reward}}},""premium"":{{}}}}}}";

        [Test]
        public void EachLevelHasTheRewardTheServerSent()
        {
            var tracks = Read(ReplyJson);

            Assert.IsTrue(tracks.TryRewardAt(PassTrack.Free, 1, out var coins));
            Assert.AreEqual("bundle.pass_season_1_free_1", coins.Id);
            Assert.AreEqual(30, coins.Coins);
            Assert.AreEqual(0, coins.Crystals);
            Assert.IsEmpty(coins.Cosmetics);

            Assert.IsTrue(tracks.TryRewardAt(PassTrack.Free, 10, out var emoji));
            Assert.AreEqual(0, emoji.Coins);
            CollectionAssert.AreEqual(new[] { "emoji.s1_wave" }, emoji.Cosmetics);

            Assert.IsTrue(tracks.TryRewardAt(PassTrack.Premium, 1, out var avatar));
            Assert.AreEqual("bundle.pass_season_1_premium_1", avatar.Id);
            Assert.AreEqual(6, avatar.Crystals);
            CollectionAssert.AreEqual(new[] { "avatar.s1_navigator" }, avatar.Cosmetics);

            Assert.IsTrue(tracks.TryRewardAt(PassTrack.Premium, 2, out var crystals));
            Assert.AreEqual(6, crystals.Crystals);
            Assert.IsEmpty(crystals.Cosmetics);
        }

        [TestCase(PassTrack.Free, 2)]
        [TestCase(PassTrack.Free, 100)]
        [TestCase(PassTrack.Premium, 3)]
        [TestCase(PassTrack.Free, 0)]
        [TestCase(PassTrack.Premium, 101)]
        public void ALevelTheServerLeftEmptyHasNoReward(PassTrack track, int level)
        {
            Assert.IsFalse(Read(ReplyJson).TryRewardAt(track, level, out var reward));
            Assert.IsNull(reward);
        }

        [TestCase(@"{""tracks"":{}}")]
        [TestCase(@"{""tracks"":{""free"":null,""premium"":null}}")]
        public void AMissingOrNullTrackIsEmpty(string json)
        {
            var tracks = Read(json);

            Assert.IsFalse(tracks.TryRewardAt(PassTrack.Free, 1, out _));
            Assert.IsFalse(tracks.TryRewardAt(PassTrack.Premium, 1, out _));
        }

        [Test]
        public void TheStandingAndTheTracksReadTheSameReply()
        {
            Assert.IsTrue(PassStanding.TryRead(PassReply.FromJson(ReplyJson), out var standing));
            Assert.IsTrue(PassTracks.TryRead(ReplyJson, standing.Levels, out var tracks));

            Assert.IsTrue(tracks.TryRewardAt(PassTrack.Premium, 1, out _));
            Assert.AreEqual(PassRewardState.Locked, standing.StateAt(PassTrack.Premium, 1));
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("{")]
        [TestCase(@"{""season"":""season_1""}")]
        [TestCase(@"{""tracks"":null}")]
        [TestCase(@"{""tracks"":[]}")]
        [TestCase(@"{""tracks"":{""free"":[]}}")]
        [TestCase(@"{""tracks"":{""premium"":{""1"":30}}}")]
        public void AReplyWithoutReadableTracksIsRefused(string json)
        {
            Assert.IsFalse(PassTracks.TryRead(json, 100, out var tracks));
            Assert.IsNull(tracks);
        }

        [TestCase("0")]
        [TestCase("101")]
        [TestCase("-1")]
        [TestCase("+1")]
        [TestCase("01")]
        [TestCase(" 1")]
        [TestCase("one")]
        public void ALevelOutsideTheSeasonIsRefused(string key)
        {
            Assert.IsTrue(PassTracks.TryRead(WithFreeLevel("1", RewardJson), 100, out _));
            Assert.IsFalse(PassTracks.TryRead(WithFreeLevel(key, RewardJson), 100, out _));
        }

        [TestCase(@"{""coins"":30}")]
        [TestCase(@"{""id"":"""",""coins"":30}")]
        [TestCase(@"{""id"":7,""coins"":30}")]
        [TestCase(@"{""id"":""bundle.a""}")]
        [TestCase(@"{""id"":""bundle.a"",""coins"":-1}")]
        [TestCase(@"{""id"":""bundle.a"",""coins"":1.5}")]
        [TestCase(@"{""id"":""bundle.a"",""coins"":""30""}")]
        [TestCase(@"{""id"":""bundle.a"",""coins"":0,""crystals"":-6}")]
        [TestCase(@"{""id"":""bundle.a"",""coins"":0,""crystals"":""6""}")]
        [TestCase(@"{""id"":""bundle.a"",""coins"":0,""cosmetics"":""avatar.fox""}")]
        [TestCase(@"{""id"":""bundle.a"",""coins"":0,""cosmetics"":[7]}")]
        public void ARewardThatDoesNotHoldTogetherIsRefused(string reward)
        {
            Assert.IsFalse(PassTracks.TryRead(WithFreeLevel("1", reward), 100, out var tracks));
            Assert.IsNull(tracks);
        }
    }
}
