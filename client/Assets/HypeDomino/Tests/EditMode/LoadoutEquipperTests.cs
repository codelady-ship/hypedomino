using System;
using System.Collections.Generic;
using HypeDomino.Game.Inventory;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Localization;
using NUnit.Framework;

namespace HypeDomino.Tests.EditMode
{
    public sealed class LoadoutEquipperTests
    {
        const string ManifestJson = @"{""schemaVersion"":1,""manifestVersion"":5,""items"":[
{""id"":""avatar.starter"",""type"":""avatar"",""rarityId"":""common"",""availability"":""live"",""staticAddress"":""cosmetic/avatar/starter/static"",""sources"":[{""type"":""default""}]},
{""id"":""avatar.moon_owl"",""type"":""avatar"",""rarityId"":""rare"",""availability"":""retired"",""staticAddress"":""cosmetic/avatar/moon_owl/static"",""sources"":[{""type"":""event"",""ref"":""event.moon""}]}]}";

        const string InventoryJson = @"{""items"":[
{""id"":""avatar.starter"",""type"":""avatar"",""source_type"":""default""},
{""id"":""avatar.moon_owl"",""type"":""avatar"",""source_type"":""event"",""source_ref"":""event.moon"",""granted_at"":""2026-11-01T10:00:00Z""}],
""loadout"":{""avatar"":""avatar.moon_owl""},""emoji_slots"":6}";

        static readonly string[] ShownKeys =
        {
            EquipKeys.NotOwned,
            EquipKeys.Unavailable,
            EquipKeys.SlotsFull,
            EquipKeys.Offline,
            EquipKeys.RateLimited,
            InventoryLoadKeys.Offline,
            InventoryLoadKeys.RateLimited,
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
        LoadoutEquipper equipper;
        CosmeticCollection avatars;
        List<EquipResult> results;

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
            equipper = new LoadoutEquipper(new ApiClient(api, session));
            Assert.IsTrue(CosmeticManifest.TryParse(ManifestJson, out var manifest));
            avatars = new CosmeticCollection(manifest);
            avatars.Show(InventoryReply.FromJson(InventoryJson));
            avatars.Filter(CosmeticAddresses.Avatar, null);
            results = new List<EquipResult>();
        }

        EquipRequest WearStarter()
        {
            Assert.IsTrue(avatars.TryEquip(0, out var request));
            return request;
        }

        [Test]
        public void EquipSendsTheIntentAndTheLoadoutChangesOnlyFromTheServersReply()
        {
            Assert.IsTrue(equipper.Equip(WearStarter(), results.Add));
            Assert.IsTrue(equipper.Busy);
            Assert.AreEqual(1, api.Requests.Count);
            Assert.AreEqual("PUT", api.Requests[0].Method);
            Assert.AreEqual("/inventory/equip", api.Requests[0].PathAndQuery);
            Assert.AreEqual(@"{""slot"":""avatar"",""items"":[""avatar.starter""]}", api.Requests[0].Body);
            Assert.IsEmpty(results);
            Assert.IsFalse(avatars[0].Equipped);
            Assert.IsTrue(avatars[1].Equipped);

            api.Requests[0].Done(new HttpReply(200, @"{""loadout"":{""avatar"":""avatar.starter""}}"));
            Assert.IsFalse(equipper.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(EquipOutcome.Equipped, results[0].Outcome);
            Assert.IsNull(results[0].MessageKey);
            Assert.IsFalse(results[0].InventoryStale);

            avatars.Apply(results[0].Equipped);
            Assert.IsTrue(avatars[0].Equipped);
            Assert.IsFalse(avatars[1].Equipped);
        }

        [Test]
        public void ASecondEquipWhileOneIsInFlightSendsNothing()
        {
            var request = WearStarter();
            Assert.IsTrue(equipper.Equip(request, results.Add));
            Assert.IsFalse(equipper.Equip(request, results.Add));
            Assert.AreEqual(1, api.Requests.Count);

            api.Requests[0].Done(new HttpReply(403, "{\"error\":\"item.not_owned\"}"));
            Assert.IsTrue(equipper.Equip(request, results.Add));
            Assert.AreEqual(2, api.Requests.Count);
        }

        [Test]
        public void NothingIsSentWithoutAnEquipRequest()
        {
            Assert.IsFalse(equipper.Equip(null, results.Add));
            Assert.IsFalse(equipper.Equip(new EquipRequest(), results.Add));
            Assert.IsFalse(equipper.Equip(new EquipRequest { slot = CosmeticAddresses.Avatar }, results.Add));
            Assert.IsFalse(equipper.Equip(new EquipRequest { items = new[] { "avatar.starter" } }, results.Add));

            Assert.IsFalse(equipper.Busy);
            Assert.IsEmpty(api.Requests);
            Assert.IsEmpty(results);
        }

        [TestCase(HttpReply.NoResponse, null, EquipOutcome.Offline, "inventory.equip.offline", false)]
        [TestCase(429, "{\"error\":\"rate_limited\"}", EquipOutcome.RateLimited, "inventory.equip.rate_limited", false)]
        [TestCase(401, "{\"error\":\"access_rejected\"}", EquipOutcome.SignInRequired, null, false)]
        [TestCase(403, "{\"error\":\"item.not_owned\"}", EquipOutcome.NotOwned, "inventory.equip.not_owned", true)]
        [TestCase(409, "{\"error\":\"item.disabled\"}", EquipOutcome.Unavailable, "inventory.equip.unavailable", true)]
        [TestCase(404, "{\"error\":\"item.unknown\"}", EquipOutcome.Unavailable, "inventory.equip.unavailable", true)]
        [TestCase(400, "{\"error\":\"slot.too_many\"}", EquipOutcome.SlotsFull, "inventory.equip.slots_full", false)]
        [TestCase(409, "{\"error\":\"profile.setup_required\"}", EquipOutcome.ProfileRequired, null, false)]
        [TestCase(409, "{\"error\":\"item.wrong_type\"}", EquipOutcome.Failed, "notice.generic", false)]
        [TestCase(400, "{\"error\":\"slot.unknown\"}", EquipOutcome.Failed, "notice.generic", false)]
        [TestCase(400, "{\"error\":\"malformed\"}", EquipOutcome.Failed, "notice.generic", false)]
        [TestCase(500, "{\"error\":\"internal\"}", EquipOutcome.Failed, "notice.generic", false)]
        [TestCase(200, "", EquipOutcome.Failed, "notice.generic", false)]
        [TestCase(200, "not json", EquipOutcome.Failed, "notice.generic", false)]
        public void ARefusedEquipLeavesTheLoadoutAsTheServerLastSentIt(int status, string body, EquipOutcome outcome, string key, bool stale)
        {
            equipper.Equip(WearStarter(), results.Add);
            api.Requests[0].Done(new HttpReply(status, body));

            Assert.IsFalse(equipper.Busy);
            Assert.AreEqual(1, results.Count);
            Assert.AreEqual(outcome, results[0].Outcome);
            Assert.AreEqual(key, results[0].MessageKey);
            Assert.AreEqual(stale, results[0].InventoryStale);
            Assert.IsNull(results[0].Equipped);

            avatars.Apply(results[0].Equipped);
            Assert.IsFalse(avatars[0].Equipped);
            Assert.IsTrue(avatars[1].Equipped);
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
