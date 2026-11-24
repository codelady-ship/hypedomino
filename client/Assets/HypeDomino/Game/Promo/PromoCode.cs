using System.Text;

namespace HypeDomino.Game.Promo
{
    public static class PromoCode
    {
        public const string Alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789";
        public const int MinLength = 4;
        public const int MaxLength = 32;

        public static bool IsWellFormed(string typed)
        {
            var kept = CountKept(typed);
            return kept >= MinLength && kept <= MaxLength;
        }

        public static bool TryNormalize(string typed, out string code)
        {
            code = IsWellFormed(typed) ? Keep(typed) : null;
            return code != null;
        }

        static int CountKept(string typed)
        {
            if (typed == null)
            {
                return 0;
            }
            var count = 0;
            foreach (var character in typed)
            {
                if (IsSeparator(character))
                {
                    continue;
                }
                if (Alphabet.IndexOf(char.ToUpperInvariant(character)) < 0)
                {
                    return 0;
                }
                count++;
            }
            return count;
        }

        static string Keep(string typed)
        {
            var kept = new StringBuilder(typed.Length);
            foreach (var character in typed)
            {
                if (!IsSeparator(character))
                {
                    kept.Append(char.ToUpperInvariant(character));
                }
            }
            return kept.ToString();
        }

        static bool IsSeparator(char character) => character == ' ' || character == '-' || character == '_';
    }
}
