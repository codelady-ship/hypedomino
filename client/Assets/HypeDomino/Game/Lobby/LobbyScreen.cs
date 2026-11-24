using System;
using HypeDomino.Net;

namespace HypeDomino.Game.Lobby
{
    [Flags]
    public enum LobbyChanges
    {
        None = 0,
        Header = 1,
        Play = 2,
        Countdown = 4,
        All = Header | Play | Countdown,
    }

    public enum QueueCountdown
    {
        None,
        Searching,
        SearchStartsIn,
        BanEndsIn,
    }

    public sealed class LobbyScreen
    {
        readonly MatchQueue queue;
        LobbyChanges pending = LobbyChanges.All;

        public LobbyScreen(LobbyHeader header, LobbyPlay play, MatchQueue queue)
        {
            Header = header;
            Play = play;
            this.queue = queue;
            header.Changed += OnHeaderChanged;
            play.Changed += OnPlayChanged;
        }

        public LobbyHeader Header { get; }
        public LobbyPlay Play { get; }
        public QueueCountdown Countdown { get; private set; }
        public int CountdownSeconds { get; private set; }

        public void ShowAll() => pending = LobbyChanges.All;

        public LobbyChanges Tick(ServerClock clock, double nowSeconds)
        {
            var changes = pending;
            pending = LobbyChanges.None;
            if (ShowCountdown(clock, nowSeconds)) changes |= LobbyChanges.Countdown;
            return changes;
        }

        bool ShowCountdown(ServerClock clock, double nowSeconds)
        {
            var countdown = QueueCountdown.None;
            var seconds = 0;
            if (queue.Phase == QueuePhase.Searching)
            {
                var untilSearch = queue.SecondsUntilSearch(clock, nowSeconds);
                countdown = untilSearch > 0d ? QueueCountdown.SearchStartsIn : QueueCountdown.Searching;
                seconds = untilSearch > 0d ? WholeLeft(untilSearch) : WholePassed(queue.SecondsSearching(clock, nowSeconds));
            }
            else if (queue.Phase == QueuePhase.Refused)
            {
                var untilBanEnds = queue.SecondsUntilBanEnds(clock, nowSeconds);
                countdown = untilBanEnds > 0d ? QueueCountdown.BanEndsIn : QueueCountdown.None;
                seconds = WholeLeft(untilBanEnds);
            }
            if (countdown == Countdown && seconds == CountdownSeconds) return false;
            Countdown = countdown;
            CountdownSeconds = seconds;
            return true;
        }

        static int WholeLeft(double seconds) => (int)Math.Min(int.MaxValue, Math.Ceiling(seconds));

        static int WholePassed(double seconds) => (int)Math.Min(int.MaxValue, Math.Floor(seconds));

        void OnHeaderChanged() => pending |= LobbyChanges.Header;

        void OnPlayChanged() => pending |= LobbyChanges.Play;
    }
}
