using HypeDomino.UI.Cosmetics;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class CosmeticCollectionTests
    {
        const string ManifestJson = @"{""schemaVersion"":1,""manifestVersion"":5,""items"":[
{""id"":""avatar.starter"",""type"":""avatar"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/starter/static"",""sources"":[{""type"":""default""}]},
{""id"":""frame.starter"",""type"":""frame"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/frame/starter/static"",""sources"":[{""type"":""default""}]},
{""id"":""avatar.comet_fox"",""type"":""avatar"",""rarityId"":""rare"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/comet_fox/static"",""sources"":[{""type"":""shop"",""ref"":""offer.avatar_comet_fox""}]},
{""id"":""avatar.moon_owl"",""type"":""avatar"",""rarityId"":""rare"",""availability"":""retired"",""staticAddress"":""cosmetic/avatar/moon_owl/static"",""sources"":[{""type"":""event"",""ref"":""event.moon""}]},
{""id"":""avatar.paper_kite"",""type"":""avatar"",""rarityId"":""epic"",""availability"":""disabled"",""staticAddress"":""cosmetic/avatar/paper_kite/static"",""sources"":[{""type"":""pass"",""ref"":""pass.s1.10""}]},
{""id"":""emoji.smile"",""type"":""emoji"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/emoji/smile/static"",""sources"":[{""type"":""default""}]},
{""id"":""emoji.wave"",""type"":""emoji"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/emoji/wave/static"",""sources"":[{""type"":""raffle"",""ref"":""raffle.wave""},{""type"":""mission"",""ref"":""mission.wave""},{""type"":""event"",""ref"":""event.wave""}]}]}";

        const string InventoryJson = @"{""items"":[
{""id"":""avatar.starter"",""type"":""avatar"",""source_type"":""default""},
{""id"":""frame.starter"",""type"":""frame"",""source_type"":""default""},
{""id"":""emoji.smile"",""type"":""emoji"",""source_type"":""default""},
{""id"":""avatar.moon_owl"",""type"":""avatar"",""source_type"":""event"",""source_ref"":""event.moon"",""granted_at"":""2026-11-01T10:00:00Z""},
{""id"":""avatar.paper_kite"",""type"":""avatar"",""source_type"":""pass"",""source_ref"":""pass.s1.10"",""granted_at"":""2026-11-02T10:00:00Z""}],
""loadout"":{""avatar"":""avatar.moon_owl"",""frame"":""frame.starter"",""emojis"":[""emoji.smile""]}}";

        static CosmeticCollection Collection(string type, string rarityId = null, bool withInventory = true)
        {
            Assert.IsTrue(CosmeticManifest.TryParse(ManifestJson, out var manifest));
            var collection = new CosmeticCollection(manifest);
            if (withInventory) collection.Show(InventoryReply.FromJson(InventoryJson));
            collection.Filter(type, rarityId);
            return collection;
        }

        static string[] Ids(CosmeticCollection collection)
        {
            var ids = new string[collection.Count];
            for (var at = 0; at < ids.Length; at++) ids[at] = collection[at].Item.Id;
            return ids;
        }

        [Test]
        public void ATabListsTheLiveAndRetiredItemsOfItsTypeInManifestOrder()
        {
            Assert.AreEqual(new[] { "avatar.starter", "avatar.comet_fox", "avatar.moon_owl" }, Ids(Collection(CosmeticAddresses.Avatar)));
            Assert.AreEqual(new[] { "frame.starter" }, Ids(Collection(CosmeticAddresses.Frame)));
            Assert.AreEqual(0, Collection(CosmeticAddresses.Table).Count);
        }

        [Test]
        public void AHeldDisabledItemIsNotShown()
        {
            CollectionAssert.DoesNotContain(Ids(Collection(CosmeticAddresses.Avatar)), "avatar.paper_kite");
            CollectionAssert.DoesNotContain(Ids(Collection(CosmeticAddresses.Avatar, "epic")), "avatar.paper_kite");
        }

        [Test]
        public void HeldAndEquippedComeFromTheServerReply()
        {
            var avatars = Collection(CosmeticAddresses.Avatar);

            Assert.IsTrue(avatars[0].Held);
            Assert.IsFalse(avatars[0].Equipped);
            Assert.IsFalse(avatars[1].Held);
            Assert.IsFalse(avatars[1].Equipped);
            Assert.IsTrue(avatars[2].Held);
            Assert.IsTrue(avatars[2].Equipped);
        }

        [Test]
        public void ARetiredItemIsNoLongerObtainable()
        {
            var avatars = Collection(CosmeticAddresses.Avatar);

            Assert.IsTrue(avatars[0].Obtainable);
            Assert.IsTrue(avatars[1].Obtainable);
            Assert.IsFalse(avatars[2].Obtainable);
        }

        [Test]
        public void EquippedEmojiAreMarkedFromTheLoadoutList()
        {
            var emoji = Collection(CosmeticAddresses.Emoji);

            Assert.AreEqual(new[] { "emoji.smile", "emoji.wave" }, Ids(emoji));
            Assert.IsTrue(emoji[0].Equipped);
            Assert.IsFalse(emoji[1].Equipped);
            Assert.IsFalse(emoji[1].Held);
        }

        [Test]
        public void TheRarityFilterKeepsOneTierAndAnEmptyFilterKeepsAll()
        {
            var avatars = Collection(CosmeticAddresses.Avatar, "rare");
            Assert.AreEqual(new[] { "avatar.comet_fox", "avatar.moon_owl" }, Ids(avatars));

            avatars.Filter(CosmeticAddresses.Avatar, "");
            Assert.AreEqual(3, avatars.Count);

            avatars.Filter(CosmeticAddresses.Avatar, "mythic");
            Assert.AreEqual(0, avatars.Count);
        }

        [Test]
        public void BeforeTheServerReplyNothingIsHeldOrEquipped()
        {
            var avatars = Collection(CosmeticAddresses.Avatar, withInventory: false);

            Assert.AreEqual(3, avatars.Count);
            for (var at = 0; at < avatars.Count; at++)
            {
                Assert.IsFalse(avatars[at].Held);
                Assert.IsFalse(avatars[at].Equipped);
                Assert.IsFalse(avatars.TryEquip(at, out _));
            }
        }

        [Test]
        public void AHeldItemThatIsNotWornGivesTheEquipIntent()
        {
            var avatars = Collection(CosmeticAddresses.Avatar);

            Assert.IsTrue(avatars.TryEquip(0, out var request));
            Assert.AreEqual(@"{""slot"":""avatar"",""items"":[""avatar.starter""]}", request.ToJson());
            Assert.AreEqual("/inventory/equip", CosmeticCollection.EquipPath);
        }

        [Test]
        public void NoIntentForAnItemNotHeldAlreadyWornOrOutOfRange()
        {
            var avatars = Collection(CosmeticAddresses.Avatar);

            Assert.IsFalse(avatars.TryEquip(1, out var notHeld));
            Assert.IsNull(notHeld);
            Assert.IsFalse(avatars.TryEquip(2, out _));
            Assert.IsFalse(avatars.TryEquip(-1, out _));
            Assert.IsFalse(avatars.TryEquip(3, out _));
        }

        [Test]
        public void AnEmojiNotHeldGivesNoIntent()
        {
            var emoji = Collection(CosmeticAddresses.Emoji);

            Assert.IsFalse(emoji.TryEquip(1, out var request));
            Assert.IsNull(request);
        }

        [TestCase(2, 1, true, @"{""slot"":""emoji"",""items"":[""emoji.smile"",""emoji.wave""]}")]
        [TestCase(1, 1, false, null)]
        [TestCase(0, 1, false, null)]
        [TestCase(2, 0, true, @"{""slot"":""emoji"",""items"":[]}")]
        [TestCase(1, 0, true, @"{""slot"":""emoji"",""items"":[]}")]
        public void AHeldEmojiTogglesInTheSlotListUpToTheSlotCount(int slots, int index, bool sent, string json)
        {
            var emoji = Collection(CosmeticAddresses.Emoji);
            emoji.Show(InventoryReply.FromJson(@"{""items"":[{""id"":""emoji.smile""},{""id"":""emoji.wave""}],""loadout"":{""emojis"":[""emoji.smile""]},""emoji_slots"":" + slots + "}"));

            Assert.AreEqual(sent, emoji.TryEquip(index, out var request));
            Assert.AreEqual(json, request?.ToJson());
        }

        [Test]
        public void TheIntentChangesNothingUntilTheServerAnswers()
        {
            var avatars = Collection(CosmeticAddresses.Avatar);

            Assert.IsTrue(avatars.TryEquip(0, out _));
            Assert.IsFalse(avatars[0].Equipped);
            Assert.IsTrue(avatars[2].Equipped);

            avatars.Apply(EquipReply.FromJson(@"{""loadout"":{""avatar"":""avatar.starter"",""frame"":""frame.starter"",""emojis"":[""emoji.smile""]}}"));

            Assert.IsTrue(avatars[0].Equipped);
            Assert.IsFalse(avatars[2].Equipped);
        }

        [Test]
        public void AnItemNotHeldShowsTheWaysToEarnItFromItsSources()
        {
            var avatars = Collection(CosmeticAddresses.Avatar);

            Assert.AreEqual(new[] { "shop" }, avatars[1].EarnPaths);
            Assert.AreEqual(new[] { "shop" }, avatars[1].Item.Sources);
        }

        [Test]
        public void AnUnknownSourceTypeIsSkippedAndTheOthersKeepManifestOrder()
        {
            var emoji = Collection(CosmeticAddresses.Emoji);

            Assert.AreEqual(new[] { "mission", "event" }, emoji[1].EarnPaths);
        }

        [Test]
        public void AHeldItemShowsNoWayToEarnIt()
        {
            var avatars = Collection(CosmeticAddresses.Avatar);

            Assert.AreEqual(new[] { "default" }, avatars[0].Item.Sources);
            Assert.IsEmpty(avatars[0].EarnPaths);
            Assert.IsEmpty(avatars[2].EarnPaths);
        }

        [Test]
        public void ARetiredItemNotHeldShowsNoWayToEarnIt()
        {
            var avatars = Collection(CosmeticAddresses.Avatar);

            avatars.Show(InventoryReply.FromJson(@"{""items"":[{""id"":""avatar.starter""}],""loadout"":{""avatar"":""avatar.starter""}}"));

            Assert.IsFalse(avatars[2].Held);
            Assert.AreEqual(new[] { "event" }, avatars[2].Item.Sources);
            Assert.IsEmpty(avatars[2].EarnPaths);
        }

        [Test]
        public void ANewInventoryReplyReplacesWhatWasHeld()
        {
            var avatars = Collection(CosmeticAddresses.Avatar);

            avatars.Show(InventoryReply.FromJson(@"{""items"":[{""id"":""avatar.starter""},{""id"":""avatar.comet_fox""}],""loadout"":{""avatar"":""avatar.comet_fox""}}"));

            Assert.IsTrue(avatars[1].Held);
            Assert.IsTrue(avatars[1].Equipped);
            Assert.IsFalse(avatars[2].Held);
            Assert.IsFalse(avatars[2].Equipped);
        }
    }
}
