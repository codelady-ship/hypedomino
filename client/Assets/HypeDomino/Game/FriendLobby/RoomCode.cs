using System;

namespace HypeDomino.Game.FriendLobby
{
    public static class RoomCode
    {
        public const string Alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
        public const int Length = 6;

        public static bool TryNormalize(string typed, out string code)
        {
            code = null;
            if (typed == null)
            {
                return false;
            }
            Span<char> kept = stackalloc char[Length];
            var count = 0;
            foreach (var character in typed)
            {
                if (character == ' ')
                {
                    continue;
                }
                var upper = char.ToUpperInvariant(character);
                if (count == Length || Alphabet.IndexOf(upper) < 0)
                {
                    return false;
                }
                kept[count++] = upper;
            }
            if (count != Length)
            {
                return false;
            }
            code = new string(kept);
            return true;
        }
    }
}
