using System;
using System.Collections.Generic;
using HypeDomino.Game.Shop;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.Net.Integrity;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ShopCheckoutTests
    {
        const string ShopJson = @"{""version"":12,""sections"":[
{""section"":""featured"",""offers"":[{""id"":""offer.starter"",""kind"":""bundle"",""name_key"":""shop.offer.starter"",""product_id"":""offer_starter"",""crystals"":300,""cosmetics"":[""avatar.comet_fox""],""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]},
{""section"":""daily_rotation"",""ends_at"":""2026-11-15T00:00:00Z"",""offers"":[{""id"":""offer.avatar_comet_fox"",""kind"":""cosmetic"",""name_key"":""shop.offer.avatar_comet_fox"",""price_coins"":400,""cosmetics"":[""avatar.comet_fox""],""limit"":1,""bought"":1,""owned"":[""avatar.comet_fox""],""compensation"":{""avatar.comet_fox"":80},""purchasable"":false,""reason"":""offer.limit_reached""}]},
{""section"":""weekly_rotation"",""ends_at"":""2026-11-16T00:00:00Z"",""offers"":[{""id"":""offer.table_night_market"",""kind"":""cosmetic"",""name_key"":""shop.offer.table_night_market"",""price_coins"":3000,""reference_price_coins"":3500,""cosmetics"":[""table.night_market""],""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]},
{""section"":""chests"",""offers"":[{""id"":""offer.chest_avatar"",""kind"":""chest"",""name_key"":""shop.offer.chest_avatar"",""price_coins"":500,""cosmetics"":[""avatar.paper_kite"",""avatar.moon_owl""],""bought"":0,""owned"":[],""compensation"":{},""odds"":[{""rarity"":""common"",""percent"":85},{""rarity"":""rare"",""percent"":15}],""opens_to_pity"":10,""purchasable"":true}]},
{""section"":""pass"",""offers"":[{""id"":""offer.pass_premium"",""kind"":""pass"",""name_key"":""shop.offer.pass_premium"",""product_id"":""pass_premium"",""limit"":1,""bought"":0,""owned"":[],""compensation"":{},""purchasable"":true}]}]}";

        const string StarterCheckoutJson = "{\"offer\":\"offer.starter\",\"product_id\":\"offer_starter\",\"account_token\":\"account-token-test\"}";
        const string StarterGrantedJson = "{\"platform\":\"google\",\"transaction_id\":\"GPA.1234\",\"offer\":\"offer.starter\",\"state\":\"granted\",\"repeated\":false,\"crystals\":300,\"finished\":true}";
        const string TableBoughtJson = "{\"offer\":\"offer.table_night_market\",\"currency\":\"coin\",\"balance\":200,\"grant\":{\"cosmetics\":[\"table.night_market\"]},\"repeated\":false}";

        sealed class Sent
        {
            public string PathAndQuery;
            public string Body;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { PathAndQuery = pathAndQuery, Body = body, Done = done });
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

        sealed class Billing : IStoreBilling
        {
            public readonly List<string> Products = new();
            public readonly List<StoreReceipt> Finished = new();
            public Action<StoreBillingResult> Answer;

            public bool CanPurchase => true;

            public void Purchase(string productId, string accountToken, Action<StoreBillingResult> done)
            {
                Products.Add(productId);
                Answer = done;
            }

            public void Finish(StoreReceipt receipt, bool finishedByServer) => Finished.Add(receipt);
        }

        static void NoProof(IntegrityPurpose purpose, Action<IntegrityProof> done) => done(null);

        Api api;
        ApiClient client;
        Billing billing;
        ShopBuyer buyer;
        ShopCheckout checkout;
        ShopShowcase showcase;
        List<ShopCheckoutResult> results;

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
            client = new ApiClient(api, session);
            billing = new Billing();
            buyer = new ShopBuyer(client);
            checkout = new ShopCheckout(buyer, new ShopPurchase(client, billing, NoProof));
            Assert.IsTrue(ShopShowcase.TryRead(ShopJson, out showcase));
            results = new List<ShopCheckoutResult>();
        }

        OfferCard StoreOffer => showcase.Sections[0].Offers[0];
        OfferCard OwnedOffer => showcase.Sections[1].Offers[0];
        OfferCard TableOffer => showcase.Sections[2].Offers[0];
        OfferCard ChestOffer => showcase.Sections[3].Offers[0];
        OfferCard PassOffer => showcase.Sections[4].Offers[0];

        void PayForTheStarterInTheStore()
        {
            checkout.Open(StoreOffer);
            Assert.IsTrue(checkout.Confirm(results.Add));
            api.Requests[0].Done(new HttpReply(200, StarterCheckoutJson));
            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Purchased, new StoreReceipt(StorePlatform.Google, "purchase-token")));
        }

        string SentOffer(int index) => JsonUtility.FromJson<ShopBuyRequest>(api.Requests[index].Body).offer;

        [Test]
        public void APurchaseTakesTwoTaps()
        {
            var taps = 0;
            taps += checkout.Open(TableOffer) ? 1 : 0;
            taps += checkout.Confirm(results.Add) ? 1 : 0;

            Assert.AreEqual(2, taps);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual(ShopBuyer.Path, api.Requests[0].PathAndQuery);
            Assert.AreEqual("offer.table_night_market", SentOffer(0));
            Assert.AreEqual(ShopCheckoutStep.Buying, checkout.Step);

            api.Requests[0].Done(new HttpReply(200, TableBoughtJson));
            Assert.AreEqual(ShopCheckoutStep.Done, checkout.Step);
            Assert.IsFalse(checkout.Result.InStore);
            Assert.IsTrue(checkout.Result.Granted);
            Assert.AreEqual(ShopBuyOutcome.Bought, checkout.Result.Bought.Outcome);
            Assert.AreEqual(ShopBuyKeys.Bought, checkout.Result.MessageKey);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(200, results[0].Bought.Purchase.balance);
            Assert.IsEmpty(billing.Products);
        }

        [Test]
        public void ThePreviewShowsTheCardAsTheServerSentIt()
        {
            Assert.IsTrue(checkout.Open(OwnedOffer));

            Assert.AreEqual(ShopCheckoutStep.Preview, checkout.Step);
            Assert.AreSame(OwnedOffer, checkout.Offer);
            Assert.AreEqual(400, checkout.Offer.Price.Amount.Amount);
            Assert.IsTrue(checkout.Offer.TryCompensationCoins("avatar.comet_fox", out var coins));
            Assert.AreEqual(80, coins);
            Assert.AreEqual(ShopBuyKeys.LimitReached, checkout.Offer.RefusalKey);
        }

        [Test]
        public void AnOfferTheServerRefusesCannotBeConfirmed()
        {
            checkout.Open(OwnedOffer);

            Assert.IsFalse(checkout.CanConfirm);
            Assert.IsFalse(checkout.Confirm(results.Add));
            Assert.AreEqual(0, api.Requests.Count);
            Assert.AreEqual(ShopCheckoutStep.Preview, checkout.Step);
        }

        [Test]
        public void AStoreProductIsPaidInTheStoreAndNeverWithTheInGameIntent()
        {
            checkout.Open(StoreOffer);
            Assert.IsTrue(checkout.CanConfirm);
            Assert.IsTrue(checkout.Confirm(results.Add));
            Assert.AreEqual(ShopCheckoutStep.Buying, checkout.Step);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual(ShopPurchase.BeginPath, api.Requests[0].PathAndQuery);
            Assert.AreEqual("{\"offer\":\"offer.starter\"}", api.Requests[0].Body);

            api.Requests[0].Done(new HttpReply(200, StarterCheckoutJson));
            CollectionAssert.AreEqual(new[] { "offer_starter" }, billing.Products);
            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Purchased, new StoreReceipt(StorePlatform.Google, "purchase-token")));
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual(ShopPurchase.CompletePath, api.Requests[1].PathAndQuery);
            Assert.AreEqual(ShopCheckoutStep.Buying, checkout.Step);

            api.Requests[1].Done(new HttpReply(200, StarterGrantedJson));
            Assert.AreEqual(ShopCheckoutStep.Done, checkout.Step);
            Assert.IsTrue(checkout.Result.InStore);
            Assert.IsTrue(checkout.Result.Granted);
            Assert.AreEqual(ShopPurchaseOutcome.Granted, checkout.Result.Paid.Outcome);
            Assert.AreEqual(300, checkout.Result.Paid.Reply.crystals);
            Assert.AreEqual(ShopBuyKeys.Bought, checkout.Result.MessageKey);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(1, billing.Finished.Count);
            Assert.IsFalse(checkout.CanConfirm);
            Assert.IsFalse(checkout.Confirm(results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void WithoutStoreBillingAStoreProductEndsAsUnavailableAndSendsNothing()
        {
            var unbilled = new ShopCheckout(buyer, new ShopPurchase(client, new NoStoreBilling(), NoProof));
            unbilled.Open(StoreOffer);

            Assert.IsTrue(unbilled.Confirm(results.Add));
            Assert.AreEqual(0, api.Requests.Count);
            Assert.AreEqual(ShopCheckoutStep.Done, unbilled.Step);
            Assert.IsTrue(unbilled.Result.InStore);
            Assert.IsFalse(unbilled.Result.Granted);
            Assert.AreEqual(ShopPurchaseOutcome.StoreUnavailable, unbilled.Result.Paid.Outcome);
            Assert.AreEqual(ShopPurchaseKeys.StoreUnavailable, unbilled.Result.MessageKey);
            Assert.AreEqual(1, results.Count);
            Assert.IsTrue(unbilled.Close());
        }

        [Test]
        public void ThePremiumPassIsAShopOfferPaidInTheStore()
        {
            checkout.Open(PassOffer);
            Assert.IsTrue(checkout.Confirm(results.Add));
            Assert.AreEqual(ShopPurchase.BeginPath, api.Requests[0].PathAndQuery);
            Assert.AreEqual("{\"offer\":\"offer.pass_premium\"}", api.Requests[0].Body);

            api.Requests[0].Done(new HttpReply(409, "{\"error\":\"offer.owned\"}"));
            Assert.AreEqual(ShopCheckoutStep.Done, checkout.Step);
            Assert.AreEqual(ShopPurchaseOutcome.Owned, checkout.Result.Paid.Outcome);
            Assert.AreEqual(ShopBuyKeys.Owned, checkout.Result.MessageKey);
            Assert.IsEmpty(billing.Products);
            Assert.IsFalse(checkout.CanConfirm);
        }

        [Test]
        public void AStorePaymentInFlightKeepsThePreviewOpen()
        {
            checkout.Open(StoreOffer);
            checkout.Confirm(results.Add);
            api.Requests[0].Done(new HttpReply(200, StarterCheckoutJson));

            Assert.IsFalse(checkout.CanConfirm);
            Assert.IsFalse(checkout.Confirm(results.Add));
            Assert.IsFalse(checkout.Close());
            Assert.IsFalse(checkout.Open(TableOffer));
            Assert.AreSame(StoreOffer, checkout.Offer);

            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Cancelled));
            Assert.AreEqual(ShopCheckoutStep.Done, checkout.Step);
            Assert.AreEqual(ShopPurchaseOutcome.Cancelled, checkout.Result.Paid.Outcome);
            Assert.IsNull(checkout.Result.MessageKey);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.IsTrue(checkout.Close());
        }

        [Test]
        public void AnUnfinishedStorePaymentIsSentAgainWithTheSameReceipt()
        {
            PayForTheStarterInTheStore();
            api.Requests[1].Done(new HttpReply(503, "{\"error\":\"receipt.store_unavailable\"}"));

            Assert.AreEqual(ShopCheckoutStep.Done, checkout.Step);
            Assert.AreEqual(ShopPurchaseOutcome.Unfinished, checkout.Result.Paid.Outcome);
            Assert.AreEqual(ShopPurchaseKeys.Unfinished, checkout.Result.MessageKey);
            Assert.IsTrue(checkout.CanConfirm);
            Assert.IsTrue(checkout.Confirm(results.Add));
            Assert.AreEqual(3, api.Requests.Count);
            Assert.AreEqual(ShopPurchase.CompletePath, api.Requests[2].PathAndQuery);
            Assert.AreEqual(api.Requests[1].Body, api.Requests[2].Body);
            Assert.AreEqual(1, billing.Products.Count);

            api.Requests[2].Done(new HttpReply(200, StarterGrantedJson));
            Assert.IsTrue(checkout.Result.Granted);
            Assert.IsFalse(checkout.CanConfirm);
            Assert.AreEqual(2, results.Count);
        }

        [Test]
        public void AnotherStoreOfferWaitsUntilTheUnfinishedPaymentIsSettled()
        {
            PayForTheStarterInTheStore();
            api.Requests[1].Done(new HttpReply(HttpReply.NoResponse, null));

            Assert.IsTrue(checkout.Open(PassOffer));
            Assert.IsFalse(checkout.CanConfirm);
            Assert.IsFalse(checkout.Confirm(results.Add));
            Assert.AreEqual(ShopCheckoutStep.Preview, checkout.Step);

            Assert.IsTrue(checkout.Open(TableOffer));
            Assert.IsTrue(checkout.CanConfirm);

            Assert.IsTrue(checkout.Open(StoreOffer));
            Assert.IsTrue(checkout.CanConfirm);
            Assert.IsTrue(checkout.Confirm(results.Add));
            Assert.AreEqual(3, api.Requests.Count);
            Assert.AreEqual(ShopPurchase.CompletePath, api.Requests[2].PathAndQuery);
        }

        [Test]
        public void AStoreAnswerBeforePaymentIsBoughtAgainFromAFreshPreview()
        {
            checkout.Open(StoreOffer);
            checkout.Confirm(results.Add);
            api.Requests[0].Done(new HttpReply(HttpReply.NoResponse, null));

            Assert.AreEqual(ShopPurchaseOutcome.Offline, checkout.Result.Paid.Outcome);
            Assert.AreEqual(ShopBuyKeys.Offline, checkout.Result.MessageKey);
            Assert.IsFalse(checkout.CanConfirm);
            Assert.IsTrue(checkout.Open(StoreOffer));
            Assert.IsTrue(checkout.Confirm(results.Add));
            Assert.AreEqual(ShopPurchase.BeginPath, api.Requests[1].PathAndQuery);
        }

        [Test]
        public void NothingIsConfirmedWithoutAnOpenPreview()
        {
            Assert.AreEqual(ShopCheckoutStep.Closed, checkout.Step);
            Assert.IsFalse(checkout.CanConfirm);
            Assert.IsFalse(checkout.Confirm(results.Add));
            Assert.IsFalse(checkout.Open(null));
            Assert.AreEqual(0, api.Requests.Count);
        }

        [Test]
        public void ASecondConfirmWhileBuyingSendsNothing()
        {
            checkout.Open(TableOffer);
            checkout.Confirm(results.Add);

            Assert.IsFalse(checkout.CanConfirm);
            Assert.IsFalse(checkout.Confirm(results.Add));
            Assert.AreEqual(1, api.Requests.Count);
        }

        [Test]
        public void ThePreviewStaysOpenUntilTheServerAnswers()
        {
            checkout.Open(TableOffer);
            checkout.Confirm(results.Add);

            Assert.IsFalse(checkout.Close());
            Assert.IsFalse(checkout.Open(ChestOffer));
            Assert.AreSame(TableOffer, checkout.Offer);

            api.Requests[0].Done(new HttpReply(402, "{\"error\":\"wallet.insufficient\"}"));
            Assert.IsTrue(checkout.Close());
            Assert.AreEqual(ShopCheckoutStep.Closed, checkout.Step);
            Assert.IsNull(checkout.Offer);
        }

        [Test]
        public void InsufficientBalanceEndsWithItsKeyAndNoRetry()
        {
            checkout.Open(ChestOffer);
            checkout.Confirm(results.Add);
            api.Requests[0].Done(new HttpReply(402, "{\"error\":\"wallet.insufficient\"}"));

            Assert.AreEqual(ShopCheckoutStep.Done, checkout.Step);
            Assert.AreEqual(ShopBuyOutcome.Insufficient, checkout.Result.Bought.Outcome);
            Assert.AreEqual(ShopBuyKeys.Insufficient, checkout.Result.MessageKey);
            Assert.IsFalse(checkout.CanConfirm);
            Assert.IsFalse(checkout.Confirm(results.Add));
            Assert.AreEqual(1, api.Requests.Count);
        }

        [TestCase(HttpReply.NoResponse, null, ShopBuyOutcome.Offline)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", ShopBuyOutcome.RateLimited)]
        [TestCase(503, "{\"error\":\"shop.unavailable\"}", ShopBuyOutcome.Failed)]
        [TestCase(503, "{\"error\":\"purchases.disabled\"}", ShopBuyOutcome.PurchasesDisabled)]
        [TestCase(200, "not json", ShopBuyOutcome.Failed)]
        public void AnUnsettledFailureRetriesTheSameIntent(int status, string body, ShopBuyOutcome outcome)
        {
            checkout.Open(TableOffer);
            checkout.Confirm(results.Add);
            var first = JsonUtility.FromJson<ShopBuyRequest>(api.Requests[0].Body).request_id;
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.AreEqual(ShopCheckoutStep.Done, checkout.Step);
            Assert.AreEqual(outcome, checkout.Result.Bought.Outcome);
            Assert.IsTrue(checkout.CanConfirm);
            Assert.IsTrue(checkout.Confirm(results.Add));
            Assert.AreEqual(first, JsonUtility.FromJson<ShopBuyRequest>(api.Requests[1].Body).request_id);

            api.Requests[1].Done(new HttpReply(200, TableBoughtJson));
            Assert.AreEqual(ShopBuyOutcome.Bought, checkout.Result.Bought.Outcome);
            Assert.IsFalse(checkout.CanConfirm);
            Assert.AreEqual(2, results.Count);
        }

        [TestCase(409, "{\"error\":\"offer.store_only\"}")]
        [TestCase(409, "{\"error\":\"offer.owned\"}")]
        public void ASettledRefusalIsNotRetried(int status, string body)
        {
            checkout.Open(TableOffer);
            checkout.Confirm(results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.AreEqual(ShopCheckoutStep.Done, checkout.Step);
            Assert.IsFalse(checkout.CanConfirm);
        }

        [Test]
        public void ReopeningAfterAnAnswerStartsAFreshPreview()
        {
            checkout.Open(TableOffer);
            checkout.Confirm(results.Add);
            api.Requests[0].Done(new HttpReply(HttpReply.NoResponse, null));

            Assert.IsTrue(checkout.Open(ChestOffer));
            Assert.AreEqual(ShopCheckoutStep.Preview, checkout.Step);
            Assert.AreSame(ChestOffer, checkout.Offer);
            Assert.IsTrue(checkout.CanConfirm);
            Assert.IsTrue(checkout.Confirm(results.Add));
            Assert.AreEqual("offer.chest_avatar", SentOffer(1));
        }
    }
}
