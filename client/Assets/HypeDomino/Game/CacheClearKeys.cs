using HypeDomino.Content;

namespace HypeDomino.Game
{
    public static class CacheClearKeys
    {
        public const string Cleared = "settings.cache.cleared";
        public const string AtNextStart = "settings.cache.cleared_at_next_start";
        public const string NotInMatch = "settings.cache.not_in_match";

        public static string Of(CacheClearResult result) => result switch
        {
            CacheClearResult.AtNextStart => AtNextStart,
            CacheClearResult.NotInMatch => NotInMatch,
            _ => Cleared,
        };
    }
}
