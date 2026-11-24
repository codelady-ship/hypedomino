using HypeDomino.Core.Analytics;
using HypeDomino.Game.Lobby;
using HypeDomino.Net.Auth;
using Hypedomino.V1;

namespace HypeDomino.Game.Analytics
{
    public sealed class AppAnalytics
    {
        public const string RankedOneVsOne = "ranked_1v1";
        public const string RankedTwoVsTwo = "ranked_2v2";
        public const string CasualOneVsOne = "casual_1v1";
        public const string CasualTwoVsTwo = "casual_2v2";

        readonly AnalyticsQueue queue;

        public AppAnalytics(AnalyticsQueue queue)
        {
            this.queue = queue;
        }

        public bool AppOpenTracked { get; private set; }

        public void Follow(AuthSession auth, MatchQueue matches)
        {
            auth.ProviderSignedIn += SignedIn;
            matches.JoinSent += JoinSent;
        }

        public void Tick()
        {
            if (AppOpenTracked || !queue.Consented) return;
            AppOpenTracked = queue.Track(AnalyticsEvents.AppOpen);
        }

        public static string QueueLabel(QueueKind kind, Mode mode) => kind == QueueKind.Ranked
            ? mode == Mode._2V2 ? RankedTwoVsTwo : RankedOneVsOne
            : mode == Mode._2V2 ? CasualTwoVsTwo : CasualOneVsOne;

        void SignedIn(AuthProvider provider) => queue.Track(AnalyticsEvents.SignIn, AuthSession.ProviderName(provider));

        void JoinSent(Mode mode, QueueKind kind) => queue.Track(AnalyticsEvents.QueueJoin, QueueLabel(kind, mode));
    }
}
