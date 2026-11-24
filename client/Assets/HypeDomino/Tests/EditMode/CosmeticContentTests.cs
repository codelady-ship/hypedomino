using System.IO;
using System.Linq;
using System.Reflection;
using HypeDomino.Content;
using HypeDomino.Editor;
using NUnit.Framework;
using UnityEditor;
using UnityEditor.AddressableAssets;
using UnityEditor.AddressableAssets.Settings;
using UnityEditor.AddressableAssets.Settings.GroupSchemas;
using UnityEditor.U2D;
using UnityEngine;
using UnityEngine.U2D;

namespace HypeDomino.Tests.EditMode
{
    public sealed class CosmeticContentTests
    {
        const string Root = "Assets/CosmeticContentTests";
        const string Slug = "test_loop";
        const string Remote = Root + "/remote/avatar/" + Slug;
        const string Builtin = Root + "/builtin/avatar/" + Slug;
        const string Table = Root + "/remote/table/" + Slug;
        const string AddressBase = "cosmetic/avatar/" + Slug;
        const int Canvas = 8;
        const int AnimatedFrameCanvas = 256;
        const int LoopFrames = 16;
        const int AtlasOfSixteenAnimatedFrames = 1024;

        AddressableAssetSettings settings;

        [SetUp]
        public void LoadSettings()
        {
            settings = AddressableAssetSettingsDefaultObject.Settings;
            Assert.IsNotNull(settings, "Assets/AddressableAssetsData is missing; run ContentSetup.Apply");
        }

        [TearDown]
        public void RemoveTestContent()
        {
            foreach (var name in new[] { CosmeticContent.GroupName("avatar", Slug), CosmeticContent.GroupName("table", Slug) })
            {
                var group = settings.FindGroup(name);
                if (group != null) settings.RemoveGroup(group);
            }
            var builtin = settings.FindGroup(ContentSetup.BuiltinGroupName);
            foreach (var entry in builtin.entries.Where(entry => entry.address.StartsWith(AddressBase + "/")).ToList()) builtin.RemoveAssetEntry(entry);
            AssetDatabase.DeleteAsset(Root);
        }

        static void WriteFrame(string path, byte shade, int canvas = Canvas)
        {
            Directory.CreateDirectory(Path.GetDirectoryName(path));
            var texture = new Texture2D(canvas, canvas, TextureFormat.RGBA32, false);
            texture.SetPixels32(Enumerable.Repeat(new Color32(shade, shade, shade, 255), canvas * canvas).ToArray());
            File.WriteAllBytes(path, texture.EncodeToPNG());
            Object.DestroyImmediate(texture);
        }

        static void WriteItem(string folder, params string[] frames)
        {
            WriteFrame($"{folder}/static.png", 200);
            for (var at = 0; at < frames.Length; at++) WriteFrame($"{folder}/{frames[at]}", (byte)(at * 40));
            AssetDatabase.Refresh();
        }

        static Texture2D[] PackedPages(SpriteAtlas atlas)
        {
            var preview = typeof(SpriteAtlasExtensions).GetMethod("GetPreviewTextures", BindingFlags.Static | BindingFlags.NonPublic);
            Assert.IsNotNull(preview, "SpriteAtlasExtensions.GetPreviewTextures is gone from this Unity version");
            return (Texture2D[])preview.Invoke(null, new object[] { atlas });
        }

        static AddressableAssetEntry Entry(AddressableAssetGroup group, string address) => group.entries.Single(entry => entry.address == address);

        [Test]
        public void ARemoteItemGetsItsOwnRemoteGroupWithTheManifestAddresses()
        {
            WriteItem(Remote, "anim/0002.png", "anim/0001.png");
            var group = CosmeticContent.Import(Remote);
            Assert.AreEqual("cosmetic-avatar-test_loop", group.Name);
            CollectionAssert.AreEquivalent(new[] { AddressBase + "/static", AddressBase + "/anim" }, group.entries.Select(entry => entry.address).ToArray());
            var bundles = group.GetSchema<BundledAssetGroupSchema>();
            Assert.AreEqual(AddressableAssetSettings.kRemoteLoadPath, bundles.LoadPath.GetName(settings));
            Assert.AreEqual(BundledAssetGroupSchema.BundlePackingMode.PackSeparately, bundles.BundleMode);
            Assert.IsFalse(group.GetSchema<ContentUpdateGroupSchema>().StaticContent);
        }

        [Test]
        public void TheStaticCarriesTheStaticLabelAndTheLoopDoesNot()
        {
            WriteItem(Remote, "anim/0001.png");
            var group = CosmeticContent.Import(Remote);
            CollectionAssert.AreEquivalent(new[] { "cosmetic", "cosmetic-avatar", "cosmetic-static" }, Entry(group, AddressBase + "/static").labels.ToArray());
            CollectionAssert.AreEquivalent(new[] { "cosmetic", "cosmetic-avatar" }, Entry(group, AddressBase + "/anim").labels.ToArray());
        }

        [Test]
        public void TheLoopHoldsItsFramesInNameOrderAsUnmippedFullRectSprites()
        {
            WriteItem(Remote, "anim/0002.png", "anim/0001.png", "anim/0003.png");
            CosmeticContent.Import(Remote);
            var loop = AssetDatabase.LoadAssetAtPath<SpriteLoop>(Remote + "/anim.asset");
            CollectionAssert.AreEqual(new[] { "0001", "0002", "0003" }, loop.Frames.Select(frame => frame.name).ToArray());
            var importer = (TextureImporter)AssetImporter.GetAtPath(Remote + "/anim/0001.png");
            var texture = new TextureImporterSettings();
            importer.ReadTextureSettings(texture);
            Assert.AreEqual(TextureImporterType.Sprite, importer.textureType);
            Assert.IsFalse(importer.mipmapEnabled);
            Assert.AreEqual(SpriteMeshType.FullRect, texture.spriteMeshType);
            Assert.AreEqual(ProjectSetup.SpriteFormat, importer.GetPlatformTextureSettings("Android").format);
        }

        [Test]
        public void EachLoopHasAnAtlasWithTheProjectFormatThatAUiImageCanDraw()
        {
            WriteItem(Remote, "anim/0001.png");
            CosmeticContent.Import(Remote);
            var importer = (SpriteAtlasImporter)AssetImporter.GetAtPath(Remote + "/anim.spriteatlasv2");
            Assert.IsNotNull(importer);
            Assert.IsFalse(importer.packingSettings.enableRotation);
            Assert.IsFalse(importer.packingSettings.enableTightPacking);
            Assert.IsFalse(importer.textureSettings.generateMipMaps);
            foreach (var platform in new[] { "Android", "iPhone" })
            {
                var platformSettings = importer.GetPlatformSettings(platform);
                Assert.IsTrue(platformSettings.overridden, platform);
                Assert.AreEqual(ProjectSetup.SpriteFormat, platformSettings.format, platform);
            }
        }

        [Test]
        public void SixteenEdgeToEdgeFramesOfTheAnimatedFrameCanvasFillOneAtlasOfFourByFourFrames()
        {
            WriteFrame($"{Remote}/static.png", 200, AnimatedFrameCanvas);
            for (var at = 1; at <= LoopFrames; at++) WriteFrame($"{Remote}/anim/{at:0000}.png", (byte)(at * 15), AnimatedFrameCanvas);
            AssetDatabase.Refresh();
            CosmeticContent.Import(Remote);
            var path = Remote + "/anim.spriteatlasv2";
            Assert.AreEqual(CosmeticContent.LoopAtlasPadding, ((SpriteAtlasImporter)AssetImporter.GetAtPath(path)).packingSettings.padding);
            var pages = PackedPages(AssetDatabase.LoadAssetAtPath<SpriteAtlas>(path));
            Assert.AreEqual(1, pages.Length, "atlas pages");
            Assert.AreEqual(new Vector2Int(AtlasOfSixteenAnimatedFrames, AtlasOfSixteenAnimatedFrames), new Vector2Int(pages[0].width, pages[0].height));
        }

        [Test]
        public void ADefaultItemGoesToTheBuiltinGroupAndLeavesItsRemoteGroup()
        {
            WriteItem(Remote);
            CosmeticContent.Import(Remote);
            AssetDatabase.DeleteAsset(Remote);
            WriteItem(Builtin);
            var group = CosmeticContent.Import(Builtin);
            Assert.AreEqual(ContentSetup.BuiltinGroupName, group.Name);
            Assert.IsNotNull(Entry(group, AddressBase + "/static"));
            Assert.IsNull(settings.FindGroup(CosmeticContent.GroupName("avatar", Slug)));
        }

        [Test]
        public void ReimportingAfterTheLoopIsGoneLeavesOnlyTheStatic()
        {
            WriteItem(Remote, "anim/0001.png");
            CosmeticContent.Import(Remote);
            AssetDatabase.DeleteAsset(Remote + "/anim/0001.png");
            var group = CosmeticContent.Import(Remote);
            CollectionAssert.AreEqual(new[] { AddressBase + "/static" }, group.entries.Select(entry => entry.address).ToArray());
            Assert.IsNull(AssetDatabase.LoadAssetAtPath<SpriteLoop>(Remote + "/anim.asset"));
            Assert.IsFalse(File.Exists(Remote + "/anim.spriteatlasv2"));
        }

        [Test]
        public void EveryTableLayerGetsItsOwnLoopAddress()
        {
            WriteItem(Table, "layer/sky/0001.png", "layer/sky/0002.png", "layer/cloth/0001.png");
            var group = CosmeticContent.Import(Table);
            const string tableBase = "cosmetic/table/" + Slug;
            CollectionAssert.AreEquivalent(new[] { tableBase + "/static", tableBase + "/layer/cloth", tableBase + "/layer/sky" }, group.entries.Select(entry => entry.address).ToArray());
            Assert.AreEqual(2, AssetDatabase.LoadAssetAtPath<SpriteLoop>(Table + "/layer/sky.asset").Frames.Length);
            CollectionAssert.AreEquivalent(new[] { "cosmetic", "cosmetic-table" }, Entry(group, tableBase + "/layer/sky").labels.ToArray());
        }

        [Test]
        public void TheTableSliceBecomesTheBorderOfTheStaticAndOfEveryLayerFrame()
        {
            WriteItem(Table, "layer/sky/0001.png", "layer/sky/0002.png", "layer/cloth/0001.png");
            File.WriteAllText($"{Table}/{CosmeticContent.SliceFile}", "{\"depth\":2,\"step\":4}\n");
            AssetDatabase.Refresh();
            CosmeticContent.Import(Table);
            var expected = new Vector4(2f, 2f, 2f, 2f);
            Assert.AreEqual(expected, AssetDatabase.LoadAssetAtPath<Sprite>(Table + "/static.png").border);
            foreach (var frame in AssetDatabase.LoadAssetAtPath<SpriteLoop>(Table + "/layer/sky.asset").Frames) Assert.AreEqual(expected, frame.border);
            Assert.AreEqual(expected, AssetDatabase.LoadAssetAtPath<SpriteLoop>(Table + "/layer/cloth.asset").Frames[0].border);
        }

        [Test]
        public void AnItemWithoutASliceHasNoBorder()
        {
            WriteItem(Table, "layer/sky/0001.png");
            CosmeticContent.Import(Table);
            Assert.AreEqual(Vector4.zero, AssetDatabase.LoadAssetAtPath<Sprite>(Table + "/static.png").border);
        }

        [Test]
        public void TheBorderKeepsTheRingDepthAndCentresOneEdgeStep()
        {
            var border = CosmeticContent.Border(new CosmeticContent.TableSlice { depth = 100, step = 132 }, 1280);
            Assert.AreEqual(new Vector4(574f, 100f, 574f, 100f), border);
        }
    }
}
