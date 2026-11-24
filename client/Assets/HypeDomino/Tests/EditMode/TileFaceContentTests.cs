using System.IO;
using System.Linq;
using HypeDomino.Editor;
using HypeDomino.UI.Board;
using NUnit.Framework;
using UnityEditor;
using UnityEditor.AddressableAssets;
using UnityEditor.U2D;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class TileFaceContentTests
    {
        const string Slug = TileFaceContent.DefaultSet;

        static readonly string[] TexturePlatforms = { "Android", "iPhone" };

        [Test]
        public void TheFolderHoldsExactlyTheTwentyEightFacesInSetOrder()
        {
            var expected = TileFaceContent.FacePaths(Slug);
            Assert.AreEqual(TileFaces.SetSize, expected.Length);
            Assert.AreEqual($"{TileFaceContent.FacesFolder(Slug)}/0_0.png", expected[0]);
            Assert.AreEqual($"{TileFaceContent.FacesFolder(Slug)}/1_1.png", expected[7]);
            Assert.AreEqual($"{TileFaceContent.FacesFolder(Slug)}/6_6.png", expected[27]);
            var files = Directory.GetFiles(TileFaceContent.FacesFolder(Slug), "*.png").Select(path => path.Replace('\\', '/'));
            CollectionAssert.AreEquivalent(expected, files);
        }

        [Test]
        public void EveryFaceIsAMipFreeCentredSpriteWithAstcOnMobile()
        {
            foreach (var path in TileFaceContent.FacePaths(Slug))
            {
                var importer = (TextureImporter)AssetImporter.GetAtPath(path);
                var settings = new TextureImporterSettings();
                importer.ReadTextureSettings(settings);
                Assert.AreEqual(TextureImporterType.Sprite, importer.textureType, path);
                Assert.AreEqual(SpriteImportMode.Single, importer.spriteImportMode, path);
                Assert.AreEqual(SpriteMeshType.FullRect, settings.spriteMeshType, path);
                Assert.AreEqual((int)SpriteAlignment.Center, settings.spriteAlignment, path);
                Assert.AreEqual(Vector4.zero, settings.spriteBorder, path);
                Assert.IsFalse(importer.mipmapEnabled, path);
                Assert.IsTrue(importer.alphaIsTransparency, path);
                Assert.IsTrue(importer.sRGBTexture, path);
                foreach (var platform in TexturePlatforms)
                {
                    var platformSettings = importer.GetPlatformTextureSettings(platform);
                    Assert.IsTrue(platformSettings.overridden, $"{path} {platform}");
                    Assert.AreEqual(ProjectSetup.SpriteFormat, platformSettings.format, $"{path} {platform}");
                }
                var sprite = AssetDatabase.LoadAssetAtPath<Sprite>(path);
                Assert.IsNotNull(sprite, path);
                Assert.AreEqual(new Vector2(TileFaceContent.FaceWidth, TileFaceContent.FaceHeight), sprite.rect.size, path);
            }
        }

        [Test]
        public void OneAtlasPacksTheFacesWithoutRotationTightPackingOrMipmaps()
        {
            var importer = (SpriteAtlasImporter)AssetImporter.GetAtPath(TileFaceContent.AtlasPath(Slug));
            Assert.IsNotNull(importer, TileFaceContent.AtlasPath(Slug));
            Assert.IsFalse(importer.packingSettings.enableRotation);
            Assert.IsFalse(importer.packingSettings.enableTightPacking);
            Assert.AreEqual(TileFaceContent.AtlasPadding, importer.packingSettings.padding);
            Assert.IsFalse(importer.textureSettings.generateMipMaps);
            foreach (var platform in TexturePlatforms)
            {
                var settings = importer.GetPlatformSettings(platform);
                Assert.IsTrue(settings.overridden, platform);
                Assert.AreEqual(ProjectSetup.SpriteFormat, settings.format, platform);
                Assert.AreEqual(TileFaceContent.MaxAtlasSize, settings.maxTextureSize, platform);
            }
        }

        [Test]
        public void TheSetGivesEachTileTheFaceNamedByItsValues()
        {
            var set = AssetDatabase.LoadAssetAtPath<TileFaceSet>(TileFaceContent.SetPath(Slug));
            Assert.IsNotNull(set, TileFaceContent.SetPath(Slug));
            Assert.IsTrue(set.IsComplete);
            for (byte first = 0; first <= TileFaces.HighestValue; first++)
            for (byte second = 0; second <= TileFaces.HighestValue; second++)
            {
                var low = System.Math.Min(first, second);
                var high = System.Math.Max(first, second);
                var expected = AssetDatabase.LoadAssetAtPath<Sprite>($"{TileFaceContent.FacesFolder(Slug)}/{low}_{high}.png");
                Assert.AreSame(expected, set.Face(new Domino(first, second)), $"{first}-{second}");
            }
        }

        [Test]
        public void TheDefaultSetIsBuiltInUnderItsAddress()
        {
            var settings = AddressableAssetSettingsDefaultObject.Settings;
            Assert.IsNotNull(settings);
            var entry = settings.FindAssetEntry(AssetDatabase.AssetPathToGUID(TileFaceContent.SetPath(Slug)));
            Assert.IsNotNull(entry, TileFaceContent.SetPath(Slug));
            Assert.AreEqual("tile/ivory/faces", entry.address);
            Assert.AreEqual(ContentSetup.BuiltinGroupName, entry.parentGroup.Name);
        }
    }
}
