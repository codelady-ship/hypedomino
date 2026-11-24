using System;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.Game.Ranking
{
    public sealed class LeaderboardView : MonoBehaviour, IRowBinder
    {
        [SerializeField] RecyclingList list;
        [SerializeField] LeaderboardRowSlot rowTemplate;
        [SerializeField] Sprite[] tierBadges;

        GameStrings strings;
        Action<LeaderboardLoadResult> loaded;
        LeaderboardStanding standing;
        string[] positions = Array.Empty<string>();
        string[] lps = Array.Empty<string>();
        int aroundStart;
        int rowCount;

        public void Configure(RecyclingList rows, LeaderboardRowSlot template, Sprite[] tierBadgeSprites)
        {
            Unbind();
            list = rows;
            rowTemplate = template;
            tierBadges = tierBadgeSprites;
        }

        public void Bind(LeaderboardLoader loader, string mode, string board, GameStrings texts)
        {
            Unbind();
            strings = texts;
            standing = null;
            loaded ??= OnLoaded;
            if (list.RowCount == 0)
            {
                var template = (RectTransform)rowTemplate.transform;
                list.Build(template, template.rect.height, TokenSpacing.Sm);
            }
            strings.Changed += ShowRows;
            ShowRows();
            loader.Load(mode, board, loaded);
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (strings == null) return;
            strings.Changed -= ShowRows;
            strings = null;
        }

        void OnLoaded(LeaderboardLoadResult result)
        {
            if (strings == null) return;
            standing = result.Outcome == LeaderboardLoadOutcome.Loaded ? result.Standing : null;
            ShowRows();
        }

        void ShowRows()
        {
            aroundStart = standing == null ? 0 : FirstAroundBelowTop(standing);
            rowCount = standing == null ? 0 : standing.TopCount + standing.AroundCount - aroundStart;
            if (positions.Length < rowCount)
            {
                positions = new string[rowCount];
                lps = new string[rowCount];
            }
            for (var index = 0; index < rowCount; index++)
            {
                var row = RowAt(index);
                positions[index] = MessageFormat.Number(row.Position, strings.Language);
                lps[index] = MessageFormat.Number(row.LP, strings.Language);
            }
            list.Show(rowCount, this);
        }

        public void Bind(RectTransform row, int index)
        {
            var shown = RowAt(index);
            var own = standing.Listed && string.Equals(shown.AccountId, standing.Me.AccountId, StringComparison.Ordinal);
            row.GetComponent<LeaderboardRowSlot>().Show(positions[index], shown.Nickname, shown.Badge, tierBadges, lps[index], own);
        }

        LeaderboardRow RowAt(int index) =>
            index < standing.TopCount ? standing.TopAt(index) : standing.AroundAt(aroundStart + index - standing.TopCount);

        static int FirstAroundBelowTop(LeaderboardStanding board)
        {
            if (board.TopCount == 0) return 0;
            var lastTop = board.TopAt(board.TopCount - 1).Position;
            var first = 0;
            while (first < board.AroundCount && board.AroundAt(first).Position <= lastTop) first++;
            return first;
        }
    }
}
