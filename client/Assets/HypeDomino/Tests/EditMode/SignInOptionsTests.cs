using HypeDomino.Net.Auth;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class SignInOptionsTests
    {
        [Test]
        public void AndroidOffersGoogleOnly()
        {
            CollectionAssert.AreEqual(new[] { AuthProvider.Google }, SignInOptions.For(RuntimePlatform.Android));
        }

        [Test]
        public void IosOffersAppleNextToGoogle()
        {
            CollectionAssert.AreEqual(new[] { AuthProvider.Google, AuthProvider.Apple }, SignInOptions.For(RuntimePlatform.IPhonePlayer));
        }

        [TestCase(RuntimePlatform.WindowsEditor)]
        [TestCase(RuntimePlatform.OSXEditor)]
        [TestCase(RuntimePlatform.LinuxEditor)]
        [TestCase(RuntimePlatform.WindowsPlayer)]
        [TestCase(RuntimePlatform.OSXPlayer)]
        public void PlatformsWithoutANativeSignInOfferNothing(RuntimePlatform platform)
        {
            CollectionAssert.IsEmpty(SignInOptions.For(platform));
        }
    }
}
