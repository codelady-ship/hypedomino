using System;
using UnityEngine;

namespace HypeDomino.Game.Ranking
{
    [Serializable]
    public sealed class LeaderboardRowReply
    {
        public int position;
        public string account_id;
        public string nickname;
        public string rank;
        public int lp;
        public string tier;
        public int wins;
    }

    [Serializable]
    public sealed class LeaderboardReply
    {
        public string season;
        public string mode;
        public string board;
        public LeaderboardRowReply[] top;
        public LeaderboardRowReply me;
        public LeaderboardRowReply[] around;

        public static LeaderboardReply FromJson(string json) => JsonUtility.FromJson<LeaderboardReply>(json);
    }

    public readonly struct LeaderboardRow
    {
        public LeaderboardRow(int position, string accountId, string nickname, RankBadge badge, int lp, int wins)
        {
            Position = position;
            AccountId = accountId;
            Nickname = nickname;
            Badge = badge;
            LP = lp;
            Wins = wins;
        }

        public int Position { get; }

        public string AccountId { get; }

        public string Nickname { get; }

        public RankBadge Badge { get; }

        public int LP { get; }

        public int Wins { get; }
    }

    public sealed class LeaderboardStanding
    {
        public const string BoardGlobal = "global";
        public const string BoardFriends = "friends";

        const string PathPrefix = "/leaderboard/";

        readonly LeaderboardRow[] top;
        readonly LeaderboardRow[] around;
        readonly LeaderboardRow me;

        LeaderboardStanding(string season, string mode, string board, LeaderboardRow[] top, bool listed, LeaderboardRow me, LeaderboardRow[] around)
        {
            Season = season;
            Mode = mode;
            Board = board;
            this.top = top;
            Listed = listed;
            this.me = me;
            this.around = around;
        }

        public string Season { get; }

        public string Mode { get; }

        public string Board { get; }

        public int TopCount => top.Length;

        public LeaderboardRow TopAt(int index) => top[index];

        public bool Listed { get; }

        public LeaderboardRow Me => me;

        public int AroundCount => around.Length;

        public LeaderboardRow AroundAt(int index) => around[index];

        public static string PathOf(string mode, string board) =>
            PathPrefix + Uri.EscapeDataString(mode) + "?board=" + Uri.EscapeDataString(board);

        public static bool TryRead(string json, out LeaderboardStanding standing)
        {
            standing = null;
            if (!TryParse(json, out var reply) || string.IsNullOrEmpty(reply.season) || string.IsNullOrEmpty(reply.mode) || string.IsNullOrEmpty(reply.board))
            {
                return false;
            }
            if (!TryRows(reply.top, out var top) || !TryRows(reply.around, out var around))
            {
                return false;
            }
            var listed = !string.IsNullOrEmpty(reply.me?.account_id);
            var me = default(LeaderboardRow);
            if (listed && !TryRow(reply.me, out me))
            {
                return false;
            }
            standing = new LeaderboardStanding(reply.season, reply.mode, reply.board, top, listed, me, around);
            return true;
        }

        static bool TryParse(string json, out LeaderboardReply reply)
        {
            reply = null;
            if (string.IsNullOrEmpty(json))
            {
                return false;
            }
            try
            {
                reply = LeaderboardReply.FromJson(json);
            }
            catch (ArgumentException)
            {
                return false;
            }
            return reply != null;
        }

        static bool TryRows(LeaderboardRowReply[] replies, out LeaderboardRow[] rows)
        {
            rows = Array.Empty<LeaderboardRow>();
            if (replies == null || replies.Length == 0)
            {
                return true;
            }
            var read = new LeaderboardRow[replies.Length];
            for (var i = 0; i < replies.Length; i++)
            {
                if (!TryRow(replies[i], out read[i]))
                {
                    return false;
                }
            }
            rows = read;
            return true;
        }

        static bool TryRow(LeaderboardRowReply reply, out LeaderboardRow row)
        {
            row = default;
            if (reply == null || string.IsNullOrEmpty(reply.account_id) || reply.position < 1 || reply.lp < 0 || reply.wins < 0 || !TryBadge(reply.rank, reply.tier, out var badge))
            {
                return false;
            }
            row = new LeaderboardRow(reply.position, reply.account_id, reply.nickname ?? string.Empty, badge, reply.lp, reply.wins);
            return true;
        }

        static bool TryBadge(string rank, string tier, out RankBadge badge)
        {
            if (!RankBadge.TryRead(rank, out badge) || !badge.Placed)
            {
                return false;
            }
            if (!badge.Apex)
            {
                return true;
            }
            return RankBadge.TryRead(tier, out badge) && badge.Apex;
        }
    }
}
