using System;
using System.Collections.Generic;
using HypeDomino.Game.Ranking;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using HypeDomino.UI.Localization;
using NUnit.Framework;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Tests.PlayMode
{
    public sealed class RankScreenViewTests
    {
        const string StandingJson = @"{""season"":{""id"":""season.2026_11"",""name_key"":""currency.coin.name"",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":true,""placement_played"":5,""placement_matches"":5,""rank"":""Gold II"",""tier"":""Gold"",""lp"":42,""division_lp"":100,""wins"":9},{""mode"":""2v2"",""placed"":false,""placement_played"":2,""placement_matches"":5,""rank"":"""",""tier"":"""",""lp"":0,""division_lp"":100,""wins"":1}]}";
        const string ApexJson = @"{""season"":{""id"":""season.2026_11"",""name_key"":"""",""starts_at"":""2026-11-01T00:00:00Z"",""ends_at"":""2027-02-01T00:00:00Z""},""modes"":[{""mode"":""1v1"",""placed"":true,""placement_played"":5,""placement_matches"":5,""rank"":""Grandmaster"",""tier"":""Grandmaster"",""lp"":3400,""division_lp"":100,""wins"":80}]}";
        const ulong SeasonEndUnixMs = 1801440000000;

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
        GameObject panel;
        Image badge;
        Sprite[] tierBadges;
        GameObject lpBar;
        Image lpFill;
        TMP_Text lp;
        GameObject placement;
        Image placementFill;
        TMP_Text seasonName;
        TMP_Text seasonLeft;
        RankScreenView view;
        Api api;
        RankStandingLoader loader;
        LanguageChoice choice;
        GameStrings strings;
        ServerClock clock;

        [SetUp]
        public void BuildView()
        {
            root = new GameObject("Rank screen", typeof(RectTransform), typeof(Canvas));
            panel = Child("Standing", root.transform);
            badge = Child("Badge", panel.transform).AddComponent<Image>();
            lpBar = Child("LP bar", panel.transform);
            lpFill = Child("LP fill", lpBar.transform).AddComponent<Image>();
            lp = Child("LP", panel.transform).AddComponent<TextMeshProUGUI>();
            placement = Child("Placement", panel.transform);
            placementFill = Child("Placement fill", placement.transform).AddComponent<Image>();
            seasonName = Child("Season", panel.transform).AddComponent<TextMeshProUGUI>();
            seasonLeft = Child("Season left", panel.transform).AddComponent<TextMeshProUGUI>();
            tierBadges = new Sprite[(int)RankTier.Challenger + 1];
            for (var tier = 1; tier < tierBadges.Length; tier++) tierBadges[tier] = Sprite.Create(Texture2D.whiteTexture, new Rect(0, 0, 4, 4), Vector2.zero);
            view = Child("View", root.transform).AddComponent<RankScreenView>();
            view.Configure(panel, badge, tierBadges, lpBar, lpFill, lp, placement, placementFill, seasonName, seasonLeft);

            api = new Api();
            var auth = new Auth();
            var store = new Store();
            var session = new AuthSession(auth, new Identity(), store);
            store.Write(AuthSession.RefreshTokenKey, "refresh-0");
            session.Resume(_ => { });
            auth.Refreshes[0](new HttpReply(200, "{\"access_token\":\"access-1\",\"refresh_token\":\"refresh-1\",\"expires_in\":900,\"request_key\":\"" + Convert.ToBase64String(new byte[32]) + "\",\"account_id\":\"acc-1\"}"));
            loader = new RankStandingLoader(new ApiClient(api, session));
            choice = LanguageChoice.Load("az", () => NoLocales);
            strings = new GameStrings(BuiltinStrings.Load(), choice, false);
            clock = new ServerClock();
        }

        [TearDown]
        public void DestroyView() => UnityEngine.Object.Destroy(root);

        static GameObject Child(string name, Transform parent)
        {
            var child = new GameObject(name, typeof(RectTransform));
            child.transform.SetParent(parent, false);
            return child;
        }

        void Load(string mode, string json)
        {
            view.Bind(loader, mode, clock, strings);
            api.Replies[0](new HttpReply(200, json));
        }

        void SyncClockAt(TimeSpan beforeSeasonEnd)
        {
            var now = Time.realtimeSinceStartupAsDouble;
            clock.Sync(SeasonEndUnixMs - (ulong)beforeSeasonEnd.TotalMilliseconds, now, now);
        }

        [Test]
        public void NothingShowsUntilTheServerAnswers()
        {
            view.Bind(loader, "1v1", clock, strings);

            Assert.AreEqual(1, api.Paths.Count);
            Assert.AreEqual(RankStanding.Path, api.Paths[0]);
            Assert.IsFalse(panel.activeSelf);
        }

        [Test]
        public void APlacedRankShowsItsBadgeAndLpBar()
        {
            Load("1v1", StandingJson);

            Assert.IsTrue(panel.activeSelf);
            Assert.IsTrue(badge.enabled);
            Assert.AreSame(tierBadges[(int)RankTier.Gold], badge.sprite);
            Assert.IsTrue(lpBar.activeSelf);
            Assert.AreEqual(0.42f, lpFill.fillAmount, 0.0001f);
            Assert.IsTrue(lp.enabled);
            Assert.AreEqual("42", lp.text);
            Assert.IsFalse(placement.activeSelf);
            Assert.AreEqual("Sikkə", seasonName.text);
        }

        [Test]
        public void AnApexRankShowsLpWithoutABar()
        {
            Load("1v1", ApexJson);

            Assert.AreSame(tierBadges[(int)RankTier.Grandmaster], badge.sprite);
            Assert.IsFalse(lpBar.activeSelf);
            Assert.AreEqual("3.400", lp.text);
            Assert.IsFalse(seasonName.enabled);
        }

        [Test]
        public void AnUnplacedModeShowsPlacementProgressOnly()
        {
            Load("2v2", StandingJson);

            Assert.IsTrue(placement.activeSelf);
            Assert.AreEqual(0.4f, placementFill.fillAmount, 0.0001f);
            Assert.IsFalse(badge.enabled);
            Assert.IsFalse(lpBar.activeSelf);
            Assert.IsFalse(lp.enabled);
        }

        [Test]
        public void TheSeasonCountdownUsesServerTimeAndFollowsTheLanguage()
        {
            SyncClockAt(new TimeSpan(2, 3, 30, 0));
            Load("1v1", StandingJson);

            Assert.IsTrue(seasonLeft.enabled);
            Assert.AreEqual("2 gün 3 saat", seasonLeft.text);

            choice.Choose(GameLanguage.English);

            Assert.AreEqual("2 days 3 hours", seasonLeft.text);
        }

        [Test]
        public void WithoutServerTimeTheCountdownStaysHidden()
        {
            Load("1v1", StandingJson);

            Assert.IsFalse(seasonLeft.enabled);
        }

        [Test]
        public void AModeWithoutStandingOrAFailedLoadShowsNothing()
        {
            Load("3v3", StandingJson);
            Assert.IsFalse(panel.activeSelf);

            view.Bind(loader, "1v1", clock, strings);
            api.Replies[1](new HttpReply(500, "{}"));
            Assert.IsFalse(panel.activeSelf);
        }
    }
}
