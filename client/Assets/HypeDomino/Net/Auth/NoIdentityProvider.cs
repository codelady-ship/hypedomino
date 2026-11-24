using System;

namespace HypeDomino.Net.Auth
{
    public sealed class NoIdentityProvider : IIdentityProvider
    {
        public bool Supports(AuthProvider provider) => false;

        public void RequestIdToken(AuthProvider provider, string nonce, Action<IdTokenResult> done) => done(new IdTokenResult(IdTokenOutcome.Failed, null));
    }
}
