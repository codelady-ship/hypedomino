using System;
using System.Collections.Generic;
using HypeDomino.UI.Lists;
using HypeDomino.UI.Localization;
using HypeDomino.UI.Missions;
using HypeDomino.UI.Tokens;
using UnityEngine;

namespace HypeDomino.Game.Missions
{
    public sealed class MissionListView : MonoBehaviour, IRowBinder
    {
        const string TargetArgument = "target";

        [SerializeField] RecyclingList list;
        [SerializeField] MissionRowSlot rowTemplate;

        readonly Dictionary<string, object> nameArgs = new();
        GameStrings strings;
        Action<MissionLoadResult> loaded;
        MissionBoard board;
        MissionTab tab;
        string[] names = Array.Empty<string>();
        string[] progresses = Array.Empty<string>();
        string[] targets = Array.Empty<string>();

        public void Configure(RecyclingList rows, MissionRowSlot template)
        {
            Unbind();
            list = rows;
            rowTemplate = template;
        }

        public void Bind(MissionLoader loader, GameStrings texts)
        {
            Unbind();
            strings = texts;
            board = null;
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

        public void ShowTab(MissionTab shown)
        {
            tab = shown;
            if (strings != null) ShowRows();
        }

        void OnDestroy() => Unbind();

        void Unbind()
        {
            if (strings == null) return;
            strings.Changed -= ShowRows;
            strings = null;
        }

        void OnLoaded(MissionLoadResult result)
        {
            if (strings == null) return;
            board = result.Outcome == MissionLoadOutcome.Loaded ? result.Board : null;
            ShowRows();
        }

        void ShowRows()
        {
            var count = board == null ? 0 : board.Count(tab);
            if (names.Length < count)
            {
                names = new string[count];
                progresses = new string[count];
                targets = new string[count];
            }
            for (var index = 0; index < count; index++)
            {
                var card = board.Card(tab, index);
                nameArgs[TargetArgument] = card.Target;
                names[index] = strings.Text(card.NameKey, nameArgs);
                progresses[index] = MessageFormat.Number(card.ShownProgress, strings.Language);
                targets[index] = MessageFormat.Number(card.Target, strings.Language);
            }
            list.Show(count, this);
        }

        public void Bind(RectTransform row, int index)
        {
            var card = board.Card(tab, index);
            row.GetComponent<MissionRowSlot>().Show(names[index], progresses[index], targets[index], card.Fill, card.Completed);
        }
    }
}
