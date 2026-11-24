using System;
using Hypedomino.V1;

namespace HypeDomino.Net.Integrity
{
    public sealed class SessionStartProof
    {
        readonly Action<IntegrityPurpose, Action<IntegrityProof>> requestProof;
        readonly Action dial;
        readonly Action<IntegrityProof> arrived;
        IntegrityProof proof;
        bool dialWithoutProof;

        public SessionStartProof(Action<IntegrityPurpose, Action<IntegrityProof>> requestProof, Action dial)
        {
            this.requestProof = requestProof;
            this.dial = dial;
            arrived = Arrived;
        }

        public bool Proving { get; private set; }
        public bool LastHelloProved { get; private set; }

        public bool Start() => Prove(true);

        public void Attach(ClientHello hello)
        {
            hello.Integrity = proof;
            LastHelloProved = proof != null;
        }

        public void Read(ServerHello hello)
        {
            if (hello.Status == HelloStatus.Accepted) proof = null;
        }

        public void Read(Error error)
        {
            if (error.Code == ErrorCode.IntegrityFailed && !LastHelloProved) Prove(false);
        }

        bool Prove(bool withoutProof)
        {
            if (Proving) return false;
            Proving = true;
            dialWithoutProof = withoutProof;
            proof = null;
            requestProof(IntegrityPurpose.SessionStart, arrived);
            return true;
        }

        void Arrived(IntegrityProof next)
        {
            Proving = false;
            proof = next;
            if (next != null || dialWithoutProof) dial();
        }
    }
}
