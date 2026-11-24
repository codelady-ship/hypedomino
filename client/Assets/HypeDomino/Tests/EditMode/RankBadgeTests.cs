using Google.Protobuf;
using HypeDomino.Game.Ranking;
using HypeDomino.Net;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class RankBadgeTests
    {
        [TestCase("Iron IV", RankTier.Iron, 4)]
        [TestCase("Bronze III", RankTier.Bronze, 3)]
        [TestCase("Silver I", RankTier.Silver, 1)]
        [TestCase("Gold II", RankTier.Gold, 2)]
        [TestCase("Platinum IV", RankTier.Platinum, 4)]
        [TestCase("Emerald III", RankTier.Emerald, 3)]
        [TestCase("Diamond I", RankTier.Diamond, 1)]
        public void ADivisionRankGivesItsTierAndDivision(string label, RankTier tier, int division)
        {
            Assert.IsTrue(RankBadge.TryRead(label, out var badge));
            Assert.AreEqual(tier, badge.Tier);
            Assert.AreEqual(division, badge.Division);
            Assert.IsTrue(badge.Placed);
            Assert.IsFalse(badge.Apex);
        }

        [TestCase("Master", RankTier.Master)]
        [TestCase("Grandmaster", RankTier.Grandmaster)]
        [TestCase("Challenger", RankTier.Challenger)]
        public void AnApexTierHasNoDivision(string label, RankTier tier)
        {
            Assert.IsTrue(RankBadge.TryRead(label, out var badge));
            Assert.AreEqual(tier, badge.Tier);
            Assert.AreEqual(RankBadge.NoDivision, badge.Division);
            Assert.IsTrue(badge.Placed);
            Assert.IsTrue(badge.Apex);
        }

        [TestCase("")]
        [TestCase(null)]
        public void AnEmptyLabelIsAPlayerInPlacement(string label)
        {
            Assert.IsTrue(RankBadge.TryRead(label, out var badge));
            Assert.AreEqual(RankTier.None, badge.Tier);
            Assert.IsFalse(badge.Placed);
        }

        [TestCase("Gold")]
        [TestCase("Gold V")]
        [TestCase("Gold  II")]
        [TestCase("Gold II ")]
        [TestCase("gold II")]
        [TestCase("Master I")]
        [TestCase("Mithril II")]
        [TestCase(" Gold II")]
        public void AnUnknownLabelShowsNoBadge(string label)
        {
            Assert.IsFalse(RankBadge.TryRead(label, out var badge));
            Assert.IsFalse(badge.Placed);
        }

        [Test]
        public void TheRosterRankLabelOfAViewReadsAsABadge()
        {
            var view = new MatchView { State = MatchState.Loading, Seat = Seat.X, Mode = Mode._1V1 };
            view.Roster.Add(new RosterEntry { Seat = Seat.X, Nickname = "me", RankLabel = "Emerald II" });
            view.Roster.Add(new RosterEntry { Seat = Seat.Y, Nickname = "rival" });
            var snapshot = new MatchSnapshot();
            Assert.IsTrue(snapshot.TryRead(view.ToByteArray()));

            Assert.IsTrue(RankBadge.TryRead(snapshot.Roster[0].RankLabel, out var mine));
            Assert.IsTrue(RankBadge.TryRead(snapshot.Roster[1].RankLabel, out var theirs));
            Assert.AreEqual(RankTier.Emerald, mine.Tier);
            Assert.AreEqual(2, mine.Division);
            Assert.IsFalse(theirs.Placed);
        }

        [Test]
        public void ReadingALabelAllocatesNothing()
        {
            TestDelegate step = () =>
            {
                RankBadge.TryRead("Platinum III", out _);
                RankBadge.TryRead("Grandmaster", out _);
                RankBadge.TryRead("Gold V", out _);
            };
            step();
            Assert.That(step, Allocates.Nothing);
        }
    }
}
