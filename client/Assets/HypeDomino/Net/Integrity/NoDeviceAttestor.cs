using System;
using Hypedomino.V1;

namespace HypeDomino.Net.Integrity
{
    public sealed class NoDeviceAttestor : IDeviceAttestor
    {
        public bool CanAttest => false;

        public void Attest(string nonce, Action<IntegrityProof> done) => done(null);
    }
}
