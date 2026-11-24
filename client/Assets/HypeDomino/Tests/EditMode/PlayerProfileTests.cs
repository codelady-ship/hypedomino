using System.IO;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Profile;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PlayerProfileTests
    {
        const string OwnModes = @"{""1v1"":{""games"":30,""wins"":17,""losses"":13},""2v2"":{""games"":10,""wins"":5,""losses"":5}}";

        const string PublicRanks = @"{""1v1"":""Gold II"",""2v2"":""Master""}";

        const string OwnJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":""Leyla"",""setup_required"":false,""rename_required"":false,
""rename_available_at"":""2026-12-01T10:00:00Z"",""level"":7,""xp"":1250,
""loadout"":{""avatar"":""avatar.comet_fox"",""frame"":""frame.season_1_top"",""table"":""table.starter"",""tile"":""tile.starter"",""emojis"":[""emoji.smile""]},
""stats"":{""games"":40,""wins"":22,""losses"":18,""win_rate_percent"":55,""streak"":3,""best_streak"":6,
""modes"":" + OwnModes + "}}";

        const string SetupJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":"""",""setup_required"":true,""rename_required"":false,
""suggested_nickname"":""Player4821"",""level"":1,""xp"":0,""loadout"":{},""stats"":{""games"":0,""wins"":0,""losses"":0,""win_rate_percent"":0,""streak"":0,""best_streak"":0,""modes"":{}}}";

        const string PublicJson = @"{""account_id"":""3b6a9e21-7c0d-4e55-8f12-9a0b4c3d2e1f"",""nickname"":""Rauf"",""level"":12,
""loadout"":{""avatar"":""avatar.moon_owl"",""frame"":""frame.starter""},
""stats"":{""games"":80,""win_rate_percent"":61,""best_streak"":9},
""ranked"":{""ranks"":" + PublicRanks + @",""emblems"":[
{""id"":""1v1:season_1:global"",""season"":""season_1"",""mode"":""1v1"",""board"":""global"",""bracket"":100,""position"":57,""awarded_at"":""2026-10-01T00:00:00Z""},
{""id"":""2v2:season_1:country:AZ"",""season"":""season_1"",""mode"":""2v2"",""board"":""country:AZ"",""bracket"":10,""position"":4,""awarded_at"":""2026-10-01T00:00:00Z""}]},
""medals"":[
{""id"":""medal.first_win"",""revision"":2,""nameKey"":""medal.first_win.name"",""descriptionKey"":""medal.first_win.description"",""rarityId"":""rare"",""staticAddress"":""medal/first_win/static"",""sources"":[{""type"":""mission"",""ref"":""mission.achievement_wins_1""}],""availability"":""live""},
{""id"":""medal.season_1_gold"",""revision"":1,""nameKey"":""medal.season_1_gold.name"",""descriptionKey"":""medal.season_1_gold.description"",""rarityId"":""epic"",""staticAddress"":""medal/season_1_gold/static"",""sources"":[],""availability"":""retired""}]}";

        const string BarePublicJson = @"{""account_id"":""3b6a9e21-7c0d-4e55-8f12-9a0b4c3d2e1f"",""nickname"":""Rauf"",""level"":1,""loadout"":{},""stats"":{""games"":0,""win_rate_percent"":0,""best_streak"":0}}";

        static PlayerProfile Own(string json = OwnJson)
        {
            Assert.IsTrue(PlayerProfile.TryFromOwn(json, out var profile));
            return profile;
        }

        static PlayerProfile Public(string json = PublicJson)
        {
            Assert.IsTrue(PlayerProfile.TryFromPublic(json, out var profile));
            return profile;
        }

        const string RenameAppealJson = @",""rename_appeal"":{""sanction_id"":42,""token"":""appeal-1"",""expires_at"":""2026-11-22T10:30:00Z""}";

        [Test]
        public void AForcedRenameOffersItsAppeal()
        {
            var profile = Own(OwnJson.Replace(@"""rename_required"":false", @"""rename_required"":true" + RenameAppealJson));

            Assert.IsTrue(profile.RenameRequired);
            Assert.IsTrue(profile.RenameAppeal.Offered);
            Assert.AreEqual(42, profile.RenameAppeal.SanctionId);
            Assert.AreEqual("appeal-1", profile.RenameAppeal.Token);
        }

        [Test]
        public void OnlyAPendingForcedRenameOffersAnAppeal()
        {
            Assert.IsFalse(Own().RenameAppeal.Offered);
            Assert.IsFalse(Own(OwnJson.Replace(@"""rename_required"":false", @"""rename_required"":false" + RenameAppealJson)).RenameAppeal.Offered);
            Assert.IsFalse(Own(OwnJson.Replace(@"""rename_required"":false", @"""rename_required"":true")).RenameAppeal.Offered);
        }

        static void AssertStats(PlayerProfile profile, params (ProfileStat stat, int value)[] expected)
        {
            Assert.AreEqual(expected.Length, profile.StatCount);
            for (var i = 0; i < expected.Length; i++)
            {
                Assert.AreEqual(expected[i].stat, profile.Stat(i).Stat);
                Assert.AreEqual(expected[i].value, profile.Stat(i).Value);
            }
        }

        [Test]
        public void TheOwnProfileShowsTheServersIdentityLevelAndEveryOwnStat()
        {
            var profile = Own();

            Assert.IsTrue(profile.IsOwn);
            Assert.AreEqual("8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10", profile.AccountId);
            Assert.AreEqual("Leyla", profile.Nickname);
            Assert.AreEqual(7, profile.Level);
            Assert.AreEqual(1250, profile.Xp);
            Assert.AreEqual("avatar.comet_fox", profile.AvatarId);
            Assert.AreEqual("frame.season_1_top", profile.FrameId);
            AssertStats(profile, (ProfileStat.Games, 40), (ProfileStat.Wins, 22), (ProfileStat.Losses, 18), (ProfileStat.WinRatePercent, 55), (ProfileStat.Streak, 3), (ProfileStat.BestStreak, 6));
            Assert.AreEqual(0, profile.MedalCount);
            Assert.AreEqual(0, profile.EmblemCount);
        }

        [Test]
        public void TheWinRateIsTheServersValueAndNeverWorkedOutOnTheClient()
        {
            var own = Own(OwnJson.Replace(@"""win_rate_percent"":55", @"""win_rate_percent"":99"));
            var other = Public(PublicJson.Replace(@"""win_rate_percent"":61", @"""win_rate_percent"":7"));

            Assert.AreEqual(99, own.Stat(3).Value);
            Assert.AreEqual(7, other.Stat(1).Value);
        }

        [Test]
        public void AnOwnReplyWithoutAProfileIsNotShown()
        {
            Assert.IsFalse(PlayerProfile.TryFromOwn(SetupJson, out var setup));
            Assert.IsNull(setup);
            Assert.IsFalse(PlayerProfile.TryFromOwn(null, out _));
            Assert.IsFalse(PlayerProfile.TryFromOwn(OwnJson.Replace(@"""nickname"":""Leyla""", @"""nickname"":"""""), out _));
        }

        [Test]
        public void TheOwnProfileShowsTheServersCountersOfEachModeInTheServersOrder()
        {
            var profile = Own();

            Assert.AreEqual(2, profile.ModeStatsCount);
            Assert.AreEqual("1v1", profile.ModeStats(0).Mode);
            Assert.AreEqual(30, profile.ModeStats(0).Games);
            Assert.AreEqual(17, profile.ModeStats(0).Wins);
            Assert.AreEqual(13, profile.ModeStats(0).Losses);
            Assert.AreEqual("2v2", profile.ModeStats(1).Mode);
            Assert.AreEqual(10, profile.ModeStats(1).Games);
            Assert.AreEqual(5, profile.ModeStats(1).Wins);
            Assert.AreEqual(5, profile.ModeStats(1).Losses);
            Assert.AreEqual(0, profile.RankCount);
        }

        [TestCase(@"{}")]
        [TestCase(@"null")]
        public void AnOwnProfileWithoutModeCountersShowsNone(string modes)
        {
            Assert.AreEqual(0, Own(OwnJson.Replace(OwnModes, modes)).ModeStatsCount);
        }

        [TestCase(@"[]")]
        [TestCase(@"{""1v1"":[]}")]
        [TestCase(@"{""1v1"":30}")]
        [TestCase(@"{"""":{""games"":1,""wins"":1,""losses"":0}}")]
        [TestCase(@"{""1v1"":{""wins"":17,""losses"":13}}")]
        [TestCase(@"{""1v1"":{""games"":30,""losses"":13}}")]
        [TestCase(@"{""1v1"":{""games"":30,""wins"":17}}")]
        [TestCase(@"{""1v1"":{""games"":-1,""wins"":0,""losses"":0}}")]
        [TestCase(@"{""1v1"":{""games"":1.5,""wins"":1,""losses"":0}}")]
        [TestCase(@"{""1v1"":{""games"":""30"",""wins"":17,""losses"":13}}")]
        [TestCase(@"{""1v1"":{""games"":2147483648,""wins"":0,""losses"":0}}")]
        public void AnOwnReplyWithUnreadableModeCountersIsNotShown(string modes)
        {
            Assert.IsFalse(PlayerProfile.TryFromOwn(OwnJson.Replace(OwnModes, modes), out var profile));
            Assert.IsNull(profile);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("{")]
        [TestCase("[]")]
        public void AReplyThatIsNotAJsonObjectIsNotShown(string json)
        {
            Assert.IsFalse(PlayerProfile.TryFromOwn(json, out var own));
            Assert.IsNull(own);
            Assert.IsFalse(PlayerProfile.TryFromPublic(json, out var other));
            Assert.IsNull(other);
        }

        [Test]
        public void AnotherPlayersProfileShowsTheRankOfEachPlacedModeInTheServersOrder()
        {
            var profile = Public();

            Assert.AreEqual(2, profile.RankCount);
            Assert.AreEqual("1v1", profile.Rank(0).Mode);
            Assert.AreEqual("Gold II", profile.Rank(0).Label);
            Assert.AreEqual("2v2", profile.Rank(1).Mode);
            Assert.AreEqual("Master", profile.Rank(1).Label);
            Assert.AreEqual(0, profile.ModeStatsCount);
        }

        [Test]
        public void AProfileWithoutAPlacedModeShowsNoRank()
        {
            var unplaced = Public(BarePublicJson.Replace(@"}}", @"},""ranked"":{""ranks"":{},""emblems"":null}}"));

            Assert.AreEqual(0, unplaced.RankCount);
            Assert.AreEqual(0, unplaced.EmblemCount);
            Assert.AreEqual(0, Public(BarePublicJson).RankCount);
        }

        [TestCase(@"[]")]
        [TestCase(@"{""1v1"":7}")]
        [TestCase(@"{""1v1"":""""}")]
        [TestCase(@"{""1v1"":null}")]
        [TestCase(@"{"""":""Gold II""}")]
        public void APublicReplyWithUnreadableRanksIsNotShown(string ranks)
        {
            Assert.IsFalse(PlayerProfile.TryFromPublic(PublicJson.Replace(PublicRanks, ranks), out var profile));
            Assert.IsNull(profile);
        }

        [Test]
        public void APublicReplyWhoseRankedSectionIsNotAnObjectIsNotShown()
        {
            Assert.IsFalse(PlayerProfile.TryFromPublic(BarePublicJson.Replace(@"}}", @"},""ranked"":[]}"), out _));
        }

        [Test]
        public void AnotherPlayersProfileShowsOnlyThePublicStatsAndNoXp()
        {
            var profile = Public();

            Assert.IsFalse(profile.IsOwn);
            Assert.AreEqual("3b6a9e21-7c0d-4e55-8f12-9a0b4c3d2e1f", profile.AccountId);
            Assert.AreEqual("Rauf", profile.Nickname);
            Assert.AreEqual(12, profile.Level);
            Assert.AreEqual(0, profile.Xp);
            Assert.AreEqual("avatar.moon_owl", profile.AvatarId);
            Assert.AreEqual("frame.starter", profile.FrameId);
            AssertStats(profile, (ProfileStat.Games, 80), (ProfileStat.WinRatePercent, 61), (ProfileStat.BestStreak, 9));
        }

        [Test]
        public void TheShowcasesKeepTheServersOrder()
        {
            var profile = Public();

            Assert.AreEqual(2, profile.MedalCount);
            var first = profile.Medal(0);
            Assert.AreEqual("medal.first_win", first.Id);
            Assert.AreEqual("medal.first_win.name", first.NameKey);
            Assert.AreEqual("rare", first.RarityId);
            Assert.AreEqual("medal/first_win/static", first.StaticAddress);
            Assert.AreEqual("medal.season_1_gold", profile.Medal(1).Id);
            Assert.AreEqual(2, profile.EmblemCount);
            var emblem = profile.Emblem(0);
            Assert.AreEqual("1v1:season_1:global", emblem.Id);
            Assert.AreEqual("season_1", emblem.Season);
            Assert.AreEqual("1v1", emblem.Mode);
            Assert.AreEqual("global", emblem.Board);
            Assert.AreEqual(100, emblem.Bracket);
            Assert.AreEqual(57, emblem.Position);
            Assert.AreEqual("country:AZ", profile.Emblem(1).Board);
        }

        [Test]
        public void AProfileWithoutShowcasesShowsNone()
        {
            var profile = Public(BarePublicJson);

            Assert.AreEqual(0, profile.MedalCount);
            Assert.AreEqual(0, profile.EmblemCount);
            AssertStats(profile, (ProfileStat.Games, 0), (ProfileStat.WinRatePercent, 0), (ProfileStat.BestStreak, 0));
        }

        [Test]
        public void AShowcaseEntryWithoutAnIdIsLeftOut()
        {
            var profile = Public(PublicJson.Replace(@"""id"":""medal.first_win""", @"""id"":""""").Replace(@"""id"":""1v1:season_1:global""", @"""id"":"""""));

            Assert.AreEqual(1, profile.MedalCount);
            Assert.AreEqual("medal.season_1_gold", profile.Medal(0).Id);
            Assert.AreEqual(1, profile.EmblemCount);
            Assert.AreEqual("2v2:season_1:country:AZ", profile.Emblem(0).Id);
        }

        [Test]
        public void APublicReplyWithoutAPlayerIsNotShown()
        {
            Assert.IsFalse(PlayerProfile.TryFromPublic(null, out var missing));
            Assert.IsNull(missing);
            Assert.IsFalse(PlayerProfile.TryFromPublic(@"{""error"":""profile.unknown""}", out _));
        }

        [Test]
        public void TheAvatarAndFrameComeFromTheEquippedIds()
        {
            var manifestJson = File.ReadAllText(Path.GetFullPath(Path.Combine(Application.dataPath, "..", "..", "docs", "cosmetics", "examples", "catalog.json")));
            Assert.IsTrue(CosmeticManifest.TryParse(manifestJson, out var manifest));

            var look = Own().Look(manifest);

            Assert.AreEqual("cosmetic/avatar/comet_fox/anim", look.Avatar.AnimationAddress);
            Assert.AreEqual("cosmetic/frame/season_1_top/static", look.Frame.StaticAddress);
            Assert.AreEqual("epic", look.MarkRarityId);
        }

        [Test]
        public void TheProfilePathsNameTheAccount()
        {
            Assert.AreEqual("/profile/me", PlayerProfile.OwnPath);
            Assert.AreEqual("/profile/3b6a9e21-7c0d-4e55-8f12-9a0b4c3d2e1f", PlayerProfile.PublicPath("3b6a9e21-7c0d-4e55-8f12-9a0b4c3d2e1f"));
            Assert.AreEqual("/profile/a%2Fb", PlayerProfile.PublicPath("a/b"));
        }

        [Test]
        public void ReadingTheProfileAllocatesNothing()
        {
            var profile = Public();
            var own = Own();
            var sink = 0;
            TestDelegate read = () =>
            {
                for (var i = 0; i < profile.StatCount; i++) sink += profile.Stat(i).Value;
                for (var i = 0; i < profile.RankCount; i++) sink += profile.Rank(i).Label.Length;
                for (var i = 0; i < own.ModeStatsCount; i++) sink += own.ModeStats(i).Games;
                for (var i = 0; i < profile.MedalCount; i++) sink += profile.Medal(i).NameKey.Length;
                for (var i = 0; i < profile.EmblemCount; i++) sink += profile.Emblem(i).Position;
                sink += profile.Level + profile.Nickname.Length;
            };
            read();

            Assert.That(read, Allocates.Nothing);
            Assert.Greater(sink, 0);
        }
    }
}
