using System;
using System.Collections.Generic;
using HypeDomino.Game.Shop;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Shop;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ShopBuyTests
    {
        const string BoughtJson = "{\"offer\":\"offer.avatar_comet_fox\",\"currency\":\"crystal\",\"balance\":40,\"grant\":{\"cosmetics\":[\"avatar.comet_fox\"]},\"repeated\":false}";
        const string ChestJson = "{\"offer\":\"offer.chest_avatar\",\"currency\":\"coin\",\"balance\":1000,\"grant\":{\"cosmetics\":[\"avatar.moon_owl\"]},\"chest\":{\"item\":\"avatar.moon_owl\",\"rarity\":\"rare\",\"opens_to_pity\":10},\"repeated\":true}";

        [Test]
        public void APurchaseKeepsWhatTheServerAnswered()
        {
            var result = ShopBuyResult.From(new ApiReply(ApiOutcome.Ok, 200, BoughtJson));
            Assert.AreEqual(ShopBuyOutcome.Bought, result.Outcome);
            Assert.AreEqual(ShopBuyKeys.Bought, result.MessageKey);
            Assert.AreEqual("offer.avatar_comet_fox", result.Purchase.offer);
            Assert.IsFalse(result.Purchase.repeated);
            var balances = new Balances(WalletReply.FromJson("{\"coins\":1500,\"crystals\":100}")).After(result.Purchase);
            Assert.AreEqual(1500, balances.Coins.Amount);
            Assert.AreEqual(40, balances.Crystals.Amount);
            Assert.IsFalse(ChestReveal.TryFrom(result.Purchase, out _));
        }

        [Test]
        public void AChestOpenKeepsTheRevealAndTheRepeatFlag()
        {
            var result = ShopBuyResult.From(new ApiReply(ApiOutcome.Ok, 200, ChestJson));
            Assert.AreEqual(ShopBuyOutcome.Bought, result.Outcome);
            Assert.IsTrue(result.Purchase.repeated);
            Assert.IsTrue(ChestReveal.TryFrom(result.Purchase, out var reveal));
            Assert.AreEqual("avatar.moon_owl", reveal.Item);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("not json")]
        [TestCase("{}")]
        public void AnUnreadablePurchaseIsAFailure(string body)
        {
            var result = ShopBuyResult.From(new ApiReply(ApiOutcome.Ok, 200, body));
            Assert.AreEqual(ShopBuyOutcome.Failed, result.Outcome);
            Assert.AreEqual(ShopBuyKeys.Failed, result.MessageKey);
            Assert.IsNull(result.Purchase);
            Assert.IsEmpty(result.Duplicates);
        }

        [Test]
        public void APurchaseKeepsTheDuplicateCompensation()
        {
            const string body = "{\"offer\":\"offer.bundle_night\",\"currency\":\"crystal\",\"balance\":40,\"grant\":{\"coins\":0,\"cosmetics\":[\"avatar.comet_fox\"],\"duplicates\":{\"tile.moon_owl\":40,\"frame.paper_kite\":0}},\"repeated\":false}";
            var result = ShopBuyResult.From(new ApiReply(ApiOutcome.Ok, 200, body));
            Assert.AreEqual(ShopBuyOutcome.Bought, result.Outcome);
            Assert.AreEqual(2, result.Duplicates.Count);
            Assert.AreEqual(40, result.Duplicates["tile.moon_owl"]);
            Assert.AreEqual(0, result.Duplicates["frame.paper_kite"]);
        }

        [Test]
        public void APurchaseWithoutDuplicatesHasNone()
        {
            Assert.IsEmpty(ShopBuyResult.From(new ApiReply(ApiOutcome.Ok, 200, BoughtJson)).Duplicates);
            Assert.IsEmpty(ShopBuyResult.From(new ApiReply(ApiOutcome.Ok, 200, "{\"offer\":\"offer.x\",\"grant\":{\"duplicates\":{}}}")).Duplicates);
        }

        [TestCase("{\"offer\":\"offer.x\",\"grant\":[]}")]
        [TestCase("{\"offer\":\"offer.x\",\"grant\":{\"duplicates\":[\"tile.moon_owl\"]}}")]
        [TestCase("{\"offer\":\"offer.x\",\"grant\":{\"duplicates\":{\"tile.moon_owl\":-1}}}")]
        [TestCase("{\"offer\":\"offer.x\",\"grant\":{\"duplicates\":{\"tile.moon_owl\":1.5}}}")]
        public void APurchaseWithUnreadableDuplicatesIsAFailure(string body)
        {
            var result = ShopBuyResult.From(new ApiReply(ApiOutcome.Ok, 200, body));
            Assert.AreEqual(ShopBuyOutcome.Failed, result.Outcome);
            Assert.IsNull(result.Purchase);
            Assert.IsEmpty(result.Duplicates);
        }

        [TestCase(402, "wallet.insufficient", ShopBuyOutcome.Insufficient, "shop.buy.insufficient")]
        [TestCase(409, "offer.owned", ShopBuyOutcome.Owned, "shop.buy.owned")]
        [TestCase(409, "offer.limit_reached", ShopBuyOutcome.LimitReached, "shop.buy.limit_reached")]
        [TestCase(403, "offer.not_eligible", ShopBuyOutcome.NotEligible, "shop.buy.not_eligible")]
        [TestCase(409, "offer.unavailable", ShopBuyOutcome.Unavailable, "shop.buy.unavailable")]
        [TestCase(404, "offer.unknown", ShopBuyOutcome.Unknown, "shop.buy.unknown")]
        [TestCase(409, "offer.unsupported", ShopBuyOutcome.Unsupported, "shop.buy.unsupported")]
        [TestCase(409, "offer.store_only", ShopBuyOutcome.StoreOnly, "shop.buy.store_only")]
        [TestCase(409, "offer.in_game_only", ShopBuyOutcome.InGameOnly, "shop.buy.in_game_only")]
        [TestCase(403, "purchase.blocked", ShopBuyOutcome.PurchaseBlocked, "shop.buy.purchase_blocked")]
        [TestCase(409, "item.source_not_listed", ShopBuyOutcome.Failed, "notice.generic")]
        [TestCase(400, "request.invalid", ShopBuyOutcome.Failed, "notice.generic")]
        [TestCase(400, "malformed", ShopBuyOutcome.Failed, "notice.generic")]
        [TestCase(401, "bad_mac", ShopBuyOutcome.Failed, "notice.generic")]
        [TestCase(409, "offer.reason_from_a_newer_server", ShopBuyOutcome.Failed, "notice.generic")]
        public void EveryServerRefusalMapsToItsOwnKey(int status, string error, ShopBuyOutcome outcome, string key)
        {
            var result = ShopBuyResult.From(new ApiReply(ApiOutcome.Rejected, status, "{\"error\":\"" + error + "\"}", error));
            Assert.AreEqual(outcome, result.Outcome);
            Assert.AreEqual(key, result.MessageKey);
            Assert.IsNull(result.Purchase);
        }

        [TestCase(ApiOutcome.RateLimited, 429, "rate_limited", ShopBuyOutcome.RateLimited, "shop.buy.rate_limited")]
        [TestCase(ApiOutcome.Unavailable, 503, "purchases.disabled", ShopBuyOutcome.PurchasesDisabled, "shop.buy.purchases_disabled")]
        [TestCase(ApiOutcome.Unavailable, 503, "shop.unavailable", ShopBuyOutcome.Failed, "notice.generic")]
        [TestCase(ApiOutcome.Unavailable, 500, "internal", ShopBuyOutcome.Failed, "notice.generic")]
        [TestCase(ApiOutcome.Offline, HttpReply.NoResponse, null, ShopBuyOutcome.Offline, "shop.buy.offline")]
        [TestCase(ApiOutcome.SignInRequired, 401, "access_rejected", ShopBuyOutcome.SignInRequired, null)]
        public void LimitsAndTransportFailuresMapToTheirKeys(ApiOutcome api, int status, string error, ShopBuyOutcome outcome, string key)
        {
            var result = ShopBuyResult.From(new ApiReply(api, status, null, error));
            Assert.AreEqual(outcome, result.Outcome);
            Assert.AreEqual(key, result.MessageKey);
        }

        [Test]
        public void EveryOutcomeButSignInHasAMessageKey()
        {
            foreach (ShopBuyOutcome outcome in Enum.GetValues(typeof(ShopBuyOutcome)))
            {
                var key = ShopBuyKeys.Of(outcome);
                if (outcome == ShopBuyOutcome.SignInRequired)
                {
                    Assert.IsNull(key);
                    continue;
                }
                StringAssert.IsMatch("^[a-z0-9_]+(\\.[a-z0-9_]+)+$", key, outcome.ToString());
            }
        }

        [Test]
        public void TheShippedBundleHasEveryShopKeyInAzerbaijaniAndEnglish()
        {
            var keys = new HashSet<string>();
            foreach (ShopBuyOutcome outcome in Enum.GetValues(typeof(ShopBuyOutcome)))
            {
                AddShown(keys, ShopBuyKeys.Of(outcome));
            }
            foreach (ShopLoadOutcome outcome in Enum.GetValues(typeof(ShopLoadOutcome)))
            {
                AddShown(keys, ShopLoadKeys.Of(outcome));
            }
            foreach (var section in new[] { "featured", "daily_rotation", "weekly_rotation", "event", "chests", "pass", "crystals" })
            {
                AddShown(keys, ShopSectionKeys.Title(section));
            }
            Assert.AreEqual(24, keys.Count);
            var strings = BuiltinStrings.Load();
            foreach (var key in keys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }

        static void AddShown(HashSet<string> keys, string key)
        {
            if (key != null) keys.Add(key);
        }

        sealed class Sent
        {
            public string Method;
            public string PathAndQuery;
            public string Body;
            public string Mac;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, PathAndQuery = pathAndQuery, Body = body, Mac = mac, Done = done });
        }

        sealed class Auth : IAuthTransport
        {
            public readonly List<Action<HttpReply>> Refreshes = new();

            public void Post(string path, string json, Action<HttpReply> done) => Refreshes.Add(done);
        }

        sealed class Identity : IIdentityProvider
        {
            public bool Supports(AuthProvider provider) => false;

            public void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done) => throw new NotSupportedException();
        }

        sealed class Store : ISecureStore
        {
            readonly Dictionary<string, string> values = new();

            public string Read(string key) => values.TryGetValue(key, out var value) ? value : null;

            public void Write(string key, string value) => values[key] = value;

            public void Delete(string key) => values.Remove(key);
        }

        Api api;
        ShopBuyer buyer;
        List<ShopBuyResult> results;

        [SetUp]
        public void SignIn()
        {
            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            buyer = new ShopBuyer(new ApiClient(api, session));
            results = new List<ShopBuyResult>();
        }

        ShopBuyRequest Body(int index) => JsonUtility.FromJson<ShopBuyRequest>(api.Requests[index].Body);

        [Test]
        public void TheIntentCarriesOnlyTheOfferAndARequestId()
        {
            Assert.IsTrue(buyer.Buy("offer.avatar_comet_fox", results.Add));
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual("/shop/buy", api.Requests[0].PathAndQuery);
            Assert.IsNotEmpty(api.Requests[0].Mac);
            var body = Body(0);
            Assert.AreEqual("offer.avatar_comet_fox", body.offer);
            StringAssert.IsMatch("^[0-9a-f]{32}$", body.request_id);
            Assert.AreEqual("{\"offer\":\"offer.avatar_comet_fox\",\"request_id\":\"" + body.request_id + "\"}", api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, BoughtJson));
            Assert.AreEqual(ShopBuyOutcome.Bought, results[0].Outcome);
            Assert.AreEqual(40, results[0].Purchase.balance);
        }

        [TestCase(null)]
        [TestCase("")]
        public void AnIntentWithoutAnOfferIsNotSent(string offer)
        {
            Assert.IsFalse(buyer.Buy(offer, results.Add));
            Assert.AreEqual(0, api.Requests.Count);
            Assert.IsEmpty(results);
        }

        [Test]
        public void OnlyOnePurchaseIsInFlight()
        {
            Assert.IsTrue(buyer.Buy("offer.avatar_comet_fox", results.Add));
            Assert.IsTrue(buyer.Busy);
            Assert.IsFalse(buyer.Buy("offer.chest_avatar", results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(402, "{\"error\":\"wallet.insufficient\"}"));
            Assert.IsFalse(buyer.Busy);
            Assert.AreEqual(ShopBuyOutcome.Insufficient, results[0].Outcome);
            Assert.IsTrue(buyer.Buy("offer.chest_avatar", results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [TestCase(HttpReply.NoResponse, null)]
        [TestCase(503, "{\"error\":\"shop.unavailable\"}")]
        [TestCase(500, "{\"error\":\"internal\"}")]
        [TestCase(429, "{\"error\":\"rate_limited\"}")]
        [TestCase(200, "not json")]
        public void ARetryAfterAnUnsettledAnswerRepeatsTheSameRequestId(int status, string body)
        {
            buyer.Buy("offer.chest_avatar", results.Add);
            var first = Body(0).request_id;
            api.Requests[0].Done(new HttpReply(status, body));

            buyer.Buy("offer.chest_avatar", results.Add);
            Assert.AreEqual(first, Body(1).request_id);
            api.Requests[1].Done(new HttpReply(200, ChestJson));
            Assert.AreEqual(ShopBuyOutcome.Bought, results[1].Outcome);
            Assert.IsTrue(results[1].Purchase.repeated);
        }

        [TestCase(200, ChestJson)]
        [TestCase(402, "{\"error\":\"wallet.insufficient\"}")]
        [TestCase(409, "{\"error\":\"offer.unavailable\"}")]
        public void ASettledAnswerStartsANewIntent(int status, string body)
        {
            buyer.Buy("offer.chest_avatar", results.Add);
            var first = Body(0).request_id;
            api.Requests[0].Done(new HttpReply(status, body));

            buyer.Buy("offer.chest_avatar", results.Add);
            Assert.AreNotEqual(first, Body(1).request_id);
        }

        [Test]
        public void AnotherOfferNeverReusesTheUnsettledRequestId()
        {
            buyer.Buy("offer.chest_avatar", results.Add);
            var first = Body(0).request_id;
            api.Requests[0].Done(new HttpReply(HttpReply.NoResponse, null));

            buyer.Buy("offer.avatar_comet_fox", results.Add);
            Assert.AreNotEqual(first, Body(1).request_id);
        }
    }
}
