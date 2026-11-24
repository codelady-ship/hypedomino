using System;
using HypeDomino.Net.Auth;

namespace HypeDomino.Game
{
    public sealed class NetStart
    {
        readonly AuthSession auth;
        readonly AppFlow flow;
        readonly Action dial;
        readonly Action<AuthOutcome> stalled;

        public NetStart(AuthSession auth, AppFlow flow, Action dial, Action<AuthOutcome> stalled)
        {
            this.auth = auth;
            this.flow = flow;
            this.dial = dial;
            this.stalled = stalled;
        }

        public void Begin()
        {
            if (!auth.HasStoredSession)
            {
                flow.SignInNeeded();
                return;
            }
            auth.Resume(Resumed);
        }

        public void Unauthenticated() => auth.Resume(Resumed);

        void Resumed(AuthResult result)
        {
            switch (result.Outcome)
            {
                case AuthOutcome.SignedIn:
                    dial();
                    return;
                case AuthOutcome.SignInRequired:
                    flow.SignInNeeded();
                    return;
                default:
                    stalled(result.Outcome);
                    return;
            }
        }
    }
}
