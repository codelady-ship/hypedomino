using HypeDomino.Game;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class StoreLinkTests
    {
        [Test]
        public void AndroidOpensTheStorePageOfThisApp() =>
            Assert.AreEqual("https://play.google.com/store/apps/details?id=com.example.game", StoreLink.Of(RuntimePlatform.Android, "com.example.game"));

        [Test]
        public void TheApplicationIdIsEscaped() =>
            Assert.AreEqual("https://play.google.com/store/apps/details?id=a%26b", StoreLink.Of(RuntimePlatform.Android, "a&b"));

        [TestCase(RuntimePlatform.IPhonePlayer, "com.example.game")]
        [TestCase(RuntimePlatform.WindowsEditor, "com.example.game")]
        [TestCase(RuntimePlatform.Android, "")]
        [TestCase(RuntimePlatform.Android, null)]
        public void NoLinkWithoutAKnownStorePage(RuntimePlatform platform, string applicationId) =>
            Assert.AreEqual(string.Empty, StoreLink.Of(platform, applicationId));
    }
}
