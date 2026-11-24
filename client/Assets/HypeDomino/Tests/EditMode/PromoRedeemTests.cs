using System;
using System.Collections.Generic;
using HypeDomino.Game.Promo;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PromoRedeemTests
    {
        const string RedemptionJson = "{\"promo\":\"promo-1\",\"campaign\":\"campaign.backers\",\"grant\":{\"id\":7,\"account_id\":\"acc-1\",\"bundle_id\":\"bundle.backers\",\"revision\":1,\"source\":{\"type\":\"promo\",\"ref\":\"campaign.backers\"},\"reason\":\"promo\",\"key\":\"promo:promo-1:acc-1\",\"config_version\":3,\"coins\":500,\"cosmetics\":[\"avatar.comet_fox\",\"frame.paper_kite\"],\"duplicates\":{\"tile.moon_owl\":40},\"medals\":[],\"medals_skipped\":[],\"pass_xp\":300,\"pass_xp_dropped\":0,\"at\":\"2026-11-14T10:00:00Z\"}}";

        [TestCase("ABCD2345", "ABCD2345")]
        [TestCase("abcd2345", "ABCD2345")]
        [TestCase("ab-cd 23_45", "ABCD2345")]
        [TestCase("  wxyz  ", "WXYZ")]
        [TestCase("ABCDEFGHJKMNPQRSTUVWXYZ234567892", "ABCDEFGHJKMNPQRSTUVWXYZ234567892")]
        public void ATypedCodeIsNormalisedAsTheServerDoes(string typed, string expected)
        {
            Assert.IsTrue(PromoCode.IsWellFormed(typed));
            Assert.IsTrue(PromoCode.TryNormalize(typed, out var code));
            Assert.AreEqual(expected, code);
        }

        [TestCase(null)]
        [TestCase("")]
        [TestCase("- _ ")]
        [TestCase("ABC")]
        [TestCase("ABCDEFGHJKMNPQRSTUVWXYZ2345678923")]
        [TestCase("ABCD0234")]
        [TestCase("ABCDO234")]
        [TestCase("ABCD1234")]
        [TestCase("ABCDI234")]
        [TestCase("ABCDL234")]
        [TestCase("abcdi234")]
        [TestCase("ABCDı234")]
        [TestCase("ABCD.234")]
        [TestCase("ABCD\n2345")]
        [TestCase("ABCDÉ234")]
        public void ACodeTheServerWouldRefuseForItsFormatIsRefusedBeforeSending(string typed)
        {
            Assert.IsFalse(PromoCode.IsWellFormed(typed));
            Assert.IsFalse(PromoCode.TryNormalize(typed, out var code));
            Assert.IsNull(code);
        }

        static string checkedCode;

        static void CheckCode() => PromoCode.IsWellFormed(checkedCode);

        [TestCase("ab-cd 23_45")]
        [TestCase("ABCD1234")]
        public void CheckingATypedCodeAllocatesNothing(string typed)
        {
            checkedCode = typed;
            CheckCode();

            Assert.That(CheckCode, Allocates.Nothing);
        }

        [Test]
        public void AGrantShowsWhatTheServerGranted()
        {
            var result = PromoRedeemResult.From(new ApiReply(ApiOutcome.Ok, 200, RedemptionJson));
            Assert.AreEqual(PromoRedeemOutcome.Granted, result.Outcome);
            Assert.AreEqual(PromoRedeemKeys.Granted, result.MessageKey);
            Assert.AreEqual(500, result.Coins);
            Assert.AreEqual(new[] { "avatar.comet_fox", "frame.paper_kite" }, result.Cosmetics);
            Assert.AreEqual(300, result.PassXp);
            Assert.AreEqual(0, result.PassXpDropped);
            Assert.AreEqual(1, result.Duplicates.Count);
            Assert.AreEqual(40, result.Duplicates["tile.moon_owl"]);
            Assert.IsFalse(result.PassPremium);
        }

        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500},\"pass_premium\":true}", true, "promo.redeem.granted_pass_premium")]
        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500},\"pass_premium\":false}", false, "promo.redeem.granted")]
        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500}}", false, "promo.redeem.granted")]
        public void AGrantSaysWhetherThePremiumPassWasUnlocked(string body, bool premium, string key)
        {
            var result = PromoRedeemResult.From(new ApiReply(ApiOutcome.Ok, 200, body));
            Assert.AreEqual(PromoRedeemOutcome.Granted, result.Outcome);
            Assert.AreEqual(500, result.Coins);
            Assert.AreEqual(premium, result.PassPremium);
            Assert.AreEqual(key, result.MessageKey);
        }

        [Test]
        public void ARefusalNeverUnlocksThePremiumPass()
        {
            var result = PromoRedeemResult.From(new ApiReply(ApiOutcome.Rejected, 409, "{\"error\":\"promo.pass_premium_owned\",\"pass_premium\":true}", "promo.pass_premium_owned"));
            Assert.AreEqual(PromoRedeemOutcome.PassPremiumOwned, result.Outcome);
            Assert.IsFalse(result.PassPremium);
        }

        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500,\"cosmetics\":[]}}")]
        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500,\"cosmetics\":[],\"duplicates\":null}}")]
        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500,\"cosmetics\":[],\"duplicates\":{}}}")]
        public void AGrantWithoutDuplicatesHasNone(string body)
        {
            var result = PromoRedeemResult.From(new ApiReply(ApiOutcome.Ok, 200, body));
            Assert.AreEqual(PromoRedeemOutcome.Granted, result.Outcome);
            Assert.AreEqual(500, result.Coins);
            Assert.IsEmpty(result.Duplicates);
        }

        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500,\"duplicates\":[\"tile.moon_owl\"]}}")]
        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500,\"duplicates\":{\"tile.moon_owl\":\"40\"}}}")]
        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500,\"duplicates\":{\"tile.moon_owl\":-40}}}")]
        [TestCase("{\"promo\":\"promo-1\",\"grant\":{\"coins\":500,\"duplicates\":{\"tile.moon_owl\":40.5}}}")]
        public void AGrantWithUnreadableDuplicatesIsAFailure(string body)
        {
            var result = PromoRedeemResult.From(new ApiReply(ApiOutcome.Ok, 200, body));
            Assert.AreEqual(PromoRedeemOutcome.Failed, result.Outcome);
            Assert.AreEqual(0, result.Coins);
            Assert.IsEmpty(result.Duplicates);
        }

        [Test]
        public void AGrantWithoutABodyIsAFailureAndShowsNothingGranted()
        {
            var result = PromoRedeemResult.From(new ApiReply(ApiOutcome.Ok, 200, "not json"));
            Assert.AreEqual(PromoRedeemOutcome.Failed, result.Outcome);
            Assert.AreEqual(0, result.Coins);
            Assert.IsEmpty(result.Cosmetics);
            Assert.IsEmpty(result.Duplicates);
        }

        [TestCase(400, "promo.code_invalid", PromoRedeemOutcome.CodeInvalid, "promo.redeem.code_invalid")]
        [TestCase(404, "promo.unknown", PromoRedeemOutcome.Unknown, "promo.redeem.unknown")]
        [TestCase(409, "promo.not_active", PromoRedeemOutcome.NotActive, "promo.redeem.not_active")]
        [TestCase(409, "promo.disabled", PromoRedeemOutcome.Disabled, "promo.redeem.disabled")]
        [TestCase(409, "promo.not_started", PromoRedeemOutcome.NotStarted, "promo.redeem.not_started")]
        [TestCase(409, "promo.expired", PromoRedeemOutcome.Expired, "promo.redeem.expired")]
        [TestCase(409, "promo.exhausted", PromoRedeemOutcome.Exhausted, "promo.redeem.exhausted")]
        [TestCase(409, "promo.already_redeemed", PromoRedeemOutcome.AlreadyRedeemed, "promo.redeem.already_redeemed")]
        [TestCase(403, "promo.not_eligible", PromoRedeemOutcome.NotEligible, "promo.redeem.not_eligible")]
        [TestCase(409, "promo.no_pass_season", PromoRedeemOutcome.NoPassSeason, "promo.redeem.no_pass_season")]
        [TestCase(409, "promo.pass_premium_owned", PromoRedeemOutcome.PassPremiumOwned, "promo.redeem.pass_premium_owned")]
        [TestCase(409, "item.source_not_listed", PromoRedeemOutcome.Failed, "notice.generic")]
        [TestCase(400, "malformed", PromoRedeemOutcome.Failed, "notice.generic")]
        [TestCase(401, "bad_mac", PromoRedeemOutcome.Failed, "notice.generic")]
        [TestCase(409, "promo.reason_from_a_newer_server", PromoRedeemOutcome.Failed, "notice.generic")]
        public void EveryServerRefusalMapsToItsOwnKey(int status, string error, PromoRedeemOutcome outcome, string key)
        {
            var result = PromoRedeemResult.From(new ApiReply(ApiOutcome.Rejected, status, "{\"error\":\"" + error + "\"}", error));
            Assert.AreEqual(outcome, result.Outcome);
            Assert.AreEqual(key, result.MessageKey);
            Assert.AreEqual(0, result.Coins);
            Assert.IsEmpty(result.Cosmetics);
        }

        [TestCase(ApiOutcome.RateLimited, 429, "promo.rate_limited", PromoRedeemOutcome.RateLimited, "promo.redeem.rate_limited")]
        [TestCase(ApiOutcome.RateLimited, 429, "rate_limited", PromoRedeemOutcome.RateLimited, "promo.redeem.rate_limited")]
        [TestCase(ApiOutcome.RateLimited, 429, "promo.locked", PromoRedeemOutcome.Locked, "promo.redeem.locked")]
        [TestCase(ApiOutcome.Unavailable, 503, "promo.unavailable", PromoRedeemOutcome.Failed, "notice.generic")]
        [TestCase(ApiOutcome.Unavailable, 500, "internal", PromoRedeemOutcome.Failed, "notice.generic")]
        [TestCase(ApiOutcome.Offline, HttpReply.NoResponse, null, PromoRedeemOutcome.Offline, "promo.redeem.offline")]
        [TestCase(ApiOutcome.SignInRequired, 401, "access_rejected", PromoRedeemOutcome.SignInRequired, null)]
        public void LimitsLocksAndTransportFailuresMapToTheirKeys(ApiOutcome api, int status, string error, PromoRedeemOutcome outcome, string key)
        {
            var result = PromoRedeemResult.From(new ApiReply(api, status, null, error));
            Assert.AreEqual(outcome, result.Outcome);
            Assert.AreEqual(key, result.MessageKey);
        }

        [Test]
        public void EveryOutcomeButSignInHasAMessageKey()
        {
            foreach (PromoRedeemOutcome outcome in Enum.GetValues(typeof(PromoRedeemOutcome)))
            {
                var key = PromoRedeemKeys.Of(outcome);
                if (outcome == PromoRedeemOutcome.SignInRequired)
                {
                    Assert.IsNull(key);
                    continue;
                }
                StringAssert.IsMatch("^[a-z0-9_]+(\\.[a-z0-9_]+)+$", key, outcome.ToString());
            }
        }

        [Test]
        public void EveryOutcomeWithAMessageHasItsOwnKey()
        {
            var keys = new HashSet<string>();
            foreach (PromoRedeemOutcome outcome in Enum.GetValues(typeof(PromoRedeemOutcome)))
            {
                var key = PromoRedeemKeys.Of(outcome);
                if (key == null)
                {
                    continue;
                }
                Assert.IsTrue(keys.Add(key), outcome.ToString());
            }
        }

        [Test]
        public void TheShippedBundleHasEveryMessageInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            foreach (PromoRedeemOutcome outcome in Enum.GetValues(typeof(PromoRedeemOutcome)))
            {
                var key = PromoRedeemKeys.Of(outcome);
                if (key == null)
                {
                    continue;
                }
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
            Assert.AreEqual("Kod qəbul olundu. Premium Hype Pass açıldı", strings.Value(PromoRedeemKeys.GrantedPassPremium, "az"));
            Assert.AreEqual("Code redeemed. Premium Hype Pass unlocked", strings.Value(PromoRedeemKeys.GrantedPassPremium, GameLanguage.English));
            Assert.AreEqual("Bu kod etibarsızdır", strings.Value(PromoRedeemKeys.CodeInvalid, "az"));
            Assert.AreEqual("Bu kodun vaxtı keçib", strings.Value(PromoRedeemKeys.Expired, "az"));
            Assert.AreEqual("Bu kod artıq istifadə olunub", strings.Value(PromoRedeemKeys.AlreadyRedeemed, "az"));
            Assert.AreEqual("This code has expired", strings.Value(PromoRedeemKeys.Expired, GameLanguage.English));
        }

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
        PromoRedeemer redeemer;
        List<PromoRedeemResult> results;

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
            redeemer = new PromoRedeemer(new ApiClient(api, session));
            results = new List<PromoRedeemResult>();
        }

        [Test]
        public void TheNormalisedCodeIsPostedAsTheOnlyField()
        {
            Assert.IsTrue(redeemer.Redeem("abcd-2345", results.Add));
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("POST", api.Requests[0].Method);
            Assert.AreEqual("/promo/redeem", api.Requests[0].PathAndQuery);
            Assert.AreEqual("{\"code\":\"ABCD2345\"}", api.Requests[0].Body);
            Assert.IsEmpty(results);

            api.Requests[0].Done(new HttpReply(200, RedemptionJson));
            Assert.AreEqual(PromoRedeemOutcome.Granted, results[0].Outcome);
            Assert.AreEqual(500, results[0].Coins);
        }

        [Test]
        public void AMalformedCodeIsAnsweredWithoutARequest()
        {
            Assert.IsTrue(redeemer.Redeem("ab1", results.Add));
            Assert.AreEqual(0, api.Requests.Count);
            Assert.AreEqual(PromoRedeemOutcome.CodeInvalid, results[0].Outcome);
            Assert.AreEqual(PromoRedeemKeys.CodeInvalid, results[0].MessageKey);
            Assert.IsFalse(redeemer.Busy);
        }

        [Test]
        public void ASecondSubmitWhileOneIsInFlightSendsNothing()
        {
            Assert.IsTrue(redeemer.Redeem("ABCD2345", results.Add));
            Assert.IsTrue(redeemer.Busy);
            Assert.IsFalse(redeemer.Redeem("ABCD2345", results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(409, "{\"error\":\"promo.already_redeemed\"}"));
            Assert.IsFalse(redeemer.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(PromoRedeemOutcome.AlreadyRedeemed, results[0].Outcome);

            Assert.IsTrue(redeemer.Redeem("WXYZ2345", results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        int changes;

        void CountChange() => changes++;

        [Test]
        public void TheEntryOffersSubmitOnlyForAWellFormedCode()
        {
            changes = 0;
            var entry = new PromoCodeEntry(redeemer);
            entry.Changed += CountChange;
            Assert.AreEqual(string.Empty, entry.Text);
            Assert.IsFalse(entry.CanSubmit);

            entry.Edit("abc");
            Assert.IsFalse(entry.WellFormed);
            Assert.IsFalse(entry.CanSubmit);

            entry.Edit("abcd");
            Assert.AreEqual("abcd", entry.Text);
            Assert.IsTrue(entry.WellFormed);
            Assert.IsTrue(entry.CanSubmit);

            entry.Edit("abcd");
            Assert.AreEqual(2, changes);

            entry.Edit("abc1");
            Assert.IsFalse(entry.CanSubmit);
            Assert.IsFalse(entry.Submit());
            Assert.AreEqual(0, api.Requests.Count);

            entry.Edit(null);
            Assert.AreEqual(string.Empty, entry.Text);
            Assert.IsFalse(entry.CanSubmit);
        }

        [Test]
        public void PastingReplacesTheTextAndAnEmptyClipboardKeepsIt()
        {
            var entry = new PromoCodeEntry(redeemer);
            entry.Edit("WX");

            entry.Paste(string.Empty);
            entry.Paste(null);
            Assert.AreEqual("WX", entry.Text);

            entry.Paste("abcd-2345");
            Assert.AreEqual("abcd-2345", entry.Text);
            Assert.IsTrue(entry.CanSubmit);

            entry.Paste("ABCD\n2345");
            Assert.AreEqual("ABCD\n2345", entry.Text);
            Assert.IsFalse(entry.CanSubmit);
        }

        [Test]
        public void SubmittingSendsOneRequestAndShowsTheReplyUntilTheNextEdit()
        {
            var entry = new PromoCodeEntry(redeemer);
            entry.Redeemed += results.Add;
            entry.Paste("abcd-2345");

            Assert.IsTrue(entry.Submit());
            Assert.IsTrue(entry.Busy);
            Assert.IsFalse(entry.CanSubmit);
            Assert.IsFalse(entry.Submit());
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("{\"code\":\"ABCD2345\"}", api.Requests[0].Body);

            api.Requests[0].Done(new HttpReply(409, "{\"error\":\"promo.expired\"}"));
            Assert.IsFalse(entry.Busy);
            Assert.AreEqual(PromoRedeemKeys.Expired, entry.MessageKey);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(PromoRedeemOutcome.Expired, results[0].Outcome);

            entry.Edit("abcd-2346");
            Assert.IsNull(entry.MessageKey);
            Assert.IsTrue(entry.Submit());
            api.Requests[1].Done(new HttpReply(200, RedemptionJson));
            Assert.AreEqual(PromoRedeemKeys.Granted, entry.MessageKey);
            Assert.AreEqual(500, results[1].Coins);
        }
    }
}
