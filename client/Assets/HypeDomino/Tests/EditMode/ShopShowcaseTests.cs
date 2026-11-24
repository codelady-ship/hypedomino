using HypeDomino.Game.Shop;
using HypeDomino.Net;
using HypeDomino.UI.Shop;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ShopShowcaseTests
    {
        const ulong DailyEndsUnixMs = 1794700800000;
        const ulong WeeklyEndsUnixMs = 1794787200000;

        const string ShopJson = @"{""version"":12,""sections"":[
{""section"":""featured"",""offers"":[{""id"":""offer.starter"",""kind"":""bundle"",""name_key"":""shop.offer.starter"",""product_id"":""offer_starter"",""crystals"":300,""cosmetics"":[""avatar.comet_fox""],""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]},
{""section"":""daily_rotation"",""ends_at"":""2026-11-15T00:00:00Z"",""offers"":[{""id"":""offer.avatar_comet_fox"",""kind"":""cosmetic"",""name_key"":""shop.offer.avatar_comet_fox"",""price_coins"":400,""cosmetics"":[""avatar.comet_fox""],""limit"":1,""bought"":1,""owned"":[""avatar.comet_fox""],""compensation"":{""avatar.comet_fox"":80},""purchasable"":false,""reason"":""offer.limit_reached""}]},
{""section"":""weekly_rotation"",""ends_at"":""2026-11-16T00:00:00Z"",""offers"":[{""id"":""offer.table_night_market"",""kind"":""cosmetic"",""name_key"":""shop.offer.table_night_market"",""price_coins"":3000,""reference_price_coins"":3500,""cosmetics"":[""table.night_market""],""ends_at"":""2027-01-01T00:00:00Z"",""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]},
{""section"":""event"",""offers"":[]},
{""section"":""chests"",""offers"":[{""id"":""offer.chest_avatar"",""kind"":""chest"",""name_key"":""shop.offer.chest_avatar"",""price_coins"":500,""cosmetics"":[""avatar.paper_kite"",""avatar.moon_owl""],""bought"":0,""owned"":[],""compensation"":{},""odds"":[{""rarity"":""common"",""percent"":85},{""rarity"":""rare"",""percent"":15}],""opens_to_pity"":10,""purchasable"":true}]},
{""section"":""pass"",""offers"":[{""id"":""offer.pass_premium"",""kind"":""pass"",""name_key"":""shop.offer.pass_premium"",""product_id"":""pass_premium"",""cosmetics"":[],""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":false,""reason"":""offer.unsupported""}]},
{""section"":""lucky_corner"",""offers"":[{""id"":""offer.lucky"",""kind"":""cosmetic"",""name_key"":""shop.offer.lucky"",""price_coins"":1,""cosmetics"":[],""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]},
{""section"":""crystals"",""offers"":[{""id"":""offer.crystals_tier1"",""kind"":""crystals"",""name_key"":""shop.offer.crystals_tier1"",""product_id"":""crystals_tier1"",""crystals"":100,""cosmetics"":[],""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true},{""id"":""offer.crystals_tier2"",""kind"":""crystals"",""name_key"":""shop.offer.crystals_tier2"",""product_id"":""crystals_tier2"",""crystals"":550,""cosmetics"":[],""bought"":0,""owned"":[],""compensation"":{},""purchasable"":false,""reason"":""purchase.blocked""}]}]}";

        static ShopShowcase Read(string json)
        {
            Assert.IsTrue(ShopShowcase.TryRead(json, out var showcase));
            return showcase;
        }

        static string Offer(string fields) =>
            @"{""version"":1,""sections"":[{""section"":""featured"",""offers"":[{""id"":""offer.x"",""kind"":""bundle"",""name_key"":""shop.offer.x"",""cosmetics"":[],""owned"":[],""purchasable"":true" + fields + "}]}]}";

        [Test]
        public void SectionsAndCardsKeepTheServerOrder()
        {
            var showcase = Read(ShopJson);

            Assert.AreEqual(12, showcase.Version);
            var ids = new string[showcase.Sections.Length];
            for (var i = 0; i < ids.Length; i++) ids[i] = showcase.Sections[i].Id;
            Assert.AreEqual(new[] { "featured", "daily_rotation", "weekly_rotation", "chests", "pass", "crystals" }, ids);
            var crystals = showcase.Sections[5];
            Assert.AreEqual("offer.crystals_tier1", crystals.Offers[0].OfferId);
            Assert.AreEqual("offer.crystals_tier2", crystals.Offers[1].OfferId);
            Assert.AreEqual(100, crystals.Offers[0].GrantCrystals);
            Assert.AreEqual(550, crystals.Offers[1].GrantCrystals);
        }

        [Test]
        public void EverySectionHasItsTitleKey()
        {
            var showcase = Read(ShopJson);

            Assert.AreEqual(ShopSectionKeys.Featured, showcase.Sections[0].TitleKey);
            Assert.AreEqual("shop.section.featured.title", ShopSectionKeys.Title("featured"));
            Assert.AreEqual("shop.section.daily_rotation.title", ShopSectionKeys.Title("daily_rotation"));
            Assert.AreEqual("shop.section.weekly_rotation.title", ShopSectionKeys.Title("weekly_rotation"));
            Assert.AreEqual("shop.section.event.title", ShopSectionKeys.Title("event"));
            Assert.AreEqual("shop.section.chests.title", ShopSectionKeys.Title("chests"));
            Assert.AreEqual("shop.section.pass.title", ShopSectionKeys.Title("pass"));
            Assert.AreEqual("shop.section.crystals.title", ShopSectionKeys.Title("crystals"));
            Assert.IsNull(ShopSectionKeys.Title("lucky_corner"));
        }

        [Test]
        public void PricesAreShownAsTheServerSentThem()
        {
            var showcase = Read(ShopJson);

            var starter = showcase.Sections[0].Offers[0];
            Assert.IsTrue(starter.Price.InStore);
            Assert.AreEqual("offer_starter", starter.Price.ProductId);
            Assert.IsFalse(starter.HasReferencePrice);
            var fox = showcase.Sections[1].Offers[0];
            Assert.IsFalse(fox.Price.InStore);
            Assert.AreEqual(Currency.Coin, fox.Price.Amount.Currency);
            Assert.AreEqual(400, fox.Price.Amount.Amount);
            Assert.IsFalse(fox.HasReferencePrice);
            var table = showcase.Sections[2].Offers[0];
            Assert.AreEqual(3000, table.Price.Amount.Amount);
            Assert.IsTrue(table.HasReferencePrice);
            Assert.AreEqual(Currency.Coin, table.ReferencePrice.Currency);
            Assert.AreEqual(3500, table.ReferencePrice.Amount);

            var crystal = Read(Offer(@",""price_crystals"":60,""reference_price_crystals"":80,""reference_price_coins"":900")).Sections[0].Offers[0];
            Assert.AreEqual(Currency.Crystal, crystal.Price.Amount.Currency);
            Assert.AreEqual(60, crystal.Price.Amount.Amount);
            Assert.AreEqual(Currency.Crystal, crystal.ReferencePrice.Currency);
            Assert.AreEqual(80, crystal.ReferencePrice.Amount);
        }

        [Test]
        public void OwnedAndLimitStateAreAsReported()
        {
            var showcase = Read(ShopJson);

            var fox = showcase.Sections[1].Offers[0];
            Assert.IsTrue(fox.IsOwned("avatar.comet_fox"));
            Assert.AreEqual(1, fox.Limit);
            Assert.AreEqual(1, fox.Bought);
            Assert.IsFalse(fox.Purchasable);
            Assert.AreEqual("offer.limit_reached", fox.Refusal);
            Assert.AreEqual(ShopBuyKeys.LimitReached, fox.RefusalKey);
            var starter = showcase.Sections[0].Offers[0];
            Assert.IsTrue(starter.Purchasable);
            Assert.AreEqual(string.Empty, starter.Refusal);
            Assert.IsNull(starter.RefusalKey);
            Assert.AreEqual(new[] { "avatar.comet_fox" }, starter.Cosmetics);
            Assert.IsFalse(starter.IsOwned("avatar.comet_fox"));
            Assert.AreEqual(ShopBuyKeys.Unsupported, showcase.Sections[4].Offers[0].RefusalKey);
            Assert.AreEqual(ShopBuyKeys.PurchaseBlocked, showcase.Sections[5].Offers[1].RefusalKey);
            Assert.AreEqual(0, showcase.Sections[5].Offers[0].Limit);
        }

        [Test]
        public void AChestOfferCarriesItsChestCard()
        {
            var showcase = Read(ShopJson);

            var chest = showcase.Sections[3].Offers[0].Chest;
            Assert.IsNotNull(chest);
            Assert.AreEqual(85, chest.Odds[0].Percent);
            Assert.AreEqual(10, chest.OpensToPity);
            Assert.IsNull(showcase.Sections[0].Offers[0].Chest);
        }

        [Test]
        public void CountdownsRunToTheServerEndsOnTheServerClock()
        {
            var showcase = Read(ShopJson);
            var clock = new ServerClock();
            clock.Sync(DailyEndsUnixMs - 90_000, 0, 0);

            var daily = showcase.Sections[1];
            Assert.IsTrue(daily.TrySecondsLeft(clock, 0, out var sectionLeft));
            Assert.AreEqual(90, sectionLeft, 0.001);
            Assert.IsTrue(daily.Offers[0].TrySecondsLeft(clock, 30.5, out var cardLeft));
            Assert.AreEqual(59.5, cardLeft, 0.001);
            Assert.IsTrue(daily.Offers[0].TrySecondsLeft(clock, 120, out var past));
            Assert.AreEqual(0, past);
            Assert.AreEqual(WeeklyEndsUnixMs, showcase.Sections[2].Offers[0].EndsUnixMs);
            Assert.IsFalse(showcase.Sections[0].TrySecondsLeft(clock, 0, out _));
            Assert.IsFalse(showcase.Sections[0].Offers[0].TrySecondsLeft(clock, 0, out _));
        }

        [Test]
        public void AnOfferEndingBeforeItsRotationCountsDownToItsOwnEnd()
        {
            var json = @"{""version"":1,""sections"":[{""section"":""weekly_rotation"",""ends_at"":""2026-11-16T00:00:00Z"",""offers"":[{""id"":""offer.x"",""kind"":""cosmetic"",""name_key"":""shop.offer.x"",""price_coins"":10,""ends_at"":""2026-11-15T18:00:00Z"",""cosmetics"":[],""owned"":[],""purchasable"":true}]}]}";
            var section = Read(json).Sections[0];

            Assert.AreEqual(WeeklyEndsUnixMs, section.EndsUnixMs);
            Assert.AreEqual(1794765600000UL, section.Offers[0].EndsUnixMs);
        }

        [Test]
        public void NoCountdownRunsBeforeTheClockIsSynced()
        {
            var daily = Read(ShopJson).Sections[1];

            Assert.IsFalse(daily.TrySecondsLeft(new ServerClock(), 10, out var seconds));
            Assert.AreEqual(0, seconds);
            Assert.IsFalse(daily.Offers[0].TrySecondsLeft(new ServerClock(), 10, out _));
            Assert.IsFalse(daily.TrySecondsLeft(null, 10, out _));
            Assert.IsFalse(daily.Offers[0].TrySecondsLeft(null, 10, out _));
        }

        [TestCase("")]
        [TestCase(@",""price_coins"":10,""price_crystals"":10")]
        [TestCase(@",""price_coins"":10,""product_id"":""offer_starter""")]
        [TestCase(@",""price_coins"":-5")]
        [TestCase(@",""price_coins"":10,""ends_at"":""soon""")]
        public void AnOfferThatDoesNotHoldTogetherRefusesTheReply(string fields)
        {
            Assert.IsFalse(ShopShowcase.TryRead(Offer(fields), out var showcase));
            Assert.IsNull(showcase);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("not json")]
        [TestCase(@"{""version"":1,""sections"":[{""section"":""daily_rotation"",""ends_at"":""tomorrow"",""offers"":[]}]}")]
        [TestCase(@"{""version"":1,""sections"":[{""section"":""featured"",""offers"":[{""kind"":""bundle"",""name_key"":""shop.offer.x"",""price_crystals"":5}]}]}")]
        public void AnUnreadableReplyIsRefused(string json)
        {
            Assert.IsFalse(ShopShowcase.TryRead(json, out _));
        }

        static string OwnedOffer(string compensation) =>
            @"{""version"":1,""sections"":[{""section"":""featured"",""offers"":[{""id"":""offer.x"",""kind"":""bundle"",""name_key"":""shop.offer.x"",""price_coins"":10,""cosmetics"":[""avatar.x""],""owned"":[""avatar.x""],""compensation"":" + compensation + @",""purchasable"":true}]}]}";

        [Test]
        public void OwnedItemsShowTheCompensationTheServerSent()
        {
            var showcase = Read(ShopJson);

            var fox = showcase.Sections[1].Offers[0];
            Assert.IsTrue(fox.TryCompensationCoins("avatar.comet_fox", out var coins));
            Assert.AreEqual(80, coins);
            Assert.AreEqual(1, fox.CompensationCoins.Count);
            var starter = showcase.Sections[0].Offers[0];
            Assert.IsFalse(starter.TryCompensationCoins("avatar.comet_fox", out _));
            Assert.AreEqual(0, starter.CompensationCoins.Count);
            Assert.AreEqual(0, showcase.Sections[3].Offers[0].CompensationCoins.Count);
        }

        [Test]
        public void EachOfferKeepsItsOwnCompensation()
        {
            var json = @"{""version"":3,""sections"":[
{""section"":""featured"",""offers"":[]},
{""section"":""daily_rotation"",""ends_at"":""2026-11-15T00:00:00Z"",""offers"":[
{""id"":""offer.a"",""kind"":""bundle"",""name_key"":""shop.offer.a"",""price_coins"":900,""coins"":100,""cosmetics"":[""avatar.comet_fox"",""emoji.wave""],""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true},
{""id"":""offer.b"",""kind"":""bundle"",""name_key"":""shop.offer.b"",""price_crystals"":200,""crystals"":20,""cosmetics"":[""avatar.comet_fox"",""table.night_market""],""bought"":0,""owned"":[""avatar.comet_fox"",""table.night_market""],""compensation"":{""avatar.comet_fox"":80,""table.night_market"":0},""purchasable"":true}]}]}";
            var offers = Read(json).Sections[0].Offers;

            Assert.AreEqual(0, offers[0].CompensationCoins.Count);
            Assert.IsTrue(offers[1].TryCompensationCoins("avatar.comet_fox", out var fox));
            Assert.AreEqual(80, fox);
            Assert.IsTrue(offers[1].TryCompensationCoins("table.night_market", out var table));
            Assert.AreEqual(0, table);
        }

        [TestCase("{}")]
        [TestCase("null")]
        public void NoCompensationIsAnEmptyMap(string compensation)
        {
            var card = Read(OwnedOffer(compensation)).Sections[0].Offers[0];

            Assert.AreEqual(0, card.CompensationCoins.Count);
            Assert.IsTrue(card.IsOwned("avatar.x"));
        }

        [TestCase(@"{""avatar.x"":-1}")]
        [TestCase(@"{""avatar.x"":1.5}")]
        [TestCase(@"{""avatar.x"":""80""}")]
        [TestCase(@"{""avatar.x"":null}")]
        [TestCase(@"{""avatar.y"":10}")]
        [TestCase(@"{"""":10}")]
        [TestCase("[]")]
        [TestCase("80")]
        public void CompensationThatIsNotACoinAmountPerOwnedItemRefusesTheReply(string compensation)
        {
            Assert.IsFalse(ShopShowcase.TryRead(OwnedOffer(compensation), out var showcase));
            Assert.IsNull(showcase);
        }

        [Test]
        public void TickingEveryCountdownAllocatesNothing()
        {
            var showcase = Read(ShopJson);
            var clock = new ServerClock();
            clock.Sync(DailyEndsUnixMs - 90_000, 0, 0);
            var now = 0.0;
            TestDelegate step = () =>
            {
                foreach (var section in showcase.Sections)
                {
                    section.TrySecondsLeft(clock, now, out _);
                    foreach (var card in section.Offers) card.TrySecondsLeft(clock, now, out _);
                }
                now += 0.016;
            };
            step();

            Assert.That(step, Allocates.Nothing);
        }
    }
}
