using System;
using System.Collections.Generic;
using System.Linq;
using HypeDomino.Game.Medals;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Medals;
using HypeDomino.UI.Motion;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class MedalGridViewTests
    {
        const int Columns = 2;
        const float RowHeight = 100f;
        const float ViewportHeight = 1000f;
        const string CollectionJson = @"{""version"":7,""slots"":3,""medals"":[
{""id"":""medal.first_win"",""nameKey"":""medal.first_win.name"",""descriptionKey"":""medal.first_win.description"",""rarityId"":""common"",""staticAddress"":""medal/first_win/static"",""sources"":[{""type"":""mission"",""ref"":""mission.first_win""}]},
{""id"":""medal.hundred_wins"",""nameKey"":""medal.hundred_wins.name"",""descriptionKey"":""medal.hundred_wins.description"",""rarityId"":""epic"",""staticAddress"":""medal/hundred_wins/static"",""sources"":[{""type"":""mission"",""ref"":""mission.hundred_wins""}]},
{""id"":""medal.season_top"",""nameKey"":""medal.season_top.name"",""descriptionKey"":""medal.season_top.description"",""rarityId"":""legendary"",""staticAddress"":""medal/season_top/static"",""sources"":[{""type"":""rank_reward"",""ref"":""rank.s1.top""}],""group"":""rank_reward""},
{""id"":""medal.domino_ten"",""nameKey"":""medal.domino_ten.name"",""descriptionKey"":""medal.domino_ten.description"",""rarityId"":""rare"",""staticAddress"":""medal/domino_ten/static"",""sources"":[{""type"":""mission"",""ref"":""mission.domino_ten""}],""group"":""achievement""},
{""id"":""medal.moon_event"",""nameKey"":""medal.moon_event.name"",""descriptionKey"":""medal.moon_event.description"",""rarityId"":""rare"",""staticAddress"":""medal/moon_event/static"",""sources"":[{""type"":""mission"",""ref"":""mission.moon_event""}],""group"":""event""}],
""owned"":[
{""medal"":""medal.first_win"",""earned_at"":""2026-11-01T10:00:00Z""},
{""medal"":""medal.moon_event"",""earned_at"":""2026-11-02T10:00:00Z""}],
""showcase"":[]}";

        sealed class Api : IApiTransport
        {
            public readonly List<Action<HttpReply>> Replies = new();
            public readonly List<string> Paths = new();

            public void Send(string method, string pathAndQuery, string body, string authorization, string mac, Action<HttpReply> done)
            {
                Paths.Add(pathAndQuery);
                Replies.Add(done);
            }
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

        static readonly string[] NoLocales = { };

        GameObject root;
        ScrollRect scroll;
        MedalGridView view;
        Api api;
        MedalLoader loader;
        FakeArtSource<Sprite> stills;
        CosmeticStage stage;
        GameStrings strings;

        [SetUp]
        public void BuildView()
        {
            root = new GameObject("Medals screen", typeof(RectTransform), typeof(Canvas));
            var holder = new GameObject("List", typeof(RectTransform), typeof(ScrollRect));
            holder.transform.SetParent(root.transform, false);
            ((RectTransform)holder.transform).sizeDelta = new Vector2(600f, ViewportHeight);
            var viewport = Stretched("Viewport", holder.transform);
            viewport.gameObject.AddComponent<RectMask2D>();
            var content = Stretched("Content", viewport);
            scroll = holder.GetComponent<ScrollRect>();
            scroll.viewport = viewport;
            scroll.content = content;
            scroll.horizontal = false;
            var list = holder.AddComponent<RecyclingList>();

            var template = new GameObject("Row", typeof(RectTransform));
            template.transform.SetParent(root.transform, false);
            ((RectTransform)template.transform).sizeDelta = new Vector2(600f, RowHeight);
            var cells = new MedalCellSlot[Columns];
            for (var column = 0; column < Columns; column++)
            {
                var cell = Child("Cell " + column, template.transform);
                var icon = new GameObject("Icon", typeof(RectTransform), typeof(Image)).AddComponent<AnimatedImage>().gameObject.AddComponent<CosmeticImage>();
                icon.transform.SetParent(cell.transform, false);
                cells[column] = cell.AddComponent<MedalCellSlot>();
                cells[column].Configure(icon, Text("Name", cell.transform), Child("Locked", cell.transform));
            }
            var row = template.AddComponent<MedalRowSlot>();
            row.Configure(cells);
            view = Child("View", root.transform).AddComponent<MedalGridView>();
            view.Configure(list, row);

            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            loader = new MedalLoader(new ApiClient(api, session));
            stills = new FakeArtSource<Sprite>();
            stage = new CosmeticStage(stills, new FakeArtSource<LoopArt>(), null, null, null, EffectLevel.Medium, false);
            strings = new GameStrings(BuiltinStrings.Load(), LanguageChoice.Load("az", () => NoLocales), true);
        }

        [TearDown]
        public void DestroyView() => UnityEngine.Object.Destroy(root);

        static GameObject Child(string name, Transform parent)
        {
            var child = new GameObject(name, typeof(RectTransform));
            child.transform.SetParent(parent, false);
            return child;
        }

        static TMP_Text Text(string name, Transform parent) => Child(name, parent).AddComponent<TextMeshProUGUI>();

        static RectTransform Stretched(string name, Transform parent)
        {
            var rect = new GameObject(name, typeof(RectTransform)).GetComponent<RectTransform>();
            rect.SetParent(parent, false);
            rect.anchorMin = Vector2.zero;
            rect.anchorMax = Vector2.one;
            rect.sizeDelta = Vector2.zero;
            return rect;
        }

        void Load(string json)
        {
            view.Bind(loader, stage, strings);
            api.Replies[api.Replies.Count - 1](new HttpReply(200, json));
        }

        List<Transform> ShownCells() =>
            scroll.content.Cast<Transform>().Where(row => row.gameObject.activeSelf).OrderBy(row => -((RectTransform)row).anchoredPosition.y)
                .SelectMany(row => row.Cast<Transform>()).Where(cell => cell.gameObject.activeSelf).ToList();

        [Test]
        public void TheGridShowsTheServersMedalsRowByRowInGridOrder()
        {
            Load(CollectionJson);

            Assert.AreEqual(MedalCollection.CollectionPath, api.Paths[0]);
            Assert.AreEqual(new[] { "[!medal.domino_ten.name]", "[!medal.moon_event.name]", "[!medal.season_top.name]", "[!medal.first_win.name]", "[!medal.hundred_wins.name]" },
                ShownCells().Select(cell => cell.Find("Name").GetComponent<TMP_Text>().text).ToArray());
        }

        [Test]
        public void UnearnedMedalsShowTheLockedLook()
        {
            Load(CollectionJson);

            Assert.AreEqual(new[] { true, false, true, false, true }, ShownCells().Select(cell => cell.Find("Locked").gameObject.activeSelf).ToArray());
        }

        [Test]
        public void EachCellShowsTheStillOfItsMedal()
        {
            Load(CollectionJson);
            var sprite = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);

            stills.Finish("medal/moon_event/static", sprite);

            CollectionAssert.AreEquivalent(new[] { "medal/domino_ten/static", "medal/moon_event/static", "medal/season_top/static", "medal/first_win/static", "medal/hundred_wins/static" },
                stills.Loads);
            Assert.AreSame(sprite, ShownCells()[1].Find("Icon").GetComponent<Image>().sprite);
        }

        [Test]
        public void AFailedLoadShowsNoMedals()
        {
            Load(CollectionJson);
            view.Bind(loader, stage, strings);
            api.Replies[1](new HttpReply(500, "{}"));

            Assert.AreEqual(0, ShownCells().Count);
        }
    }
}
