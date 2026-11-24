using System;
using System.Collections.Generic;
using HypeDomino.Net;
using HypeDomino.UI.Localization;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace HypeDomino.Game.Ranking
{
    public sealed class RankScreenView : MonoBehaviour
    {
        const string DaysArgument = "days";
        const string HoursArgument = "hours";
        const string MinutesArgument = "minutes";
        const string SecondsArgument = "seconds";
        const long NoCountdown = -1;
        const long Unshown = long.MinValue;

        [SerializeField] GameObject standingPanel;
        [SerializeField] Image badge;
        [SerializeField] Sprite[] tierBadges;
        [SerializeField] GameObject lpBar;
        [SerializeField] Image lpFill;
        [SerializeField] TMP_Text lp;
        [SerializeField] GameObject placement;
        [SerializeField] Image placementFill;
        [SerializeField] TMP_Text seasonName;
        [SerializeField] TMP_Text seasonLeft;

        readonly Dictionary<string, object> durationArgs = new();
        ServerClock clock;
        GameStrings strings;
        Action<RankStandingLoadResult> loaded;
        RankStanding standing;
        RankModeStanding modeStanding;
        string mode;
        bool shown;
        long shownSecondsLeft = Unshown;

        public void Configure(GameObject standingRoot, Image badgeImage, Sprite[] tierBadgeSprites, GameObject lpBarRoot, Image lpFillImage, TMP_Text lpText,
            GameObject placementRoot, Image placementFillImage, TMP_Text seasonNameText, TMP_Text seasonLeftText)
        {
            Unbind();
            standingPanel = standingRoot;
            badge = badgeImage;
            tierBadges = tierBadgeSprites;
            lpBar = lpBarRoot;
            lpFill = lpFillImage;
            lp = lpText;
            placement = placementRoot;
            placementFill = placementFillImage;
            seasonName = seasonNameText;
            seasonLeft = seasonLeftText;
        }

        public void Bind(RankStandingLoader loader, string rankedMode, ServerClock serverClock, GameStrings texts)
        {
            Unbind();
            mode = rankedMode;
            clock = serverClock;
            strings = texts;
            standing = null;
            loaded ??= OnLoaded;
            strings.Changed += ShowTexts;
            Show();
            loader.Load(loaded);
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (strings == null) return;
            strings.Changed -= ShowTexts;
            clock = null;
            strings = null;
        }

        void OnLoaded(RankStandingLoadResult result)
        {
            if (strings == null) return;
            standing = result.Outcome == RankStandingLoadOutcome.Loaded ? result.Standing : null;
            Show();
        }

        void Update()
        {
            if (shown) ShowSecondsLeft();
        }

        void Show()
        {
            shown = standing != null && standing.TryFind(mode, out modeStanding);
            standingPanel.SetActive(shown);
            if (!shown) return;
            RankBadgeArt.Show(badge, tierBadges, modeStanding.Badge);
            lpBar.SetActive(modeStanding.HasLpBar);
            lpFill.fillAmount = modeStanding.LpFill;
            placement.SetActive(!modeStanding.Placed);
            placementFill.fillAmount = (float)modeStanding.PlacementPlayed / modeStanding.PlacementMatches;
            ShowTexts();
        }

        void ShowTexts()
        {
            if (!shown) return;
            lp.enabled = modeStanding.Placed;
            lp.text = MessageFormat.Number(modeStanding.LP, strings.Language);
            var named = standing.SeasonNameKey.Length > 0;
            seasonName.enabled = named;
            seasonName.text = named ? strings.Text(standing.SeasonNameKey) : string.Empty;
            shownSecondsLeft = Unshown;
            ShowSecondsLeft();
        }

        void ShowSecondsLeft()
        {
            var known = standing.TrySecondsLeft(clock, Time.realtimeSinceStartupAsDouble, out var seconds);
            var whole = known ? (long)seconds : NoCountdown;
            if (whole == shownSecondsLeft) return;
            shownSecondsLeft = whole;
            seasonLeft.enabled = known;
            if (!known) return;
            var left = TimeSpan.FromSeconds(whole);
            durationArgs[DaysArgument] = (long)left.Days;
            durationArgs[HoursArgument] = (long)left.Hours;
            durationArgs[MinutesArgument] = (long)left.Minutes;
            durationArgs[SecondsArgument] = (long)left.Seconds;
            seasonLeft.text = strings.Text(MessageFormat.DurationKey(left), durationArgs);
        }
    }
}
