using System;
using System.Collections.Generic;
using HypeDomino.Game.Push;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class PushDeviceTests
    {
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

        static readonly HttpReply NoContent = new(204, "");

        Api api;
        ApiClient client;
        PushDevice device;

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
            device = new PushDevice(client, PushDevice.Android);
        }

        Sent Last => api.Requests[api.Requests.Count - 1];

        static string Put(string token, string platform) => "{\"token\":\"" + token + "\",\"platform\":\"" + platform + "\"}";

        [TestCase(null)]
        [TestCase("")]
        [TestCase("windows")]
        [TestCase("Android")]
        public void OnlyAndroidAndIosAreKnownPlatforms(string platform)
        {
            Assert.Throws<ArgumentException>(() => new PushDevice(null, platform));
        }

        [Test]
        public void RegisterSendsTheTokenAndPlatform()
        {
            var ios = new PushDevice(client, PushDevice.Ios);
            Assert.IsTrue(ios.Register("apns-1"));
            Assert.AreEqual("PUT", Last.Method);
            Assert.AreEqual("/push/devices", Last.PathAndQuery);
            Assert.AreEqual(Put("apns-1", "ios"), Last.Body);
        }

        [Test]
        public void ATokenCountsAsRegisteredOnlyAfterTheServerAccepts()
        {
            Assert.IsTrue(device.Register("fcm-1"));
            Assert.AreEqual(Put("fcm-1", "android"), Last.Body);
            Assert.IsNull(device.Registered);

            Last.Done(NoContent);
            Assert.AreEqual("fcm-1", device.Registered);
        }

        [TestCase(null)]
        [TestCase("")]
        public void AnEmptyTokenSendsNothing(string token)
        {
            Assert.IsFalse(device.Register(token));
            Assert.AreEqual(0, api.Requests.Count);
        }

        [Test]
        public void TheRegisteredTokenIsNotSentAgain()
        {
            device.Register("fcm-1");
            Last.Done(NoContent);
            Assert.IsFalse(device.Register("fcm-1"));
            Assert.AreEqual(1, api.Requests.Count);
        }

        [Test]
        public void AFailedRegistrationIsSentAgainOnTheNextRegister()
        {
            device.Register("fcm-1");
            Last.Done(new HttpReply(HttpReply.NoResponse, null));
            Assert.IsNull(device.Registered);
            Assert.AreEqual(1, api.Requests.Count);

            Assert.IsTrue(device.Register("fcm-1"));
            Assert.AreEqual(2, api.Requests.Count);
            Last.Done(NoContent);
            Assert.AreEqual("fcm-1", device.Registered);
        }

        [Test]
        public void ATokenRotatedInFlightIsSentAfterTheCurrentRequest()
        {
            device.Register("fcm-1");
            Assert.IsTrue(device.Register("fcm-2"));
            Assert.IsTrue(device.Register("fcm-3"));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(NoContent);
            Assert.AreEqual("fcm-1", device.Registered);
            Assert.AreEqual(2, api.Requests.Count);
            Assert.AreEqual(Put("fcm-3", "android"), Last.Body);

            Last.Done(NoContent);
            Assert.AreEqual("fcm-3", device.Registered);
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void UnregisterRemovesTheTokenAndFinishesWhateverTheServerAnswers()
        {
            device.Register("fcm-1");
            Last.Done(NoContent);
            var finished = false;
            device.Unregister(() => finished = true);
            Assert.IsNull(device.Registered);
            Assert.AreEqual("DELETE", Last.Method);
            Assert.AreEqual("/push/devices", Last.PathAndQuery);
            StringAssert.StartsWith("{\"token\":\"fcm-1\"", Last.Body);
            Assert.IsFalse(finished);

            Last.Done(new HttpReply(HttpReply.NoResponse, null));
            Assert.IsTrue(finished);
        }

        [Test]
        public void UnregisterWithoutATokenFinishesAtOnce()
        {
            var finished = false;
            device.Unregister(() => finished = true);
            Assert.IsTrue(finished);
            Assert.AreEqual(0, api.Requests.Count);
        }

        [Test]
        public void ARegistrationAnsweredAfterUnregisterIsForgotten()
        {
            device.Register("fcm-1");
            device.Unregister(() => { });
            Assert.AreEqual("DELETE", Last.Method);

            api.Requests[0].Done(NoContent);
            Assert.IsNull(device.Registered);
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void TheSameTokenRegistersAgainAfterUnregister()
        {
            device.Register("fcm-1");
            Last.Done(NoContent);
            device.Unregister(() => { });
            Last.Done(NoContent);

            Assert.IsTrue(device.Register("fcm-1"));
            Assert.AreEqual("PUT", Last.Method);
        }
    }
}
