using System.Text.RegularExpressions;

namespace HypeDomino.Core.Diagnostics
{
    public static class ErrorScrub
    {
        public const string Email = "[email]";
        public const string Id = "[id]";
        public const string Token = "[token]";
        public const string Query = "[query]";

        static readonly Regex EmailPattern = new Regex(@"[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}", RegexOptions.CultureInvariant);
        static readonly Regex UuidPattern = new Regex(@"\b[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}\b", RegexOptions.CultureInvariant);
        static readonly Regex BearerPattern = new Regex(@"\bBearer\s+\S+", RegexOptions.CultureInvariant | RegexOptions.IgnoreCase);
        static readonly Regex JwtPattern = new Regex(@"\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+", RegexOptions.CultureInvariant);
        static readonly Regex QueryPattern = new Regex(@"(\b[a-z][a-z0-9+.-]*://[^\s?#]+)\?[^\s#]*", RegexOptions.CultureInvariant | RegexOptions.IgnoreCase);

        public static string Scrub(string text)
        {
            if (string.IsNullOrEmpty(text)) return text ?? string.Empty;
            text = QueryPattern.Replace(text, "$1?" + Query);
            text = BearerPattern.Replace(text, "Bearer " + Token);
            text = JwtPattern.Replace(text, Token);
            text = EmailPattern.Replace(text, Email);
            return UuidPattern.Replace(text, Id);
        }
    }
}
