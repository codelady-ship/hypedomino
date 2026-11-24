using HypeDomino.UI.Localization;
using HypeDomino.UI.Rarity;
using NUnit.Framework;
using TMPro;
using UnityEngine;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class RarityLabelTests
    {
        const string Table = @"{""tiers"":[
{""id"":""plain"",""ordinal"":1,""nameKey"":""rarity.plain.name"",""fill"":""#808080"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":1},
{""id"":""bright"",""ordinal"":2,""nameKey"":""rarity.bright.name"",""fill"":""#FF0000"",""on"":""#000000"",""text"":""#00FF00"",""pips"":2}]}";
        const string Names = @"{""entries"":[
{""key"":""rarity.plain.name"",""en"":""Plain"",""az"":""Sadə""},
{""key"":""rarity.bright.name"",""en"":""Bright"",""az"":""Parlaq""}]}";

        static readonly string[] NoLocales = { };

        GameObject root;
        RarityTable table;
        LanguageChoice choice;
        GameStrings strings;

        [SetUp]
        public void BuildScreen()
        {
            root = new GameObject("Screen", typeof(RectTransform), typeof(Canvas));
            table = RarityTable.Parse(Table);
            choice = LanguageChoice.Load(GameLanguage.English, () => NoLocales);
            strings = new GameStrings(BuiltinStrings.Parse(Names), choice, false);
        }

        [TearDown]
        public void DestroyScreen() => Object.Destroy(root);

        RarityLabel NewLabel()
        {
            var text = new GameObject("Label", typeof(RectTransform)).AddComponent<TextMeshProUGUI>();
            text.transform.SetParent(root.transform, false);
            return text.gameObject.AddComponent<RarityLabel>();
        }

        [Test]
        public void TheLabelShowsTheTierNameInTheTierTextShade()
        {
            var label = NewLabel();
            label.Show(table, strings, "bright");
            var text = label.GetComponent<TMP_Text>();
            Assert.IsTrue(label.IsShown);
            Assert.AreEqual("Bright", text.text);
            Assert.AreEqual(Color.green, text.color);
        }

        [Test]
        public void ShowingAnotherTierReplacesNameAndShade()
        {
            var label = NewLabel();
            label.Show(table, strings, "bright");
            label.Show(table, strings, "plain");
            var text = label.GetComponent<TMP_Text>();
            Assert.AreEqual("Plain", text.text);
            Assert.AreEqual(Color.white, text.color);
        }

        [Test]
        public void AnUnknownTierShowsNoLabel()
        {
            var label = NewLabel();
            label.Show(table, strings, "bright");
            label.Show(table, strings, "unknown");
            Assert.IsFalse(label.IsShown);
        }

        [Test]
        public void TheLabelFollowsALanguageChange()
        {
            var label = NewLabel();
            label.Show(table, strings, "bright");
            choice.Choose("az");
            Assert.AreEqual("Parlaq", label.GetComponent<TMP_Text>().text);
        }

        [Test]
        public void ADestroyedLabelNoLongerFollowsTheLanguage()
        {
            var label = NewLabel();
            label.Show(table, strings, "bright");
            Object.DestroyImmediate(label.gameObject);
            Assert.DoesNotThrow(() => choice.Choose("az"));
        }
    }
}
