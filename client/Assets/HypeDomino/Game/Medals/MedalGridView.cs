using System;
using HypeDomino.UI.Cosmetics;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Medals;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.Game.Medals
{
    public sealed class MedalGridView : MonoBehaviour, IRowBinder
    {
        [SerializeField] RecyclingList list;
        [SerializeField] MedalRowSlot rowTemplate;

        GameStrings strings;
        CosmeticStage stage;
        MedalCollection collection;
        Action<MedalLoadResult> loaded;
        string[] names = Array.Empty<string>();

        public void Configure(RecyclingList rows, MedalRowSlot template)
        {
            Unbind();
            list = rows;
            rowTemplate = template;
        }

        public void Bind(MedalLoader loader, CosmeticStage cosmetics, GameStrings texts)
        {
            Unbind();
            strings = texts;
            stage = cosmetics;
            collection = new MedalCollection(cosmetics.Rarities);
            loaded ??= OnLoaded;
            if (list.RowCount == 0)
            {
                var template = (RectTransform)rowTemplate.transform;
                list.Build(template, template.rect.height, TokenSpacing.Sm);
            }
            strings.Changed += ShowRows;
            ShowRows();
            loader.Load(loaded);
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (strings == null) return;
            strings.Changed -= ShowRows;
            strings = null;
        }

        void OnLoaded(MedalLoadResult result)
        {
            if (strings == null) return;
            collection.Show(result.Outcome == MedalLoadOutcome.Loaded ? result.Collection : null);
            ShowRows();
        }

        void ShowRows()
        {
            var count = collection.ShownCount;
            if (names.Length < count) names = new string[count];
            for (var at = 0; at < count; at++) names[at] = strings.Text(collection.ShownAt(at).nameKey);
            var columns = rowTemplate.Columns;
            list.Show((count + columns - 1) / columns, this);
        }

        public void Bind(RectTransform row, int index)
        {
            var slot = row.GetComponent<MedalRowSlot>();
            for (var column = 0; column < slot.Columns; column++)
            {
                var at = index * slot.Columns + column;
                var cell = slot.CellAt(column);
                if (at >= collection.ShownCount)
                {
                    cell.Hide();
                    continue;
                }
                var medal = collection.ShownAt(at);
                collection.TryGet(medal.id, out var entry);
                cell.Show(stage, medal.staticAddress, names[at], entry.Earned);
            }
        }
    }
}
