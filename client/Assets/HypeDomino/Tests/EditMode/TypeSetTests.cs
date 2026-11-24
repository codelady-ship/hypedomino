using System;
using System.Collections.Generic;
using HypeDomino.Editor;
using HypeDomino.UI.Typography;
using HypeDomino.UI.Tokens;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.TextCore;
using Object = UnityEngine.Object;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TypeSetTests
    {
        static readonly TokenTextStyle[] Styles = (TokenTextStyle[])Enum.GetValues(typeof(TokenTextStyle));

        static TypeSet Set
        {
            get
            {
                var set = TypeSet.Loaded;
                Assert.IsNotNull(set, "Resources/TypeSet is missing, run HypeDomino.Editor.FontSetup.GenerateAndExit");
                return set;
            }
        }

        [Test]
        public void EveryTextStyleHasAStaticFontOfItsFamilyAndWeight()
        {
            foreach (var style in Styles)
            {
                var font = Set.FontOf(style);
                Assert.IsNotNull(font, style.ToString());
                Assert.AreEqual(AtlasPopulationMode.Static, font.atlasPopulationMode, style.ToString());
            }
        }

        [Test]
        public void EveryTextStyleDrawsTheLaunchLanguagesAndTheBuiltinStrings()
        {
            var required = FontSetup.RequiredCharacters(FontSetup.LoadManifest());
            foreach (var style in Styles)
            {
                var font = Set.FontOf(style);
                var missing = new List<char>();
                foreach (var character in required)
                    if (!font.HasCharacter(character, true)) missing.Add(character);
                Assert.IsEmpty(missing, $"{style} draws a box for these characters");
            }
        }

        [Test]
        public void CurrencySignsComeOnlyFromTheUiFamily()
        {
            var uiOnly = FontSetup.LoadManifest().uiOnly;
            var allowed = new List<TokenFontFamily>();
            foreach (var family in uiOnly.families) allowed.Add(FontSetup.FamilyToken(family));
            foreach (var style in Styles)
                foreach (var sign in uiOnly.characters)
                {
                    var holder = Holder(Set.FontOf(style), sign);
                    Assert.IsNotNull(holder, $"{style} has no {sign}");
                    CollectionAssert.Contains(allowed, FamilyOf(holder), $"{style} sets {sign} in {holder.name}");
                }
        }

        [Test]
        public void ApplySetsTheFontSizeAndLineHeightOfTheStyle()
        {
            var owner = new GameObject("text", typeof(RectTransform));
            try
            {
                var text = owner.AddComponent<TextMeshProUGUI>();
                Set.Apply(text, TokenTextStyle.Score);
                Assert.AreSame(Set.FontOf(TokenTextStyle.Score), text.font);
                Assert.AreEqual(TokenTextSize.Score, text.fontSize);
                var lineHeight = TokenTypography.LineHeightOfStyle[(int)TokenTextStyle.Score];
                Assert.AreEqual(TypeSet.LineSpacing(text.font.faceInfo, lineHeight), text.lineSpacing, 1e-4f);
            }
            finally
            {
                Object.DestroyImmediate(owner);
            }
        }

        [Test]
        public void LineSpacingIsTheGapBetweenTheFontLineAndTheTokenLineHeightInHundredthsOfAnEm()
        {
            var face = new FaceInfo { pointSize = 100, lineHeight = 120 };
            Assert.AreEqual(10f, TypeSet.LineSpacing(face, 1.3f), 1e-4f);
            Assert.AreEqual(-20f, TypeSet.LineSpacing(face, 1f), 1e-4f);
        }

        static TMP_FontAsset Holder(TMP_FontAsset font, char character)
        {
            if (font.characterLookupTable.ContainsKey(character)) return font;
            foreach (var fallback in font.fallbackFontAssetTable)
            {
                var holder = Holder(fallback, character);
                if (holder != null) return holder;
            }
            return null;
        }

        static TokenFontFamily FamilyOf(TMP_FontAsset font)
        {
            for (var at = 0; at < Set.FaceCount; at++)
                if (Set.FaceAt(at).font == font) return Set.FaceAt(at).family;
            throw new AssertionException($"{font.name} is not a face of the type set");
        }
    }
}
