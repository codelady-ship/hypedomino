using System.IO;
using System.Linq;
using HypeDomino.Editor;
using NUnit.Framework;
using UnityEditor;
using UnityEditor.U2D;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class EffectsContentTests
    {
        static readonly string[] TexturePlatforms = { "Android", "iPhone" };

        [Test]
        public void EverySpriteIsAMipFreeTightSpriteWithAstcOnMobile()
        {
            var paths = EffectsContent.SpritePaths();
            Assert.IsNotEmpty(paths);
            foreach (var path in paths)
            {
                var importer = (TextureImporter)AssetImporter.GetAtPath(path);
                var settings = new TextureImporterSettings();
                importer.ReadTextureSettings(settings);
                Assert.AreEqual(TextureImporterType.Sprite, importer.textureType, path);
                Assert.AreEqual(SpriteImportMode.Single, importer.spriteImportMode, path);
                Assert.AreEqual(SpriteMeshType.Tight, settings.spriteMeshType, path);
                Assert.AreEqual(Vector4.zero, settings.spriteBorder, path);
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
        public void TheTableListsEverySpriteWithOneOfTheTwoMaterials()
        {
            var table = EffectsContent.ReadTable();
            var listed = table.sprites.Select(sprite => sprite.name).ToArray();
            var files = EffectsContent.SpritePaths().Select(Path.GetFileNameWithoutExtension).ToArray();
            CollectionAssert.AreEquivalent(files, listed);
            foreach (var sprite in table.sprites)
            {
                Assert.IsNotNull(AssetDatabase.LoadAssetAtPath<Sprite>(EffectsContent.SpritePath(sprite.name)), sprite.name);
                Assert.That(sprite.material, Is.EqualTo(EffectsContent.AdditiveMaterial).Or.EqualTo(EffectsContent.AlphaMaterial), sprite.name);
            }
        }

        [Test]
        public void TheRarityHaloSpritesAreInTheEffects()
        {
            Assert.IsNotNull(AssetDatabase.LoadAssetAtPath<Sprite>(EffectsContent.SpritePath("halo")));
            Assert.IsNotNull(AssetDatabase.LoadAssetAtPath<Sprite>(EffectsContent.SpritePath("flow_halo")));
        }

        [Test]
        public void OneAtlasPacksTheSpritesWithoutRotationOrTightPacking()
        {
            var importer = (SpriteAtlasImporter)AssetImporter.GetAtPath(EffectsContent.AtlasPath);
            Assert.IsNotNull(importer, EffectsContent.AtlasPath);
            Assert.IsFalse(importer.packingSettings.enableRotation);
            Assert.IsFalse(importer.packingSettings.enableTightPacking);
            Assert.IsFalse(importer.textureSettings.generateMipMaps);
            foreach (var platform in TexturePlatforms)
            {
                var settings = importer.GetPlatformSettings(platform);
                Assert.IsTrue(settings.overridden, platform);
                Assert.AreEqual(ProjectSetup.SpriteFormat, settings.format, platform);
                Assert.AreEqual(EffectsContent.MaxAtlasSize, settings.maxTextureSize, platform);
            }
        }

        [Test]
        public void TheFolderHoldsOnlyTheAdditiveAndTheAlphaMaterial()
        {
            var materials = AssetDatabase.FindAssets("t:Material", new[] { EffectsContent.Root }).Select(AssetDatabase.GUIDToAssetPath).OrderBy(path => path).ToArray();
            CollectionAssert.AreEqual(new[] { EffectsContent.AdditiveMaterialPath, EffectsContent.AlphaMaterialPath }, materials);
            Assert.AreEqual(EffectsContent.AdditiveShaderName, AssetDatabase.LoadAssetAtPath<Material>(EffectsContent.AdditiveMaterialPath).shader.name);
            Assert.AreEqual(EffectsContent.AlphaShaderName, AssetDatabase.LoadAssetAtPath<Material>(EffectsContent.AlphaMaterialPath).shader.name);
        }

        [Test]
        public void TheAdditiveShaderCompiles()
        {
            var shader = Shader.Find(EffectsContent.AdditiveShaderName);
            Assert.IsNotNull(shader);
            Assert.IsFalse(ShaderUtil.ShaderHasError(shader));
        }
    }
}
