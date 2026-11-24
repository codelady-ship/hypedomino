namespace HypeDomino.UI.Localization
{
    public static class PluralRules
    {
        public const string One = "one";
        public const string Few = "few";
        public const string Many = "many";
        public const string Other = "other";

        public static string Category(string language, long number)
        {
            var n = number < 0 ? -number : number;
            if (language == "ru") return Russian(n);
            return n == 1 ? One : Other;
        }

        static string Russian(long n)
        {
            var last = n % 10;
            var lastTwo = n % 100;
            if (last == 1 && lastTwo != 11) return One;
            if (last >= 2 && last <= 4 && (lastTwo < 12 || lastTwo > 14)) return Few;
            return Many;
        }
    }
}
