using HypeDomino.Core.Diagnostics;
using HypeDomino.Net;

namespace HypeDomino.Game
{
    public sealed class NetStartPhases
    {
        public const string Resume = "resume";
        public const string Proof = "proof";
        public const string Dial = "dial";

        readonly StartupPhases phases;
        ISocketTransport dialing;
        bool resumed;
        bool proved;
        bool dialed;

        public NetStartPhases(StartupPhases phases) => this.phases = phases;

        public void Resumed()
        {
            if (resumed) return;
            resumed = true;
            phases.Mark(Resume);
        }

        public void Proved()
        {
            if (proved) return;
            proved = true;
            phases.Mark(Proof);
        }

        public ISocketTransport Dialing(ISocketTransport socket)
        {
            if (!dialed && !phases.Closed) dialing = socket;
            return socket;
        }

        public void Tick()
        {
            if (dialing == null) return;
            if (phases.Closed)
            {
                dialing = null;
                return;
            }
            if (dialing.State != TransportState.Open) return;
            dialing = null;
            dialed = true;
            phases.Mark(Dial);
        }
    }
}
