using HypeDomino.Game.Privacy;
using HypeDomino.Net;
using HypeDomino.Net.Api;
using HypeDomino.Net.Auth;
using Hypedomino.V1;

namespace HypeDomino.Game
{
    public static class PrivacyBoot
    {
        static AuthSession auth;

        public static ConsentGates Gates { get; private set; }

        public static ConsentStartup Consents { get; private set; }

        public static void Start(ApiClient api, AuthSession session, Session net)
        {
            auth = session;
            Gates = new ConsentGates(AnalyticsBoot.Queue, CrashReportingBoot.Reports, PrivacySettings.RandomAnalyticsId);
            Consents = new ConsentStartup(api, Gates);
            net.HelloReceived += SignedIn;
        }

        static void SignedIn(ServerHello hello)
        {
            if (hello.Status == HelloStatus.Accepted) Consents.SignedIn(auth.AccountId);
        }
    }
}
