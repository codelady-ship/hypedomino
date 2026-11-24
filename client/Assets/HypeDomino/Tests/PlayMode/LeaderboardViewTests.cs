using System;
using System.Collections.Generic;
using System.Linq;
using HypeDomino.Game.Ranking;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Localization;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class LeaderboardViewTests
    {
        const float RowHeight = 100f;
        const float ViewportHeight = 1000f;
        const string BoardJson = @"{""season"":""season.s1"",""mode"":""1v1"",""board"":""global"",""top"":[{""position"":1,""account_id"":""a"",""nickname"":""Ayla"",""rank"":""Grandmaster"",""tier"":""Grandmaster"",""lp"":1200,""wins"":90},{""position"":2,""account_id"":""b"",""nickname"":"""",""rank"":""Diamond I"",""tier"":""Diamond"",""lp"":80,""wins"":70}],""me"":{""position"":12345,""account_id"":""acc-1"",""nickname"":""Me"",""rank"":""Gold II"",""tier"":""Gold"",""lp"":42,""wins"":9},""around"":[{""position"":12344,""account_id"":""c"",""nickname"":""Cem"",""rank"":""Gold II"",""tier"":""Gold"",""lp"":43,""wins"":9},{""position"":12345,""account_id"":""acc-1"",""nickname"":""Me"",""rank"":""Gold II"",""tier"":""Gold"",""lp"":42,""wins"":9}]}";
        const string OverlapJson = @"{""season"":""season.s1"",""mode"":""1v1"",""board"":""friends"",""top"":[{""position"":1,""account_id"":""acc-1"",""nickname"":""Me"",""rank"":""Gold II"",""tier"":""Gold"",""lp"":42,""wins"":9},{""position"":2,""account_id"":""c"",""nickname"":""Cem"",""rank"":""Gold III"",""tier"":""Gold"",""lp"":10,""wins"":3}],""me"":{""position"":1,""account_id"":""acc-1"",""nickname"":""Me"",""rank"":""Gold II"",""tier"":""Gold"",""lp"":42,""wins"":9},""around"":[{""position"":1,""account_id"":""acc-1"",""nickname"":""Me"",""rank"":""Gold II"",""tier"":""Gold"",""lp"":42,""wins"":9},{""position"":2,""account_id"":""c"",""nickname"":""Cem"",""rank"":""Gold III"",""tier"":""Gold"",""lp"":10,""wins"":3}]}";

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
        Sprite[] tierBadges;
        LeaderboardView view;
        Api api;
        LeaderboardLoader loader;
        LanguageChoice choice;
        GameStrings strings;

        [SetUp]
        public void BuildView()
        {
            root = new GameObject("Leaderboard screen", typeof(RectTransform), typeof(Canvas));
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
            var slot = template.AddComponent<LeaderboardRowSlot>();
            slot.Configure(Text("Position", template.transform), Text("Nickname", template.transform), Child("Badge", template.transform).AddComponent<Image>(),
                Text("LP", template.transform), Child("Own", template.transform));
            tierBadges = new Sprite[(int)RankTier.Challenger + 1];
            for (var tier = 1; tier < tierBadges.Length; tier++) tierBadges[tier] = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);
            view = Child("View", root.transform).AddComponent<LeaderboardView>();
            view.Configure(list, slot, tierBadges);

            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            loader = new LeaderboardLoader(new ApiClient(api, session));
            choice = LanguageChoice.Load("az", () => NoLocales);
            strings = new GameStrings(BuiltinStrings.Load(), choice, false);
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

        void Load(string board, string json)
        {
            view.Bind(loader, "1v1", board, strings);
            api.Replies[api.Replies.Count - 1](new HttpReply(200, json));
        }

        List<Transform> ShownRows() =>
            scroll.content.Cast<Transform>().Where(row => row.gameObject.activeSelf).OrderBy(row => -((RectTransform)row).anchoredPosition.y).ToList();

        static string TextOf(Transform row, string name) => row.Find(name).GetComponent<TMP_Text>().text;

        [Test]
        public void TheBoardShowsTheTopThenTheViewersNeighbours()
        {
            Load(LeaderboardStanding.BoardGlobal, BoardJson);

            Assert.AreEqual("/leaderboard/1v1?board=global", api.Paths[0]);
            var rows = ShownRows();
            Assert.AreEqual(new[] { "1", "2", "12.344", "12.345" }, rows.Select(row => TextOf(row, "Position")).ToArray());
            Assert.AreEqual(new[] { "Ayla", "", "Cem", "Me" }, rows.Select(row => TextOf(row, "Nickname")).ToArray());
            Assert.AreEqual(new[] { "1.200", "80", "43", "42" }, rows.Select(row => TextOf(row, "LP")).ToArray());
            Assert.AreEqual(new[] { false, false, false, true }, rows.Select(row => row.Find("Own").gameObject.activeSelf).ToArray());
        }

        [Test]
        public void EachRowShowsTheBadgeOfItsRankOrApexTier()
        {
            Load(LeaderboardStanding.BoardGlobal, BoardJson);

            var badges = ShownRows().Select(row => row.Find("Badge").GetComponent<Image>().sprite).ToArray();
            Assert.AreSame(tierBadges[(int)RankTier.Grandmaster], badges[0]);
            Assert.AreSame(tierBadges[(int)RankTier.Diamond], badges[1]);
            Assert.AreSame(tierBadges[(int)RankTier.Gold], badges[3]);
        }

        [Test]
        public void NeighboursInsideTheTopAreNotRepeated()
        {
            Load(LeaderboardStanding.BoardFriends, OverlapJson);

            var rows = ShownRows();
            Assert.AreEqual(new[] { "1", "2" }, rows.Select(row => TextOf(row, "Position")).ToArray());
            Assert.IsTrue(rows[0].Find("Own").gameObject.activeSelf);
        }

        [Test]
        public void NumbersFollowTheChosenLanguage()
        {
            Load(LeaderboardStanding.BoardGlobal, BoardJson);

            choice.Choose(GameLanguage.English);

            Assert.AreEqual("12,345", TextOf(ShownRows()[3], "Position"));
        }

        [Test]
        public void AFailedLoadShowsNoRows()
        {
            Load(LeaderboardStanding.BoardGlobal, BoardJson);
            view.Bind(loader, "1v1", LeaderboardStanding.BoardGlobal, strings);
            api.Replies[1](new HttpReply(500, "{}"));

            Assert.AreEqual(0, ShownRows().Count);
        }
    }
}
