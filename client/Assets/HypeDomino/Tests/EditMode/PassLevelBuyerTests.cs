using System;
using System.Collections.Generic;
using HypeDomino.Game.Pass;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Shop;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PassLevelBuyerTests
    {
        const string StandingJson = @"{""season"":""season_1"",""ends_at"":""2027-02-01T00:00:00Z"",""levels"":100,""xp"":1700,""level"":97,""level_xp"":1500,""next_level_xp"":2000,""premium"":false}";
        const string PricedJson = @"{""season"":""season_1"",""ends_at"":""2027-02-01T00:00:00Z"",""levels"":100,""xp"":1700,""level"":97,""level_xp"":1500,""next_level_xp"":2000,""premium"":false,""level_coin_price"":200,""level_crystal_price"":40}";
        const string LastLevelJson = @"{""season"":""season_1"",""ends_at"":""2027-02-01T00:00:00Z"",""levels"":100,""xp"":2000,""level"":100,""level_xp"":2000,""next_level_xp"":0,""premium"":false}";
        const string BoughtJson = @"{""season"":""season_1"",""count"":2,""currency"":""crystal"",""price"":80,""level"":99,""xp"":1900,""balance"":20,""repeated"":false}";
        const string RepeatedJson = @"{""season"":""season_1"",""count"":2,""currency"":""coin"",""price"":400,""level"":99,""xp"":1900,""balance"":1100,""repeated"":true}";

        static readonly string[] ShownKeys =
        {
            PassLevelBuyKeys.Insufficient,
            PassLevelBuyKeys.TooManyLevels,
            PassLevelBuyKeys.SeasonOver,
            PassLevelBuyKeys.Offline,
            PassLevelBuyKeys.RateLimited,
            PassLoadKeys.Offline,
            PassLoadKeys.RateLimited,
            PassStanding.NoSeason,
        };

        sealed class Sent
        {
            public string Method;
            public string PathAndQuery;
            public string Body;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, PathAndQuery = pathAndQuery, Body = body, Done = done });
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
        PassLevelBuyer buyer;
        PassStanding standing;
        List<PassLevelBuyResult> results;

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
            buyer = new PassLevelBuyer(new ApiClient(api, session));
            standing = Standing(StandingJson);
            results = new List<PassLevelBuyResult>();
        }

        static PassStanding Standing(string json)
        {
            Assert.IsTrue(PassStanding.TryRead(PassReply.FromJson(json), out var read));
            return read;
        }

        PassLevelsRequest Body(int index) => JsonUtility.FromJson<PassLevelsRequest>(api.Requests[index].Body);

        [TestCase(1, Currency.Coin, 200L)]
        [TestCase(3, Currency.Coin, 600L)]
        [TestCase(2, Currency.Crystal, 80L)]
        public void TheShownPriceIsTheServerLevelPriceTimesTheCount(int count, Currency currency, long expected)
        {
            Assert.IsTrue(PassLevelBuyer.TryPriceOf(Standing(PricedJson), count, currency, out var price));
            Assert.AreEqual(expected, price);
        }

        [TestCase(0)]
        [TestCase(4)]
        public void ACountTheSeasonCannotHoldHasNoPrice(int count)
        {
            Assert.IsFalse(PassLevelBuyer.TryPriceOf(Standing(PricedJson), count, Currency.Coin, out var price));
            Assert.AreEqual(0, price);
            Assert.IsFalse(PassLevelBuyer.TryPriceOf(null, 1, Currency.Coin, out _));
            Assert.IsFalse(PassLevelBuyer.TryPriceOf(Standing(LastLevelJson), 1, Currency.Crystal, out _));
        }

        [Test]
        public void WithoutAServerPriceNoPriceIsShown()
        {
            Assert.IsFalse(PassLevelBuyer.TryPriceOf(standing, 1, Currency.Coin, out var price));
            Assert.AreEqual(0, price);
            Assert.IsFalse(PassLevelBuyer.TryPriceOf(standing, 1, Currency.Crystal, out _));
        }

        [Test]
        public void TheIntentCarriesOnlyTheCountTheCurrencyAndARequestId()
        {
            Assert.IsTrue(buyer.Buy(standing, 2, Currency.Crystal, results.Add));
            Assert.IsTrue(buyer.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual("/pass/levels", api.Requests[0].PathAndQuery);
            var body = Body(0);
            StringAssert.IsMatch("^[0-9a-f]{32}$", body.request_id);
            Assert.AreEqual("{\"count\":2,\"currency\":\"crystal\",\"request_id\":\"" + body.request_id + "\"}", api.Requests[0].Body);
            Assert.IsEmpty(results);

            buyer.Buy(standing, 1, Currency.Coin, results.Add);
            Assert.AreEqual(1, api.Requests.Count);
            api.Requests[0].Done(new HttpReply(200, BoughtJson));
            Assert.IsFalse(buyer.Busy);
            Assert.IsTrue(buyer.Buy(standing, 1, Currency.Coin, results.Add));
            Assert.AreEqual("coin", Body(1).currency);
        }

        [Test]
        public void APurchaseKeepsWhatTheServerAnswered()
        {
            buyer.Buy(standing, 2, Currency.Crystal, results.Add);
            api.Requests[0].Done(new HttpReply(200, BoughtJson));

            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(PassLevelBuyOutcome.Bought, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.AreEqual(99, results[0].Purchase.level);
            Assert.AreEqual(1900, results[0].Purchase.xp);
            Assert.AreEqual(80, results[0].Purchase.price);
            Assert.IsFalse(results[0].Purchase.repeated);
            Assert.AreEqual(Currency.Crystal, results[0].Balance.Currency);
            Assert.AreEqual(20, results[0].Balance.Amount);
        }

        [Test]
        public void NothingIsSentForACountTheSeasonCannotHold()
        {
            Assert.AreEqual(3, PassLevelBuyer.MostLevelsFor(standing));
            Assert.AreEqual(0, PassLevelBuyer.MostLevelsFor(null));
            Assert.AreEqual(0, PassLevelBuyer.MostLevelsFor(Standing(LastLevelJson)));
            Assert.IsFalse(buyer.Buy(standing, 0, Currency.Coin, results.Add));
            Assert.IsFalse(buyer.Buy(standing, -1, Currency.Coin, results.Add));
            Assert.IsFalse(buyer.Buy(standing, 4, Currency.Coin, results.Add));
            Assert.IsFalse(buyer.Buy(null, 1, Currency.Coin, results.Add));
            Assert.IsFalse(buyer.Buy(Standing(LastLevelJson), 1, Currency.Coin, results.Add));

            Assert.IsFalse(buyer.Busy);
            Assert.IsEmpty(api.Requests);
            Assert.IsEmpty(results);
        }

        [TestCase(HttpReply.NoResponse, null, PassLevelBuyOutcome.Offline, "pass.levels.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", PassLevelBuyOutcome.RateLimited, "pass.levels.rate_limited")]
        [TestCase(401, "{\"error\":\"access_rejected\"}", PassLevelBuyOutcome.SignInRequired, null)]
        [TestCase(402, "{\"error\":\"wallet.insufficient\"}", PassLevelBuyOutcome.Insufficient, "pass.levels.insufficient")]
        [TestCase(409, "{\"error\":\"pass.level_count\"}", PassLevelBuyOutcome.TooManyLevels, "pass.levels.too_many")]
        [TestCase(409, "{\"error\":\"pass.no_season\"}", PassLevelBuyOutcome.SeasonOver, "pass.levels.season_over")]
        [TestCase(400, "{\"error\":\"request.invalid\"}", PassLevelBuyOutcome.Failed, "notice.generic")]
        [TestCase(400, "{\"error\":\"malformed\"}", PassLevelBuyOutcome.Failed, "notice.generic")]
        [TestCase(500, "{\"error\":\"internal\"}", PassLevelBuyOutcome.Failed, "notice.generic")]
        [TestCase(200, "", PassLevelBuyOutcome.Failed, "notice.generic")]
        [TestCase(200, "not json", PassLevelBuyOutcome.Failed, "notice.generic")]
        [TestCase(200, "{}", PassLevelBuyOutcome.Failed, "notice.generic")]
        [TestCase(200, @"{""season"":""season_1"",""count"":2,""currency"":""gem"",""price"":80,""level"":99,""xp"":1900,""balance"":20}", PassLevelBuyOutcome.Failed, "notice.generic")]
        [TestCase(200, @"{""season"":""season_1"",""count"":2,""currency"":""coin"",""price"":-400,""level"":99,""xp"":1900,""balance"":20}", PassLevelBuyOutcome.Failed, "notice.generic")]
        [TestCase(200, @"{""season"":""season_1"",""count"":2,""currency"":""coin"",""price"":400,""level"":0,""xp"":1900,""balance"":20}", PassLevelBuyOutcome.Failed, "notice.generic")]
        public void ARefusedPurchaseEndsWithoutAPurchase(int status, string body, PassLevelBuyOutcome outcome, string key)
        {
            buyer.Buy(standing, 2, Currency.Coin, results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(buyer.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.IsNull(results[0].Purchase);
        }

        [TestCase(HttpReply.NoResponse, null)]
        [TestCase(500, "{\"error\":\"internal\"}")]
        [TestCase(429, "{\"error\":\"rate_limited\"}")]
        [TestCase(200, "not json")]
        public void ARetryAfterAnUnsettledAnswerRepeatsTheSameRequestId(int status, string body)
        {
            buyer.Buy(standing, 2, Currency.Coin, results.Add);
            var first = Body(0).request_id;
            api.Requests[0].Done(new HttpReply(status, body));
            Assert.IsTrue(buyer.IsUnsettled(2, Currency.Coin));

            buyer.Buy(standing, 2, Currency.Coin, results.Add);
            Assert.AreEqual(first, Body(1).request_id);
            api.Requests[1].Done(new HttpReply(200, RepeatedJson));
            Assert.AreEqual(PassLevelBuyOutcome.Bought, results[1].Outcome);
            Assert.IsTrue(results[1].Purchase.repeated);
            Assert.IsFalse(buyer.IsUnsettled(2, Currency.Coin));
        }

        [TestCase(200, BoughtJson)]
        [TestCase(402, "{\"error\":\"wallet.insufficient\"}")]
        [TestCase(409, "{\"error\":\"pass.level_count\"}")]
        public void ASettledAnswerStartsANewIntent(int status, string body)
        {
            buyer.Buy(standing, 2, Currency.Coin, results.Add);
            var first = Body(0).request_id;
            api.Requests[0].Done(new HttpReply(status, body));
            Assert.IsFalse(buyer.IsUnsettled(2, Currency.Coin));

            buyer.Buy(standing, 2, Currency.Coin, results.Add);
            Assert.AreNotEqual(first, Body(1).request_id);
        }

        [Test]
        public void AnotherCountOrCurrencyNeverReusesTheUnsettledRequestId()
        {
            buyer.Buy(standing, 2, Currency.Coin, results.Add);
            var first = Body(0).request_id;
            api.Requests[0].Done(new HttpReply(HttpReply.NoResponse, null));

            buyer.Buy(standing, 2, Currency.Crystal, results.Add);
            var second = Body(1).request_id;
            Assert.AreNotEqual(first, second);
            api.Requests[1].Done(new HttpReply(HttpReply.NoResponse, null));

            buyer.Buy(standing, 1, Currency.Crystal, results.Add);
            Assert.AreNotEqual(second, Body(2).request_id);
        }

        [Test]
        public void TheShippedBundleHasEveryShownKeyInAzerbaijaniAndEnglish()
        {
            Assert.AreEqual(ShownKeys.Length, new HashSet<string>(ShownKeys).Count);
            var strings = BuiltinStrings.Load();
            foreach (var key in ShownKeys)
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
