using System.IO;
using System.Linq;
using HypeDomino.Editor;
using NUnit.Framework;
using UnityEditor;
using UnityEditor.U2D;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class UiKitContentTests
    {
        static readonly string[] TexturePlatforms = { "Android", "iPhone" };

        static TextureImporter Importer(string path) => (TextureImporter)AssetImporter.GetAtPath(path);

        [Test]
        public void EverySpriteIsAMipFreeFullRectSpriteWithAstcOnMobile()
        {
            var paths = UiKitContent.SpritePaths();
            Assert.IsNotEmpty(paths);
            foreach (var path in paths)
            {
                var importer = Importer(path);
                var settings = new TextureImporterSettings();
                importer.ReadTextureSettings(settings);
                Assert.AreEqual(TextureImporterType.Sprite, importer.textureType, path);
                Assert.AreEqual(SpriteImportMode.Single, importer.spriteImportMode, path);
                Assert.AreEqual(SpriteMeshType.FullRect, settings.spriteMeshType, path);
                Assert.IsFalse(importer.mipmapEnabled, path);
                Assert.IsTrue(importer.alphaIsTransparency, path);
                foreach (var platform in TexturePlatforms)
                {
                    var platformSettings = importer.GetPlatformTextureSettings(platform);
                    Assert.IsTrue(platformSettings.overridden, $"{path} {platform}");
                    Assert.AreEqual(ProjectSetup.SpriteFormat, platformSettings.format, $"{path} {platform}");
                }
            }
        }

        [Test]
        public void EveryKitSpriteCarriesItsNineSliceBorderAndPixelsPerUnit()
        {
            var kit = UiKitContent.ReadKit();
            Assert.IsNotEmpty(kit.sprites);
            foreach (var entry in kit.sprites)
            {
                var path = $"{UiKitContent.PanelsFolder}/{entry.name}.png";
                var sprite = AssetDatabase.LoadAssetAtPath<Sprite>(path);
                Assert.IsNotNull(sprite, path);
                Assert.AreEqual(new Vector2(entry.width, entry.height), sprite.rect.size, path);
                Assert.AreEqual(new Vector4(entry.border[0], entry.border[1], entry.border[2], entry.border[3]), sprite.border, path);
                Assert.AreEqual(entry.pixelsPerUnit, sprite.pixelsPerUnit, path);
            }
        }

        [Test]
        public void EveryPanelSpriteIsListedInTheKit()
        {
            var listed = UiKitContent.ReadKit().sprites.Select(entry => entry.name).ToArray();
            var files = Directory.GetFiles(UiKitContent.PanelsFolder, "*.png").Select(Path.GetFileNameWithoutExtension).ToArray();
            CollectionAssert.AreEquivalent(files, listed);
        }

        [Test]
        public void IconsAndRarityMarksHaveNoBorder()
        {
            foreach (var path in UiKitContent.SpritePaths().Where(path => !path.StartsWith(UiKitContent.PanelsFolder + "/")))
                Assert.AreEqual(Vector4.zero, AssetDatabase.LoadAssetAtPath<Sprite>(path).border, path);
        }

        [Test]
        public void RarityHasTheDiamondAndOneOverlayPerPipCountUpToSix()
        {
            Assert.IsNotNull(AssetDatabase.LoadAssetAtPath<Sprite>($"{UiKitContent.RarityFolder}/diamond.png"));
            for (var pips = 1; pips <= 6; pips++)
                Assert.IsNotNull(AssetDatabase.LoadAssetAtPath<Sprite>($"{UiKitContent.RarityFolder}/pips_{pips}.png"), $"pips_{pips}");
        }

        [Test]
        public void OneAtlasPacksTheWholeKitWithoutRotationOrTightPacking()
        {
            var importer = (SpriteAtlasImporter)AssetImporter.GetAtPath(UiKitContent.AtlasPath);
            Assert.IsNotNull(importer, UiKitContent.AtlasPath);
            Assert.IsFalse(importer.packingSettings.enableRotation);
            Assert.IsFalse(importer.packingSettings.enableTightPacking);
            Assert.IsFalse(importer.textureSettings.generateMipMaps);
            foreach (var platform in TexturePlatforms)
            {
                var settings = importer.GetPlatformSettings(platform);
                Assert.IsTrue(settings.overridden, platform);
                Assert.AreEqual(ProjectSetup.SpriteFormat, settings.format, platform);
                Assert.AreEqual(UiKitContent.MaxAtlasSize, settings.maxTextureSize, platform);
            }
        }
    }
}
