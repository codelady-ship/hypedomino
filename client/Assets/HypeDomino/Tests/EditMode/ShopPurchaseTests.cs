using System;
using System.Collections.Generic;
using HypeDomino.Core.Analytics;
using HypeDomino.Game.Shop;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.Net.Integrity;
using HypeDomino.UI.Localization;
using Hypedomino.V1;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class ShopPurchaseTests
    {
        const string Offer = "offer.crystals_tier2";
        const string CheckoutJson = "{\"offer\":\"offer.crystals_tier2\",\"product_id\":\"crystals_tier2\",\"account_token\":\"account-token-test\"}";
        const string GrantedJson = "{\"platform\":\"google\",\"transaction_id\":\"GPA.1234\",\"offer\":\"offer.crystals_tier2\",\"state\":\"granted\",\"repeated\":false,\"crystals\":650,\"finished\":true}";
        const string PendingJson = "{\"platform\":\"google\",\"transaction_id\":\"GPA.1234\",\"offer\":\"offer.crystals_tier2\",\"state\":\"pending\",\"repeated\":false,\"crystals\":0,\"finished\":false}";

        sealed class Sent
        {
            public string Method;
            public string PathAndQuery;
            public string Body;
            public string Mac;
            public IReadOnlyList<ApiHeader> Headers;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiHeaderTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, PathAndQuery = pathAndQuery, Body = body, Mac = mac, Done = done });

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, IReadOnlyList<ApiHeader> headers, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, PathAndQuery = pathAndQuery, Body = body, Mac = mac, Headers = headers, Done = done });
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
            public readonly List<string> AccountTokens = new();
            public readonly List<StoreReceipt> Finished = new();
            public readonly List<bool> FinishedByServer = new();
            public Action<StoreBillingResult> Answer;

            public bool CanPurchase { get; set; } = true;

            public void Purchase(string productId, string accountToken, Action<StoreBillingResult> done)
            {
                Products.Add(productId);
                AccountTokens.Add(accountToken);
                Answer = done;
            }

            public void Finish(StoreReceipt receipt, bool finishedByServer)
            {
                Finished.Add(receipt);
                FinishedByServer.Add(finishedByServer);
            }
        }

        sealed class Proofs
        {
            public readonly List<IntegrityPurpose> Purposes = new();
            public IntegrityProof Next;

            public void Request(IntegrityPurpose purpose, Action<IntegrityProof> done)
            {
                Purposes.Add(purpose);
                done(Next);
            }
        }

        sealed class AnalyticsSink : IAnalyticsSender
        {
            public readonly List<AnalyticsEvent> Sent = new();

            public void Send(string analyticsId, IReadOnlyList<AnalyticsEvent> batch, Action<bool> done)
            {
                for (var i = 0; i < batch.Count; i++) Sent.Add(batch[i]);
                done(true);
            }
        }

        Api api;
        Billing billing;
        Proofs proofs;
        AnalyticsSink analytics;
        AnalyticsQueue events;
        ShopPurchase purchase;
        List<ShopPurchaseResult> results;

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
            billing = new Billing();
            proofs = new Proofs { Next = new IntegrityProof { PlayIntegrityToken = "play-token" } };
            analytics = new AnalyticsSink();
            events = new AnalyticsQueue(analytics, () => 1700000000000, 10, 10);
            events.Consent("3f2b8c1e-5a4d-4e8f-9b7a-1c2d3e4f5a6b");
            purchase = new ShopPurchase(new ApiClient(api, session), billing, proofs.Request, events);
            results = new List<ShopPurchaseResult>();
        }

        void PayInStore(string token = "purchase-token")
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Purchased, new StoreReceipt(StorePlatform.Google, token)));
        }

        [Test]
        public void ABuyBeginsPaysInTheStoreAndCompletesWithTheReceiptAndAPurchaseProof()
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            Assert.IsTrue(purchase.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual("/shop/begin", api.Requests[0].PathAndQuery);
            Assert.AreEqual("{\"offer\":\"offer.crystals_tier2\"}", api.Requests[0].Body);
            Assert.IsNotEmpty(api.Requests[0].Mac);
            Assert.IsNull(api.Requests[0].Headers);

            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            CollectionAssert.AreEqual(new[] { "crystals_tier2" }, billing.Products);
            CollectionAssert.AreEqual(new[] { "account-token-test" }, billing.AccountTokens);
            Assert.IsEmpty(proofs.Purposes);

            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Purchased, new StoreReceipt(StorePlatform.Google, "purchase-token")));
            CollectionAssert.AreEqual(new[] { IntegrityPurpose.Purchase }, proofs.Purposes);
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual("/shop/complete", api.Requests[1].PathAndQuery);
            Assert.AreEqual("{\"offer\":\"offer.crystals_tier2\",\"platform\":\"google\",\"token\":\"purchase-token\"}", api.Requests[1].Body);
            Assert.IsNotEmpty(api.Requests[1].Mac);
            Assert.AreEqual(1, api.Requests[1].Headers.Count);
            Assert.AreEqual("X-HD-Play-Integrity", api.Requests[1].Headers[0].Name);
            Assert.AreEqual("play-token", api.Requests[1].Headers[0].Value);
            Assert.IsEmpty(results);

            api.Requests[1].Done(new HttpReply(200, GrantedJson));
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(ShopPurchaseOutcome.Granted, results[0].Outcome);
            Assert.AreEqual(ShopBuyKeys.Bought, results[0].MessageKey);
            Assert.AreEqual(650, results[0].Reply.crystals);
            Assert.AreEqual("GPA.1234", results[0].Reply.transaction_id);
            Assert.AreEqual("purchase-token", billing.Finished[0].Token);
            Assert.IsTrue(billing.FinishedByServer[0]);
            Assert.IsFalse(purchase.Busy);
            Assert.IsFalse(purchase.CanRetry);
            Assert.IsNull(purchase.UnsettledOffer);
        }

        [Test]
        public void AnAppleReceiptNamesItsPlatform()
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Purchased, new StoreReceipt(StorePlatform.Apple, "signed-transaction")));
            Assert.AreEqual("{\"offer\":\"offer.crystals_tier2\",\"platform\":\"apple\",\"token\":\"signed-transaction\"}", api.Requests[1].Body);
        }

        [Test]
        public void WithoutAProofTheReceiptIsSentWithoutIntegrityHeaders()
        {
            proofs.Next = null;
            PayInStore();
            Assert.AreEqual("/shop/complete", api.Requests[1].PathAndQuery);
            Assert.IsNull(api.Requests[1].Headers);
        }

        [Test]
        public void WithoutStoreBillingNothingIsSent()
        {
            var unavailable = new ShopPurchase(null, new NoStoreBilling(), proofs.Request);
            Assert.IsTrue(unavailable.Buy(Offer, results.Add));
            Assert.IsEmpty(api.Requests);
            Assert.AreEqual(ShopPurchaseOutcome.StoreUnavailable, results[0].Outcome);
            Assert.AreEqual(ShopPurchaseKeys.StoreUnavailable, results[0].MessageKey);
            Assert.IsFalse(unavailable.Busy);
        }

        [TestCase(null)]
        [TestCase("")]
        public void ABuyWithoutAnOfferIsNotSent(string offer)
        {
            Assert.IsFalse(purchase.Buy(offer, results.Add));
            Assert.IsEmpty(api.Requests);
        }

        [Test]
        public void OnlyOnePurchaseIsInFlight()
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            Assert.IsFalse(purchase.Buy("offer.pass_premium", results.Add));
            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            Assert.IsFalse(purchase.Buy("offer.pass_premium", results.Add));
            Assert.IsFalse(purchase.Retry(results.Add));
            Assert.AreEqual(1, api.Requests.Count);
        }

        [TestCase(409, "offer.in_game_only", ShopPurchaseOutcome.InGameOnly, ShopBuyKeys.InGameOnly)]
        [TestCase(503, "purchases.disabled", ShopPurchaseOutcome.PurchasesDisabled, ShopBuyKeys.PurchasesDisabled)]
        [TestCase(403, "purchase.blocked", ShopPurchaseOutcome.PurchaseBlocked, ShopBuyKeys.PurchaseBlocked)]
        [TestCase(403, "offer.not_eligible", ShopPurchaseOutcome.NotEligible, ShopBuyKeys.NotEligible)]
        [TestCase(409, "offer.owned", ShopPurchaseOutcome.Owned, ShopBuyKeys.Owned)]
        [TestCase(409, "offer.limit_reached", ShopPurchaseOutcome.LimitReached, ShopBuyKeys.LimitReached)]
        [TestCase(409, "offer.unavailable", ShopPurchaseOutcome.Unavailable, ShopBuyKeys.Unavailable)]
        [TestCase(409, "offer.unsupported", ShopPurchaseOutcome.Unsupported, ShopBuyKeys.Unsupported)]
        [TestCase(404, "offer.unknown", ShopPurchaseOutcome.Unknown, ShopBuyKeys.Unknown)]
        [TestCase(409, "item.source_not_listed", ShopPurchaseOutcome.Failed, ShopBuyKeys.Failed)]
        [TestCase(400, "malformed", ShopPurchaseOutcome.Failed, ShopBuyKeys.Failed)]
        [TestCase(503, "shop.unavailable", ShopPurchaseOutcome.Failed, ShopBuyKeys.Failed)]
        [TestCase(500, "internal", ShopPurchaseOutcome.Failed, ShopBuyKeys.Failed)]
        [TestCase(429, "rate_limited", ShopPurchaseOutcome.RateLimited, ShopBuyKeys.RateLimited)]
        public void ABeginRefusalEndsThePurchaseBeforeTheStore(int status, string error, ShopPurchaseOutcome outcome, string key)
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(status, "{\"error\":\"" + error + "\"}"));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(ShopPurchaseStep.Begin, results[0].Step);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsEmpty(billing.Products);
            Assert.IsFalse(purchase.Busy);
            Assert.IsFalse(purchase.CanRetry);
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
        }

        [Test]
        public void NoResponseToTheBeginEndsThePurchaseBeforeTheStore()
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(HttpReply.NoResponse, null));
            Assert.AreEqual(ShopPurchaseOutcome.Offline, results[0].Outcome);
            Assert.AreEqual(ShopBuyKeys.Offline, results[0].MessageKey);
            Assert.IsEmpty(billing.Products);
            Assert.IsFalse(purchase.CanRetry);
        }

        [TestCase("{}")]
        [TestCase("{\"offer\":\"offer.crystals_tier2\",\"product_id\":\"crystals_tier2\"}")]
        [TestCase("{\"offer\":\"offer.crystals_tier2\",\"account_token\":\"account-token-test\"}")]
        [TestCase("not json")]
        public void AnUnreadableCheckoutIsAFailure(string body)
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(200, body));
            Assert.AreEqual(ShopPurchaseOutcome.Failed, results[0].Outcome);
            Assert.AreEqual(ShopBuyKeys.Failed, results[0].MessageKey);
            Assert.IsEmpty(billing.Products);
        }

        [TestCase(StoreBillingOutcome.Cancelled, ShopPurchaseOutcome.Cancelled, null)]
        [TestCase(StoreBillingOutcome.Deferred, ShopPurchaseOutcome.Pending, ShopPurchaseKeys.Pending)]
        [TestCase(StoreBillingOutcome.Unavailable, ShopPurchaseOutcome.StoreUnavailable, ShopPurchaseKeys.StoreUnavailable)]
        [TestCase(StoreBillingOutcome.Failed, ShopPurchaseOutcome.Failed, ShopBuyKeys.Failed)]
        public void AStorePurchaseWithoutAReceiptSendsNoCompletion(StoreBillingOutcome billed, ShopPurchaseOutcome outcome, string key)
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            billing.Answer(new StoreBillingResult(billed));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(ShopPurchaseStep.Billing, results[0].Step);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.IsEmpty(proofs.Purposes);
            Assert.IsFalse(purchase.Busy);
            Assert.IsFalse(purchase.CanRetry);
        }

        [Test]
        public void APurchasedResultWithoutATokenIsAFailure()
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Purchased, new StoreReceipt(StorePlatform.Google, "")));
            Assert.AreEqual(ShopPurchaseOutcome.Failed, results[0].Outcome);
            Assert.AreEqual(1, api.Requests.Count);
        }

        [TestCase(403, "integrity_failed", ShopPurchaseOutcome.IntegrityFailed)]
        [TestCase(400, "bad_integrity_proof", ShopPurchaseOutcome.IntegrityFailed)]
        [TestCase(400, "receipt.malformed", ShopPurchaseOutcome.ReceiptRefused)]
        [TestCase(400, "receipt.signature", ShopPurchaseOutcome.ReceiptRefused)]
        [TestCase(400, "receipt.unknown", ShopPurchaseOutcome.ReceiptRefused)]
        [TestCase(403, "receipt.account", ShopPurchaseOutcome.ReceiptRefused)]
        [TestCase(403, "receipt.reused", ShopPurchaseOutcome.ReceiptRefused)]
        [TestCase(409, "receipt.product", ShopPurchaseOutcome.ReceiptRefused)]
        [TestCase(409, "receipt.cancelled", ShopPurchaseOutcome.ReceiptRefused)]
        [TestCase(409, "receipt.environment", ShopPurchaseOutcome.ReceiptRefused)]
        [TestCase(409, "receipt.app", ShopPurchaseOutcome.ReceiptRefused)]
        [TestCase(403, "purchase.blocked", ShopPurchaseOutcome.PurchaseBlocked)]
        [TestCase(404, "offer.unknown", ShopPurchaseOutcome.Unknown)]
        [TestCase(409, "purchase.platform_unavailable", ShopPurchaseOutcome.StoreUnavailable)]
        [TestCase(409, "offer.in_game_only", ShopPurchaseOutcome.InGameOnly)]
        [TestCase(409, "offer.unsupported", ShopPurchaseOutcome.Unsupported)]
        [TestCase(409, "pass.no_season", ShopPurchaseOutcome.NoPassSeason)]
        [TestCase(409, "pass.premium_owned", ShopPurchaseOutcome.Owned)]
        [TestCase(409, "item.source_not_listed", ShopPurchaseOutcome.Failed)]
        [TestCase(400, "malformed", ShopPurchaseOutcome.Failed)]
        public void ACompletionRefusalSettlesTheReceiptAsNotGranted(int status, string error, ShopPurchaseOutcome outcome)
        {
            PayInStore();
            api.Requests[1].Done(new HttpReply(status, "{\"error\":\"" + error + "\"}"));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(ShopPurchaseStep.Complete, results[0].Step);
            Assert.AreEqual(ShopPurchaseKeys.NotGranted, results[0].MessageKey);
            Assert.IsEmpty(billing.Finished);
            Assert.IsFalse(purchase.CanRetry);
            Assert.IsNull(purchase.UnsettledOffer);
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
        }

        [TestCase(503, "{\"error\":\"integrity_unavailable\"}", ShopPurchaseOutcome.Unfinished, ShopPurchaseKeys.Unfinished)]
        [TestCase(502, "{\"error\":\"receipt.store_unavailable\"}", ShopPurchaseOutcome.Unfinished, ShopPurchaseKeys.Unfinished)]
        [TestCase(503, "{\"error\":\"shop.unavailable\"}", ShopPurchaseOutcome.Unfinished, ShopPurchaseKeys.Unfinished)]
        [TestCase(500, "{\"error\":\"internal\"}", ShopPurchaseOutcome.Unfinished, ShopPurchaseKeys.Unfinished)]
        [TestCase(200, "not json", ShopPurchaseOutcome.Unfinished, ShopPurchaseKeys.Unfinished)]
        [TestCase(200, "{}", ShopPurchaseOutcome.Unfinished, ShopPurchaseKeys.Unfinished)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", ShopPurchaseOutcome.RateLimited, ShopBuyKeys.RateLimited)]
        [TestCase(HttpReply.NoResponse, null, ShopPurchaseOutcome.Offline, ShopBuyKeys.Offline)]
        public void AnUnsettledCompletionKeepsTheReceiptForARetryWithAFreshProof(int status, string body, ShopPurchaseOutcome outcome, string key)
        {
            PayInStore();
            api.Requests[1].Done(new HttpReply(status, body));
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsTrue(results[0].KeepsReceipt);
            Assert.IsEmpty(billing.Finished);
            Assert.IsFalse(purchase.Busy);
            Assert.IsTrue(purchase.CanRetry);
            Assert.AreEqual(Offer, purchase.UnsettledOffer);
            Assert.IsFalse(purchase.Buy("offer.pass_premium", results.Add));

            proofs.Next = new IntegrityProof { PlayIntegrityToken = "play-token-2" };
            Assert.IsTrue(purchase.Retry(results.Add));
            Assert.IsFalse(purchase.Retry(results.Add));
            Assert.AreEqual(2, proofs.Purposes.Count);
            Assert.AreEqual(IntegrityPurpose.Purchase, proofs.Purposes[1]);
            Assert.AreEqual(3, api.Requests.Count);
            Assert.AreEqual("/shop/complete", api.Requests[2].PathAndQuery);
            Assert.AreEqual(api.Requests[1].Body, api.Requests[2].Body);
            Assert.AreEqual("play-token-2", api.Requests[2].Headers[0].Value);
            Assert.AreEqual(1, billing.Products.Count);

            api.Requests[2].Done(new HttpReply(200, GrantedJson));
            Assert.AreEqual(ShopPurchaseOutcome.Granted, results[1].Outcome);
            Assert.AreEqual(1, billing.Finished.Count);
            Assert.IsFalse(purchase.CanRetry);
        }

        [Test]
        public void APendingStorePurchaseIsRecordedAndNotFinished()
        {
            PayInStore();
            api.Requests[1].Done(new HttpReply(200, PendingJson));
            Assert.AreEqual(ShopPurchaseOutcome.Pending, results[0].Outcome);
            Assert.AreEqual(ShopPurchaseKeys.Pending, results[0].MessageKey);
            Assert.IsEmpty(billing.Finished);
            Assert.IsFalse(purchase.CanRetry);
        }

        [Test]
        public void ARepeatedTransactionTheServerDidNotGrantNowIsFinishedButNotGranted()
        {
            PayInStore();
            api.Requests[1].Done(new HttpReply(200, "{\"platform\":\"google\",\"transaction_id\":\"GPA.1234\",\"offer\":\"offer.crystals_tier2\",\"state\":\"refunded\",\"repeated\":true,\"crystals\":0,\"finished\":false}"));
            Assert.AreEqual(ShopPurchaseOutcome.Failed, results[0].Outcome);
            Assert.AreEqual(ShopPurchaseKeys.NotGranted, results[0].MessageKey);
            Assert.AreEqual(1, billing.Finished.Count);
            Assert.IsFalse(billing.FinishedByServer[0]);
            Assert.IsFalse(purchase.CanRetry);
        }

        void AssertTracked(int index, string name)
        {
            Assert.AreEqual(name, analytics.Sent[index].Name);
            Assert.AreEqual("crystals_tier2", analytics.Sent[index].Label);
            Assert.AreEqual(0, analytics.Sent[index].Value);
        }

        [Test]
        public void PurchaseStartIsTrackedWithTheStoreProductIdWhenTheStorePurchaseOpens()
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            Assert.AreEqual(0, events.Count);

            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            Assert.AreEqual(1, billing.Products.Count);
            Assert.AreEqual(1, events.Count);

            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Purchased, new StoreReceipt(StorePlatform.Google, "purchase-token")));
            api.Requests[1].Done(new HttpReply(200, GrantedJson));
            events.Flush();

            Assert.AreEqual(ShopPurchaseOutcome.Granted, results[0].Outcome);
            Assert.AreEqual(1, analytics.Sent.Count);
            AssertTracked(0, AnalyticsEvents.PurchaseStart);
        }

        [Test]
        public void AStoreCancelTracksPurchaseCancelWithTheStoreProductId()
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Cancelled, null));
            events.Flush();

            Assert.AreEqual(ShopPurchaseOutcome.Cancelled, results[0].Outcome);
            Assert.AreEqual(2, analytics.Sent.Count);
            AssertTracked(0, AnalyticsEvents.PurchaseStart);
            AssertTracked(1, AnalyticsEvents.PurchaseCancel);
        }

        [TestCase(StoreBillingOutcome.Deferred)]
        [TestCase(StoreBillingOutcome.Unavailable)]
        [TestCase(StoreBillingOutcome.Failed)]
        public void AStoreAnswerOtherThanACancelTracksNoCancel(StoreBillingOutcome billed)
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            billing.Answer(new StoreBillingResult(billed, null));
            events.Flush();

            Assert.AreEqual(1, analytics.Sent.Count);
            AssertTracked(0, AnalyticsEvents.PurchaseStart);
        }

        [TestCase(409, "{\"error\":\"offer.owned\"}")]
        [TestCase(503, "{\"error\":\"purchases.disabled\"}")]
        [TestCase(HttpReply.NoResponse, null)]
        [TestCase(200, "{}")]
        public void APurchaseThatNeverReachesTheStoreTracksNothing(int status, string body)
        {
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsEmpty(billing.Products);
            Assert.AreEqual(0, events.Count);
        }

        [Test]
        public void WithoutStoreBillingNothingIsTracked()
        {
            var unavailable = new ShopPurchase(null, new NoStoreBilling(), proofs.Request, events);
            Assert.IsTrue(unavailable.Buy(Offer, results.Add));

            Assert.AreEqual(ShopPurchaseOutcome.StoreUnavailable, results[0].Outcome);
            Assert.AreEqual(0, events.Count);
        }

        [Test]
        public void ARetryOfAnUnsettledReceiptTracksNoNewStart()
        {
            PayInStore();
            api.Requests[1].Done(new HttpReply(HttpReply.NoResponse, null));
            Assert.IsTrue(purchase.Retry(results.Add));
            api.Requests[2].Done(new HttpReply(200, GrantedJson));
            events.Flush();

            Assert.AreEqual(ShopPurchaseOutcome.Granted, results[1].Outcome);
            Assert.AreEqual(1, analytics.Sent.Count);
            AssertTracked(0, AnalyticsEvents.PurchaseStart);
        }

        [Test]
        public void WithoutAnalyticsConsentThePurchaseRunsAndNothingIsTracked()
        {
            events.Withdraw();
            Assert.IsTrue(purchase.Buy(Offer, results.Add));
            api.Requests[0].Done(new HttpReply(200, CheckoutJson));
            billing.Answer(new StoreBillingResult(StoreBillingOutcome.Cancelled, null));

            Assert.AreEqual(1, billing.Products.Count);
            Assert.AreEqual(ShopPurchaseOutcome.Cancelled, results[0].Outcome);
            Assert.AreEqual(0, events.Count);
        }

        [Test]
        public void EveryOutcomeButCancelAndSignInHasAMessageKeyWithShippedTexts()
        {
            var strings = BuiltinStrings.Load();
            foreach (ShopPurchaseOutcome outcome in Enum.GetValues(typeof(ShopPurchaseOutcome)))
            {
                foreach (ShopPurchaseStep step in Enum.GetValues(typeof(ShopPurchaseStep)))
                {
                    var key = ShopPurchaseKeys.Of(outcome, step);
                    if (outcome is ShopPurchaseOutcome.Cancelled or ShopPurchaseOutcome.SignInRequired)
                    {
                        Assert.IsNull(key);
                        continue;
                    }
                    StringAssert.IsMatch("^[a-z0-9_]+(\\.[a-z0-9_]+)+$", key, outcome.ToString());
                    Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                    Assert.IsNotNull(strings.Value(key, "az"), key);
                }
            }
        }
    }
}
