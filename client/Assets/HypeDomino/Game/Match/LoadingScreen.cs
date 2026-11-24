using System;
using HypeDomino.Net;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Match;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.Game.Match
{
    public sealed class LoadingScreen : ILoadingScreen
    {
        public const string ReadyKey = "match.loading.ready";
        public const string BotPlayingKey = "match.loading.bot_playing";

        readonly LoadingScreenView view;
        readonly CosmeticStage stage;
        readonly ActiveCosmeticManifest manifest;
        readonly GameStrings strings;
        readonly bool[] shown;
        readonly TableBackground table;

        public LoadingScreen(LoadingScreenView view, CosmeticStage stage, ActiveCosmeticManifest manifest, GameStrings strings)
        {
            this.view = view ? view : throw new ArgumentNullException(nameof(view));
            this.stage = stage ?? throw new ArgumentNullException(nameof(stage));
            this.manifest = manifest ?? throw new ArgumentNullException(nameof(manifest));
            this.strings = strings ?? throw new ArgumentNullException(nameof(strings));
            shown = new bool[view.CardCount];
            table = new TableBackground(stage, view.Table);
        }

        public void ShowCards(MatchSnapshot match, LoadingCards cards)
        {
            view.SetOpen(true);
            table.Show(match, manifest.Current);
            Array.Clear(shown, 0, shown.Length);
            Span<int> placed = stackalloc int[LoadingScreenView.Rows];
            for (var at = 0; at < cards.Count; at++)
            {
                var card = cards[at];
                if (card.Row >= LoadingScreenView.Rows || placed[card.Row] >= LoadingScreenView.CardsPerRow) continue;
                var place = placed[card.Row]++;
                shown[card.Row * LoadingScreenView.CardsPerRow + place] = true;
                Show(view.Card(card.Row, place), match, card);
            }
            for (var index = 0; index < shown.Length; index++)
                if (!shown[index]) view.Card(index).SetVisible(false);
            view.ShowTotal(cards.Progress);
        }

        public void Close() => view.SetOpen(false);

        public void RefreshEffects()
        {
            table.RefreshEffects();
            for (var index = 0; index < shown.Length; index++)
                if (shown[index]) view.Card(index).RefreshEffects();
        }

        void Show(LoadingCardView card, MatchSnapshot match, in LoadingCard loading)
        {
            card.SetVisible(true);
            if (loading.RosterIndex >= 0)
            {
                var entry = match.Roster[loading.RosterIndex];
                card.ShowPlayer(stage, AvatarLook.Of(manifest.Current, entry.Avatar, entry.Frame), entry.Nickname);
            }
            card.ShowProgress(loading.Progress / (float)LoadingCards.FullProgress, StatusText(loading.Status), StatusColor(loading.Status));
        }

        string StatusText(LoadingStatus status) => status switch
        {
            LoadingStatus.Ready => strings.Text(ReadyKey),
            LoadingStatus.BotPlaying => strings.Text(BotPlayingKey),
            _ => null,
        };

        static Color32 StatusColor(LoadingStatus status) =>
            status == LoadingStatus.BotPlaying ? TokenColor.RoleWarningText : TokenColor.RoleSuccessText;
    }
}
