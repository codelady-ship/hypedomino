using System;
using System.Collections.Generic;

namespace HypeDomino.UI.Localization
{
    public sealed class LanguageChoice
    {
        LanguageChoice(string language)
        {
            Language = language;
        }

        public event Action Changed;

        public string Language { get; private set; }

        public static LanguageChoice Load(string saved, Func<IReadOnlyList<string>> readDeviceLocales) =>
            new(GameLanguage.IsLaunch(saved) ? saved : GameLanguage.Pick(null, readDeviceLocales()));

        public bool Choose(string language)
        {
            if (!GameLanguage.IsLaunch(language)) return false;
            if (Language == language) return true;
            Language = language;
            Changed?.Invoke();
            return true;
        }
    }
}
