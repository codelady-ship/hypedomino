using HypeDomino.UI.Localization;
using HypeDomino.UI.Shop;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ShopTests
    {
        const string ShopJson = @"{""version"":4,""sections"":[
{""section"":""daily_rotation"",""ends_at"":""2026-11-03T00:00:00Z"",""offers"":[{""id"":""offer.avatar_comet_fox"",""kind"":""cosmetic"",""name_key"":""shop.offer.avatar_comet_fox"",""price_crystals"":60,""cosmetics"":[""avatar.comet_fox""],""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]},
{""section"":""chests"",""offers"":[{""id"":""offer.chest_avatar"",""kind"":""chest"",""name_key"":""shop.offer.chest_avatar"",""price_coins"":500,""cosmetics"":[""avatar.paper_kite"",""avatar.moon_owl""],""bought"":0,""owned"":[""avatar.paper_kite""],""compensation"":{},""odds"":[{""rarity"":""common"",""percent"":70},{""rarity"":""rare"",""percent"":30}],""opens_to_pity"":4,""purchasable"":false,""reason"":""wallet.insufficient""}]}]}";

        [Test]
        public void TheChestCardShowsWhatTheServerSentWithoutComputingIt()
        {
            var shop = ShopReply.FromJson(ShopJson);
            Assert.IsFalse(ChestCard.TryFrom(shop.sections[0].offers[0], out _));
            Assert.IsTrue(ChestCard.TryFrom(shop.sections[1].offers[0], out var card));
            Assert.AreEqual("offer.chest_avatar", card.OfferId);
            Assert.AreEqual(Currency.Coin, card.Price.Currency);
            Assert.AreEqual(500, card.Price.Amount);
            Assert.AreEqual(new[] { "avatar.paper_kite", "avatar.moon_owl" }, card.Pool);
            Assert.IsTrue(card.IsOwned("avatar.paper_kite"));
            Assert.IsFalse(card.IsOwned("avatar.moon_owl"));
            Assert.AreEqual(2, card.Odds.Length);
            Assert.AreEqual("rarity.common.name", card.Odds[0].RarityNameKey);
            Assert.AreEqual(70, card.Odds[0].Percent);
            Assert.AreEqual(30, card.Odds[1].Percent);
            Assert.AreEqual(4, card.OpensToPity);
            Assert.IsFalse(card.Purchasable);
            Assert.AreEqual("wallet.insufficient", card.Refusal);
        }

        [Test]
        public void TheRevealShowsOnlyTheGrantedItem()
        {
            var opened = PurchaseReply.FromJson(@"{""offer"":""offer.chest_avatar"",""currency"":""coin"",""balance"":1000,""grant"":{""cosmetics"":[""avatar.moon_owl""]},""chest"":{""item"":""avatar.moon_owl"",""rarity"":""rare"",""opens_to_pity"":10},""repeated"":false}");
            Assert.IsTrue(ChestReveal.TryFrom(opened, out var reveal));
            Assert.AreEqual("avatar.moon_owl", reveal.Item);
            Assert.AreEqual("rarity.rare.name", reveal.RarityNameKey);
            Assert.AreEqual(10, reveal.OpensToPity);
            var bought = PurchaseReply.FromJson(@"{""offer"":""offer.avatar_comet_fox"",""currency"":""crystal"",""balance"":40,""repeated"":false}");
            Assert.IsFalse(ChestReveal.TryFrom(bought, out _));
        }

        [Test]
        public void BalancesKeepCoinsAndCrystalsApartAndFollowThePurchaseCurrency()
        {
            var balances = new Balances(WalletReply.FromJson(@"{""coins"":1500,""crystals"":100,""entries"":[]}"));
            Assert.AreEqual(1500, balances.Coins.Amount);
            Assert.AreEqual(100, balances.Crystals.Amount);
            Assert.AreEqual("currency.coin.name", balances.Coins.NameKey);
            Assert.AreEqual("currency.crystal.name", balances.Crystals.NameKey);
            var afterChest = balances.After(PurchaseReply.FromJson(@"{""currency"":""coin"",""balance"":1000}"));
            Assert.AreEqual(1000, afterChest.Coins.Amount);
            Assert.AreEqual(100, afterChest.Crystals.Amount);
            var afterCrystals = afterChest.After(PurchaseReply.FromJson(@"{""currency"":""crystal"",""balance"":40}"));
            Assert.AreEqual(1000, afterCrystals.Coins.Amount);
            Assert.AreEqual(40, afterCrystals.Crystals.Amount);
            var unknown = afterCrystals.After(PurchaseReply.FromJson(@"{""currency"":""gold"",""balance"":5}"));
            Assert.AreEqual(1000, unknown.Coins.Amount);
            Assert.AreEqual(40, unknown.Crystals.Amount);
        }

        [TestCase("currency.coin.name", "en", "Coin")]
        [TestCase("currency.coin.name", "az", "Sikkə")]
        [TestCase("currency.coin.name", "tr", "Coin")]
        [TestCase("currency.coin.name", "ru", "Монета")]
        [TestCase("currency.crystal.name", "en", "Crystal")]
        [TestCase("currency.crystal.name", "az", "Kristal")]
        [TestCase("currency.crystal.name", "tr", "Kristal")]
        [TestCase("currency.crystal.name", "ru", "Кристалл")]
        public void CurrencyNamesComeFromTheBuiltinBundle(string key, string language, string text)
        {
            var strings = BuiltinStrings.Load();
            Assert.AreEqual(text, strings.Value(key, language));
        }
    }
}
