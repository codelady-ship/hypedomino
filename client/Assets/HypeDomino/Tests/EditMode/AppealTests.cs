using System;
using System.Collections.Generic;
using HypeDomino.Game.Lobby;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using Hypedomino.V1;
using NUnit.Framework;
using UnityEngine;

namespace HypeDomino.Tests.EditMode
{
    public sealed class AppealTests
    {
        const string Token = "appeal-token-1";

        sealed class Sent
        {
            public string Method;
            public string PathAndQuery;
            public string Body;
            public string Authorization;
            public string Mac;
            public Action<HttpReply> Done;
        }

        sealed class Api : IApiTransport
        {
            public readonly List<Sent> Requests = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done) =>
                Requests.Add(new Sent { Method = method, PathAndQuery = pathAndQuery, Body = body, Authorization = authorization, Mac = mac, Done = done });

            public void Answer(int status, string body) => Requests[Requests.Count - 1].Done(new HttpReply(status, body));
        }

        Api api;
        AppealSender sender;
        List<AppealResult> results;

        static readonly SanctionAppeal Granted = new(42, Token, 1_800_000_000_000);

        [SetUp]
        public void Build()
        {
            api = new Api();
            sender = new AppealSender(api);
            results = new List<AppealResult>();
        }

        [Test]
        public void TheSenderNeedsATransport()
        {
            Assert.Throws<ArgumentNullException>(() => new AppealSender(null));
        }

        [Test]
        public void AnAppealPostsTheSanctionAndTextWithTheGrantTokenAsBearer()
        {
            Assert.IsTrue(sender.Send(Granted, "I did not \"cheat\"", results.Add));

            Assert.AreEqual(1, api.Requests.Count);
            var sent = api.Requests[0];
            Assert.AreEqual("POST", sent.Method);
            Assert.AreEqual(AppealSender.Path, sent.PathAndQuery);
            Assert.AreEqual("Bearer " + Token, sent.Authorization);
            Assert.IsNull(sent.Mac);
            var body = JsonUtility.FromJson<AppealRequest>(sent.Body);
            Assert.AreEqual(42, body.sanction_id);
            Assert.AreEqual("I did not \"cheat\"", body.text);
        }

        [Test]
        public void TheTextGoesAsWrittenAndTheServerJudgesItsLength()
        {
            var longText = new string('a', 5000);

            Assert.IsTrue(sender.Send(Granted, longText, results.Add));
            Assert.AreEqual(longText, JsonUtility.FromJson<AppealRequest>(api.Requests[0].Body).text);

            api.Answer(400, "{\"error\":\"text_too_long\"}");
            Assert.AreEqual(AppealKeys.TextTooLong, results[0].MessageKey);
        }

        [Test]
        public void NoAppealIsSentWithoutAGrant()
        {
            Assert.IsFalse(sender.Send(default, "text", results.Add));
            Assert.IsFalse(sender.Send(new SanctionAppeal(42, "", 0), "text", results.Add));
            Assert.IsFalse(sender.Send(new SanctionAppeal(0, Token, 0), "text", results.Add));
            Assert.AreEqual(0, api.Requests.Count);
            Assert.Throws<ArgumentNullException>(() => sender.Send(Granted, "text", null));
        }

        [Test]
        public void OneAppealIsSentAtATime()
        {
            Assert.IsTrue(sender.Send(Granted, "text", results.Add));
            Assert.IsTrue(sender.Busy);
            Assert.IsFalse(sender.Send(Granted, "text", results.Add));

            api.Answer(201, "{\"id\":7,\"sanction_id\":42,\"state\":\"open\",\"created_at\":\"2026-11-22T10:00:00Z\"}");

            Assert.IsFalse(sender.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.IsTrue(results[0].Filed);
            Assert.AreEqual(AppealKeys.Sent, results[0].MessageKey);
            Assert.IsTrue(sender.Send(Granted, "text", results.Add));
        }

        [TestCase(401, "appeal_token_expired", "appeal.refused.expired")]
        [TestCase(401, "appeal_token_rejected", "appeal.refused.expired")]
        [TestCase(403, "sanction_mismatch", "appeal.refused.expired")]
        [TestCase(400, "text_required", "appeal.refused.text_required")]
        [TestCase(400, "text_too_long", "appeal.refused.text_too_long")]
        [TestCase(409, "appeal_open", "appeal.refused.open")]
        [TestCase(409, "appeal_rejected", "appeal.refused.rejected")]
        [TestCase(409, "sanction_ended", "appeal.refused.ended")]
        [TestCase(409, "not_appealable", "appeal.refused.not_appealable")]
        [TestCase(400, "malformed", "notice.generic")]
        [TestCase(404, "unknown_sanction", "notice.generic")]
        [TestCase(409, "a_future_refusal", "notice.generic")]
        public void EveryRefusalNamesItsText(int status, string error, string key)
        {
            sender.Send(Granted, "text", results.Add);
            api.Answer(status, "{\"error\":\"" + error + "\"}");

            Assert.AreEqual(ApiOutcome.Rejected, results[0].Outcome);
            Assert.IsFalse(results[0].Filed);
            Assert.AreEqual(status, results[0].Status);
            Assert.AreEqual(error, results[0].Error);
            Assert.AreEqual(key, results[0].MessageKey);
        }

        [TestCase(HttpReply.NoResponse, "", ApiOutcome.Offline, "appeal.offline")]
        [TestCase(429, "{\"error\":\"rate_limited\"}", ApiOutcome.RateLimited, "appeal.unavailable")]
        [TestCase(503, "", ApiOutcome.Unavailable, "appeal.unavailable")]
        [TestCase(500, "{\"error\":\"internal\"}", ApiOutcome.Unavailable, "appeal.unavailable")]
        [TestCase(400, "not json", ApiOutcome.Rejected, "notice.generic")]
        public void TransportFailuresNameTheirText(int status, string body, ApiOutcome outcome, string key)
        {
            sender.Send(Granted, "text", results.Add);
            api.Answer(status, body);

            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
        }

        [Test]
        public void AGrantIsReadFromJson()
        {
            var appeal = SanctionAppeal.Of(JsonUtility.FromJson<AppealGrantReply>("{\"sanction_id\":42,\"token\":\"t\",\"expires_at\":\"2026-11-22T10:30:00Z\"}"));

            Assert.IsTrue(appeal.Offered);
            Assert.AreEqual(42, appeal.SanctionId);
            Assert.AreEqual("t", appeal.Token);
            Assert.AreEqual((ulong)new DateTimeOffset(2026, 11, 22, 10, 30, 0, TimeSpan.Zero).ToUnixTimeMilliseconds(), appeal.ExpiresUnixMs);
        }

        [TestCase("{}")]
        [TestCase("{\"sanction_id\":0,\"token\":\"t\"}")]
        [TestCase("{\"sanction_id\":-1,\"token\":\"t\"}")]
        [TestCase("{\"sanction_id\":42,\"token\":\"\"}")]
        public void AnIncompleteJsonGrantOffersNoAppeal(string json)
        {
            Assert.IsFalse(SanctionAppeal.Of(JsonUtility.FromJson<AppealGrantReply>(json)).Offered);
            Assert.IsFalse(SanctionAppeal.Of((AppealGrantReply)null).Offered);
        }

        [Test]
        public void AGrantIsReadFromTheWire()
        {
            var appeal = SanctionAppeal.FromWire(new AppealGrant { SanctionId = 42, Token = "t", ExpiresUnixMs = 1_800_000_000_000 });

            Assert.IsTrue(appeal.Offered);
            Assert.AreEqual(42, appeal.SanctionId);
            Assert.AreEqual("t", appeal.Token);
            Assert.AreEqual(1_800_000_000_000UL, appeal.ExpiresUnixMs);
        }

        [Test]
        public void AnIncompleteWireGrantOffersNoAppeal()
        {
            Assert.IsFalse(SanctionAppeal.FromWire(null).Offered);
            Assert.IsFalse(SanctionAppeal.FromWire(new AppealGrant { Token = "t" }).Offered);
            Assert.IsFalse(SanctionAppeal.FromWire(new AppealGrant { SanctionId = 42 }).Offered);
            Assert.IsFalse(SanctionAppeal.FromWire(new AppealGrant { SanctionId = ulong.MaxValue, Token = "t" }).Offered);
        }

        [Test]
        public void EveryAppealTextIsInTheShippedBundleInAzerbaijaniAndEnglish()
        {
            var strings = BuiltinStrings.Load();
            foreach (var key in new[]
            {
                AppealKeys.Sent, AppealKeys.Expired, AppealKeys.TextRequired, AppealKeys.TextTooLong, AppealKeys.Open,
                AppealKeys.Rejected, AppealKeys.Ended, AppealKeys.NotAppealable, AppealKeys.Offline, AppealKeys.Unavailable, AppealKeys.Failed,
            })
            {
                Assert.IsNotNull(strings.Value(key, GameLanguage.English), key);
                Assert.IsNotNull(strings.Value(key, "az"), key);
            }
        }
    }
}
