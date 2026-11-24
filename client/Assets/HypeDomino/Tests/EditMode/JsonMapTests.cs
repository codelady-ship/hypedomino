using System.Linq;
using HypeDomino.Net;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class JsonMapTests
    {
        static JsonMap Parse(string json)
        {
            Assert.IsTrue(JsonMap.TryParse(json, out var map));
            return map;
        }

        [Test]
        public void TheKeysAreTheObjectMembersAsSent()
        {
            var map = Parse(@"{""1v1"":""Silver I"",""2v2"":""Gold II""}");

            CollectionAssert.AreEquivalent(new[] { "1v1", "2v2" }, map.Keys.ToArray());
            Assert.IsTrue(map.TryString("1v1", out var oneVersusOne));
            Assert.AreEqual("Silver I", oneVersusOne);
            Assert.IsTrue(map.TryString("2v2", out var twoVersusTwo));
            Assert.AreEqual("Gold II", twoVersusTwo);
        }

        [Test]
        public void NestedObjectsAreMapsToo()
        {
            var map = Parse(@"{""modes"":{""1v1"":{""games"":10,""wins"":7}}}");

            Assert.IsTrue(map.TryMap("modes", out var modes));
            Assert.IsTrue(modes.TryMap("1v1", out var mode));
            Assert.IsTrue(mode.TryInteger("games", out var games));
            Assert.AreEqual(10, games);
            Assert.IsTrue(mode.TryInteger("wins", out var wins));
            Assert.AreEqual(7, wins);
        }

        [Test]
        public void NumbersAreReadAsWholeIntegers()
        {
            var map = Parse(@"{""avatar.fox"":120,""zero"":0,""negative"":-5,""largest"":9007199254740992,""half"":1.5,""beyond"":9007199254740994,""text"":""120""}");

            Assert.IsTrue(map.TryInteger("avatar.fox", out var compensation));
            Assert.AreEqual(120, compensation);
            Assert.IsTrue(map.TryInteger("zero", out var zero));
            Assert.AreEqual(0, zero);
            Assert.IsTrue(map.TryInteger("negative", out var negative));
            Assert.AreEqual(-5, negative);
            Assert.IsTrue(map.TryInteger("largest", out var largest));
            Assert.AreEqual(9_007_199_254_740_992L, largest);
            Assert.IsFalse(map.TryInteger("half", out _));
            Assert.IsFalse(map.TryInteger("beyond", out _));
            Assert.IsFalse(map.TryInteger("text", out _));
        }

        [Test]
        public void StringListsHoldOnlyStrings()
        {
            var map = Parse(@"{""cosmetics"":[""avatar.fox"",""emoji.wave""],""empty"":[],""mixed"":[""avatar.fox"",1]}");

            Assert.IsTrue(map.TryStrings("cosmetics", out var cosmetics));
            CollectionAssert.AreEqual(new[] { "avatar.fox", "emoji.wave" }, cosmetics);
            Assert.IsTrue(map.TryStrings("empty", out var empty));
            Assert.IsEmpty(empty);
            Assert.IsFalse(map.TryStrings("mixed", out var mixed));
            Assert.IsNull(mixed);
        }

        [Test]
        public void ObjectListsHoldOnlyObjects()
        {
            var map = Parse(@"{""offers"":[{""id"":""offer.a""},{""id"":""offer.b""}],""empty"":[],""mixed"":[{""id"":""offer.a""},null],""texts"":[""offer.a""],""object"":{}}");

            Assert.IsTrue(map.TryMaps("offers", out var offers));
            Assert.AreEqual(2, offers.Length);
            Assert.IsTrue(offers[1].TryString("id", out var second));
            Assert.AreEqual("offer.b", second);
            Assert.IsTrue(map.TryMaps("empty", out var empty));
            Assert.IsEmpty(empty);
            Assert.IsFalse(map.TryMaps("mixed", out var mixed));
            Assert.IsNull(mixed);
            Assert.IsFalse(map.TryMaps("texts", out _));
            Assert.IsFalse(map.TryMaps("object", out _));
            Assert.IsFalse(map.TryMaps("missing", out _));
        }

        [Test]
        public void AValueOfAnotherKindIsNotRead()
        {
            var map = Parse(@"{""text"":""a"",""number"":1,""flag"":true,""nothing"":null,""list"":[],""object"":{}}");

            Assert.IsFalse(map.TryString("number", out var text));
            Assert.IsNull(text);
            Assert.IsFalse(map.TryInteger("text", out var number));
            Assert.AreEqual(0, number);
            Assert.IsFalse(map.TryMap("list", out var nested));
            Assert.IsNull(nested);
            Assert.IsFalse(map.TryStrings("object", out _));
            Assert.IsFalse(map.TryString("flag", out _));
            Assert.IsFalse(map.TryMap("nothing", out _));
            Assert.IsFalse(map.TryString("missing", out _));
        }

        [Test]
        public void OnlyAMissingKeyOrNullIsAbsent()
        {
            var map = Parse(@"{""nothing"":null,""zero"":0,""empty"":"""",""list"":[]}");

            Assert.IsTrue(map.IsAbsent("nothing"));
            Assert.IsTrue(map.IsAbsent("missing"));
            Assert.IsFalse(map.IsAbsent("zero"));
            Assert.IsFalse(map.IsAbsent("empty"));
            Assert.IsFalse(map.IsAbsent("list"));
        }

        [Test]
        public void EscapedStringsAreUnescaped()
        {
            var map = Parse(@"{""name"":""a \""b\"" ç""}");

            Assert.IsTrue(map.TryString("name", out var name));
            Assert.AreEqual("a \"b\" ç", name);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("{")]
        [TestCase("[]")]
        [TestCase("1")]
        [TestCase("\"text\"")]
        [TestCase("null")]
        [TestCase("{a:1}")]
        [TestCase("{\"a\":}")]
        [TestCase("{\"a\":1,}")]
        [TestCase("{\"a\":1}x")]
        [TestCase("{\"a\":1}{\"b\":2}")]
        [TestCase("{\"a\":'b'}")]
        public void MalformedJsonIsRefused(string json)
        {
            Assert.IsFalse(JsonMap.TryParse(json, out var map));
            Assert.IsNull(map);
        }
    }
}
