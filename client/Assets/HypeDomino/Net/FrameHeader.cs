namespace HypeDomino.Net
{
    public readonly struct FrameHeader
    {
        public readonly uint ProtocolVersion;
        public readonly ulong Sequence;
        public readonly ulong AckedSequence;
        public readonly ulong SessionNonce;

        public FrameHeader(uint protocolVersion, ulong sequence, ulong ackedSequence, ulong sessionNonce)
        {
            ProtocolVersion = protocolVersion;
            Sequence = sequence;
            AckedSequence = ackedSequence;
            SessionNonce = sessionNonce;
        }
    }
}
