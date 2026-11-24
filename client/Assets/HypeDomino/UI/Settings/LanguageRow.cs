using HypeDomino.UI.Localization;
using TMPro;
using UnityEngine;

namespace HypeDomino.UI.Settings
{
    public sealed class LanguageRow : MonoBehaviour
    {
        [SerializeField] TMP_Dropdown dropdown;

        LanguageChoice choice;

        public void Configure(TMP_Dropdown languages)
        {
            Unbind();
            dropdown = languages;
        }

        public void Bind(LanguageChoice languageChoice)
        {
            Unbind();
            choice = languageChoice;
            if (dropdown.options.Count != GameLanguage.LaunchCount) FillOwnNames();
            dropdown.SetValueWithoutNotify(IndexOf(choice.Language));
            dropdown.onValueChanged.AddListener(OnPicked);
        }

        void FillOwnNames()
        {
            dropdown.ClearOptions();
            for (var i = 0; i < GameLanguage.LaunchCount; i++)
                dropdown.options.Add(new TMP_Dropdown.OptionData(GameLanguage.OwnName(GameLanguage.LaunchAt(i))));
        }

        static int IndexOf(string language)
        {
            for (var i = 0; i < GameLanguage.LaunchCount; i++)
                if (GameLanguage.LaunchAt(i) == language) return i;
            return 0;
        }

        void OnPicked(int index) => choice.Choose(GameLanguage.LaunchAt(index));

        void Unbind()
        {
            if (choice == null) return;
            dropdown.onValueChanged.RemoveListener(OnPicked);
            choice = null;
        }

        void OnDestroy() => Unbind();
    }
}
