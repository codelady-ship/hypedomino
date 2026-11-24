using System.Collections.Generic;
using System.Text;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;
using UnityEngine.Networking;

namespace HypeDomino.Tests.EditMode
{
    public sealed class HttpExchangeTests
    {
        static List<string> HeaderLines(HttpRequestPlan plan)
        {
            var lines = new List<string>();
            foreach (var header in plan.Headers) lines.Add(header.Name + ": " + header.Value);
            return lines;
        }

        [TestCase("https://api.example", "/v1/me", "https://api.example/v1/me")]
        [TestCase("https://api.example/", "/v1/me", "https://api.example/v1/me")]
        [TestCase("https://api.example//", "/v1/me?x=1", "https://api.example/v1/me?x=1")]
        public void UrlJoinsBaseWithoutTrailingSlashAndPath(string baseUrl, string path, string expected)
        {
            var plan = HttpExchange.Plan(baseUrl, "GET", path, null, null, null, null);

            Assert.AreEqual(expected, plan.Url);
            Assert.AreEqual("GET", plan.Method);
        }

        [Test]
        public void RequestWithoutBodyOrCredentialsHasNoBodyAndNoHeaders()
        {
            var plan = HttpExchange.Plan("https://api.example", "GET", "/v1/me", null, null, null, null);

            Assert.IsNull(plan.Body);
            Assert.IsEmpty(plan.Headers);
        }

        [Test]
        public void BodyIsUtf8EncodedJsonWithContentType()
        {
            var plan = HttpExchange.Plan("https://api.example", "POST", "/v1/nick", "{\"n\":\"Ləman\"}", null, null, null);

            CollectionAssert.AreEqual(Encoding.UTF8.GetBytes("{\"n\":\"Ləman\"}"), plan.Body);
            CollectionAssert.AreEqual(new[] { "Content-Type: application/json" }, HeaderLines(plan));
        }

        [Test]
        public void HeadersFollowContentTypeAuthorizationMacThenExtrasInOrder()
        {
            var extras = new[] { new ApiHeader("X-HD-Play-Integrity", "token"), new ApiHeader("X-HD-Attest-Key", "key") };

            var plan = HttpExchange.Plan("https://api.example", "POST", "/shop/complete", "{}", "Bearer abc", "mac-1", extras);

            CollectionAssert.AreEqual(new[]
            {
                "Content-Type: application/json",
                "Authorization: Bearer abc",
                RequestMac.Header + ": mac-1",
                "X-HD-Play-Integrity: token",
                "X-HD-Attest-Key: key",
            }, HeaderLines(plan));
        }

        [Test]
        public void EmptyBodyStillSendsJsonContentType()
        {
            var plan = HttpExchange.Plan("https://api.example", "POST", "/v1/ping", "", null, null, null);

            Assert.IsEmpty(plan.Body);
            CollectionAssert.AreEqual(new[] { "Content-Type: application/json" }, HeaderLines(plan));
        }

        [TestCase(UnityWebRequest.Result.Success, 200, "{\"ok\":true}")]
        [TestCase(UnityWebRequest.Result.ProtocolError, 409, "{\"error\":\"conflict\"}")]
        [TestCase(UnityWebRequest.Result.ProtocolError, 500, "")]
        public void AnsweredResultKeepsStatusAndBody(UnityWebRequest.Result result, long code, string text)
        {
            var reply = HttpExchange.Reply(result, code, text);

            Assert.AreEqual((int)code, reply.Status);
            Assert.AreEqual(text, reply.Body);
        }

        [TestCase(UnityWebRequest.Result.ConnectionError)]
        [TestCase(UnityWebRequest.Result.DataProcessingError)]
        [TestCase(UnityWebRequest.Result.InProgress)]
        public void UnansweredResultMapsToNoResponseWithoutBody(UnityWebRequest.Result result)
        {
            var reply = HttpExchange.Reply(result, 0, "partial");

            Assert.AreEqual(HttpReply.NoResponse, reply.Status);
            Assert.IsNull(reply.Body);
        }
    }
}
