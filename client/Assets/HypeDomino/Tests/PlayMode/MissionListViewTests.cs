using System;
using System.Collections.Generic;
using System.Linq;
using HypeDomino.Game.Missions;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Missions;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class MissionListViewTests
    {
        const float RowHeight = 100f;
        const float ViewportHeight = 1000f;
        const string BoardJson = @"{""version"":7,""day"":""2026-11-14"",""week"":""2026-W46"",""missions"":[
{""mission"":{""id"":""mission.achievement_wins_10"",""type"":""achievement"",""nameKey"":""mission.achievement_wins_10.name"",""target"":10},""period"":""once"",""progress"":12,""completed_at"":""2026-11-02T18:04:11.5Z""},
{""mission"":{""id"":""mission.daily_play_1"",""type"":""daily"",""nameKey"":""mission.daily_play_1.name"",""target"":1},""period"":""2026-11-14"",""progress"":0,""completed_at"":null},
{""mission"":{""id"":""mission.daily_points_1500"",""type"":""daily"",""nameKey"":""mission.daily_points_1500.name"",""target"":1500},""period"":""2026-11-14"",""progress"":1450,""completed_at"":null},
{""mission"":{""id"":""mission.weekly_wins_15"",""type"":""weekly"",""nameKey"":""mission.weekly_wins_15.name"",""target"":15},""period"":""2026-W46"",""progress"":15,""completed_at"":""2026-11-13T09:00:00Z""}],
""daily_rerolls"":1,""weekly_rerolls"":0}";

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
        MissionListView view;
        Api api;
        MissionLoader loader;
        LanguageChoice choice;
        GameStrings strings;

        [SetUp]
        public void BuildView()
        {
            root = new GameObject("Missions screen", typeof(RectTransform), typeof(Canvas));
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
            var slot = template.AddComponent<MissionRowSlot>();
            slot.Configure(Text("Name", template.transform), Text("Progress", template.transform), Text("Target", template.transform),
                Child("Fill", template.transform).AddComponent<Image>(), Child("Completed", template.transform));
            view = Child("View", root.transform).AddComponent<MissionListView>();
            view.Configure(list, slot);

            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            loader = new MissionLoader(new ApiClient(api, session));
            choice = LanguageChoice.Load("az", () => NoLocales);
            strings = new GameStrings(BuiltinStrings.Load(), choice, true);
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
            view.Bind(loader, strings);
            api.Replies[api.Replies.Count - 1](new HttpReply(200, json));
        }

        List<Transform> ShownRows() =>
            scroll.content.Cast<Transform>().Where(row => row.gameObject.activeSelf).OrderBy(row => -((RectTransform)row).anchoredPosition.y).ToList();

        static string TextOf(Transform row, string name) => row.Find(name).GetComponent<TMP_Text>().text;

        static float FillOf(Transform row) => row.Find("Fill").GetComponent<Image>().fillAmount;

        static bool CompletedOf(Transform row) => row.Find("Completed").gameObject.activeSelf;

        [Test]
        public void TheDailyTabShowsTheServersCardsInOrder()
        {
            Load(BoardJson);

            Assert.AreEqual(MissionBoard.BoardPath, api.Paths[0]);
            var rows = ShownRows();
            Assert.AreEqual(new[] { "[!mission.daily_play_1.name]", "[!mission.daily_points_1500.name]" }, rows.Select(row => TextOf(row, "Name")).ToArray());
            Assert.AreEqual(new[] { "0", "1.450" }, rows.Select(row => TextOf(row, "Progress")).ToArray());
            Assert.AreEqual(new[] { "1", "1.500" }, rows.Select(row => TextOf(row, "Target")).ToArray());
            Assert.AreEqual(new[] { 0f, 1450f / 1500f }, rows.Select(FillOf).ToArray());
            Assert.AreEqual(new[] { false, false }, rows.Select(CompletedOf).ToArray());
        }

        [Test]
        public void AnotherTabShowsItsOwnCardsAsCompletedByTheServer()
        {
            Load(BoardJson);

            view.ShowTab(MissionTab.Weekly);

            var rows = ShownRows();
            Assert.AreEqual(1, rows.Count);
            Assert.AreEqual("15", TextOf(rows[0], "Progress"));
            Assert.AreEqual(1f, FillOf(rows[0]));
            Assert.IsTrue(CompletedOf(rows[0]));
        }

        [Test]
        public void ACompletedCardShowsItsTargetAsProgress()
        {
            Load(BoardJson);

            view.ShowTab(MissionTab.Achievement);

            var row = ShownRows().Single();
            Assert.AreEqual("10", TextOf(row, "Progress"));
            Assert.AreEqual("10", TextOf(row, "Target"));
            Assert.IsTrue(CompletedOf(row));
        }

        [Test]
        public void NumbersFollowTheChosenLanguage()
        {
            Load(BoardJson);

            choice.Choose(GameLanguage.English);

            Assert.AreEqual("1,450", TextOf(ShownRows()[1], "Progress"));
        }

        [Test]
        public void AFailedLoadShowsNoRows()
        {
            Load(BoardJson);
            view.Bind(loader, strings);
            api.Replies[1](new HttpReply(500, "{}"));

            Assert.AreEqual(0, ShownRows().Count);
        }
    }
}
