using System;
using Hypedomino.V1;

namespace HypeDomino.Net.Integrity
{
    public interface IDeviceAttestor
    {
        bool CanAttest { get; }

        void Attest(string nonce, Action<IntegrityProof> done);
    }
}
