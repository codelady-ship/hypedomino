using System;
using System.Collections.Generic;
using HypeDomino.Game.Match;
using HypeDomino.Net;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ConnectionNoticeKeysTests
    {
        static readonly string[] AllKeys =
        {
            ConnectionNoticeKeys.Reconnecting,
            ConnectionNoticeKeys.Weak,
            ConnectionNoticeKeys.Dropped,
            ConnectionNoticeKeys.BotPlaying,
            ConnectionNoticeKeys.OneTimeoutFromBot,
            ConnectionNoticeKeys.TakeBack,
        };

        [Test]
        public void EveryNetworkNoticeHasAKeyAndNoneHasNone()
        {
            Assert.IsNull(ConnectionNoticeKeys.Of(NetworkNotice.None));
            Assert.AreEqual(ConnectionNoticeKeys.Reconnecting, ConnectionNoticeKeys.Of(NetworkNotice.Reconnecting));
            Assert.AreEqual(ConnectionNoticeKeys.Weak, ConnectionNoticeKeys.Of(NetworkNotice.Weak));
            foreach (NetworkNotice notice in Enum.GetValues(typeof(NetworkNotice)))
            {
                Assert.AreEqual(notice == NetworkNotice.None, ConnectionNoticeKeys.Of(notice) == null, notice.ToString());
            }
        }

        [Test]
        public void EverySeatNoticeHasAKeyAndNoneHasNone()
        {
            Assert.IsNull(ConnectionNoticeKeys.Of(SeatNotice.None));
            Assert.AreEqual(ConnectionNoticeKeys.Dropped, ConnectionNoticeKeys.Of(SeatNotice.Dropped));
            Assert.AreEqual(ConnectionNoticeKeys.BotPlaying, ConnectionNoticeKeys.Of(SeatNotice.BotPlaying));
            foreach (SeatNotice notice in Enum.GetValues(typeof(SeatNotice)))
            {
                Assert.AreEqual(notice == SeatNotice.None, ConnectionNoticeKeys.Of(notice) == null, notice.ToString());
            }
        }

        [Test]
        public void TheKeysAreDistinct() =>
            Assert.AreEqual(AllKeys.Length, new HashSet<string>(AllKeys).Count);

        [Test]
        public void TheShippedBundleHasEveryKeyInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            foreach (var key in AllKeys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
            Assert.AreEqual("Bot sənin yerinə oynayır — qayıt", strings.Value(ConnectionNoticeKeys.TakeBack, "az"));
        }
    }
}
