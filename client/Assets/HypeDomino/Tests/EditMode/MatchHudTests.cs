using HypeDomino.Game.Match;
using HypeDomino.UI.Match;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class MatchHudTests
    {
        MatchHud hud;

        [SetUp]
        public void CreateHud() => hud = new MatchHud();

        [Test]
        public void AShownHandIsHeldAndCountsAsAChange()
        {
            var hand = new HandChoice();

            hud.ShowHand(hand);
            hud.ShowHand(hand);

            Assert.AreSame(hand, hud.Hand);
            Assert.AreEqual(2, hud.HandVersion);
        }

        [Test]
        public void TheTurnTimerIsHeldWithoutCountingAChange()
        {
            var timer = new TurnTimer();

            hud.ShowTurn(timer);

            Assert.AreSame(timer, hud.Timer);
            Assert.AreEqual(0, hud.HandVersion);
            Assert.AreEqual(0, hud.NoticesVersion);
            Assert.AreEqual(0, hud.PassVersion);
        }

        [Test]
        public void ShownNoticesAreHeldAndCountAsAChange()
        {
            var notices = new MatchNotices();

            hud.ShowNotices(notices);

            Assert.AreSame(notices, hud.Notices);
            Assert.AreEqual(1, hud.NoticesVersion);
        }

        [Test]
        public void APassLabelShowsOnlyAtItsSideUntilHidden()
        {
            Assert.IsTrue(PassNotice.At(0f, false, out var notice));

            hud.ShowPass(TableSide.Left, notice);

            Assert.IsTrue(hud.TryGetPass(TableSide.Left, out var shown));
            Assert.AreEqual(notice.Alpha, shown.Alpha);
            Assert.IsFalse(hud.TryGetPass(TableSide.Right, out _));

            hud.HidePass(TableSide.Left);

            Assert.IsFalse(hud.TryGetPass(TableSide.Left, out _));
            Assert.AreEqual(2, hud.PassVersion);
        }

        [Test]
        public void ClearDropsEverythingAndCountsAsAChangeForEveryPart()
        {
            Assert.IsTrue(PassNotice.At(0f, false, out var notice));
            hud.ShowHand(new HandChoice());
            hud.ShowTurn(new TurnTimer());
            hud.ShowNotices(new MatchNotices());
            hud.ShowPass(TableSide.Top, notice);

            hud.Clear();

            Assert.IsNull(hud.Hand);
            Assert.IsNull(hud.Timer);
            Assert.IsNull(hud.Notices);
            Assert.IsFalse(hud.TryGetPass(TableSide.Top, out _));
            Assert.AreEqual(2, hud.HandVersion);
            Assert.AreEqual(2, hud.NoticesVersion);
            Assert.AreEqual(2, hud.PassVersion);
        }

        [Test]
        public void ShowingAllocatesNothing()
        {
            var hand = new HandChoice();
            hud.ShowHand(hand);

            Assert.That(() => hud.ShowHand(hand), Allocates.Nothing);
        }
    }
}
