using HypeDomino.Core.Diagnostics;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ErrorScrubTests
    {
        [TestCase("account 3f2504e0-4f89-11d3-9a0c-0305e82c3301 failed", "account [id] failed")]
        [TestCase("analytics id 3F2504E0-4F89-11D3-9A0C-0305E82C3301", "analytics id [id]")]
        [TestCase("support for jane.doe+x@mail.example.org", "support for [email]")]
        [TestCase("Authorization: Bearer abc.def-ghi", "Authorization: Bearer [token]")]
        [TestCase("token eyJhbGciOiJub25lIn0.eyJ4IjoxfQ.c2ln expired", "token [token] expired")]
        [TestCase("GET https://play.example.com/auth/refresh?token=abc&x=1 failed", "GET https://play.example.com/auth/refresh?[query] failed")]
        [TestCase("wss://play.example.com/ws?resume=r1#f", "wss://play.example.com/ws?[query]#f")]
        public void IdsEmailsTokensAndQueriesAreRemoved(string text, string scrubbed)
        {
            Assert.AreEqual(scrubbed, ErrorScrub.Scrub(text));
        }

        [TestCase("NullReferenceException: Object reference not set to an instance of an object")]
        [TestCase("Content manifest catalog_1.4.2-57.bin missing at https://cdn.example.com/Android/catalog.bin")]
        [TestCase("HypeDomino.Game.Match.MatchView.Play (System.Int32 tile) (at Assets/HypeDomino/Game/Match/MatchView.cs:42)")]
        public void OrdinaryErrorTextIsKept(string text)
        {
            Assert.AreEqual(text, ErrorScrub.Scrub(text));
        }

        [TestCase(null, "")]
        [TestCase("", "")]
        public void NoTextStaysEmpty(string text, string scrubbed)
        {
            Assert.AreEqual(scrubbed, ErrorScrub.Scrub(text));
        }
    }
}
