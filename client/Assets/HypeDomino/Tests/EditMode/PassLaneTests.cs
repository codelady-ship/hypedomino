using System;
using HypeDomino.Game.Pass;
using HypeDomino.UI.Cosmetics;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PassLaneTests
    {
        const string ReplyJson = @"{""season"":""season_1"",""starts_at"":""2026-12-01T00:00:00Z"",""ends_at"":""2027-03-01T00:00:00Z"",""config_version"":4,""levels"":3,""xp"":150,""level"":2,""level_xp"":100,""next_level_xp"":300,""premium"":false,""tracks"":{""free"":{""1"":{""id"":""bundle.pass_season_1_free_1"",""revision"":1,""coins"":30,""cosmetics"":null,""medals"":null,""passXp"":0},""3"":{""id"":""bundle.pass_season_1_free_3"",""revision"":1,""coins"":0,""cosmetics"":[""emoji.s1_cheer""],""medals"":null,""passXp"":0}},""premium"":{""1"":{""id"":""bundle.pass_season_1_premium_1"",""revision"":1,""coins"":0,""crystals"":6,""cosmetics"":[""avatar.s1_navigator"",""avatar.s1_captain"",""avatar.s1_unknown"",""emoji.s1_storm""],""medals"":null,""passXp"":0},""2"":{""id"":""bundle.pass_season_1_premium_2"",""revision"":1,""coins"":0,""crystals"":6,""cosmetics"":null,""medals"":null,""passXp"":0}}}}";

        const string ManifestJson = @"{""schemaVersion"":1,""manifestVersion"":7,""items"":[
{""id"":""avatar.s1_navigator"",""type"":""avatar"",""rarityId"":""epic"",""availability"":""live"",""animated"":true,""staticAddress"":""cosmetic/avatar/s1_navigator/static"",""animationAddress"":""cosmetic/avatar/s1_navigator/anim"",""sources"":[{""type"":""pass"",""ref"":""pass.season_1.premium.1""}]},
{""id"":""avatar.s1_captain"",""type"":""avatar"",""rarityId"":""rare"",""availability"":""disabled"",""staticAddress"":""cosmetic/avatar/s1_captain/static"",""sources"":[{""type"":""pass"",""ref"":""pass.season_1.premium.1""}]},
{""id"":""emoji.s1_storm"",""type"":""emoji"",""rarityId"":""common"",""availability"":""retired"",""staticAddress"":""cosmetic/emoji/s1_storm/static"",""sources"":[{""type"":""pass"",""ref"":""pass.season_1.premium.1""}]},
{""id"":""emoji.s1_cheer"",""type"":""emoji"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/emoji/s1_cheer/static"",""sources"":[{""type"":""pass"",""ref"":""pass.season_1.free.3""}]}]}";

        static PassLane Lane(bool premium = false)
        {
            var json = premium ? ReplyJson.Replace(@"""premium"":false", @"""premium"":true") : ReplyJson;
            Assert.IsTrue(PassStanding.TryRead(PassReply.FromJson(json), out var standing));
            Assert.IsTrue(PassTracks.TryRead(json, standing.Levels, out var tracks));
            Assert.IsTrue(CosmeticManifest.TryParse(ManifestJson, out var manifest));
            return PassLane.Of(standing, tracks, manifest);
        }

        [Test]
        public void EachRowIsOneLevelOfTheSeason()
        {
            var lane = Lane();

            Assert.AreEqual(3, lane.Count);
            for (var row = 0; row < lane.Count; row++)
            {
                Assert.AreEqual(row + 1, lane.CellAt(PassTrack.Free, row).Level);
                Assert.AreEqual(row + 1, lane.CellAt(PassTrack.Premium, row).Level);
            }
        }

        [Test]
        public void TheFreeTrackIsEarnedUpToTheServerLevel()
        {
            var lane = Lane();

            Assert.AreEqual(PassRewardState.Earned, lane.CellAt(PassTrack.Free, 0).State);
            Assert.AreEqual(PassRewardState.Earned, lane.CellAt(PassTrack.Free, 1).State);
            Assert.AreEqual(PassRewardState.Upcoming, lane.CellAt(PassTrack.Free, 2).State);
            Assert.IsFalse(lane.CellAt(PassTrack.Free, 2).IsLocked);
        }

        [Test]
        public void EveryPremiumCellIsLockedWithoutPremium()
        {
            var lane = Lane();

            for (var row = 0; row < lane.Count; row++)
            {
                Assert.AreEqual(PassRewardState.Locked, lane.CellAt(PassTrack.Premium, row).State);
                Assert.IsTrue(lane.CellAt(PassTrack.Premium, row).IsLocked);
            }
        }

        [Test]
        public void PremiumCellsFollowTheServerLevelOnceThePassIsPremium()
        {
            var lane = Lane(premium: true);

            Assert.AreEqual(PassRewardState.Earned, lane.CellAt(PassTrack.Premium, 1).State);
            Assert.AreEqual(PassRewardState.Upcoming, lane.CellAt(PassTrack.Premium, 2).State);
            Assert.IsFalse(lane.CellAt(PassTrack.Premium, 0).IsLocked);
        }

        [Test]
        public void EachCellCarriesTheRewardTheServerSent()
        {
            var lane = Lane();

            var coins = lane.CellAt(PassTrack.Free, 0);
            Assert.IsTrue(coins.HasReward);
            Assert.AreEqual("bundle.pass_season_1_free_1", coins.Reward.Id);
            Assert.AreEqual(30, coins.Reward.Coins);
            Assert.AreEqual(0, coins.CosmeticCount);

            var crystals = lane.CellAt(PassTrack.Premium, 1);
            Assert.AreEqual("bundle.pass_season_1_premium_2", crystals.Reward.Id);
            Assert.AreEqual(6, crystals.Reward.Crystals);
        }

        [Test]
        public void ALevelTheServerLeftEmptyHasNoReward()
        {
            var empty = Lane().CellAt(PassTrack.Free, 1);

            Assert.IsFalse(empty.HasReward);
            Assert.IsNull(empty.Reward);
            Assert.AreEqual(0, empty.CosmeticCount);
            Assert.AreEqual(PassRewardState.Earned, empty.State);
        }

        [Test]
        public void CosmeticsShowTheirManifestItemWithItsAnimation()
        {
            var cell = Lane().CellAt(PassTrack.Premium, 0);

            var navigator = cell.CosmeticAt(0);
            Assert.AreEqual("avatar.s1_navigator", navigator.Id);
            Assert.AreEqual("epic", navigator.RarityId);
            Assert.IsTrue(navigator.Art.IsAnimated);
            Assert.AreEqual("cosmetic/avatar/s1_navigator/anim", navigator.Art.AnimationAddress);

            var cheer = Lane().CellAt(PassTrack.Free, 2);
            Assert.AreEqual(1, cheer.CosmeticCount);
            Assert.AreEqual("emoji.s1_cheer", cheer.CosmeticAt(0).Id);
            Assert.IsFalse(cheer.CosmeticAt(0).Art.IsAnimated);
        }

        [Test]
        public void ACosmeticTheManifestDoesNotRenderIsLeftOut()
        {
            var cell = Lane().CellAt(PassTrack.Premium, 0);

            Assert.AreEqual(2, cell.CosmeticCount);
            Assert.AreEqual("avatar.s1_navigator", cell.CosmeticAt(0).Id);
            Assert.AreEqual("emoji.s1_storm", cell.CosmeticAt(1).Id);
            Assert.AreEqual(4, cell.Reward.Cosmetics.Count);
        }

        [Test]
        public void ALaneNeedsTheStandingTheTracksAndTheManifest()
        {
            Assert.IsTrue(PassStanding.TryRead(PassReply.FromJson(ReplyJson), out var standing));
            Assert.IsTrue(PassTracks.TryRead(ReplyJson, standing.Levels, out var tracks));
            Assert.IsTrue(CosmeticManifest.TryParse(ManifestJson, out var manifest));

            Assert.Throws<ArgumentNullException>(() => PassLane.Of(null, tracks, manifest));
            Assert.Throws<ArgumentNullException>(() => PassLane.Of(standing, null, manifest));
            Assert.Throws<ArgumentNullException>(() => PassLane.Of(standing, tracks, null));
        }

        [Test]
        public void ReadingEveryCellAllocatesNothing()
        {
            var lane = Lane(premium: true);
            TestDelegate bindAll = () =>
            {
                for (var row = 0; row < lane.Count; row++)
                {
                    BindCell(lane.CellAt(PassTrack.Free, row));
                    BindCell(lane.CellAt(PassTrack.Premium, row));
                }
            };
            bindAll();

            Assert.That(bindAll, Allocates.Nothing);
        }

        static void BindCell(in PassCell cell)
        {
            _ = cell.Level;
            _ = cell.State;
            _ = cell.IsLocked;
            if (!cell.HasReward) return;
            _ = cell.Reward.Coins;
            _ = cell.Reward.Crystals;
            for (var at = 0; at < cell.CosmeticCount; at++) _ = cell.CosmeticAt(at).Art;
        }
    }
}
