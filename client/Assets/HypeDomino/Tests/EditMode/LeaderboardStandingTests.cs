using HypeDomino.Game.Ranking;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LeaderboardStandingTests
    {
        const string GlobalJson = @"{""season"":""season.s1"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""top"",""nickname"":""Ayla"",""rank"":""Master"",""lp"":700,""tier"":""Challenger"",""wins"":90},{""position"":2,""account_id"":""second"",""nickname"":""Kamran"",""rank"":""Gold II"",""lp"":50,""tier"":""Gold"",""wins"":10}],""me"":{""position"":2,""account_id"":""second"",""nickname"":""Kamran"",""rank"":""Gold II"",""lp"":50,""tier"":""Gold"",""wins"":10},""around"":[{""position"":1,""account_id"":""top"",""nickname"":""Ayla"",""rank"":""Master"",""lp"":700,""tier"":""Challenger"",""wins"":90},{""position"":2,""account_id"":""second"",""nickname"":""Kamran"",""rank"":""Gold II"",""lp"":50,""tier"":""Gold"",""wins"":10},{""position"":3,""account_id"":""third"",""nickname"":"""",""rank"":""Iron IV"",""lp"":0,""tier"":""Iron"",""wins"":0}]}";
        const string UnlistedJson = @"{""season"":""season.s2"",""mode"":""2v2"",""board"":""country:AZ"",""top"":[],""me"":null,""around"":[]}";

        [Test]
        public void ReadsTheBoardTheServerSends()
        {
            Assert.IsTrue(LeaderboardStanding.TryRead(GlobalJson, out var standing));
            Assert.AreEqual("season.s1", standing.Season);
            Assert.AreEqual("1v1", standing.Mode);
            Assert.AreEqual(LeaderboardStanding.BoardGlobal, standing.Board);
            Assert.AreEqual(2, standing.TopCount);
            Assert.AreEqual(3, standing.AroundCount);

            var first = standing.TopAt(0);
            Assert.AreEqual(1, first.Position);
            Assert.AreEqual("top", first.AccountId);
            Assert.AreEqual("Ayla", first.Nickname);
            Assert.AreEqual(700, first.LP);
            Assert.AreEqual(90, first.Wins);
            Assert.AreEqual(RankTier.Challenger, first.Badge.Tier);
            Assert.AreEqual(RankBadge.NoDivision, first.Badge.Division);

            var second = standing.TopAt(1);
            Assert.AreEqual(RankTier.Gold, second.Badge.Tier);
            Assert.AreEqual(2, second.Badge.Division);

            Assert.IsTrue(standing.Listed);
            Assert.AreEqual(2, standing.Me.Position);
            Assert.AreEqual("second", standing.Me.AccountId);
            Assert.AreEqual("Kamran", standing.Me.Nickname);
            Assert.AreEqual(50, standing.Me.LP);
            Assert.AreEqual("third", standing.AroundAt(2).AccountId);
            Assert.AreEqual(string.Empty, standing.AroundAt(2).Nickname);
            Assert.AreEqual(0, standing.AroundAt(2).LP);
            Assert.AreEqual(RankTier.Iron, standing.AroundAt(2).Badge.Tier);
            Assert.AreEqual(RankBadge.LowestDivision, standing.AroundAt(2).Badge.Division);
        }

        [Test]
        public void AViewerOffTheBoardHasNoOwnRow()
        {
            Assert.IsTrue(LeaderboardStanding.TryRead(UnlistedJson, out var standing));
            Assert.AreEqual("country:AZ", standing.Board);
            Assert.AreEqual(0, standing.TopCount);
            Assert.AreEqual(0, standing.AroundCount);
            Assert.IsFalse(standing.Listed);
        }

        [Test]
        public void ApexRowsTakeTheirTierFromTheServerTier()
        {
            const string json = @"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""a"",""rank"":""Master"",""tier"":""Grandmaster"",""wins"":5},{""position"":2,""account_id"":""b"",""rank"":""Master"",""tier"":""Master"",""wins"":4}]}";

            Assert.IsTrue(LeaderboardStanding.TryRead(json, out var standing));
            Assert.AreEqual(RankTier.Grandmaster, standing.TopAt(0).Badge.Tier);
            Assert.AreEqual(RankTier.Master, standing.TopAt(1).Badge.Tier);
            Assert.IsFalse(standing.Listed);
        }

        [Test]
        public void ARowWithoutANicknameReadsAsAnEmptyName()
        {
            const string json = @"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""a"",""rank"":""Gold II"",""tier"":""Gold"",""wins"":1}]}";

            Assert.IsTrue(LeaderboardStanding.TryRead(json, out var standing));
            Assert.AreEqual(string.Empty, standing.TopAt(0).Nickname);
            Assert.AreEqual(0, standing.TopAt(0).LP);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("not json")]
        [TestCase(@"{""mode"":""1v1"",""board"":""global"",""top"":[]}")]
        [TestCase(@"{""season"":""s"",""board"":""global"",""top"":[]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""top"":[]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""rank"":""Gold II"",""tier"":""Gold"",""wins"":1}]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":0,""account_id"":""a"",""rank"":""Gold II"",""tier"":""Gold"",""wins"":1}]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""a"",""rank"":""Gold II"",""tier"":""Gold"",""wins"":-1}]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""a"",""rank"":""Gold II"",""lp"":-1,""tier"":""Gold"",""wins"":1}]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""a"",""rank"":"""",""tier"":"""",""wins"":1}]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""a"",""rank"":""Gold V"",""tier"":""Gold"",""wins"":1}]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""a"",""rank"":""Master"",""tier"":""Gold"",""wins"":1}]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""around"":[{""position"":1,""account_id"":""a"",""rank"":""Mud I"",""tier"":""Mud"",""wins"":1}]}")]
        [TestCase(@"{""season"":""s"",""mode"":""1v1"",""board"":""global"",""me"":{""position"":1,""account_id"":""a"",""rank"":""Gold"",""tier"":""Gold"",""wins"":1}}")]
        public void ARefusedReplyHasNoStanding(string json)
        {
            Assert.IsFalse(LeaderboardStanding.TryRead(json, out var standing));
            Assert.IsNull(standing);
        }

        [TestCase("1v1", "global", "/leaderboard/1v1?board=global")]
        [TestCase("2v2", "friends", "/leaderboard/2v2?board=friends")]
        [TestCase("1v1", "country:AZ", "/leaderboard/1v1?board=country%3AAZ")]
        public void ThePathNamesTheModeAndTheBoard(string mode, string board, string path)
        {
            Assert.AreEqual(path, LeaderboardStanding.PathOf(mode, board));
        }

        [Test]
        public void ReadingTheRowsAllocatesNothing()
        {
            LeaderboardStanding.TryRead(GlobalJson, out var standing);
            TestDelegate step = () =>
            {
                for (var i = 0; i < standing.TopCount; i++) _ = standing.TopAt(i).Badge.Tier;
                for (var i = 0; i < standing.AroundCount; i++) _ = standing.AroundAt(i).Position + standing.AroundAt(i).LP;
                _ = standing.Me.Nickname;
                _ = standing.Me.Wins;
            };
            step();
            Assert.That(step, Allocates.Nothing);
        }
    }
}
