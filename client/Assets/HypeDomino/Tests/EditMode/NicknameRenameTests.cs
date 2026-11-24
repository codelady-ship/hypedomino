using HypeDomino.Net;
using HypeDomino.UI.Profile;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class NicknameRenameTests
    {
        const ulong RenameAvailableUnixMs = 1796119200000;

        const string Cooldown = @"""rename_required"":false,""rename_available_at"":""2026-12-01T10:00:00Z"",";

        const string OwnTemplate = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":""Leyla"",""setup_required"":false," + Cooldown + @"
""level"":7,""xp"":1250,""loadout"":{""avatar"":""avatar.comet_fox"",""frame"":""frame.season_1_top""},
""stats"":{""games"":40,""wins"":22,""losses"":18,""win_rate_percent"":55,""streak"":3,""best_streak"":6,""modes"":{}}}";

        const string SetupJson = @"{""account_id"":""8d1f0c52-5a4e-4f7b-9a43-0c7d2b1e6f10"",""nickname"":"""",""setup_required"":true,""rename_required"":false,
""suggested_nickname"":""Player4821"",""level"":1,""xp"":0,""loadout"":{},""stats"":{""games"":0,""wins"":0,""losses"":0,""win_rate_percent"":0,""streak"":0,""best_streak"":0,""modes"":{}}}";

        const string PublicJson = @"{""account_id"":""3b6a9e21-7c0d-4e55-8f12-9a0b4c3d2e1f"",""nickname"":""Rauf"",""level"":12,""loadout"":{},
""stats"":{""games"":80,""win_rate_percent"":61,""best_streak"":9}}";

        static string OwnJson(string rename = Cooldown) => OwnTemplate.Replace(Cooldown, rename);

        static PlayerProfile Own(string rename = Cooldown)
        {
            Assert.IsTrue(PlayerProfile.TryFromOwn(OwnJson(rename), out var profile));
            return profile;
        }

        static ServerClock SyncedAt(ulong serverUnixMs)
        {
            var clock = new ServerClock();
            clock.Sync(serverUnixMs, 0, 0);
            return clock;
        }

        [Test]
        public void TheOwnProfileCountsDownToTheServersRenameTime()
        {
            var profile = Own();

            Assert.IsFalse(profile.RenameRequired);
            Assert.AreEqual(RenameAvailableUnixMs, profile.RenameAvailableUnixMs);
            Assert.IsTrue(profile.TryRenameSecondsLeft(SyncedAt(RenameAvailableUnixMs - 90000), 30, out var seconds));
            Assert.AreEqual(60, seconds, 0.001);
            Assert.IsTrue(profile.TryRenameSecondsLeft(SyncedAt(RenameAvailableUnixMs), 30, out var passed));
            Assert.AreEqual(0, passed);
        }

        [Test]
        public void ARenameIsOpenWhenTheServerSendsNoRenameTime()
        {
            var free = Own(@"""rename_required"":false,");

            Assert.AreEqual(0, free.RenameAvailableUnixMs);
            Assert.IsFalse(free.TryRenameSecondsLeft(SyncedAt(RenameAvailableUnixMs), 0, out _));
        }

        [Test]
        public void AModeratorsRenameDemandComesFromTheServer()
        {
            var flagged = Own(@"""rename_required"":true,");

            Assert.IsTrue(flagged.RenameRequired);
            Assert.AreEqual(0, flagged.RenameAvailableUnixMs);
        }

        [Test]
        public void TheCountdownNeedsASyncedServerClock()
        {
            var profile = Own();

            Assert.IsFalse(profile.TryRenameSecondsLeft(new ServerClock(), 0, out _));
            Assert.IsFalse(profile.TryRenameSecondsLeft(null, 0, out _));
        }

        [TestCase(@"""rename_available_at"":""soon"",")]
        [TestCase(@"""rename_available_at"":""1960-01-01T00:00:00Z"",")]
        public void AnOwnReplyWithAnUnreadableRenameTimeIsNotShown(string rename)
        {
            Assert.IsFalse(PlayerProfile.TryFromOwn(OwnJson(rename), out var profile));
            Assert.IsNull(profile);
        }

        [Test]
        public void AnotherPlayersProfileCarriesNoRenameState()
        {
            Assert.IsTrue(PlayerProfile.TryFromPublic(PublicJson, out var other));

            Assert.IsFalse(other.RenameRequired);
            Assert.AreEqual(0, other.RenameAvailableUnixMs);
        }

        [Test]
        public void TheRequestSendsTheTypedNicknameAsIsForTheServerToJudge()
        {
            Assert.AreEqual("/profile/nickname", NicknameRename.Path);
            Assert.IsTrue(NicknameRename.TryRequest(" Leyla_07 ", out var request));
            Assert.AreEqual(@"{""nickname"":"" Leyla_07 ""}", request.ToJson());
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("   ")]
        public void ABlankNicknameIsNotSent(string nickname)
        {
            Assert.IsFalse(NicknameRename.TryRequest(nickname, out var request));
            Assert.IsNull(request);
        }

        [Test]
        public void AnAcceptedRenameShowsTheServersNewProfile()
        {
            var outcome = NicknameRenameOutcome.Read(OwnJson().Replace(@"""Leyla""", @"""Leyla07"""));

            Assert.AreEqual(NicknameRenameStatus.Accepted, outcome.Status);
            Assert.AreEqual("Leyla07", outcome.Profile.Nickname);
            Assert.AreEqual(RenameAvailableUnixMs, outcome.Profile.RenameAvailableUnixMs);
            Assert.IsNull(outcome.Reason);
        }

        [TestCase(@"{""error"":""nickname.taken""}", "nickname.taken")]
        [TestCase(@"{""error"":""nickname.not_allowed""}", "nickname.not_allowed")]
        [TestCase(@"{""error"":""nickname.invalid_length""}", "nickname.invalid_length")]
        [TestCase(@"{""error"":""malformed""}", "malformed")]
        public void ARefusalCarriesTheServersReasonCode(string json, string reason)
        {
            var outcome = NicknameRenameOutcome.Read(json);

            Assert.AreEqual(NicknameRenameStatus.Refused, outcome.Status);
            Assert.AreEqual(reason, outcome.Reason);
            Assert.AreEqual(0, outcome.RetryAtUnixMs);
            Assert.IsNull(outcome.Profile);
        }

        [Test]
        public void ACooldownRefusalCarriesTheServersRetryTime()
        {
            var outcome = NicknameRenameOutcome.Read(@"{""error"":""nickname.cooldown"",""retry_at"":""2026-12-01T10:00:00Z""}");

            Assert.AreEqual(NicknameRenameStatus.Refused, outcome.Status);
            Assert.AreEqual("nickname.cooldown", outcome.Reason);
            Assert.AreEqual(RenameAvailableUnixMs, outcome.RetryAtUnixMs);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("{")]
        [TestCase("[]")]
        [TestCase(@"{""error"":""""}")]
        [TestCase(@"{""error"":7}")]
        [TestCase(@"{""error"":null}")]
        [TestCase(SetupJson)]
        public void AReplyThatIsNeitherAProfileNorARefusalIsUnreadable(string json)
        {
            var outcome = NicknameRenameOutcome.Read(json);

            Assert.AreEqual(NicknameRenameStatus.Unreadable, outcome.Status);
            Assert.IsNull(outcome.Profile);
            Assert.IsNull(outcome.Reason);
        }

        [Test]
        public void CountingDownTheRenameAllocatesNothing()
        {
            var profile = Own();
            var clock = SyncedAt(RenameAvailableUnixMs - 90000);
            var sink = 0d;
            TestDelegate countDown = () =>
            {
                if (profile.TryRenameSecondsLeft(clock, 30, out var seconds)) sink += seconds;
            };
            countDown();

            Assert.That(countDown, Allocates.Nothing);
            Assert.Greater(sink, 0);
        }
    }
}
