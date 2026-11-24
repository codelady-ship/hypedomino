using System;
using HypeDomino.Net;
using HypeDomino.UI.Board;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Match;
using HypeDomino.UI.Tokens;
using TMPro;
using UnityEngine;
using UnityEngine.Events;
using UnityEngine.UI;

namespace HypeDomino.Game.Match
{
    public sealed class MatchHudView : MonoBehaviour
    {
        static readonly Color RingColor = TokenColor.GoldDefault;
        static readonly Color RingWarningColor = TokenColor.RoleWarningFill;

        [SerializeField] TileFaceSet faceSet;
        [SerializeField] Button[] handTiles = Array.Empty<Button>();
        [SerializeField] GameObject[] legalMarks = Array.Empty<GameObject>();
        [SerializeField] Button draw;
        [SerializeField] Button pass;
        [SerializeField] Button endTurn;
        [SerializeField] Image turnRing;
        [SerializeField] TMP_Text network;
        [SerializeField] TMP_Text afkWarning;
        [SerializeField] Button takeBack;
        [SerializeField] TMP_Text takeBackText;
        [SerializeField] TMP_Text[] seatBadges = new TMP_Text[TableSides.Count];
        [SerializeField] CanvasGroup[] passLabels = new CanvasGroup[TableSides.Count];

        UnityAction[] tileTaps = Array.Empty<UnityAction>();
        MatchHost host;
        Connection connection;
        GameStrings strings;
        int handVersion = -1;
        int noticesVersion = -1;
        int passVersion = -1;

        MatchScreen LiveScreen => host?.Root?.Screen;

        public void Configure(TileFaceSet faces, Button[] tiles, GameObject[] marks, Button drawButton, Button passButton, Button endTurnButton, Image ring,
            TMP_Text networkText, TMP_Text afkText, Button takeBackButton, TMP_Text takeBackLabel, TMP_Text[] badges, CanvasGroup[] passGroups)
        {
            Unbind();
            faceSet = faces;
            handTiles = tiles ?? Array.Empty<Button>();
            legalMarks = marks ?? Array.Empty<GameObject>();
            if (legalMarks.Length != handTiles.Length) throw new ArgumentException($"{legalMarks.Length} marks for {handTiles.Length} tiles", nameof(marks));
            draw = drawButton;
            pass = passButton;
            endTurn = endTurnButton;
            turnRing = ring;
            network = networkText;
            afkWarning = afkText;
            takeBack = takeBackButton;
            takeBackText = takeBackLabel;
            seatBadges = badges;
            passLabels = passGroups;
        }

        public void Bind(MatchHost matchHost, Connection matchConnection, GameStrings texts)
        {
            Unbind();
            host = matchHost ?? throw new ArgumentNullException(nameof(matchHost));
            connection = matchConnection ?? throw new ArgumentNullException(nameof(matchConnection));
            strings = texts ?? throw new ArgumentNullException(nameof(texts));
            strings.Changed += RedrawTexts;
            tileTaps = new UnityAction[handTiles.Length];
            for (var index = 0; index < handTiles.Length; index++)
            {
                var slot = index;
                tileTaps[index] = () => LiveScreen?.TapTile(slot);
                handTiles[index].onClick.AddListener(tileTaps[index]);
            }
            draw.onClick.AddListener(Draw);
            pass.onClick.AddListener(Pass);
            endTurn.onClick.AddListener(EndTurn);
            takeBack.onClick.AddListener(TakeBack);
            handVersion = noticesVersion = passVersion = -1;
            Refresh();
        }

        public void Unbind()
        {
            if (host == null) return;
            strings.Changed -= RedrawTexts;
            for (var index = 0; index < tileTaps.Length; index++) handTiles[index].onClick.RemoveListener(tileTaps[index]);
            tileTaps = Array.Empty<UnityAction>();
            draw.onClick.RemoveListener(Draw);
            pass.onClick.RemoveListener(Pass);
            endTurn.onClick.RemoveListener(EndTurn);
            takeBack.onClick.RemoveListener(TakeBack);
            host = null;
            connection = null;
            strings = null;
        }

        public void Refresh()
        {
            if (host == null) return;
            var hud = host.Hud;
            if (hud.HandVersion != handVersion)
            {
                handVersion = hud.HandVersion;
                ShowHand(hud.Hand);
            }
            ShowTimer(hud.Timer);
            if (hud.NoticesVersion != noticesVersion)
            {
                noticesVersion = hud.NoticesVersion;
                ShowNotices(hud.Notices);
            }
            if (hud.PassVersion != passVersion)
            {
                passVersion = hud.PassVersion;
                ShowPasses(hud);
            }
        }

        void LateUpdate() => Refresh();

        void OnDestroy() => Unbind();

        void RedrawTexts() => noticesVersion = -1;

        void Draw() => LiveScreen?.Draw();

        void Pass() => LiveScreen?.Pass();

        void EndTurn() => LiveScreen?.EndTurn();

        void TakeBack() => LiveScreen?.TapTakeBack(connection);

        void ShowHand(HandChoice hand)
        {
            var count = hand?.HandCount ?? 0;
            for (var index = 0; index < handTiles.Length; index++)
            {
                var shown = index < count;
                SetActive(handTiles[index].gameObject, shown);
                var selectable = shown && hand.IsSelectable(index);
                SetActive(legalMarks[index], selectable);
                handTiles[index].interactable = selectable;
                if (!shown) continue;
                var tile = hand.TileAt(index);
                handTiles[index].image.sprite = faceSet.Face(new Domino((byte)tile.Low, (byte)tile.High));
            }
            SetActive(draw.gameObject, hand != null && hand.CanDraw);
            SetActive(pass.gameObject, hand != null && hand.CanPass);
            SetActive(endTurn.gameObject, hand != null && hand.CanEndTurn);
        }

        void ShowTimer(TurnTimer timer)
        {
            var running = timer != null && timer.Running;
            SetActive(turnRing.gameObject, running);
            if (!running) return;
            turnRing.fillAmount = timer.Fill;
            turnRing.color = timer.IsWarning ? RingWarningColor : RingColor;
        }

        void ShowNotices(MatchNotices notices)
        {
            ShowKey(network, notices == null ? null : ConnectionNoticeKeys.Of(notices.Network));
            ShowKey(afkWarning, notices != null && notices.AfkWarning ? ConnectionNoticeKeys.OneTimeoutFromBot : null);
            var banner = notices != null && notices.TakeBack.Shown;
            SetActive(takeBack.gameObject, banner);
            if (banner) takeBackText.text = strings.Text(ConnectionNoticeKeys.TakeBack);
            for (var side = 0; side < seatBadges.Length; side++)
                ShowKey(seatBadges[side], notices == null ? null : ConnectionNoticeKeys.Of(notices.Seats.At((TableSide)side)));
        }

        void ShowPasses(MatchHud hud)
        {
            for (var side = 0; side < passLabels.Length; side++)
            {
                var label = passLabels[side];
                if (label == null) continue;
                var shown = hud.TryGetPass((TableSide)side, out var notice);
                SetActive(label.gameObject, shown);
                if (!shown) continue;
                label.alpha = notice.Alpha;
                label.transform.localScale = new Vector3(notice.Scale, notice.Scale, 1f);
            }
        }

        void ShowKey(TMP_Text text, string key)
        {
            if (text == null) return;
            text.enabled = key != null;
            if (key != null) text.text = strings.Text(key);
        }

        static void SetActive(GameObject target, bool active)
        {
            if (target.activeSelf != active) target.SetActive(active);
        }
    }
}
