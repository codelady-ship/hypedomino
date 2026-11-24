using System.Collections;
using HypeDomino.UI.Motion;
using HypeDomino.UI.Rarity;
using NUnit.Framework;
using UnityEngine;
using UnityEngine.TestTools;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class RarityMarksTests
    {
        const float MinimumSize = 72f;
        const float Period = 2f;
        const string Table = @"{""diamondMinSize"":72,""flowPeriodMs"":2000,""tiers"":[
{""id"":""plain"",""ordinal"":1,""nameKey"":""rarity.plain.name"",""fill"":""#808080"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":1,""glowAlpha"":0,""glowScale"":1,""flow"":false},
{""id"":""bright"",""ordinal"":2,""nameKey"":""rarity.bright.name"",""fill"":""#FF0000"",""on"":""#00FF00"",""text"":""#FFFFFF"",""pips"":2,""glowAlpha"":0.5,""glowScale"":1.25,""flow"":false},
{""id"":""top"",""ordinal"":3,""nameKey"":""rarity.top.name"",""fill"":""#0000FF"",""on"":""#000000"",""text"":""#FFFFFF"",""pips"":3,""glowAlpha"":0.75,""glowScale"":1.5,""flow"":true}]}";

        GameObject root;
        RarityTable table;
        Sprite[] overlays;
        Sprite haloSprite;
        Sprite flowSprite;

        [SetUp]
        public void BuildArt()
        {
            root = new GameObject("Screen", typeof(RectTransform), typeof(Canvas));
            table = RarityTable.Parse(Table);
            overlays = new[] { NewSprite(), NewSprite(), NewSprite() };
            haloSprite = NewSprite();
            flowSprite = NewSprite();
        }

        [TearDown]
        public void DestroyScreen() => Object.Destroy(root);

        static Sprite NewSprite() => Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);

        Image NewImage(string name, Transform parent)
        {
            var image = new GameObject(name, typeof(RectTransform), typeof(Image)).GetComponent<Image>();
            image.transform.SetParent(parent, false);
            return image;
        }

        RarityDiamond NewDiamond(float size)
        {
            var fill = NewImage("Diamond", root.transform);
            fill.rectTransform.sizeDelta = new Vector2(size, size);
            var pips = NewImage("Pips", fill.transform);
            var diamond = fill.gameObject.AddComponent<RarityDiamond>();
            diamond.Configure(fill, pips, overlays);
            return diamond;
        }

        RarityHalo NewHalo()
        {
            var halo = NewImage("Halo", root.transform).gameObject.AddComponent<RarityHalo>();
            halo.Configure(haloSprite, flowSprite);
            return halo;
        }

        [Test]
        public void TheDiamondTakesItsColoursAndPipsFromTheTier()
        {
            var diamond = NewDiamond(MinimumSize);
            diamond.Show(table, "bright");
            var fill = diamond.GetComponent<Image>();
            var pips = diamond.transform.GetChild(0).GetComponent<Image>();
            Assert.IsTrue(diamond.IsShown);
            Assert.AreEqual(Color.red, fill.color);
            Assert.AreEqual(Color.green, pips.color);
            Assert.AreSame(overlays[1], pips.sprite);
        }

        [Test]
        public void EachTierDrawsItsOwnPipOverlay()
        {
            var diamond = NewDiamond(MinimumSize);
            var pips = diamond.transform.GetChild(0).GetComponent<Image>();
            diamond.Show(table, "plain");
            var first = pips.sprite;
            diamond.Show(table, "top");
            Assert.AreNotSame(first, pips.sprite);
        }

        [Test]
        public void TheDiamondAndItsPipsUseOneMaterialSoTheyBatch()
        {
            var diamond = NewDiamond(MinimumSize);
            diamond.Show(table, "top");
            var pips = diamond.transform.GetChild(0).GetComponent<Image>();
            Assert.AreSame(diamond.GetComponent<Image>().materialForRendering, pips.materialForRendering);
            Assert.AreSame(diamond.GetComponent<Image>().mainTexture, pips.mainTexture);
        }

        [Test]
        public void NoDiamondIsDrawnBelowTheMinimumSize()
        {
            var diamond = NewDiamond(MinimumSize - 1f);
            diamond.Show(table, "top");
            Assert.IsFalse(diamond.IsShown);
            Assert.IsFalse(diamond.transform.GetChild(0).GetComponent<Image>().enabled);
        }

        [UnityTest]
        public IEnumerator TheDiamondFollowsItsBoxWhenResized()
        {
            var diamond = NewDiamond(MinimumSize);
            diamond.Show(table, "top");
            ((RectTransform)diamond.transform).sizeDelta = new Vector2(MinimumSize - 1f, MinimumSize);
            yield return null;
            Assert.IsFalse(diamond.IsShown);
            ((RectTransform)diamond.transform).sizeDelta = new Vector2(MinimumSize, MinimumSize);
            yield return null;
            Assert.IsTrue(diamond.IsShown);
        }

        [Test]
        public void AnUnknownTierDrawsNoMark()
        {
            var diamond = NewDiamond(MinimumSize);
            var halo = NewHalo();
            diamond.Show(table, "unknown");
            halo.Show(table, "unknown", EffectLevel.High, false, 0f);
            Assert.IsFalse(diamond.IsShown);
            Assert.IsFalse(halo.IsShown);
        }

        [Test]
        public void ATierWithoutGlowDrawsNoHalo()
        {
            var halo = NewHalo();
            halo.Show(table, "plain", EffectLevel.High, false, 0f);
            Assert.IsFalse(halo.IsShown);
            Assert.IsFalse(halo.IsRotating);
        }

        [Test]
        public void TheHaloTakesItsTintAlphaAndScaleFromTheTier()
        {
            var halo = NewHalo();
            halo.Show(table, "bright", EffectLevel.High, false, 0f);
            var image = halo.GetComponent<Image>();
            Assert.IsTrue(halo.IsShown);
            Assert.IsFalse(halo.IsRotating);
            Assert.AreSame(haloSprite, image.sprite);
            Assert.AreEqual(new Color(1f, 0f, 0f, 0.5f), image.color);
            Assert.AreEqual(1.25f, halo.transform.localScale.x, 0.0001f);
        }

        [Test]
        public void TheFlowHaloTurnsOnceInEachPeriod()
        {
            var halo = NewHalo();
            halo.Show(table, "top", EffectLevel.Medium, false, 10f);
            Assert.AreSame(flowSprite, halo.GetComponent<Image>().sprite);
            Assert.IsTrue(halo.IsRotating);
            halo.Advance(10f + Period / 4f);
            Assert.AreEqual(270f, halo.transform.localEulerAngles.z, 0.01f);
            halo.Advance(10f + Period);
            Assert.AreEqual(0f, Mathf.DeltaAngle(0f, halo.transform.localEulerAngles.z), 0.01f);
        }

        [TestCase(EffectLevel.Low, false)]
        [TestCase(EffectLevel.High, true)]
        public void TheFlowHaloRestsAtTheTopOnLowOrWithReduceMotion(EffectLevel level, bool reduceMotion)
        {
            var halo = NewHalo();
            halo.Show(table, "top", EffectLevel.High, false, 0f);
            halo.Advance(Period / 4f);
            halo.Show(table, "top", level, reduceMotion, 0f);
            halo.Advance(Period / 4f);
            Assert.IsTrue(halo.IsShown);
            Assert.IsFalse(halo.IsRotating);
            Assert.AreEqual(Quaternion.identity, halo.transform.localRotation);
        }

        [Test]
        public void TurningTheHaloDoesNotAllocate()
        {
            var halo = NewHalo();
            halo.Show(table, "top", EffectLevel.Medium, false, 0f);
            halo.Advance(0.25f);
            Assert.That(() => halo.Advance(0.5f), Allocates.Nothing);
        }
    }
}
